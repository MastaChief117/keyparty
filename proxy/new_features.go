package proxy

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"math"
	"net/http"
	"regexp"
	"strings"
	"sync"
	"time"

	"ai-gateway/provider"
	"ai-gateway/store"
)

// ── PROMPT FIREWALL ───────────────────────────────────────────────────────

var outboundPIIPatterns = map[string]*regexp.Regexp{
	"email":  regexp.MustCompile(`[a-zA-Z0-9._%+\-]+@[a-zA-Z0-9.\-]+\.[a-zA-Z]{2,}`),
	"phone":  regexp.MustCompile(`\b\d{3}[-.]?\d{3}[-.]?\d{4}\b`),
	"ssn":    regexp.MustCompile(`\b\d{3}-\d{2}-\d{4}\b`),
	"credit": regexp.MustCompile(`\b\d{4}[\s-]?\d{4}[\s-]?\d{4}[\s-]?\d{4}\b`),
	"api_key": regexp.MustCompile(`(?i)(api[_-]?key|secret[_-]?key|access[_-]?key|bearer)\s*[:=]\s*["']?[A-Za-z0-9\-_\.]{20,}`),
}

var outboundSecretPatterns = []*regexp.Regexp{
	regexp.MustCompile(`(?i)(password|passwd|pwd)\s*[:=]\s*["']?[^\s"']{8,}`),
	regexp.MustCompile(`(?i)(private[_-]?key)\s*[:=]\s*["']?[A-Za-z0-9\-_\.]{20,}`),
}

type FirewallResult struct {
	Blocked   bool     `json:"blocked"`
	Patterns  []string `json:"patterns,omitempty"`
	Redacted  string   `json:"redacted,omitempty"`
}

func (p *Proxy) CheckOutboundFirewall(message string, config map[string]string) FirewallResult {
	if config["enabled"] != "true" {
		return FirewallResult{Blocked: false}
	}

	var matched []string
	redacted := message

	for name, pattern := range outboundPIIPatterns {
		if pattern.MatchString(message) {
			matched = append(matched, name)
			redacted = pattern.ReplaceAllString(redacted, "[REDACTED_"+strings.ToUpper(name)+"]")
		}
	}

	for _, pattern := range outboundSecretPatterns {
		if pattern.MatchString(message) {
			matched = append(matched, "secret")
			redacted = pattern.ReplaceAllString(redacted, "[REDACTED_SECRET]")
		}
	}

	if len(matched) > 0 && config["action"] == "block" {
		return FirewallResult{Blocked: true, Patterns: matched}
	}

	if len(matched) > 0 {
		return FirewallResult{Blocked: false, Patterns: matched, Redacted: redacted}
	}

	return FirewallResult{Blocked: false}
}

func (p *Proxy) GetFirewallConfig() map[string]string {
	config := map[string]string{
		"enabled": "false",
		"action":  "warn",
	}
	rows, err := p.store.GetFirewallConfig()
	if err != nil {
		return config
	}
	for k, v := range rows {
		config[k] = v
	}
	return config
}

func (p *Proxy) SetFirewallConfig(key, value string) error {
	return p.store.SetFirewallConfig(key, value)
}

// ── IP ALLOWLIST ──────────────────────────────────────────────────────────

// ── PROVIDER BUDGETS ──────────────────────────────────────────────────────

func (p *Proxy) HandleProviderBudgets(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	switch r.Method {
	case "GET":
		budgets, err := p.store.GetProviderBudgets()
		if err != nil {
			http.Error(w, `{"error":"Failed"}`, 500)
			return
		}
		json.NewEncoder(w).Encode(budgets)
	case "POST":
		var req struct {
			Provider      string  `json:"provider"`
			MonthlyTokens int64   `json:"monthly_tokens"`
			MonthlyCost   float64 `json:"monthly_cost"`
			ResetDay      int     `json:"reset_day"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, `{"error":"Invalid JSON"}`, 400)
			return
		}
		if req.Provider == "" {
			http.Error(w, `{"error":"provider required"}`, 400)
			return
		}
		id, err := p.store.AddProviderBudget(req.Provider, req.MonthlyTokens, req.MonthlyCost, req.ResetDay)
		if err != nil {
			http.Error(w, `{"error":"Failed"}`, 500)
			return
		}
		json.NewEncoder(w).Encode(map[string]interface{}{"id": id, "status": "created"})
	case "DELETE":
		var req struct {
			ID int64 `json:"id"`
		}
		json.NewDecoder(r.Body).Decode(&req)
		if err := p.store.DeleteProviderBudget(req.ID); err != nil {
			http.Error(w, `{"error":"Failed"}`, 500)
			return
		}
		json.NewEncoder(w).Encode(map[string]string{"status": "deleted"})
	default:
		http.Error(w, "Method not allowed", 405)
	}
}

// ── REQUEST QUEUING ───────────────────────────────────────────────────────

func (p *Proxy) HandleQueue(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	switch r.Method {
	case "GET":
		path := r.URL.Path
		if strings.HasSuffix(path, "/stats") {
			stats, err := p.store.GetQueueStats()
			if err != nil {
				http.Error(w, `{"error":"Failed"}`, 500)
				return
			}
			json.NewEncoder(w).Encode(stats)
			return
		}
		logs, err := p.store.GetQueueLogs(50)
		if err != nil {
			http.Error(w, `{"error":"Failed"}`, 500)
			return
		}
		json.NewEncoder(w).Encode(logs)
	case "POST":
		var req struct {
			VirtualKey string `json:"virtual_key"`
			Provider   string `json:"provider"`
			Model      string `json:"model"`
			Body       string `json:"body"`
			Priority   int    `json:"priority"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, `{"error":"Invalid JSON"}`, 400)
			return
		}
		if req.Provider == "" || req.Body == "" {
			http.Error(w, `{"error":"provider and body required"}`, 400)
			return
		}
		id, err := p.store.EnqueueRequest(req.VirtualKey, req.Provider, req.Model, req.Body, req.Priority, 3)
		if err != nil {
			http.Error(w, `{"error":"Failed"}`, 500)
			return
		}
		json.NewEncoder(w).Encode(map[string]interface{}{"id": id, "status": "queued"})
	default:
		http.Error(w, "Method not allowed", 405)
	}
}

func (p *Proxy) ProcessQueue() {
	ticker := time.NewTicker(5 * time.Second)
	for range ticker.C {
		reqs, err := p.store.DequeueRequests(5)
		if err != nil || len(reqs) == 0 {
			continue
		}
		for _, req := range reqs {
			go p.processQueuedRequest(req)
		}
	}
}

func (p *Proxy) processQueuedRequest(req store.QueuedRequest) {
	p.store.MarkQueueProcessing(req.ID)

	keys, err := p.store.GetKeysByProvider(req.Provider)
	if err != nil || len(keys) == 0 {
		p.store.MarkQueueFailed(req.ID, "No keys for provider", 30)
		return
	}

	key := keys[0]
	provDef, ok := provider.GetByName(req.Provider)
	if !ok {
		p.store.MarkQueueFailed(req.ID, "Unknown provider", 30)
		return
	}

	baseURL := provDef.BaseURL
	if !strings.HasSuffix(baseURL, "/") {
		baseURL += "/"
	}
	endpoint := baseURL + "chat/completions"

	proxyReq, err := http.NewRequest("POST", endpoint, strings.NewReader(req.Body))
	if err != nil {
		p.store.MarkQueueFailed(req.ID, err.Error(), 30)
		return
	}

	proxyReq.Header.Set("Content-Type", "application/json")
	proxyReq.Header.Set("Authorization", "Bearer "+key.Key)
	if req.Provider == "anthropic" {
		proxyReq.Header.Set("x-api-key", key.Key)
		proxyReq.Header.Set("anthropic-version", "2023-06-01")
	}

	resp, err := p.client.Do(proxyReq)
	if err != nil {
		p.store.MarkQueueFailed(req.ID, err.Error(), 30)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusTooManyRequests {
		p.store.MarkQueueFailed(req.ID, "Rate limited", 60)
		return
	}

	if resp.StatusCode >= 400 {
		body, _ := io.ReadAll(resp.Body)
		p.store.MarkQueueFailed(req.ID, fmt.Sprintf("HTTP %d: %s", resp.StatusCode, string(body[:min(len(body), 200)])), 30)
		return
	}

	p.store.MarkQueueDone(req.ID)
}

// ── HEALTH DASHBOARD ──────────────────────────────────────────────────────

func (p *Proxy) HandleHealthDashboard(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", 405)
		return
	}

	health, err := p.store.GetProviderHealth()
	if err != nil {
		http.Error(w, `{"error":"Failed"}`, 500)
		return
	}

	keys, _ := p.store.GetKeys()
	keyCount := make(map[string]int)
	for _, k := range keys {
		if k.Enabled {
			keyCount[k.Provider]++
		}
	}

	type healthWithKeys struct {
		store.ProviderHealth
		ActiveKeys int `json:"active_keys"`
	}

	var result []healthWithKeys
	for _, h := range health {
		result = append(result, healthWithKeys{
			ProviderHealth: h,
			ActiveKeys:     keyCount[h.Provider],
		})
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"providers": result,
		"total":     len(result),
	})
}

// ── A/B TESTING ───────────────────────────────────────────────────────────

func (p *Proxy) HandleABTest(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	switch r.Method {
	case "GET":
		tests, err := p.store.GetABTests(20)
		if err != nil {
			http.Error(w, `{"error":"Failed"}`, 500)
			return
		}
		json.NewEncoder(w).Encode(tests)
	case "POST":
		var req struct {
			Name      string `json:"name"`
			Prompt    string `json:"prompt"`
			ProviderA string `json:"provider_a"`
			ModelA    string `json:"model_a"`
			ProviderB string `json:"provider_b"`
			ModelB    string `json:"model_b"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, `{"error":"Invalid JSON"}`, 400)
			return
		}
		if req.Prompt == "" {
			req.Prompt = "Explain quantum computing in one paragraph."
		}
		if req.Name == "" {
			req.Name = fmt.Sprintf("A/B Test #%d", time.Now().Unix()%10000)
		}

		id, err := p.store.CreateABTest(req.Name, req.Prompt, req.ProviderA, req.ModelA, req.ProviderB, req.ModelB)
		if err != nil {
			http.Error(w, `{"error":"Failed"}`, 500)
			return
		}

		go p.runABTest(id, req)

		json.NewEncoder(w).Encode(map[string]interface{}{
			"id":     id,
			"status": "started",
			"name":   req.Name,
		})
	default:
		http.Error(w, "Method not allowed", 405)
	}
}

func (p *Proxy) runABTest(id int64, req struct {
	Name      string `json:"name"`
	Prompt    string `json:"prompt"`
	ProviderA string `json:"provider_a"`
	ModelA    string `json:"model_a"`
	ProviderB string `json:"provider_b"`
	ModelB    string `json:"model_b"`
}) {
	messages := []map[string]string{{"role": "user", "content": req.Prompt}}

	type result struct {
		Reply   string
		Tokens  int
		Latency int64
		Error   string
	}

	var resultA, resultB result
	var wg sync.WaitGroup

	wg.Add(2)
	go func() {
		defer wg.Done()
		keys, err := p.store.GetKeysByProvider(req.ProviderA)
		if err != nil || len(keys) == 0 {
			resultA.Error = "No keys"
			return
		}
		start := time.Now()
		reply, tokensIn, tokensOut, err := p.callProviderForPro(req.ProviderA, keys[0], req.ModelA, messages)
		resultA.Latency = time.Since(start).Milliseconds()
		if err != nil {
			resultA.Error = err.Error()
		} else {
			resultA.Reply = reply
			resultA.Tokens = tokensIn + tokensOut
		}
	}()
	go func() {
		defer wg.Done()
		keys, err := p.store.GetKeysByProvider(req.ProviderB)
		if err != nil || len(keys) == 0 {
			resultB.Error = "No keys"
			return
		}
		start := time.Now()
		reply, tokensIn, tokensOut, err := p.callProviderForPro(req.ProviderB, keys[0], req.ModelB, messages)
		resultB.Latency = time.Since(start).Milliseconds()
		if err != nil {
			resultB.Error = err.Error()
		} else {
			resultB.Reply = reply
			resultB.Tokens = tokensIn + tokensOut
		}
	}()
	wg.Wait()

	replyA := resultA.Reply
	if resultA.Error != "" {
		replyA = "ERROR: " + resultA.Error
	}
	replyB := resultB.Reply
	if resultB.Error != "" {
		replyB = "ERROR: " + resultB.Error
	}

	p.store.UpdateABTestReplies(id, replyA, replyB, resultA.Tokens, resultB.Tokens, resultA.Latency, resultB.Latency)
}

func (p *Proxy) HandleABVote(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", 405)
		return
	}

	var req struct {
		TestID int64  `json:"test_id"`
		Winner string `json:"winner"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, `{"error":"Invalid JSON"}`, 400)
		return
	}

	if req.Winner != "a" && req.Winner != "b" {
		http.Error(w, `{"error":"winner must be 'a' or 'b'"}`, 400)
		return
	}

	if err := p.store.VoteABTest(req.TestID, req.Winner); err != nil {
		http.Error(w, `{"error":"Failed"}`, 500)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "voted"})
}

// ── SEMANTIC CACHING ──────────────────────────────────────────────────────

func (p *Proxy) HandleSemanticCache(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	switch r.Method {
	case "GET":
		entries, err := p.store.GetSemanticCacheEntries()
		if err != nil {
			http.Error(w, `{"error":"Failed"}`, 500)
			return
		}
		json.NewEncoder(w).Encode(map[string]interface{}{
			"entries": entries,
			"total":   len(entries),
		})
	case "POST":
		var req struct {
			Query    string  `json:"query"`
			Response string  `json:"response"`
			Model    string  `json:"model"`
			TTL      int     `json:"ttl_minutes"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, `{"error":"Invalid JSON"}`, 400)
			return
		}
		if req.Query == "" || req.Response == "" {
			http.Error(w, `{"error":"query and response required"}`, 400)
			return
		}
		if req.TTL <= 0 {
			req.TTL = 60
		}
		embedding := p.generateEmbedding(req.Query)
		if err := p.store.SetSemanticCache(req.Query, embedding, req.Response, req.Model, req.TTL); err != nil {
			http.Error(w, `{"error":"Failed"}`, 500)
			return
		}
		json.NewEncoder(w).Encode(map[string]string{"status": "cached"})
	default:
		http.Error(w, "Method not allowed", 405)
	}
}

func (p *Proxy) generateEmbedding(text string) string {
	words := strings.Fields(strings.ToLower(text))
	vec := make([]float64, 128)
	for i, word := range words {
		for j, c := range word {
			idx := (i*31 + j*7) % 128
			vec[idx] += float64(c)
		}
	}
	var norm float64
	for _, v := range vec {
		norm += v * v
	}
	if norm > 0 {
		norm = math.Sqrt(norm)
		for i := range vec {
			vec[i] /= norm
		}
	}
	parts := make([]string, len(vec))
	for i, v := range vec {
		parts[i] = fmt.Sprintf("%.6f", v)
	}
	return strings.Join(parts, ",")
}

func (p *Proxy) checkSemanticCache(query string, threshold float64) (*store.SemanticCacheEntry, float64) {
	embedding := p.generateEmbedding(query)
	entry, sim := p.store.GetSemanticCacheByEmbedding(embedding, threshold)
	if entry != nil {
		p.store.IncrementSemanticCacheHit(entry.ID)
	}
	return entry, sim
}

// ── HELPERS ───────────────────────────────────────────────────────────────

func init() {
	log.Println("New features loaded: Prompt Firewall, IP Allowlist, Provider Budgets, Request Queue, Health Dashboard, A/B Testing, Semantic Cache")
}
