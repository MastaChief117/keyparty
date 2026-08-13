package proxy

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"math/rand"
	"net/http"
	"strings"
	"time"

	"ai-gateway/provider"
	"ai-gateway/store"
)

// ── AI FORTUNE TELLER ──────────────────────────────────────────────────────

func (p *Proxy) HandleFortune(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var reqBody struct {
		Message string `json:"message"`
	}
	json.NewDecoder(r.Body).Decode(&reqBody)

	fortunes := []string{
		"Your code will compile on the first try... just kidding, fix line 42.",
		"A wild nil pointer appears! Your debugging journey begins now.",
		"The stars say your PR will be approved without comments. LOL.",
		"Your next commit message will be 'fixed stuff'. Embrace mediocrity.",
		"Segfault in your future. Touch grass before it touches you.",
		"A mysterious bug will appear at 3AM. It was a semicolon all along.",
		"Your code review will include the phrase 'have you tried turning it off and on again?'",
		"The algorithm gods demand a sacrifice. Offer a console.log.",
		"Your database will migrate itself. Just kidding, run the backup first.",
		"A stack overflow approaches. Reduce your recursion, mortal.",
		"Your deployment will succeed... in the staging environment only.",
		"The CI/CD pipeline smiles upon you today. Merge with confidence.",
		"A wild dependency update appears! Your build will break for 3 days.",
		"Your code is 99% bug-free. The 1% will crash production.",
		"The rubber duck on your desk has wisdom. Ask it about your logic.",
	}

	fortune := fortunes[rand.Intn(len(fortunes))]

	if reqBody.Message != "" {
		fortune = fmt.Sprintf("You asked: '%s'\n\n%s", reqBody.Message, fortune)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"fortune":  fortune,
		"provider": "keyparty",
		"mood":     "cosmic",
		"warning":  "Side effects may include: imposter syndrome, existential dread, and sudden urges to rewrite everything in Rust.",
	})
}

// ── PROVIDER ROAST MODE ───────────────────────────────────────────────────

func (p *Proxy) HandleProviderRoast(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var reqBody struct {
		Provider1 string `json:"provider1"`
		Provider2 string `json:"provider2"`
	}
	json.NewDecoder(r.Body).Decode(&reqBody)

	keys, err := p.store.GetKeys()
	if err != nil || len(keys) < 2 {
		http.Error(w, `{"error":"Need at least 2 providers"}`, 400)
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

	callProvider := func(pname string, key store.APIKey, messages []map[string]string) (string, error) {
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
			"max_tokens":  200,
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

	systemA := fmt.Sprintf("You are %s, an AI model. You're in a ROAST BATTLE against %s. Roast them HARD about their weaknesses, quirks, and failings. Be savage but funny. Max 100 words.", nameA, nameB)
	systemB := fmt.Sprintf("You are %s, an AI model. You're in a ROAST BATTLE against %s. Roast them HARD about their weaknesses, quirks, and failings. Be savage but funny. Max 100 words.", nameB, nameA)

	messagesA := []map[string]string{
		{"role": "system", "content": systemA},
		{"role": "user", "content": "Open with your first roast against " + nameB + ". Go hard."},
	}

	replyA, err := callProvider(nameA, keyA, messagesA)
	if err != nil {
		http.Error(w, fmt.Sprintf(`{"error":"%s"}`, err.Error()), 500)
		return
	}

	messagesB := []map[string]string{
		{"role": "system", "content": systemB},
		{"role": "user", "content": nameA + " just roasted you:\n\n" + replyA + "\n\nFire back harder."},
	}

	replyB, err := callProvider(nameB, keyB, messagesB)
	if err != nil {
		http.Error(w, fmt.Sprintf(`{"error":"%s"}`, err.Error()), 500)
		return
	}

	messagesA = append(messagesA, map[string]string{"role": "assistant", "content": replyA})
	messagesA = append(messagesA, map[string]string{"role": "user", "content": nameB + " responded:\n\n" + replyB + "\n\nEnd them with a final roast."})

	replyA2, err := callProvider(nameA, keyA, messagesA)
	if err != nil {
		replyA2 = replyA
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"provider_a": nameA,
		"provider_b": nameB,
		"roast_a":    replyA,
		"roast_b":    replyB,
		"finale_a":   replyA2,
		"winner":     "you, for watching two AIs insult each other",
	})
}

// ── DEBUGGING ORACLE ──────────────────────────────────────────────────────

func (p *Proxy) HandleDebugOracle(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var reqBody struct {
		Error string `json:"error"`
		Stack string `json:"stack"`
	}
	if err := json.NewDecoder(r.Body).Decode(&reqBody); err != nil {
		http.Error(w, `{"error":"Invalid JSON"}`, 400)
		return
	}
	if reqBody.Error == "" {
		http.Error(w, `{"error":"error message required"}`, 400)
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
	provDef, ok := provider.GetByName(chosen.Provider)
	if !ok {
		http.Error(w, `{"error":"Unknown provider"}`, 500)
		return
	}
	baseURL := provDef.BaseURL
	if !strings.HasSuffix(baseURL, "/") {
		baseURL += "/"
	}

	prompt := fmt.Sprintf("The user encountered this error:\n\nError: %s\n\n", reqBody.Error)
	if reqBody.Stack != "" {
		prompt += "Stack trace:\n" + reqBody.Stack + "\n\n"
	}
	prompt += "You are the Debugging Oracle. Analyze this error dramatically:\n1. What went wrong (in a dramatic tone)\n2. The most likely cause\n3. A simple fix\n4. A life lesson from this bug\n\nKeep it under 200 words. Be theatrical but helpful."

	body := map[string]interface{}{
		"model":       getDefaultModel(chosen.Provider),
		"messages":    []map[string]string{{"role": "user", "content": prompt}},
		"max_tokens":  400,
		"temperature": 0.8,
	}
	bodyBytes, _ := json.Marshal(body)

	proxyReq, _ := http.NewRequest("POST", baseURL+"chat/completions", bytes.NewReader(bodyBytes))
	proxyReq.Header.Set("Content-Type", "application/json")
	proxyReq.Header.Set("Authorization", "Bearer "+chosen.Key)
	if chosen.Provider == "anthropic" {
		proxyReq.Header.Set("x-api-key", chosen.Key)
		proxyReq.Header.Set("anthropic-version", "2023-06-01")
	}

	start := time.Now()
	resp, err := p.client.Do(proxyReq)
	latency := time.Since(start).Milliseconds()
	if err != nil {
		http.Error(w, fmt.Sprintf(`{"error":"%s"}`, err.Error()), 500)
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
		"oracle":     reply,
		"provider":   chosen.Provider,
		"latency_ms": latency,
		"mood":       "mystical",
	})
}

// ── COMMIT MESSAGE GENERATOR ──────────────────────────────────────────────

func (p *Proxy) HandleCommitMsg(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var reqBody struct {
		Diff  string `json:"diff"`
		Style string `json:"style"`
	}
	json.NewDecoder(r.Body).Decode(&reqBody)

	if reqBody.Style == "" {
		reqBody.Style = "dramatic"
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
	provDef, ok := provider.GetByName(chosen.Provider)
	if !ok {
		http.Error(w, `{"error":"Unknown provider"}`, 500)
		return
	}
	baseURL := provDef.BaseURL
	if !strings.HasSuffix(baseURL, "/") {
		baseURL += "/"
	}

	stylePrompts := map[string]string{
		"dramatic":     "Write a dramatic, epic commit message as if this code change will change the world. Use metaphors and grandiose language.",
		"professional": "Write a clean, conventional commit message following best practices.",
		"meme":         "Write a hilarious meme-style commit message. Be absurd and funny.",
		"poetic":       "Write a poetic, Shakespearean commit message about this code change.",
	}

	prompt := fmt.Sprintf("Generate a commit message for these changes:\n\n%s\n\nStyle: %s\n\nOutput ONLY the commit message, nothing else.", reqBody.Diff, stylePrompts[reqBody.Style])

	body := map[string]interface{}{
		"model":       getDefaultModel(chosen.Provider),
		"messages":    []map[string]string{{"role": "user", "content": prompt}},
		"max_tokens":  150,
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

	start := time.Now()
	resp, err := p.client.Do(proxyReq)
	latency := time.Since(start).Milliseconds()
	if err != nil {
		http.Error(w, fmt.Sprintf(`{"error":"%s"}`, err.Error()), 500)
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
		"commit_message": reply,
		"style":          reqBody.Style,
		"provider":       chosen.Provider,
		"latency_ms":     latency,
	})
}
