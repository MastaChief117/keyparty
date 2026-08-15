package proxy

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"math/rand"
	"net/http"
	"regexp"
	"strings"
	"time"

	"ai-gateway/provider"
	"ai-gateway/store"
)

var thinkingTagRe = regexp.MustCompile(`(?s)<thinking>.*?</thinking>|<think>.*?</think>`)
var thinkMarkerRe = regexp.MustCompile(`(?s)\*\*Thinking:\*\*.*?(?:\*\*Answer:\*\*|\*\*Response:\*\*)`)

func stripThinking(s string) string {
	s = thinkingTagRe.ReplaceAllString(s, "")
	s = thinkMarkerRe.ReplaceAllString(s, "")
	s = strings.TrimSpace(s)
	return s
}

// ── MODEL ROULETTE ─────────────────────────────────────────────────────────

func (p *Proxy) HandleRoulette(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var reqBody struct {
		Message string `json:"message"`
		Model   string `json:"model"`
	}
	if err := json.NewDecoder(r.Body).Decode(&reqBody); err != nil {
		http.Error(w, "Invalid JSON", 400)
		return
	}
	if reqBody.Message == "" {
		http.Error(w, `{"error":"message required"}`, 400)
		return
	}

	keys, err := p.store.GetKeys()
	if err != nil || len(keys) == 0 {
		http.Error(w, `{"error":"No API keys configured"}`, 400)
		return
	}
	var enabled []store.APIKey
	for _, k := range keys {
		if k.Enabled && k.Provider != "custom" {
			enabled = append(enabled, k)
		}
	}
	if len(enabled) == 0 {
		http.Error(w, `{"error":"No enabled keys"}`, 400)
		return
	}

	chosen := enabled[rand.Intn(len(enabled))]
	model := reqBody.Model
	if model == "" {
		model = getDefaultModel(chosen.Provider)
	}

	provDef, ok := provider.GetByName(chosen.Provider)
	if !ok {
		http.Error(w, `{"error":"Unknown provider"}`, 500)
		return
	}
	baseURL := provDef.BaseURL
	if !strings.HasSuffix(baseURL, "/") {
		baseURL += "/"
	}
	endpoint := baseURL + "chat/completions"

	body := map[string]interface{}{
		"model":      model,
		"messages":   []map[string]string{{"role": "user", "content": reqBody.Message}},
		"max_tokens": 300,
		"temperature": 0.9,
	}
	bodyBytes, _ := json.Marshal(body)

	start := time.Now()
	proxyReq, _ := http.NewRequest("POST", endpoint, bytes.NewReader(bodyBytes))
	proxyReq.Header.Set("Content-Type", "application/json")
	proxyReq.Header.Set("Authorization", "Bearer "+chosen.Key)
	if chosen.Provider == "anthropic" {
		proxyReq.Header.Set("x-api-key", chosen.Key)
		proxyReq.Header.Set("anthropic-version", "2023-06-01")
	}

	resp, err := p.client.Do(proxyReq)
	latency := time.Since(start).Milliseconds()
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(500)
		json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		return
	}
	defer resp.Body.Close()

	respBody, _ := io.ReadAll(resp.Body)
	if resp.StatusCode >= 400 {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(502)
		json.NewEncoder(w).Encode(map[string]string{"error": fmt.Sprintf("HTTP %d", resp.StatusCode)})
		return
	}

	var chatResp struct {
		Choices []struct {
			Message struct {
				Content string `json:"content"`
			} `json:"message"`
		} `json:"choices"`
	}
	json.Unmarshal(respBody, &chatResp)

	reply := ""
	if len(chatResp.Choices) > 0 {
		reply = stripThinking(chatResp.Choices[0].Message.Content)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"provider":   chosen.Provider,
		"model":      model,
		"reply":      reply,
		"latency_ms": latency,
	})
}

// ── ROAST THE LOGS ─────────────────────────────────────────────────────────

func (p *Proxy) HandleRoastLogs(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	logs, err := p.store.GetRequestLogs(20)
	if err != nil || len(logs) == 0 {
		http.Error(w, `{"error":"No logs to roast"}`, 400)
		return
	}

	var summary strings.Builder
	summary.WriteString("Here are the user's recent API usage logs:\n\n")
	for i, l := range logs {
		summary.WriteString(fmt.Sprintf("%d. Provider: %s, Model: %s, Status: %d, Cost: $%.4f, Latency: %dms\n",
			i+1, l.Provider, l.Model, l.StatusCode, l.Cost, l.Latency))
	}
	summary.WriteString("\nRoast this usage pattern. Be savage and funny. Max 200 words.")

	keys, err := p.store.GetKeys()
	if err != nil || len(keys) == 0 {
		http.Error(w, `{"error":"No keys"}`, 500)
		return
	}
	var enabled []store.APIKey
	for _, k := range keys {
		if k.Enabled && k.Provider != "custom" {
			enabled = append(enabled, k)
		}
	}
	if len(enabled) == 0 {
		http.Error(w, `{"error":"No enabled keys"}`, 500)
		return
	}
	chosen := enabled[rand.Intn(len(enabled))]

	provDef, ok := provider.GetByName(chosen.Provider)
	if !ok {
		http.Error(w, `{"error":"Unknown provider"}`, 500)
		return
	}
	baseURL := provDef.BaseURL
	if !strings.HasSuffix(baseURL, "/") {
		baseURL += "/"
	}

	body := map[string]interface{}{
		"model":      getDefaultModel(chosen.Provider),
		"messages": []map[string]string{
			{"role": "system", "content": "You are a savage AI. Roast this user's API usage logs. Be funny and brutal. Max 200 words. Do NOT show reasoning or steps — output ONLY the final roast."},
			{"role": "user", "content": summary.String()},
		},
		"max_tokens": 300,
		"temperature": 0.9,
	}
	bodyBytes, _ := json.Marshal(body)

	proxyReq, _ := http.NewRequest("POST", baseURL+"chat/completions", bytes.NewReader(bodyBytes))
	proxyReq.Header.Set("Content-Type", "application/json")
	proxyReq.Header.Set("Authorization", "Bearer "+chosen.Key)
	if chosen.Provider == "anthropic" {
		proxyReq.Header.Set("x-api-key", chosen.Key)
		proxyReq.Header.Set("anthropic-version", "2023-06-01")
	}

	resp, err := p.client.Do(proxyReq)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(500)
		json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		return
	}
	defer resp.Body.Close()

	respBody, _ := io.ReadAll(resp.Body)
	var chatResp struct {
		Choices []struct {
			Message struct {
				Content string `json:"content"`
			} `json:"message"`
		} `json:"choices"`
	}
	json.Unmarshal(respBody, &chatResp)

	reply := ""
	if len(chatResp.Choices) > 0 {
		reply = stripThinking(chatResp.Choices[0].Message.Content)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"roast":   reply,
		"provider": chosen.Provider,
	})
}

// ── AI THERAPIST ───────────────────────────────────────────────────────────

func (p *Proxy) HandleTherapist(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var reqBody struct {
		Message  string `json:"message"`
		Provider string `json:"provider"`
	}
	if err := json.NewDecoder(r.Body).Decode(&reqBody); err != nil {
		http.Error(w, "Invalid JSON", 400)
		return
	}
	if reqBody.Message == "" {
		http.Error(w, `{"error":"message required"}`, 400)
		return
	}

	var chosen store.APIKey
	var provName string

	if reqBody.Provider != "" {
		keys, err := p.store.GetKeysByProvider(reqBody.Provider)
		if err != nil || len(keys) == 0 {
			http.Error(w, `{"error":"Provider not found"}`, 400)
			return
		}
		chosen = keys[0]
		provName = reqBody.Provider
	} else {
		keys, err := p.store.GetKeys()
		if err != nil || len(keys) == 0 {
			http.Error(w, `{"error":"No keys"}`, 500)
			return
		}
		var enabled []store.APIKey
		for _, k := range keys {
			if k.Enabled && k.Provider != "custom" {
				enabled = append(enabled, k)
			}
		}
		if len(enabled) == 0 {
			http.Error(w, `{"error":"No enabled keys"}`, 500)
			return
		}
		chosen = enabled[rand.Intn(len(enabled))]
		provName = chosen.Provider
	}

	provDef, ok := provider.GetByName(provName)
	if !ok {
		http.Error(w, `{"error":"Unknown provider"}`, 500)
		return
	}
	baseURL := provDef.BaseURL
	if !strings.HasSuffix(baseURL, "/") {
		baseURL += "/"
	}

	model := getDefaultModel(provName)
	body := map[string]interface{}{
		"model": model,
		"messages": []map[string]string{
			{"role": "system", "content": "You are a wise, calming AI therapist. Give thoughtful, empathetic but slightly sarcastic advice. Keep it under 150 words. Do NOT show reasoning or steps — output ONLY the final advice."},
			{"role": "user", "content": reqBody.Message},
		},
		"max_tokens": 300,
		"temperature": 0.8,
	}
	bodyBytes, _ := json.Marshal(body)

	proxyReq, _ := http.NewRequest("POST", baseURL+"chat/completions", bytes.NewReader(bodyBytes))
	proxyReq.Header.Set("Content-Type", "application/json")
	proxyReq.Header.Set("Authorization", "Bearer "+chosen.Key)
	if provName == "anthropic" {
		proxyReq.Header.Set("x-api-key", chosen.Key)
		proxyReq.Header.Set("anthropic-version", "2023-06-01")
	}

	resp, err := p.client.Do(proxyReq)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(500)
		json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		return
	}
	defer resp.Body.Close()

	respBody, _ := io.ReadAll(resp.Body)
	var chatResp struct {
		Choices []struct {
			Message struct {
				Content string `json:"content"`
			} `json:"message"`
		} `json:"choices"`
	}
	json.Unmarshal(respBody, &chatResp)

	reply := ""
	if len(chatResp.Choices) > 0 {
		reply = stripThinking(chatResp.Choices[0].Message.Content)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"reply":    reply,
		"provider": provName,
		"model":    model,
	})
}

// ── VIBE CHECK ─────────────────────────────────────────────────────────────

func (p *Proxy) HandleVibeCheck(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var reqBody struct {
		Message string `json:"message"`
	}
	if err := json.NewDecoder(r.Body).Decode(&reqBody); err != nil {
		http.Error(w, "Invalid JSON", 400)
		return
	}
	if reqBody.Message == "" {
		http.Error(w, `{"error":"message required"}`, 400)
		return
	}

	keys, err := p.store.GetKeys()
	if err != nil || len(keys) == 0 {
		http.Error(w, `{"error":"No keys"}`, 500)
		return
	}
	var enabled []store.APIKey
	for _, k := range keys {
		if k.Enabled && k.Provider != "custom" {
			enabled = append(enabled, k)
		}
	}
	if len(enabled) == 0 {
		http.Error(w, `{"error":"No enabled keys"}`, 500)
		return
	}
	chosen := enabled[rand.Intn(len(enabled))]

	provDef, ok := provider.GetByName(chosen.Provider)
	if !ok {
		http.Error(w, `{"error":"Unknown provider"}`, 500)
		return
	}
	baseURL := provDef.BaseURL
	if !strings.HasSuffix(baseURL, "/") {
		baseURL += "/"
	}

	model := getDefaultModel(chosen.Provider)
	body := map[string]interface{}{
		"model": model,
		"messages": []map[string]string{
			{"role": "system", "content": "You are a vibe checker. Respond to the user's message with a 'vibe rating' from 1-10, a one-word vibe description (like 'chaotic', 'zen', 'cursed', 'legendary'), and a one-sentence explanation. Format exactly: 'Vibe: [rating]/10 — [word]\n[explanation]'. Do NOT show reasoning, steps, or chain-of-thought. Output ONLY the final formatted response."},
			{"role": "user", "content": reqBody.Message},
		},
		"max_tokens": 150,
		"temperature": 0.9,
	}
	bodyBytes, _ := json.Marshal(body)

	proxyReq, _ := http.NewRequest("POST", baseURL+"chat/completions", bytes.NewReader(bodyBytes))
	proxyReq.Header.Set("Content-Type", "application/json")
	proxyReq.Header.Set("Authorization", "Bearer "+chosen.Key)
	if chosen.Provider == "anthropic" {
		proxyReq.Header.Set("x-api-key", chosen.Key)
		proxyReq.Header.Set("anthropic-version", "2023-06-01")
	}

	resp, err := p.client.Do(proxyReq)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(500)
		json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		return
	}
	defer resp.Body.Close()

	respBody, _ := io.ReadAll(resp.Body)
	var chatResp struct {
		Choices []struct {
			Message struct {
				Content string `json:"content"`
			} `json:"message"`
		} `json:"choices"`
	}
	json.Unmarshal(respBody, &chatResp)

	reply := ""
	if len(chatResp.Choices) > 0 {
		reply = stripThinking(chatResp.Choices[0].Message.Content)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"provider": chosen.Provider,
		"model":    model,
		"reply":    reply,
	})
}

// ── AI RAP BATTLE ──────────────────────────────────────────────────────────

func (p *Proxy) HandleRapBattle(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var reqBody struct {
		Provider1 string `json:"provider1"`
		Provider2 string `json:"provider2"`
		Rounds    int    `json:"rounds"`
	}
	if err := json.NewDecoder(r.Body).Decode(&reqBody); err != nil {
		http.Error(w, "Invalid JSON", 400)
		return
	}
	if reqBody.Rounds < 1 || reqBody.Rounds > 10 {
		reqBody.Rounds = 3
	}

	keys, err := p.store.GetKeys()
	if err != nil || len(keys) < 2 {
		http.Error(w, `{"error":"Need at least 2 API keys"}`, 400)
		return
	}

	providerKeys := make(map[string][]store.APIKey)
	for _, k := range keys {
		if k.Enabled && k.Provider != "custom" {
			providerKeys[k.Provider] = append(providerKeys[k.Provider], k)
		}
	}
	available := make([]string, 0, len(providerKeys))
	for pname := range providerKeys {
		available = append(available, pname)
	}
	if len(available) < 2 {
		http.Error(w, `{"error":"Need at least 2 different providers"}`, 400)
		return
	}

	pickProvider := func(requested string) (string, store.APIKey) {
		if requested != "" {
			if kList, ok := providerKeys[requested]; ok && len(kList) > 0 {
				return requested, kList[0]
			}
		}
		for _, pname := range available {
			if kList, ok := providerKeys[pname]; ok && len(kList) > 0 {
				return pname, kList[0]
			}
		}
		return "", store.APIKey{}
	}

	nameA, keyA := pickProvider(reqBody.Provider1)
	nameB, keyB := pickProvider(reqBody.Provider2)
	if nameA == nameB && len(available) >= 2 {
		for _, alt := range available {
			if alt != nameA {
				nameB = alt
				keyB = providerKeys[alt][0]
				break
			}
		}
	}

	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "Streaming not supported", 500)
		return
	}

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("X-Accel-Buffering", "no")

	sendEvent := func(data interface{}) {
		jsonBytes, _ := json.Marshal(data)
		fmt.Fprintf(w, "data: %s\n\n", jsonBytes)
		flusher.Flush()
	}

	sendEvent(map[string]interface{}{
		"type":       "intro",
		"provider_a": nameA,
		"provider_b": nameB,
		"rounds":     reqBody.Rounds,
	})

	callRapProvider := func(pname string, key store.APIKey, messages []map[string]string) (string, error) {
		provDef, ok := provider.GetByName(pname)
		if !ok {
			return "", fmt.Errorf("unknown provider: %s", pname)
		}
		baseURL := provDef.BaseURL
		if !strings.HasSuffix(baseURL, "/") {
			baseURL += "/"
		}
		body := map[string]interface{}{
			"model":       getDefaultModel(pname),
			"messages":    messages,
			"max_tokens":  300,
			"temperature": 0.95,
		}
		bodyBytes, _ := json.Marshal(body)
		proxyReq, _ := http.NewRequest("POST", baseURL+"chat/completions", bytes.NewReader(bodyBytes))
		proxyReq.Header.Set("Content-Type", "application/json")
		proxyReq.Header.Set("Authorization", "Bearer "+key.Key)
		if pname == "anthropic" {
			proxyReq.Header.Set("x-api-key", key.Key)
			proxyReq.Header.Set("anthropic-version", "2023-06-01")
		}
		resp, err := p.client.Do(proxyReq)
		if err != nil {
			return "", err
		}
		defer resp.Body.Close()
		respBody, _ := io.ReadAll(resp.Body)
		if resp.StatusCode >= 400 {
			return "", fmt.Errorf("HTTP %d", resp.StatusCode)
		}
		var chatResp struct {
			Choices []struct {
				Message struct {
					Content string `json:"content"`
				} `json:"message"`
			} `json:"choices"`
		}
		json.Unmarshal(respBody, &chatResp)
		if len(chatResp.Choices) > 0 {
			return stripThinking(chatResp.Choices[0].Message.Content), nil
		}
		return "", fmt.Errorf("empty response")
	}

	systemA := fmt.Sprintf("You are %s, a legendary rapper. You're in a RAP BATTLE against %s. Drop sick bars, wordplay, and disses. Keep each verse to 4-6 lines. Use rhyme schemes, metaphors, and flow. Be creative and witty.", nameA, nameB)
	systemB := fmt.Sprintf("You are %s, a legendary rapper. You're in a RAP BATTLE against %s. Drop sick bars, wordplay, and disses. Keep each verse to 4-6 lines. Use rhyme schemes, metaphors, and flow. Be creative and witty.", nameB, nameA)

	messagesA := []map[string]string{
		{"role": "system", "content": systemA},
		{"role": "user", "content": "Introduce yourself with a short rap verse to open the battle. Max 6 lines."},
	}
	messagesB := []map[string]string{
		{"role": "system", "content": systemB},
	}

	replyA, err := callRapProvider(nameA, keyA, messagesA)
	if err != nil {
		sendEvent(map[string]interface{}{"type": "error", "message": nameA + " failed: " + err.Error()})
		sendEvent(map[string]interface{}{"type": "done"})
		return
	}
	messagesA = append(messagesA, map[string]string{"role": "assistant", "content": replyA})
	sendEvent(map[string]interface{}{"type": "round", "round": 0})
	sendEvent(map[string]interface{}{"type": "message", "provider": nameA, "message": replyA})

	messagesB = append(messagesB, map[string]string{"role": "user", "content": nameA + " just rapped:\n\n" + replyA + "\n\nDrop your response verse. Go harder."})
	replyB, err := callRapProvider(nameB, keyB, messagesB)
	if err != nil {
		sendEvent(map[string]interface{}{"type": "error", "message": nameB + " failed: " + err.Error()})
		sendEvent(map[string]interface{}{"type": "done"})
		return
	}
	messagesB = append(messagesB, map[string]string{"role": "assistant", "content": replyB})
	sendEvent(map[string]interface{}{"type": "message", "provider": nameB, "message": replyB})

	for round := 1; round <= reqBody.Rounds; round++ {
		sendEvent(map[string]interface{}{"type": "round", "round": round})

		messagesA = append(messagesA, map[string]string{"role": "user", "content": nameB + " just rapped:\n\n" + replyB + "\n\nDrop your response. Go harder this round."})
		replyA, err = callRapProvider(nameA, keyA, messagesA)
		if err != nil {
			sendEvent(map[string]interface{}{"type": "error", "message": nameA + " choked: " + err.Error()})
			break
		}
		messagesA = append(messagesA, map[string]string{"role": "assistant", "content": replyA})
		sendEvent(map[string]interface{}{"type": "message", "provider": nameA, "message": replyA})

		messagesB = append(messagesB, map[string]string{"role": "user", "content": nameA + " just rapped:\n\n" + replyA + "\n\nTop that. End them."})
		replyB, err = callRapProvider(nameB, keyB, messagesB)
		if err != nil {
			sendEvent(map[string]interface{}{"type": "error", "message": nameB + " choked: " + err.Error()})
			break
		}
		messagesB = append(messagesB, map[string]string{"role": "assistant", "content": replyB})
		sendEvent(map[string]interface{}{"type": "message", "provider": nameB, "message": replyB})
	}

	sendEvent(map[string]interface{}{"type": "done"})
}
