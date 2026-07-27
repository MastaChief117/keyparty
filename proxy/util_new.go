package proxy

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	"ai-gateway/provider"
	"ai-gateway/store"
)

// ── WEBHOOK MANAGEMENT ──────────────────────────────────────────────────────

func (p *Proxy) HandleWebhooks(w http.ResponseWriter, r *http.Request) {

	path := r.URL.Path

	if path == "/admin/webhooks/test" && r.Method == http.MethodPost {
		p.handleWebhookTest(w, r)
		return
	}

	// Check for /admin/webhooks/:id
	parts := strings.TrimPrefix(path, "/admin/webhooks")
	if parts != "" && parts != "/" {
		idStr := strings.Trim(parts, "/")
		id, err := strconv.ParseInt(idStr, 10, 64)
		if err != nil {
			http.Error(w, `{"error":"Invalid webhook ID"}`, http.StatusBadRequest)
			return
		}
		if r.Method == http.MethodDelete {
			if err := p.store.DeleteWebhook(id); err != nil {
				http.Error(w, `{"error":"Failed to delete webhook"}`, http.StatusInternalServerError)
				return
			}
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(map[string]string{"status": "deleted"})
			return
		}
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	switch r.Method {
	case http.MethodGet:
		hooks, err := p.store.GetWebhooks()
		if err != nil {
			http.Error(w, `{"error":"Failed to get webhooks"}`, http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(hooks)

	case http.MethodPost:
		var req struct {
			URL    string `json:"url"`
			Events string `json:"events"`
			Secret string `json:"secret"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, `{"error":"Invalid JSON"}`, http.StatusBadRequest)
			return
		}
		if req.URL == "" {
			http.Error(w, `{"error":"URL is required"}`, http.StatusBadRequest)
			return
		}
		if req.Events == "" {
			req.Events = "*"
		}
		id, err := p.store.AddWebhook(req.URL, req.Events, req.Secret)
		if err != nil {
			http.Error(w, `{"error":"Failed to create webhook"}`, http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{"id": id, "status": "created"})

	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

func (p *Proxy) handleWebhookTest(w http.ResponseWriter, r *http.Request) {
	hooks, err := p.store.GetWebhooks()
	if err != nil || len(hooks) == 0 {
		http.Error(w, `{"error":"No webhooks configured"}`, http.StatusBadRequest)
		return
	}

	var reqBody struct {
		WebhookID int64 `json:"webhook_id"`
	}
	json.NewDecoder(r.Body).Decode(&reqBody)

	targetHooks := hooks
	if reqBody.WebhookID > 0 {
		for _, h := range hooks {
			if h.ID == reqBody.WebhookID {
				targetHooks = []store.Webhook{h}
				break
			}
		}
	}

	testData := map[string]interface{}{
		"event":     "test",
		"timestamp": time.Now().Format(time.RFC3339),
		"message":   "This is a test webhook delivery from AI Gateway",
		"data": map[string]string{
			"type":    "test",
			"gateway": "ai-gateway",
		},
	}

	var results []map[string]interface{}
	for _, h := range targetHooks {
		status, resp := p.sendWebhook(h, "test", testData)
		results = append(results, map[string]interface{}{
			"webhook_id": h.ID,
			"url":        h.URL,
			"status":     status,
			"response":   resp,
		})
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"results": results,
	})
}

func (p *Proxy) sendWebhook(h store.Webhook, event string, data map[string]interface{}) (int, string) {
	payload := map[string]interface{}{
		"event":     event,
		"timestamp": time.Now().Format(time.RFC3339),
		"data":      data,
	}
	bodyBytes, _ := json.Marshal(payload)

	req, err := http.NewRequest("POST", h.URL, bytes.NewReader(bodyBytes))
	if err != nil {
		p.store.LogWebhookDelivery(h.ID, event, 0, err.Error())
		return 0, err.Error()
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Webhook-Event", event)
	req.Header.Set("User-Agent", "AI-Gateway-Webhook/1.0")
	if h.Secret != "" {
		req.Header.Set("X-Webhook-Secret", h.Secret)
	}

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		p.store.LogWebhookDelivery(h.ID, event, 0, err.Error())
		return 0, err.Error()
	}
	defer resp.Body.Close()

	respBody, _ := io.ReadAll(resp.Body)
	respStr := string(respBody)
	if len(respStr) > 200 {
		respStr = respStr[:200]
	}

	p.store.LogWebhookDelivery(h.ID, event, resp.StatusCode, respStr)
	return resp.StatusCode, respStr
}

func (p *Proxy) FireWebhooks(event string, data map[string]interface{}) {
	go func() {
		hooks := p.store.GetActiveWebhooks()
		var wg sync.WaitGroup
		for _, h := range hooks {
			if !p.store.ShouldFireWebhook(h, event) {
				continue
			}
			wg.Add(1)
			go func(hook store.Webhook) {
				defer wg.Done()
				p.sendWebhook(hook, event, data)
			}(h)
		}
		wg.Wait()
	}()
}

// ── REQUEST LOG SEARCH ──────────────────────────────────────────────────────

func (p *Proxy) HandleSearchLogs(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}


	q := r.URL.Query()
	provider := q.Get("provider")
	model := q.Get("model")
	vk := q.Get("virtual_key")
	status := q.Get("status")
	limit := 50
	if l := q.Get("limit"); l != "" {
		if parsed, err := strconv.Atoi(l); err == nil && parsed > 0 {
			limit = parsed
		}
	}

	logs, err := p.store.SearchRequestLogs(provider, model, vk, status, limit)
	if err != nil {
		http.Error(w, `{"error":"Failed to search logs"}`, http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(logs)
}

// ── PROMPT TEMPLATES ────────────────────────────────────────────────────────

func (p *Proxy) HandleTemplates(w http.ResponseWriter, r *http.Request) {

	path := r.URL.Path
	parts := strings.TrimPrefix(path, "/admin/templates")
	if parts != "" && parts != "/" {
		name := strings.Trim(parts, "/")
		if r.Method == http.MethodDelete {
			if err := p.store.DeleteTemplate(name); err != nil {
				http.Error(w, `{"error":"Failed to delete template"}`, http.StatusInternalServerError)
				return
			}
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(map[string]string{"status": "deleted"})
			return
		}
		if r.Method == http.MethodGet {
			t, err := p.store.GetTemplateByName(name)
			if err != nil {
				http.Error(w, `{"error":"Template not found"}`, http.StatusNotFound)
				return
			}
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(t)
			return
		}
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	switch r.Method {
	case http.MethodGet:
		templates, err := p.store.GetTemplates()
		if err != nil {
			http.Error(w, `{"error":"Failed to get templates"}`, http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(templates)

	case http.MethodPost:
		var req struct {
			Name         string `json:"name"`
			SystemPrompt string `json:"system_prompt"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, `{"error":"Invalid JSON"}`, http.StatusBadRequest)
			return
		}
		if req.Name == "" {
			http.Error(w, `{"error":"Name is required"}`, http.StatusBadRequest)
			return
		}
		id, err := p.store.AddTemplate(req.Name, req.SystemPrompt)
		if err != nil {
			http.Error(w, `{"error":"Failed to save template"}`, http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{"id": id, "status": "created"})

	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

// ── AI POLL / DEMOCRACY ─────────────────────────────────────────────────────

func (p *Proxy) HandlePoll(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var reqBody struct {
		Message   string   `json:"message"`
		Providers []string `json:"providers"`
		Model     string   `json:"model"`
	}
	if err := json.NewDecoder(r.Body).Decode(&reqBody); err != nil {
		http.Error(w, "Invalid JSON", 400)
		return
	}

	if reqBody.Message == "" {
		http.Error(w, `{"error":"message is required"}`, 400)
		return
	}

	if len(reqBody.Providers) == 0 {
		reqBody.Providers = []string{"groq", "nvidia", "deepseek"}
	}

	type pollResult struct {
		Provider string `json:"provider"`
		Model    string `json:"model"`
		Reply    string `json:"reply"`
		Latency  int64  `json:"latency_ms"`
		Error    string `json:"error,omitempty"`
	}

	var results []pollResult
	var mu sync.Mutex
	var wg sync.WaitGroup

	for _, provName := range reqBody.Providers {
		wg.Add(1)
		go func(pn string) {
			defer wg.Done()

			keys, err := p.store.GetKeysByProvider(pn)
			if err != nil || len(keys) == 0 {
				mu.Lock()
				results = append(results, pollResult{Provider: pn, Error: "No keys configured"})
				mu.Unlock()
				return
			}

			key := keys[0]
			provDef, ok := provider.GetByName(pn)
			if !ok {
				mu.Lock()
				results = append(results, pollResult{Provider: pn, Error: "Unknown provider"})
				mu.Unlock()
				return
			}

			baseURL := provDef.BaseURL
			if !strings.HasSuffix(baseURL, "/") {
				baseURL += "/"
			}
			endpoint := baseURL + "chat/completions"

			model := reqBody.Model
			if model == "" {
				model = getDefaultModel(pn)
			}

			body := map[string]interface{}{
				"model": model,
				"messages": []map[string]string{
					{"role": "user", "content": reqBody.Message},
				},
				"max_tokens": 500,
				"temperature": 0.9,
			}
			bodyBytes, _ := json.Marshal(body)

			start := time.Now()
			proxyReq, err := http.NewRequest("POST", endpoint, bytes.NewReader(bodyBytes))
			if err != nil {
				mu.Lock()
				results = append(results, pollResult{Provider: pn, Error: "Request creation failed"})
				mu.Unlock()
				return
			}

			proxyReq.Header.Set("Content-Type", "application/json")
			proxyReq.Header.Set("Authorization", "Bearer "+key.Key)
			if pn == "anthropic" {
				proxyReq.Header.Set("x-api-key", key.Key)
				proxyReq.Header.Set("anthropic-version", "2023-06-01")
			}

			resp, err := p.client.Do(proxyReq)
			latency := time.Since(start).Milliseconds()
			if err != nil {
				mu.Lock()
				results = append(results, pollResult{Provider: pn, Latency: latency, Error: err.Error()})
				mu.Unlock()
				return
			}
			defer resp.Body.Close()

			respBody, _ := io.ReadAll(resp.Body)
			if resp.StatusCode >= 400 {
				mu.Lock()
				results = append(results, pollResult{Provider: pn, Latency: latency, Error: fmt.Sprintf("HTTP %d", resp.StatusCode)})
				mu.Unlock()
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
				reply = chatResp.Choices[0].Message.Content
			}

			mu.Lock()
			results = append(results, pollResult{
				Provider: pn,
				Model:    model,
				Reply:    reply,
				Latency:  latency,
			})
			mu.Unlock()
		}(provName)
	}

	wg.Wait()

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"results": results,
		"message": reqBody.Message,
	})
}

// ── RATE LIMIT TIER MANAGEMENT ──────────────────────────────────────────────

func (p *Proxy) HandleRateLimitTiers(w http.ResponseWriter, r *http.Request) {

	path := r.URL.Path
	parts := strings.TrimPrefix(path, "/admin/rate-tiers")
	if parts != "" && parts != "/" {
		idStr := strings.Trim(parts, "/")
		id, err := strconv.ParseInt(idStr, 10, 64)
		if err != nil {
			http.Error(w, `{"error":"Invalid tier ID"}`, http.StatusBadRequest)
			return
		}
		if r.Method == http.MethodDelete {
			if err := p.store.DeleteRateLimitTier(id); err != nil {
				http.Error(w, `{"error":"Failed to delete tier"}`, http.StatusInternalServerError)
				return
			}
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(map[string]string{"status": "deleted"})
			return
		}
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	switch r.Method {
	case http.MethodGet:
		tiers, err := p.store.GetRateLimitTiers()
		if err != nil {
			http.Error(w, `{"error":"Failed to get tiers"}`, http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(tiers)

	case http.MethodPost:
		var req struct {
			Name            string  `json:"name"`
			RequestsPerMin  int     `json:"requests_per_minute"`
			RequestsPerDay  int     `json:"requests_per_day"`
			MonthlyBudget   float64 `json:"monthly_budget"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, `{"error":"Invalid JSON"}`, http.StatusBadRequest)
			return
		}
		if req.Name == "" {
			http.Error(w, `{"error":"Name is required"}`, http.StatusBadRequest)
			return
		}
		id, err := p.store.AddRateLimitTier(req.Name, req.RequestsPerMin, req.RequestsPerDay, req.MonthlyBudget)
		if err != nil {
			http.Error(w, `{"error":"Failed to save tier"}`, http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{"id": id, "status": "created"})

	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

// ── BUDGET ALERT MANAGEMENT ─────────────────────────────────────────────────

func (p *Proxy) HandleBudgetAlerts(w http.ResponseWriter, r *http.Request) {

	path := r.URL.Path

	// /admin/budget-alerts/check
	if path == "/admin/budget-alerts/check" {
		if r.Method != http.MethodGet {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}
		triggered := p.store.CheckBudgetAlerts()
		for _, a := range triggered {
			p.FireWebhooks("budget_alert", map[string]interface{}{
				"alert_id":         a.ID,
				"virtual_key_id":   a.VirtualKeyID,
				"virtual_key_name": a.VirtualKeyName,
				"threshold":        a.ThresholdPercent,
				"message":          fmt.Sprintf("Budget alert triggered: key '%s' has exceeded %d%% of its monthly budget", a.VirtualKeyName, a.ThresholdPercent),
			})
			p.store.MarkAlertNotified(a.ID)
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(triggered)
		return
	}

	// Check for /admin/budget-alerts/:id
	parts := strings.TrimPrefix(path, "/admin/budget-alerts")
	if parts != "" && parts != "/" {
		idStr := strings.Trim(parts, "/")
		id, err := strconv.ParseInt(idStr, 10, 64)
		if err != nil {
			http.Error(w, `{"error":"Invalid alert ID"}`, http.StatusBadRequest)
			return
		}
		if r.Method == http.MethodDelete {
			if err := p.store.DeleteBudgetAlert(id); err != nil {
				http.Error(w, `{"error":"Failed to delete alert"}`, http.StatusInternalServerError)
				return
			}
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(map[string]string{"status": "deleted"})
			return
		}
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	switch r.Method {
	case http.MethodGet:
		alerts, err := p.store.GetBudgetAlerts()
		if err != nil {
			http.Error(w, `{"error":"Failed to get alerts"}`, http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(alerts)

	case http.MethodPost:
		var req struct {
			VirtualKeyID     int64 `json:"virtual_key_id"`
			ThresholdPercent int   `json:"threshold_percent"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, `{"error":"Invalid JSON"}`, http.StatusBadRequest)
			return
		}
		if req.VirtualKeyID == 0 {
			http.Error(w, `{"error":"virtual_key_id is required"}`, http.StatusBadRequest)
			return
		}
		if req.ThresholdPercent <= 0 || req.ThresholdPercent > 100 {
			req.ThresholdPercent = 80
		}
		id, err := p.store.AddBudgetAlert(req.VirtualKeyID, req.ThresholdPercent)
		if err != nil {
			http.Error(w, `{"error":"Failed to save alert"}`, http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{"id": id, "status": "created"})

	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

// ── ROUTES TO ADD ───────────────────────────────────────────────────────────
// Add these lines in main.go after the existing mux.HandleFunc("/admin/rumble", ...) line:

/*
	mux.HandleFunc("/admin/webhooks", p.HandleWebhooks)
	mux.HandleFunc("/admin/webhooks/", p.HandleWebhooks)
	mux.HandleFunc("/admin/logs/search", p.HandleSearchLogs)
	mux.HandleFunc("/admin/templates", p.HandleTemplates)
	mux.HandleFunc("/admin/templates/", p.HandleTemplates)
	mux.HandleFunc("/admin/poll", p.HandlePoll)
	mux.HandleFunc("/admin/rate-tiers", p.HandleRateLimitTiers)
	mux.HandleFunc("/admin/rate-tiers/", p.HandleRateLimitTiers)
	mux.HandleFunc("/admin/budget-alerts", p.HandleBudgetAlerts)
	mux.HandleFunc("/admin/budget-alerts/", p.HandleBudgetAlerts)
*/

// ── HTML TO ADD ─────────────────────────────────────────────────────────────
// Add these new tab buttons and tab content panels to the dashboard HTML.
// Insert the tab buttons after the existing tab buttons, and the tab content
// panels after the existing tab content panels.

/*
--- TAB BUTTONS (add after existing tab buttons in the nav) ---
<button class="tab-btn" onclick="switchTab('webhooks')">Webhooks</button>
<button class="tab-btn" onclick="switchTab('searchlogs')">Search Logs</button>
<button class="tab-btn" onclick="switchTab('templates')">Templates</button>
<button class="tab-btn" onclick="switchTab('poll')">AI Poll</button>
<button class="tab-btn" onclick="switchTab('ratelimits')">Rate Limits</button>
<button class="tab-btn" onclick="switchTab('budgets')">Budget Alerts</button>

--- TAB CONTENT (add as new tab-content divs) ---

<div id="tab-webhooks" class="tab-content" style="display:none">
  <h3>Webhook Management</h3>
  <div style="margin-bottom:12px">
    <input id="wh-url" placeholder="Webhook URL" style="width:300px">
    <input id="wh-events" placeholder="Events (comma-sep or *)" style="width:200px" value="*">
    <button onclick="addWebhook()">Add Webhook</button>
  </div>
  <button onclick="testWebhooks()" style="margin-bottom:12px">Test All Webhooks</button>
  <div id="webhooks-list"></div>
</div>

<div id="tab-searchlogs" class="tab-content" style="display:none">
  <h3>Search Request Logs</h3>
  <div style="margin-bottom:12px">
    <input id="sl-provider" placeholder="Provider" style="width:120px">
    <input id="sl-model" placeholder="Model" style="width:150px">
    <input id="sl-vk" placeholder="Virtual Key" style="width:150px">
    <input id="sl-status" placeholder="Status Code" style="width:80px">
    <input id="sl-limit" placeholder="Limit" style="width:60px" value="50">
    <button onclick="searchLogs()">Search</button>
  </div>
  <div id="searchlogs-results"></div>
</div>

<div id="tab-templates" class="tab-content" style="display:none">
  <h3>Prompt Templates</h3>
  <div style="margin-bottom:12px">
    <input id="tpl-name" placeholder="Template Name" style="width:200px">
    <input id="tpl-prompt" placeholder="System Prompt" style="width:400px">
    <button onclick="addTemplate()">Save Template</button>
  </div>
  <div id="templates-list"></div>
</div>

<div id="tab-poll" class="tab-content" style="display:none">
  <h3>AI Poll / Democracy</h3>
  <div style="margin-bottom:12px">
    <input id="poll-msg" placeholder="Message to send to all providers" style="width:400px">
    <input id="poll-providers" placeholder="Providers (comma-sep)" style="width:250px" value="groq,nvidia,deepseek">
    <button onclick="runPoll()">Send Poll</button>
  </div>
  <div id="poll-results"></div>
</div>

<div id="tab-ratelimits" class="tab-content" style="display:none">
  <h3>Rate Limit Tiers</h3>
  <div style="margin-bottom:12px">
    <input id="rl-name" placeholder="Tier Name" style="width:120px">
    <input id="rl-rpm" placeholder="Requests/Min" style="width:100px" type="number">
    <input id="rl-rpd" placeholder="Requests/Day" style="width:100px" type="number">
    <input id="rl-budget" placeholder="Monthly Budget" style="width:100px" type="number" step="0.01">
    <button onclick="addRateTier()">Add Tier</button>
  </div>
  <div id="ratelimits-list"></div>
</div>

<div id="tab-budgets" class="tab-content" style="display:none">
  <h3>Budget Alerts</h3>
  <div style="margin-bottom:12px">
    <input id="ba-vkid" placeholder="Virtual Key ID" style="width:120px" type="number">
    <input id="ba-threshold" placeholder="Threshold %" style="width:100px" type="number" value="80">
    <button onclick="addBudgetAlert()">Add Alert</button>
    <button onclick="checkBudgetAlerts()">Check Alerts</button>
  </div>
  <div id="budgets-list"></div>
</div>

--- JAVASCRIPT FUNCTIONS (add to existing script block) ---

async function addWebhook() {
  const url = document.getElementById('wh-url').value;
  const events = document.getElementById('wh-events').value;
  if (!url) return alert('URL required');
  await apiCall('/admin/webhooks', 'POST', {url, events});
  document.getElementById('wh-url').value = '';
  loadWebhooks();
}

async function loadWebhooks() {
  const data = await apiCall('/admin/webhooks');
  let html = '<table><tr><th>ID</th><th>URL</th><th>Events</th><th>Enabled</th><th>Actions</th></tr>';
  (data || []).forEach(w => {
    html += `<tr><td>${w.id}</td><td>${w.url}</td><td>${w.events}</td><td>${w.enabled}</td>
      <td><button onclick="deleteWebhook(${w.id})">Delete</button></td></tr>`;
  });
  html += '</table>';
  document.getElementById('webhooks-list').innerHTML = html;
}

async function deleteWebhook(id) {
  await apiCall('/admin/webhooks/' + id, 'DELETE');
  loadWebhooks();
}

async function testWebhooks() {
  const data = await apiCall('/admin/webhooks/test', 'POST', {});
  alert(JSON.stringify(data.results, null, 2));
}

async function searchLogs() {
  const p = document.getElementById('sl-provider').value;
  const m = document.getElementById('sl-model').value;
  const vk = document.getElementById('sl-vk').value;
  const s = document.getElementById('sl-status').value;
  const l = document.getElementById('sl-limit').value;
  const params = new URLSearchParams();
  if (p) params.set('provider', p);
  if (m) params.set('model', m);
  if (vk) params.set('virtual_key', vk);
  if (s) params.set('status', s);
  if (l) params.set('limit', l);
  const data = await apiCall('/admin/logs/search?' + params.toString());
  let html = '<table><tr><th>ID</th><th>Provider</th><th>Model</th><th>Status</th><th>Cost</th><th>Latency</th><th>Time</th></tr>';
  (data || []).forEach(l => {
    html += `<tr><td>${l.id}</td><td>${l.provider}</td><td>${l.model}</td><td>${l.status_code}</td>
      <td>$${l.cost.toFixed(6)}</td><td>${l.latency_ms}ms</td><td>${l.timestamp}</td></tr>`;
  });
  html += '</table>';
  document.getElementById('searchlogs-results').innerHTML = html;
}

async function addTemplate() {
  const name = document.getElementById('tpl-name').value;
  const system_prompt = document.getElementById('tpl-prompt').value;
  if (!name) return alert('Name required');
  await apiCall('/admin/templates', 'POST', {name, system_prompt});
  document.getElementById('tpl-name').value = '';
  document.getElementById('tpl-prompt').value = '';
  loadTemplates();
}

async function loadTemplates() {
  const data = await apiCall('/admin/templates');
  let html = '<table><tr><th>Name</th><th>System Prompt</th><th>Actions</th></tr>';
  (data || []).forEach(t => {
    html += `<tr><td>${t.name}</td><td style="max-width:400px;overflow:hidden;text-overflow:ellipsis">${t.system_prompt}</td>
      <td><button onclick="deleteTemplate('${t.name}')">Delete</button></td></tr>`;
  });
  html += '</table>';
  document.getElementById('templates-list').innerHTML = html;
}

async function deleteTemplate(name) {
  await apiCall('/admin/templates/' + encodeURIComponent(name), 'DELETE');
  loadTemplates();
}

async function runPoll() {
  const message = document.getElementById('poll-msg').value;
  const providers = document.getElementById('poll-providers').value.split(',').map(s => s.trim());
  if (!message) return alert('Message required');
  document.getElementById('poll-results').innerHTML = '<p>Running poll across providers...</p>';
  const data = await apiCall('/admin/poll', 'POST', {message, providers});
  let html = '<h4>Results</h4>';
  (data.results || []).forEach(r => {
    html += `<div style="border:1px solid #444;padding:10px;margin:8px 0;border-radius:6px">
      <strong>${r.provider}</strong> (${r.model}) - ${r.latency_ms}ms
      ${r.error ? '<span style="color:red"> ERROR: ' + r.error + '</span>' : ''}
      <pre style="white-space:pre-wrap;margin-top:6px;background:#1a1a2e;padding:8px;border-radius:4px">${r.reply || ''}</pre>
    </div>`;
  });
  document.getElementById('poll-results').innerHTML = html;
}

async function addRateTier() {
  const name = document.getElementById('rl-name').value;
  const requests_per_minute = parseInt(document.getElementById('rl-rpm').value) || 0;
  const requests_per_day = parseInt(document.getElementById('rl-rpd').value) || 0;
  const monthly_budget = parseFloat(document.getElementById('rl-budget').value) || 0;
  if (!name) return alert('Name required');
  await apiCall('/admin/rate-tiers', 'POST', {name, requests_per_minute, requests_per_day, monthly_budget});
  document.getElementById('rl-name').value = '';
  loadRateTiers();
}

async function loadRateTiers() {
  const data = await apiCall('/admin/rate-tiers');
  let html = '<table><tr><th>ID</th><th>Name</th><th>Req/Min</th><th>Req/Day</th><th>Budget</th><th>Actions</th></tr>';
  (data || []).forEach(t => {
    html += `<tr><td>${t.id}</td><td>${t.name}</td><td>${t.requests_per_minute}</td><td>${t.requests_per_day}</td>
      <td>$${t.monthly_budget}</td><td><button onclick="deleteRateTier(${t.id})">Delete</button></td></tr>`;
  });
  html += '</table>';
  document.getElementById('ratelimits-list').innerHTML = html;
}

async function deleteRateTier(id) {
  await apiCall('/admin/rate-tiers/' + id, 'DELETE');
  loadRateTiers();
}

async function addBudgetAlert() {
  const virtual_key_id = parseInt(document.getElementById('ba-vkid').value);
  const threshold_percent = parseInt(document.getElementById('ba-threshold').value) || 80;
  if (!virtual_key_id) return alert('Virtual Key ID required');
  await apiCall('/admin/budget-alerts', 'POST', {virtual_key_id, threshold_percent});
  loadBudgetAlerts();
}

async function loadBudgetAlerts() {
  const data = await apiCall('/admin/budget-alerts');
  let html = '<table><tr><th>ID</th><th>Key ID</th><th>Key Name</th><th>Threshold</th><th>Notified</th><th>Actions</th></tr>';
  (data || []).forEach(a => {
    html += `<tr><td>${a.id}</td><td>${a.virtual_key_id}</td><td>${a.virtual_key_name}</td>
      <td>${a.threshold_percent}%</td><td>${a.notified ? 'Yes' : 'No'}</td>
      <td><button onclick="deleteBudgetAlert(${a.id})">Delete</button></td></tr>`;
  });
  html += '</table>';
  document.getElementById('budgets-list').innerHTML = html;
}

async function deleteBudgetAlert(id) {
  await apiCall('/admin/budget-alerts/' + id, 'DELETE');
  loadBudgetAlerts();
}

async function checkBudgetAlerts() {
  const data = await apiCall('/admin/budget-alerts/check');
  if (!data || data.length === 0) {
    alert('No triggered alerts');
  } else {
    alert(data.map(a => `Alert #${a.id}: Key "${a.virtual_key_name}" exceeded ${a.threshold_percent}% budget`).join('\n'));
  }
}

// Update the switchTab function to also load data for new tabs:
// Add cases in the existing switchTab function:
//   case 'webhooks': loadWebhooks(); break;
//   case 'templates': loadTemplates(); break;
//   case 'ratelimits': loadRateTiers(); break;
//   case 'budgets': loadBudgetAlerts(); break;
*/
