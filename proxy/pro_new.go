package proxy

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	"ai-gateway/provider"
	"ai-gateway/store"
)

func (p *Proxy) callProviderForPro(providerName string, key store.APIKey, model string, messages []map[string]string) (string, int, int, error) {
	provDef, ok := provider.GetByName(providerName)
	if !ok {
		return "", 0, 0, fmt.Errorf("unknown provider: %s", providerName)
	}
	baseURL := provDef.BaseURL
	if !strings.HasSuffix(baseURL, "/") {
		baseURL += "/"
	}
	endpoint := baseURL + "chat/completions"

	body := map[string]interface{}{
		"model":       model,
		"messages":    messages,
		"max_tokens":  2048,
		"temperature": 0.7,
	}
	bodyBytes, _ := json.Marshal(body)

	proxyReq, err := http.NewRequest("POST", endpoint, bytes.NewReader(bodyBytes))
	if err != nil {
		return "", 0, 0, err
	}
	proxyReq.Header.Set("Content-Type", "application/json")
	proxyReq.Header.Set("Authorization", "Bearer "+key.Key)
	if providerName == "anthropic" {
		proxyReq.Header.Set("x-api-key", key.Key)
		proxyReq.Header.Set("anthropic-version", "2023-06-01")
	}

	resp, err := p.client.Do(proxyReq)
	if err != nil {
		return "", 0, 0, err
	}
	defer resp.Body.Close()

	respBody, _ := io.ReadAll(resp.Body)
	if resp.StatusCode >= 400 {
		return "", 0, 0, fmt.Errorf("HTTP %d: %s", resp.StatusCode, string(respBody[:min(len(respBody), 300)]))
	}

	var chatResp struct {
		Choices []struct {
			Message struct {
				Content string `json:"content"`
			} `json:"message"`
		} `json:"choices"`
		Usage struct {
			PromptTokens     int `json:"prompt_tokens"`
			CompletionTokens int `json:"completion_tokens"`
		} `json:"usage"`
	}
	json.Unmarshal(respBody, &chatResp)

	reply := ""
	if len(chatResp.Choices) > 0 {
		reply = stripThinking(chatResp.Choices[0].Message.Content)
	}
	return reply, chatResp.Usage.PromptTokens, chatResp.Usage.CompletionTokens, nil
}

func (p *Proxy) pickRandomProviderForPro() (string, store.APIKey, error) {
	keys, err := p.store.GetKeys()
	if err != nil || len(keys) == 0 {
		return "", store.APIKey{}, fmt.Errorf("no keys available")
	}
	var enabled []store.APIKey
	for _, k := range keys {
		if k.Enabled && k.Provider != "custom" {
			enabled = append(enabled, k)
		}
	}
	if len(enabled) == 0 {
		return "", store.APIKey{}, fmt.Errorf("no enabled keys")
	}
	k := enabled[int(time.Now().UnixNano())%len(enabled)]
	return k.Provider, k, nil
}

// ── 1. HandleUsageDashboard — Per Virtual Key Usage ─────────────────────

func (p *Proxy) HandleUsageDashboard(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	w.Header().Set("Content-Type", "application/json")

	parts := strings.Split(strings.TrimPrefix(r.URL.Path, "/admin/vk-usage"), "/")
	keyID := ""
	if len(parts) > 0 && parts[0] != "" {
		keyID = parts[0]
	}

	if keyID != "" {
		logs, err := p.store.SearchRequestLogs("", "", keyID, "", 500)
		if err != nil {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(500)
			json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
			return
		}

		vkKeys, _ := p.store.GetVirtualKeys()
		vkName := ""
		for _, vk := range vkKeys {
			if vk.Key == keyID || fmt.Sprintf("%d", vk.ID) == keyID {
				vkName = vk.Name
				break
			}
		}

		var totalCost float64
		var totalLatency int64
		var errorCount int64
		providerCounts := map[string]int64{}

		for _, l := range logs {
			totalCost += l.Cost
			totalLatency += l.Latency
			if l.StatusCode >= 400 {
				errorCount++
			}
			providerCounts[l.Provider]++
		}

		topProvider := ""
		topCount := int64(0)
		for prov, cnt := range providerCounts {
			if cnt > topCount {
				topProvider = prov
				topCount = cnt
			}
		}

		avgLatency := float64(0)
		if len(logs) > 0 {
			avgLatency = float64(totalLatency) / float64(len(logs))
		}

		json.NewEncoder(w).Encode(map[string]interface{}{
			"virtual_key":    keyID,
			"vk_name":        vkName,
			"total_requests": len(logs),
			"total_cost":     totalCost,
			"avg_latency":    avgLatency,
			"top_provider":   topProvider,
			"error_count":    errorCount,
			"recent_logs":    logs,
		})
		return
	}

	details, err := p.store.GetVirtualKeyUsageDetail()
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(500)
		json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		return
	}

	vkKeys, _ := p.store.GetVirtualKeys()
	vkNameMap := map[string]string{}
	for _, vk := range vkKeys {
		vkNameMap[vk.Key] = vk.Name
		vkNameMap[fmt.Sprintf("%d", vk.ID)] = vk.Name
	}

	type vkRow struct {
		VirtualKey   string  `json:"virtual_key"`
		VKName       string  `json:"vk_name"`
		TotalReqs    int64   `json:"total_requests"`
		TotalCost    float64 `json:"total_cost"`
		AvgLatency   float64 `json:"avg_latency"`
		TopProvider  string  `json:"top_provider"`
		ErrorCount   int64   `json:"error_count"`
	}

	var result []vkRow
	for _, d := range details {
		name := vkNameMap[d.VirtualKey]
		result = append(result, vkRow{
			VirtualKey:  d.VirtualKey,
			VKName:      name,
			TotalReqs:   d.TotalReqs,
			TotalCost:   d.TotalCost,
			AvgLatency:  d.AvgLatency,
			TopProvider: d.TopProvider,
			ErrorCount:  d.ErrorCount,
		})
	}

	json.NewEncoder(w).Encode(result)
}

// ── 2. HandleAutoRotate — Auto-Rotate Keys ──────────────────────────────

func (p *Proxy) HandleAutoRotate(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	if r.URL.Path == "/admin/auto-rotate/status" {
		keys, err := p.store.GetKeys()
		if err != nil {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(500)
			json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
			return
		}

		type providerHealth struct {
			Provider     string `json:"provider"`
			TotalKeys    int    `json:"total_keys"`
			HealthyKeys  int    `json:"healthy_keys"`
			TotalErrors  int64  `json:"total_errors"`
		}

		healthMap := map[string]*providerHealth{}
		for _, k := range keys {
			if k.Provider == "custom" {
				continue
			}
			if _, ok := healthMap[k.Provider]; !ok {
				healthMap[k.Provider] = &providerHealth{Provider: k.Provider}
			}
			ph := healthMap[k.Provider]
			ph.TotalKeys++
			ph.TotalErrors += k.ErrorReqs
			if k.Enabled && k.ErrorReqs < 10 {
				ph.HealthyKeys++
			}
		}

		var result []providerHealth
		for _, ph := range healthMap {
			result = append(result, *ph)
		}
		json.NewEncoder(w).Encode(result)
		return
	}

	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var reqBody struct {
		Provider string `json:"provider"`
	}
	if err := json.NewDecoder(r.Body).Decode(&reqBody); err != nil {
		http.Error(w, `{"error":"invalid JSON"}`, 400)
		return
	}
	if reqBody.Provider == "" {
		http.Error(w, `{"error":"provider required"}`, 400)
		return
	}

	healthy, err := p.store.GetHealthyKeysByProvider(reqBody.Provider)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(500)
		json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		return
	}
	if len(healthy) == 0 {
		http.Error(w, `{"error":"no healthy keys for provider"}`, 404)
		return
	}

	best := healthy[0]
	json.NewEncoder(w).Encode(map[string]interface{}{
		"provider":    best.Provider,
		"key_id":      best.ID,
		"key_name":    best.Name,
		"model":       best.Model,
		"error_count": best.ErrorReqs,
	})
}

// ── 3. HandlePlayground — Chat Playground ───────────────────────────────

func (p *Proxy) HandlePlayground(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var reqBody struct {
		Provider string              `json:"provider"`
		Model    string              `json:"model"`
		Messages []map[string]string `json:"messages"`
		Stream   bool                `json:"stream"`
	}
	if err := json.NewDecoder(r.Body).Decode(&reqBody); err != nil {
		http.Error(w, `{"error":"invalid JSON"}`, 400)
		return
	}
	if len(reqBody.Messages) == 0 {
		http.Error(w, `{"error":"messages required"}`, 400)
		return
	}

	keys, err := p.store.GetKeysByProvider(reqBody.Provider)
	if err != nil || len(keys) == 0 {
		http.Error(w, `{"error":"no keys for provider"}`, 404)
		return
	}

	key := keys[0]
	model := reqBody.Model
	if model == "" {
		model = getDefaultModel(reqBody.Provider)
	}

	provDef, ok := provider.GetByName(reqBody.Provider)
	if !ok {
		http.Error(w, `{"error":"unknown provider"}`, 400)
		return
	}
	baseURL := provDef.BaseURL
	if !strings.HasSuffix(baseURL, "/") {
		baseURL += "/"
	}
	endpoint := baseURL + "chat/completions"

	body := map[string]interface{}{
		"model":       model,
		"messages":    reqBody.Messages,
		"max_tokens":  2048,
		"temperature": 0.7,
		"stream":      reqBody.Stream,
	}
	bodyBytes, _ := json.Marshal(body)

	proxyReq, err := http.NewRequest("POST", endpoint, bytes.NewReader(bodyBytes))
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(500)
		json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		return
	}
	proxyReq.Header.Set("Content-Type", "application/json")
	proxyReq.Header.Set("Authorization", "Bearer "+key.Key)
	if reqBody.Provider == "anthropic" {
		proxyReq.Header.Set("x-api-key", key.Key)
		proxyReq.Header.Set("anthropic-version", "2023-06-01")
	}

	start := time.Now()
	resp, err := p.client.Do(proxyReq)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(502)
		json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		return
	}
	defer resp.Body.Close()

	latency := time.Since(start).Milliseconds()

	if reqBody.Stream {
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

		var totalContent strings.Builder
		buf := make([]byte, 4096)
		for {
			n, readErr := resp.Body.Read(buf)
			if n > 0 {
				chunk := string(buf[:n])
				totalContent.WriteString(chunk)

				lines := strings.Split(chunk, "\n")
				for _, line := range lines {
					line = strings.TrimSpace(line)
					if strings.HasPrefix(line, "data: ") {
						data := strings.TrimPrefix(line, "data: ")
						if data == "[DONE]" {
							sendEvent(map[string]interface{}{"type": "done", "provider": reqBody.Provider, "model": model, "latency_ms": latency})
							return
						}
						var sseChunk struct {
							Choices []struct {
								Delta struct {
									Content string `json:"content"`
								} `json:"delta"`
							} `json:"choices"`
						}
						if json.Unmarshal([]byte(data), &sseChunk) == nil && len(sseChunk.Choices) > 0 {
							token := sseChunk.Choices[0].Delta.Content
							if token != "" {
								sendEvent(map[string]interface{}{"type": "token", "content": token})
							}
						}
					}
				}
			}
			if readErr != nil {
				break
			}
		}

		sendEvent(map[string]interface{}{"type": "done", "provider": reqBody.Provider, "model": model, "latency_ms": latency})
		return
	}

	respBody, _ := io.ReadAll(resp.Body)
	if resp.StatusCode >= 400 {
		http.Error(w, fmt.Sprintf(`{"error":"HTTP %d"}`, resp.StatusCode), resp.StatusCode)
		return
	}

	var chatResp struct {
		Choices []struct {
			Message struct {
				Content string `json:"content"`
			} `json:"message"`
		} `json:"choices"`
		Usage struct {
			PromptTokens     int `json:"prompt_tokens"`
			CompletionTokens int `json:"completion_tokens"`
		} `json:"usage"`
	}
	json.Unmarshal(respBody, &chatResp)

	reply := ""
	if len(chatResp.Choices) > 0 {
		reply = stripThinking(chatResp.Choices[0].Message.Content)
	}

	json.NewEncoder(w).Encode(map[string]interface{}{
		"provider":   reqBody.Provider,
		"model":      model,
		"reply":      reply,
		"latency_ms": latency,
		"tokens_in":  chatResp.Usage.PromptTokens,
		"tokens_out": chatResp.Usage.CompletionTokens,
	})
}

// ── 4. HandleRecap — Weekly Recap ───────────────────────────────────────

func (p *Proxy) HandleRecap(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	if r.URL.Path == "/admin/recap/generate" {
		if r.Method != http.MethodPost {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		recap, err := p.store.GetWeeklyRecap()
		if err != nil {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(500)
			json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
			return
		}

		recapBytes, _ := json.MarshalIndent(recap, "", "  ")

		providerName, key, err := p.pickRandomProviderForPro()
		if err != nil {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(500)
			json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
			return
		}

		model := getDefaultModel(providerName)
		messages := []map[string]string{
			{"role": "system", "content": "You are a snarky stats bot. Generate a funny weekly recap report based on these API gateway stats. Use emojis and memes. Keep it under 300 words. Do NOT show reasoning or steps — output ONLY the final recap."},
			{"role": "user", "content": string(recapBytes)},
		}

		reply, _, _, err := p.callProviderForPro(providerName, key, model, messages)
		if err != nil {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(502)
			json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
			return
		}

		json.NewEncoder(w).Encode(map[string]interface{}{
			"recap_text": reply,
			"provider":   providerName,
		})
		return
	}

	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	recap, err := p.store.GetWeeklyRecap()
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(500)
		json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		return
	}

	json.NewEncoder(w).Encode(recap)
}

// ── 5. HandleCostAnalytics — Cost Analytics ─────────────────────────────

func (p *Proxy) HandleCostAnalytics(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	w.Header().Set("Content-Type", "application/json")

	days := 30
	if d := r.URL.Query().Get("days"); d != "" {
		if parsed, err := strconv.Atoi(d); err == nil && parsed > 0 {
			days = parsed
		}
	}

	analytics, err := p.store.GetCostAnalytics(days)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(500)
		json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		return
	}

	json.NewEncoder(w).Encode(analytics)
}

// ── Streaming provider call helper for playground ───────────────────────

func (p *Proxy) streamProviderForPro(w http.ResponseWriter, flusher http.Flusher, providerName string, key store.APIKey, model string, messages []map[string]string) (int64, error) {
	provDef, ok := provider.GetByName(providerName)
	if !ok {
		return 0, fmt.Errorf("unknown provider: %s", providerName)
	}
	baseURL := provDef.BaseURL
	if !strings.HasSuffix(baseURL, "/") {
		baseURL += "/"
	}
	endpoint := baseURL + "chat/completions"

	body := map[string]interface{}{
		"model":       model,
		"messages":    messages,
		"max_tokens":  2048,
		"temperature": 0.7,
		"stream":      true,
	}
	bodyBytes, _ := json.Marshal(body)

	proxyReq, err := http.NewRequest("POST", endpoint, bytes.NewReader(bodyBytes))
	if err != nil {
		return 0, err
	}
	proxyReq.Header.Set("Content-Type", "application/json")
	proxyReq.Header.Set("Authorization", "Bearer "+key.Key)
	if providerName == "anthropic" {
		proxyReq.Header.Set("x-api-key", key.Key)
		proxyReq.Header.Set("anthropic-version", "2023-06-01")
	}

	start := time.Now()
	resp, err := p.client.Do(proxyReq)
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()

	buf := make([]byte, 4096)
	for {
		n, readErr := resp.Body.Read(buf)
		if n > 0 {
			w.Write(buf[:n])
			flusher.Flush()
		}
		if readErr != nil {
			break
		}
	}
	return time.Since(start).Milliseconds(), nil
}

// === ROUTES TO ADD ===
//
// Add these route registrations to your main.go router setup:
//
//   // Pro feature routes
//   mux.HandleFunc("/admin/vk-usage", proxy.HandleUsageDashboard)
//   mux.HandleFunc("/admin/vk-usage/", proxy.HandleUsageDashboard)
//   mux.HandleFunc("/admin/auto-rotate", proxy.HandleAutoRotate)
//   mux.HandleFunc("/admin/auto-rotate/status", proxy.HandleAutoRotate)
//   mux.HandleFunc("/admin/playground", proxy.HandlePlayground)
//   mux.HandleFunc("/admin/recap", proxy.HandleRecap)
//   mux.HandleFunc("/admin/recap/generate", proxy.HandleRecap)
//   mux.HandleFunc("/admin/analytics", proxy.HandleCostAnalytics)

// === HTML TO ADD ===
//
// Add this tab content and JS to your dashboard HTML template:
//
// <!-- New nav tabs -->
// <li><a href="#" onclick="showTab('vkusage')">VK Usage</a></li>
// <li><a href="#" onclick="showTab('autoRotate')">Auto-Rotate</a></li>
// <li><a href="#" onclick="showTab('playground')">Playground</a></li>
// <li><a href="#" onclick="showTab('recap')">Recap</a></li>
// <li><a href="#" onclick="showTab('analytics')">Analytics</a></li>
//
// <!-- Tab content: VK Usage -->
// <div id="tab-vkusage" class="tab-content" style="display:none">
//   <h2>Virtual Key Usage</h2>
//   <button onclick="loadVKUsage()">Refresh</button>
//   <div id="vk-usage-table"></div>
// </div>
//
// <!-- Tab content: Auto-Rotate -->
// <div id="tab-autoRotate" class="tab-content" style="display:none">
//   <h2>Auto-Rotate Keys</h2>
//   <div>
//     <select id="rotate-provider">
//       <option value="openai">OpenAI</option><option value="anthropic">Anthropic</option>
//       <option value="gemini">Gemini</option><option value="groq">Groq</option>
//       <option value="deepseek">DeepSeek</option><option value="mistral">Mistral</option>
//       <option value="together">Together</option><option value="openrouter">OpenRouter</option>
//     </select>
//     <button onclick="doAutoRotate()">Pick Best Key</button>
//   </div>
//   <div id="rotate-result"></div>
//   <h3>Provider Health</h3>
//   <button onclick="loadRotateStatus()">Refresh Health</button>
//   <div id="rotate-health"></div>
// </div>
//
// <!-- Tab content: Playground -->
// <div id="tab-playground" class="tab-content" style="display:none">
//   <h2>Chat Playground</h2>
//   <div>
//     <select id="pg-provider">
//       <option value="openai">OpenAI</option><option value="anthropic">Anthropic</option>
//       <option value="gemini">Gemini</option><option value="groq">Groq</option>
//       <option value="deepseek">DeepSeek</option><option value="mistral">Mistral</option>
//       <option value="together">Together</option><option value="openrouter">OpenRouter</option>
//     </select>
//     <input id="pg-model" placeholder="model (optional)" style="width:200px"/>
//     <label><input type="checkbox" id="pg-stream"/> Stream</label>
//   </div>
//   <div style="margin:10px 0">
//     <textarea id="pg-message" rows="4" style="width:100%" placeholder="Type your message..."></textarea>
//   </div>
//   <button onclick="sendPlayground()">Send</button>
//   <div id="pg-response" style="margin-top:10px;padding:10px;background:#1a1a2e;border-radius:6px;white-space:pre-wrap;min-height:50px"></div>
//   <div id="pg-stats" style="margin-top:5px;font-size:12px;color:#888"></div>
// </div>
//
// <!-- Tab content: Recap -->
// <div id="tab-recap" class="tab-content" style="display:none">
//   <h2>Weekly Recap</h2>
//   <button onclick="loadRecap()">Load Recap</button>
//   <button onclick="generateRecap()">Generate Snarky Report</button>
//   <div id="recap-data" style="margin-top:10px"></div>
//   <div id="recap-generated" style="margin-top:10px;padding:10px;background:#1a1a2e;border-radius:6px;white-space:pre-wrap;display:none"></div>
// </div>
//
// <!-- Tab content: Analytics -->
// <div id="tab-analytics" class="tab-content" style="display:none">
//   <h2>Cost Analytics</h2>
//   <select id="analytics-days" onchange="loadAnalytics()">
//     <option value="7">Last 7 days</option>
//     <option value="14">Last 14 days</option>
//     <option value="30" selected>Last 30 days</option>
//     <option value="90">Last 90 days</option>
//   </select>
//   <div id="analytics-summary" style="margin:10px 0"></div>
//   <div style="display:flex;gap:20px">
//     <div style="flex:1"><h3>By Provider</h3><div id="analytics-provider"></div></div>
//     <div style="flex:1"><h3>By Day</h3><div id="analytics-daily"></div></div>
//   </div>
//   <h3>Top Models</h3>
//   <div id="analytics-models"></div>
// </div>
//
// <script>
// function showTab(name) {
//   document.querySelectorAll('.tab-content').forEach(t => t.style.display = 'none');
//   var el = document.getElementById('tab-' + name);
//   if (el) el.style.display = 'block';
//   if (name === 'vkusage') loadVKUsage();
//   if (name === 'autoRotate') loadRotateStatus();
//   if (name === 'analytics') loadAnalytics();
// }
//
// async function loadVKUsage() {
//   var res = await fetch('/admin/vk-usage');
//   var data = await res.json();
//   var html = '<table style="width:100%;border-collapse:collapse">';
//   html += '<tr style="border-bottom:1px solid #333"><th>Key</th><th>Name</th><th>Requests</th><th>Cost</th><th>Avg Latency</th><th>Top Provider</th><th>Errors</th></tr>';
//   (data || []).forEach(function(vk) {
//     html += '<tr style="border-bottom:1px solid #222">';
//     html += '<td>' + (vk.virtual_key||'').substring(0,12) + '...</td>';
//     html += '<td>' + (vk.vk_name||'-') + '</td>';
//     html += '<td>' + vk.total_requests + '</td>';
//     html += '<td>$' + (vk.total_cost||0).toFixed(4) + '</td>';
//     html += '<td>' + Math.round(vk.avg_latency||0) + 'ms</td>';
//     html += '<td>' + (vk.top_provider||'-') + '</td>';
//     html += '<td style="color:' + (vk.error_count > 0 ? '#f44' : '#4f4') + '">' + vk.error_count + '</td>';
//     html += '</tr>';
//   });
//   html += '</table>';
//   document.getElementById('vk-usage-table').innerHTML = html || '<p>No usage data yet</p>';
// }
//
// async function doAutoRotate() {
//   var provider = document.getElementById('rotate-provider').value;
//   var res = await fetch('/admin/auto-rotate', {method:'POST', headers:{'Content-Type':'application/json'}, body:JSON.stringify({provider:provider})});
//   var data = await res.json();
//   if (data.error) { document.getElementById('rotate-result').innerHTML = '<span style="color:#f44">'+data.error+'</span>'; return; }
//   document.getElementById('rotate-result').innerHTML = '<div style="padding:10px;background:#1a3a1a;border-radius:6px;margin-top:10px"><strong>Best key:</strong> ' + data.key_name + ' (ID: ' + data.key_id + ', errors: ' + data.error_count + ', model: ' + data.model + ')</div>';
// }
//
// async function loadRotateStatus() {
//   var res = await fetch('/admin/auto-rotate/status');
//   var data = await res.json();
//   var html = '';
//   (data || []).forEach(function(p) {
//     var pct = p.total_keys > 0 ? Math.round(p.healthy_keys / p.total_keys * 100) : 0;
//     html += '<div style="margin:5px 0;padding:8px;background:#1a1a2e;border-radius:4px">';
//     html += '<strong>' + p.provider + '</strong> — ' + p.healthy_keys + '/' + p.total_keys + ' healthy (' + pct + '%)';
//     html += ' — <span style="color:' + (p.total_errors > 50 ? '#f44' : '#ff0') + '">' + p.total_errors + ' errors</span>';
//     html += '</div>';
//   });
//   document.getElementById('rotate-health').innerHTML = html || '<p>No providers configured</p>';
// }
//
// async function sendPlayground() {
//   var provider = document.getElementById('pg-provider').value;
//   var model = document.getElementById('pg-model').value;
//   var message = document.getElementById('pg-message').value;
//   var stream = document.getElementById('pg-stream').checked;
//   var respEl = document.getElementById('pg-response');
//   var statsEl = document.getElementById('pg-stats');
//   respEl.textContent = 'Loading...';
//   statsEl.textContent = '';
//
//   if (stream) {
//     respEl.textContent = '';
//     var res = await fetch('/admin/playground', {method:'POST', headers:{'Content-Type':'application/json'}, body:JSON.stringify({provider:provider, model:model, messages:[{role:'user',content:message}], stream:true})});
//     var reader = res.body.getReader();
//     var decoder = new TextDecoder();
//     while (true) {
//       var {done, value} = await reader.read();
//       if (done) break;
//       var text = decoder.decode(value);
//       var lines = text.split('\n');
//       for (var i = 0; i < lines.length; i++) {
//         if (lines[i].startsWith('data: ')) {
//           try {
//             var evt = JSON.parse(lines[i].substring(6));
//             if (evt.type === 'token') respEl.textContent += evt.content;
//             if (evt.type === 'done') statsEl.textContent = 'Provider: ' + evt.provider + ' | Model: ' + evt.model + ' | Latency: ' + evt.latency_ms + 'ms';
//           } catch(e) {}
//         }
//       }
//     }
//   } else {
//     var res = await fetch('/admin/playground', {method:'POST', headers:{'Content-Type':'application/json'}, body:JSON.stringify({provider:provider, model:model, messages:[{role:'user',content:message}], stream:false})});
//     var data = await res.json();
//     if (data.error) { respEl.textContent = 'Error: ' + data.error; return; }
//     respEl.textContent = data.reply;
//     statsEl.textContent = 'Provider: ' + data.provider + ' | Model: ' + data.model + ' | Latency: ' + data.latency_ms + 'ms | Tokens: ' + data.tokens_in + ' in / ' + data.tokens_out + ' out';
//   }
// }
//
// async function loadRecap() {
//   var res = await fetch('/admin/recap');
//   var data = await res.json();
//   var html = '<div style="padding:10px;background:#1a1a2e;border-radius:6px">';
//   html += '<p><strong>Period:</strong> ' + data.period + '</p>';
//   html += '<p><strong>Total Requests:</strong> ' + data.total_requests + ' | <strong>Cost:</strong> $' + (data.total_cost||0).toFixed(4) + '</p>';
//   html += '<p><strong>Cache Hits:</strong> ' + data.cache_hits + ' | <strong>Error Rate:</strong> ' + (data.error_rate||0).toFixed(1) + '%</p>';
//   html += '<p><strong>Top Provider:</strong> ' + (data.top_provider||'-') + ' | <strong>Top Model:</strong> ' + (data.top_model||'-') + '</p>';
//   html += '<p><strong>Avg Latency:</strong> ' + Math.round(data.avg_latency||0) + 'ms</p>';
//   if (data.daily_breakdown && data.daily_breakdown.length) {
//     html += '<h4>Daily Breakdown</h4>';
//     data.daily_breakdown.forEach(function(d) {
//       html += '<div style="margin:2px 0">' + d.date + ': ' + d.requests + ' reqs, $' + (d.cost||0).toFixed(4) + '</div>';
//     });
//   }
//   html += '</div>';
//   document.getElementById('recap-data').innerHTML = html;
// }
//
// async function generateRecap() {
//   var el = document.getElementById('recap-generated');
//   el.style.display = 'block';
//   el.textContent = 'Generating...';
//   var res = await fetch('/admin/recap/generate', {method:'POST'});
//   var data = await res.json();
//   if (data.error) { el.textContent = 'Error: ' + data.error; return; }
//   el.innerHTML = '<strong>Generated via ' + data.provider + ':</strong><br><br>' + (data.recap_text||'').replace(/\n/g, '<br>');
// }
//
// async function loadAnalytics() {
//   var days = document.getElementById('analytics-days').value;
//   var res = await fetch('/admin/analytics?days=' + days);
//   var data = await res.json();
//
//   document.getElementById('analytics-summary').innerHTML =
//     '<div style="display:flex;gap:20px">' +
//     '<div style="padding:10px;background:#1a1a2e;border-radius:6px;flex:1;text-align:center"><strong>$' + (data.total_cost||0).toFixed(4) + '</strong><br>Total Cost</div>' +
//     '<div style="padding:10px;background:#1a1a2e;border-radius:6px;flex:1;text-align:center"><strong>' + (data.total_tokens||0).toLocaleString() + '</strong><br>Total Tokens</div>' +
//     '<div style="padding:10px;background:#1a1a2e;border-radius:6px;flex:1;text-align:center"><strong>' + Math.round(data.avg_latency||0) + 'ms</strong><br>Avg Latency</div>' +
//     '</div>';
//
//   // Provider bar chart
//   var provHtml = '';
//   var maxCost = 0;
//   (data.by_provider||[]).forEach(function(p) { if (p.cost > maxCost) maxCost = p.cost; });
//   (data.by_provider||[]).forEach(function(p) {
//     var pct = maxCost > 0 ? (p.cost / maxCost * 100) : 0;
//     provHtml += '<div style="margin:4px 0"><div style="display:flex;justify-content:space-between"><span>' + p.provider + '</span><span>$' + p.cost.toFixed(4) + ' (' + p.requests + ' reqs)</span></div>';
//     provHtml += '<div style="background:#333;border-radius:3px;height:16px"><div style="background:#4a9eff;height:100%;width:' + pct + '%;border-radius:3px"></div></div></div>';
//   });
//   document.getElementById('analytics-provider').innerHTML = provHtml || '<p>No data</p>';
//
//   // Daily bar chart
//   var dayHtml = '';
//   var maxDayCost = 0;
//   (data.by_day||[]).forEach(function(d) { if (d.cost > maxDayCost) maxDayCost = d.cost; });
//   (data.by_day||[]).forEach(function(d) {
//     var pct = maxDayCost > 0 ? (d.cost / maxDayCost * 100) : 0;
//     dayHtml += '<div style="margin:4px 0"><div style="display:flex;justify-content:space-between"><span>' + d.date + '</span><span>$' + d.cost.toFixed(4) + '</span></div>';
//     dayHtml += '<div style="background:#333;border-radius:3px;height:16px"><div style="background:#4aff9e;height:100%;width:' + pct + '%;border-radius:3px"></div></div></div>';
//   });
//   document.getElementById('analytics-daily').innerHTML = dayHtml || '<p>No data</p>';
//
//   // Top models table
//   var modelHtml = '<table style="width:100%;border-collapse:collapse">';
//   modelHtml += '<tr style="border-bottom:1px solid #333"><th>Model</th><th>Provider</th><th>Cost</th><th>Requests</th></tr>';
//   (data.top_models||[]).forEach(function(m) {
//     modelHtml += '<tr style="border-bottom:1px solid #222"><td>' + m.model + '</td><td>' + m.provider + '</td><td>$' + m.cost.toFixed(4) + '</td><td>' + m.requests + '</td></tr>';
//   });
//   modelHtml += '</table>';
//   document.getElementById('analytics-models').innerHTML = modelHtml || '<p>No data</p>';
// }
// </script>
