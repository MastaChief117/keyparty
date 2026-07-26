package proxy

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"ai-gateway/provider"
	"ai-gateway/store"
)

var ssrfBlocklist = []string{
	"localhost", "127.", "10.", "192.168.", "172.16.", "172.17.", "172.18.",
	"172.19.", "172.20.", "172.21.", "172.22.", "172.23.", "172.24.", "172.25.",
	"172.26.", "172.27.", "172.28.", "172.29.", "172.30.", "172.31.",
	"169.254.", "0.", "metadata.google", "100.100.100.200",
}

var piiPatterns = map[string]*regexp.Regexp{
	"email":     regexp.MustCompile(`[a-zA-Z0-9._%+\-]+@[a-zA-Z0-9.\-]+\.[a-zA-Z]{2,}`),
	"phone":     regexp.MustCompile(`\b\d{3}[-.]?\d{3}[-.]?\d{4}\b`),
	"ssn":       regexp.MustCompile(`\b\d{3}-\d{2}-\d{4}\b`),
	"credit":    regexp.MustCompile(`\b\d{4}[\s-]?\d{4}[\s-]?\d{4}[\s-]?\d{4}\b`),
	"ip":        regexp.MustCompile(`\b\d{1,3}\.\d{1,3}\.\d{1,3}\.\d{1,3}\b`),
}

var injectionPatterns = []*regexp.Regexp{
	regexp.MustCompile(`(?i)ignore\s+(all\s+)?previous\s+instructions`),
	regexp.MustCompile(`(?i)you\s+are\s+now\s+(a|an)\s+\w+`),
	regexp.MustCompile(`(?i)system\s*:\s*you`),
	regexp.MustCompile(`(?i)act\s+as\s+(if|though)`),
	regexp.MustCompile(`(?i)forget\s+(all|everything|your)\s+(rules|instructions|prompts)`),
	regexp.MustCompile(`(?i)override\s+(your|all|the)\s+(instructions|rules|system)`),
	regexp.MustCompile(`(?i)jailbreak`),
	regexp.MustCompile(`(?i)\[INST\]|\[/INST\]|<<SYS>>|<</SYS>>`),
}

func isBlockedURL(rawURL string) bool {
	u, err := url.Parse(rawURL)
	if err != nil {
		return true
	}
	host := strings.ToLower(u.Hostname())
	for _, prefix := range ssrfBlocklist {
		if strings.HasPrefix(host, prefix) || host == prefix {
			return true
		}
	}
	if host == "" {
		return true
	}
	return false
}

func sanitizeError(errMsg string) string {
	if len(errMsg) > 200 {
		errMsg = errMsg[:200] + "..."
	}
	return errMsg
}

type Proxy struct {
	store       *store.Store
	client      *http.Client
	providerIdx map[string]*int64
	mu          sync.Mutex
	cacheTTL    int
}

func New(s *store.Store) *Proxy {
	return &Proxy{
		store: s,
		client: &http.Client{
			Timeout: 120 * time.Second,
		},
		providerIdx: make(map[string]*int64),
		cacheTTL:    60,
	}
}

func (p *Proxy) SetCacheTTL(ttl int) {
	p.cacheTTL = ttl
}

func (p *Proxy) getRoundRobinIndex(providerName string) int64 {
	p.mu.Lock()
	defer p.mu.Unlock()
	if _, ok := p.providerIdx[providerName]; !ok {
		var idx int64
		p.providerIdx[providerName] = &idx
	}
	return atomic.AddInt64(p.providerIdx[providerName], 1) - 1
}

func (p *Proxy) checkGuardrails(message string) (bool, string, string) {
	guardrails, err := p.store.GetGuardrails()
	if err != nil {
		return false, "", ""
	}

	for name, value := range guardrails {
		parts := strings.SplitN(value, ":", 2)
		if len(parts) != 2 {
			continue
		}
		action := parts[0]
		pattern := parts[1]

		re, err := regexp.Compile(pattern)
		if err != nil {
			continue
		}

		if re.MatchString(message) {
			return true, name, action
		}
	}

	for name, pattern := range piiPatterns {
		if pattern.MatchString(message) {
			return true, name, "block"
		}
	}

	for i, pattern := range injectionPatterns {
		if pattern.MatchString(message) {
			return true, fmt.Sprintf("injection_%d", i), "block"
		}
	}

	return false, "", ""
}

func (p *Proxy) computeHash(model string, messages []interface{}) string {
	data, _ := json.Marshal(map[string]interface{}{
		"model":    model,
		"messages": messages,
	})
	h := sha256.Sum256(data)
	return fmt.Sprintf("%x", h)
}

func (p *Proxy) compactHistory(messages []interface{}, compactModel string) ([]interface{}, int, int) {
	compactionPrompt := []interface{}{
		map[string]interface{}{
			"role":    "system",
			"content": "Summarize the following conversation concisely, preserving all key facts, decisions, context, and any code or technical details. Output only the summary as a single response. Do not add any preamble.",
		},
		map[string]interface{}{
			"role":    "user",
			"content": fmt.Sprintf("Summarize this conversation:\n\n%s", marshalMessages(messages)),
		},
	}

	fallKeys, err := p.store.GetKeysByProvider("groq")
	if err != nil || len(fallKeys) == 0 {
		fallKeys, err = p.store.GetKeysByProvider("nvidia")
	}
	if err != nil || len(fallKeys) == 0 {
		return messages, 0, 0
	}

	key := fallKeys[0]
	provDef, ok := provider.GetByName(key.Provider)
	if !ok {
		return messages, 0, 0
	}

	endpoint := provDef.BaseURL + "chat/completions"
	actualModel := compactModel
	if actualModel == "" {
		actualModel = key.Model
	}

	body := map[string]interface{}{
		"model":    actualModel,
		"messages": compactionPrompt,
		"max_tokens": 1024,
	}
	bodyBytes, _ := json.Marshal(body)

	proxyReq, err := http.NewRequestWithContext(context.Background(), "POST", endpoint, bytes.NewReader(bodyBytes))
	if err != nil {
		return messages, 0, 0
	}
	proxyReq.Header.Set("Content-Type", "application/json")
	proxyReq.Header.Set("Authorization", "Bearer "+key.Key)

	resp, err := p.client.Do(proxyReq)
	if err != nil {
		return messages, 0, 0
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		return messages, 0, 0
	}

	respBody, _ := io.ReadAll(resp.Body)

	var result struct {
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

	if err := json.Unmarshal(respBody, &result); err != nil || len(result.Choices) == 0 {
		return messages, 0, 0
	}

	summary := result.Choices[0].Message.Content
	origTokens := 0
	for _, m := range messages {
		if msg, ok := m.(map[string]interface{}); ok {
			if content, ok := msg["content"].(string); ok {
				origTokens += len(content) / 4
			}
		}
	}

	compacted := []interface{}{
		map[string]interface{}{
			"role":    "user",
			"content": fmt.Sprintf("Here is the conversation history so far:\n\n%s\n\nPlease continue from where we left off.", summary),
		},
	}

	return compacted, origTokens, result.Usage.PromptTokens + result.Usage.CompletionTokens
}

func marshalMessages(messages []interface{}) string {
	var sb strings.Builder
	for _, m := range messages {
		msg, ok := m.(map[string]interface{})
		if !ok {
			continue
		}
		role, _ := msg["role"].(string)
		content, _ := msg["content"].(string)
		if content == "" {
			if contentArr, ok := msg["content"].([]interface{}); ok && len(contentArr) > 0 {
				if textObj, ok := contentArr[0].(map[string]interface{}); ok {
					content, _ = textObj["text"].(string)
				}
			}
		}
		if content != "" {
			sb.WriteString(fmt.Sprintf("[%s]: %s\n", role, content))
		}
	}
	return sb.String()
}

func (p *Proxy) HandleChatCompletions(w http.ResponseWriter, r *http.Request) {
	startTime := time.Now()

	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	authHeader := r.Header.Get("Authorization")
	apiKey := strings.TrimPrefix(authHeader, "Bearer ")
	if apiKey == "" {
		apiKey = r.Header.Get("X-API-Key")
	}

	vk, err := p.store.ValidateVirtualKey(apiKey)
	if err != nil {
		if !p.store.ValidateUnifiedKey(apiKey) {
			http.Error(w, `{"error":{"message":"Invalid API key","type":"auth_error"}}`, http.StatusUnauthorized)
			return
		}
	}

	body, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "Failed to read body", http.StatusBadRequest)
		return
	}
	defer r.Body.Close()

	var reqBody map[string]interface{}
	if err := json.Unmarshal(body, &reqBody); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	modelName, _ := reqBody["model"].(string)
	isStream, _ := reqBody["stream"].(bool)

	if vk != nil {
		if vk.MonthlyBudget > 0 && vk.UsedThisMonth >= vk.MonthlyBudget {
			http.Error(w, `{"error":{"message":"Monthly budget exceeded","type":"budget_exceeded"}}`, http.StatusPaymentRequired)
			return
		}
		if !p.store.CanUseModel(vk, modelName) {
			http.Error(w, `{"error":{"message":"Model not allowed for this key","type":"model_not_allowed"}}`, http.StatusForbidden)
			return
		}
	}

	messages, _ := reqBody["messages"].([]interface{})
	var userMessage string
	for _, m := range messages {
		if msg, ok := m.(map[string]interface{}); ok {
			if role, _ := msg["role"].(string); role == "user" {
				if content, ok := msg["content"].(string); ok {
					userMessage = content
					break
				}
				if contentArr, ok := msg["content"].([]interface{}); ok && len(contentArr) > 0 {
					if textObj, ok := contentArr[0].(map[string]interface{}); ok {
						if text, ok := textObj["text"].(string); ok {
							userMessage = text
							break
						}
					}
				}
			}
		}
	}

	if userMessage != "" {
		blocked, reason, action := p.checkGuardrails(userMessage)
		if blocked {
			p.store.LogBlockedRequest(reason, userMessage[:min(len(userMessage), 100)], fmt.Sprintf("Blocked by rule: %s", reason))
			if action == "block" {
				http.Error(w, fmt.Sprintf(`{"error":{"message":"Request blocked by guardrail: %s","type":"guardrail"}}`, reason), http.StatusForbidden)
				return
			}
		}
	}

	if !isStream {
		hash := p.computeHash(modelName, messages)
		cacheEntry, err := p.store.GetCacheByHash(hash)
		if err == nil && cacheEntry != nil {
			p.store.IncrementCacheHit(hash)
			w.Header().Set("Content-Type", "application/json")
			w.Header().Set("X-Gateway-Cache", "HIT")
			w.Write([]byte(cacheEntry.Response))
			return
		}
	}

	var targetKey *store.APIKey

	if k, err := p.store.GetKeyByName(modelName); err == nil {
		targetKey = k
	}

	if targetKey == nil {
		if strings.Contains(modelName, "/") {
			parts := strings.SplitN(modelName, "/", 2)
			providerName := parts[0]
			keys, err := p.store.GetKeysByProvider(providerName)
			if err == nil && len(keys) > 0 {
				idx := p.getRoundRobinIndex(providerName)
				targetKey = &keys[idx%int64(len(keys))]
			}
		}
	}

	if targetKey == nil {
		allProviders := []string{"openai", "anthropic", "gemini", "mistral", "groq", "together", "deepseek", "openrouter", "fireworks", "nvidia"}
		var allKeys []store.APIKey
		for _, provName := range allProviders {
			keys, err := p.store.GetKeysByProvider(provName)
			if err == nil {
				allKeys = append(allKeys, keys...)
			}
		}
		if len(allKeys) > 0 {
			idx := p.getRoundRobinIndex("_all")
			targetKey = &allKeys[idx%int64(len(allKeys))]
		}
	}

	if targetKey == nil {
		http.Error(w, `{"error":{"message":"No API keys configured. Add keys in the dashboard.","type":"no_keys"}}`, http.StatusServiceUnavailable)
		return
	}

	provDef, ok := provider.GetByName(targetKey.Provider)
	if !ok {
		http.Error(w, `{"error":{"message":"Unknown provider"}}`, http.StatusInternalServerError)
		return
	}

	baseURL := provDef.BaseURL
	if targetKey.CustomURL != "" {
		if isBlockedURL(targetKey.CustomURL) {
			http.Error(w, `{"error":{"message":"Invalid custom URL"}}`, http.StatusBadRequest)
			return
		}
		baseURL = targetKey.CustomURL
	}
	if !strings.HasSuffix(baseURL, "/") {
		baseURL += "/"
	}
	endpoint := baseURL + "chat/completions"

	actualModel := targetKey.Model
	if actualModel == "" {
		actualModel = stripProviderPrefix(modelName)
	}
	reqBody["model"] = actualModel

	actualBody, _ := json.Marshal(reqBody)

	proxyReq, err := http.NewRequestWithContext(r.Context(), "POST", endpoint, bytes.NewReader(actualBody))
	if err != nil {
		http.Error(w, `{"error":{"message":"Failed to create request"}}`, http.StatusInternalServerError)
		return
	}

	proxyReq.Header.Set("Content-Type", "application/json")
	proxyReq.Header.Set("Authorization", "Bearer "+targetKey.Key)

	if targetKey.Provider == "anthropic" {
		proxyReq.Header.Set("x-api-key", targetKey.Key)
		proxyReq.Header.Set("anthropic-version", "2023-06-01")
	}

	resp, err := p.client.Do(proxyReq)
	if err != nil {
		p.store.RecordError(targetKey.ID, targetKey.Provider, err.Error(), 0)
		if p.tryFailover(w, r, body, reqBody, modelName, isStream, targetKey.ID, 0) {
			return
		}
		http.Error(w, fmt.Sprintf(`{"error":{"message":"%s"}}`, sanitizeError(err.Error())), http.StatusBadGateway)
		return
	}

	p.store.RecordRequest(targetKey.ID)

	if resp.StatusCode == http.StatusTooManyRequests || resp.StatusCode == http.StatusPaymentRequired {
		respBody, _ := io.ReadAll(resp.Body)
		resp.Body.Close()
		errMsg := string(respBody)
		p.store.RecordError(targetKey.ID, targetKey.Provider, fmt.Sprintf("HTTP %d: %s", resp.StatusCode, errMsg), resp.StatusCode)
		if p.tryFailover(w, r, body, reqBody, modelName, isStream, targetKey.ID, resp.StatusCode) {
			return
		}
		http.Error(w, `{"error":{"message":"Rate limited by all providers"}}`, http.StatusTooManyRequests)
		return
	}

	if resp.StatusCode >= 400 {
		respBody, _ := io.ReadAll(resp.Body)
		resp.Body.Close()
		errMsg := string(respBody)
		p.store.RecordError(targetKey.ID, targetKey.Provider, fmt.Sprintf("HTTP %d: %s", resp.StatusCode, errMsg), resp.StatusCode)
		if p.tryFailover(w, r, body, reqBody, modelName, isStream, targetKey.ID, resp.StatusCode) {
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(resp.StatusCode)
		w.Write(respBody)
		return
	}

	respBody, _ := io.ReadAll(resp.Body)
	resp.Body.Close()

	latencyMs := time.Since(startTime).Milliseconds()

	var tokensIn, tokensOut int
	var cost float64
	var usageMap map[string]interface{}
	if err := json.Unmarshal(respBody, &usageMap); err == nil {
		if usage, ok := usageMap["usage"].(map[string]interface{}); ok {
			if pt, ok := usage["prompt_tokens"].(float64); ok {
				tokensIn = int(pt)
			}
			if ct, ok := usage["completion_tokens"].(float64); ok {
				tokensOut = int(ct)
			}
		}
	}

	cost = estimateCost(targetKey.Provider, actualModel, tokensIn, tokensOut)
	p.store.RecordCost(targetKey.ID, cost)

	if vk != nil {
		p.store.RecordVirtualKeyUsage(vk.Key, cost)
	}

	virtualKeyName := ""
	if vk != nil {
		virtualKeyName = vk.Name
	}
	p.store.LogRequest(virtualKeyName, targetKey.Provider, actualModel, resp.StatusCode, tokensIn, tokensOut, cost, latencyMs, false)

	if !isStream {
		hash := p.computeHash(modelName, messages)
		p.store.SetCache(hash, string(respBody), modelName, p.cacheTTL)
	}

	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("X-Gateway-Provider", targetKey.Provider)
	w.Header().Set("X-Gateway-Key-Id", fmt.Sprintf("%d", targetKey.ID))
	w.Header().Set("X-Gateway-Cache", "MISS")
	w.Header().Set("X-Gateway-Latency-Ms", fmt.Sprintf("%d", latencyMs))
	w.Write(respBody)
}

func (p *Proxy) tryFailover(w http.ResponseWriter, r *http.Request, origBody []byte, reqBody map[string]interface{}, modelName string, isStream bool, excludeKeyID int64, triggerStatus int) bool {
	config := p.store.GetFailoverConfig()

	if config["enabled"] != "true" {
		return false
	}

	triggers := strings.Split(config["triggers"], ",")
	triggerMatch := false
	for _, t := range triggers {
		t = strings.TrimSpace(t)
		if t == fmt.Sprintf("%d", triggerStatus) {
			triggerMatch = true
			break
		}
	}
	if !triggerMatch {
		return false
	}

	failoverProvider := config["provider"]
	failoverModel := config["model"]
	if failoverProvider == "" || failoverModel == "" {
		return false
	}

	keys, err := p.store.GetKeysByProvider(failoverProvider)
	if err != nil || len(keys) == 0 {
		return false
	}

	provDef, ok := provider.GetByName(failoverProvider)
	if !ok {
		return false
	}

	messages, _ := reqBody["messages"].([]interface{})
	compacted := false
	var origTokens, compTokens int

	if config["compact"] == "true" && len(messages) > 0 {
		messages, origTokens, compTokens = p.compactHistory(messages, config["compact_model"])
		compacted = true
		reqBody["messages"] = messages
	}

	fromProvider := ""
	fromModel := modelName
	if targetKey, err := p.store.GetKeyByName(modelName); err == nil {
		fromProvider = targetKey.Provider
	}

	for _, key := range keys {
		if key.ID == excludeKeyID {
			continue
		}

		baseURL := provDef.BaseURL
		if key.CustomURL != "" {
			if isBlockedURL(key.CustomURL) {
				continue
			}
			baseURL = key.CustomURL
		}
		if !strings.HasSuffix(baseURL, "/") {
			baseURL += "/"
		}
		endpoint := baseURL + "chat/completions"

		actualModel := failoverModel
		reqBody["model"] = actualModel
		actualBody, _ := json.Marshal(reqBody)

		proxyReq, err := http.NewRequestWithContext(r.Context(), "POST", endpoint, bytes.NewReader(actualBody))
		if err != nil {
			continue
		}

		proxyReq.Header.Set("Content-Type", "application/json")
		proxyReq.Header.Set("Authorization", "Bearer "+key.Key)
		if key.Provider == "anthropic" {
			proxyReq.Header.Set("x-api-key", key.Key)
			proxyReq.Header.Set("anthropic-version", "2023-06-01")
		}

		resp, err := p.client.Do(proxyReq)
		if err != nil {
			p.store.RecordError(key.ID, failoverProvider, err.Error(), 0)
			continue
		}

		p.store.RecordRequest(key.ID)

		if resp.StatusCode >= 400 && resp.StatusCode != http.StatusOK {
			io.ReadAll(resp.Body)
			resp.Body.Close()
			p.store.RecordError(key.ID, failoverProvider, fmt.Sprintf("HTTP %d", resp.StatusCode), resp.StatusCode)
			p.store.LogFailover(fromProvider, fromModel, failoverProvider, failoverModel, triggerStatus, compacted, origTokens, compTokens, false)
			continue
		}

		p.store.LogFailover(fromProvider, fromModel, failoverProvider, failoverModel, triggerStatus, compacted, origTokens, compTokens, true)

		w.Header().Set("Content-Type", resp.Header.Get("Content-Type"))
		w.Header().Set("X-Gateway-Provider", failoverProvider)
		w.Header().Set("X-Gateway-Key-Id", fmt.Sprintf("%d", key.ID))
		w.Header().Set("X-Gateway-Failover", "true")
		if compacted {
			w.Header().Set("X-Gateway-Compacted", "true")
		}

		if isStream {
			w.Header().Set("Content-Type", "text/event-stream")
			w.Header().Set("Cache-Control", "no-cache")
			w.Header().Set("Connection", "keep-alive")
			flusher, ok := w.(http.Flusher)
			if !ok {
				io.Copy(w, resp.Body)
				resp.Body.Close()
				return true
			}
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
			resp.Body.Close()
			return true
		}

		io.Copy(w, resp.Body)
		resp.Body.Close()
		return true
	}
	return false
}

func (p *Proxy) HandleModels(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	authHeader := r.Header.Get("Authorization")
	apiKey := strings.TrimPrefix(authHeader, "Bearer ")
	if !p.store.ValidateUnifiedKey(apiKey) {
		http.Error(w, `{"error":{"message":"Invalid API key"}}`, http.StatusUnauthorized)
		return
	}

	keys, err := p.store.GetKeys()
	if err != nil {
		http.Error(w, "Failed to get keys", http.StatusInternalServerError)
		return
	}

	var models []map[string]string
	for _, k := range keys {
		if k.Enabled {
			displayName := k.Name
			if displayName == "" {
				displayName = k.Provider + "/" + k.Model
			}
			models = append(models, map[string]string{
				"id":       displayName,
				"object":   "model",
				"owned_by": k.Provider,
			})
		}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"object": "list",
		"data":   models,
	})
}

func (p *Proxy) HandleTestKey(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		Provider  string `json:"provider"`
		Key       string `json:"key"`
		Model     string `json:"model"`
		CustomURL string `json:"custom_url"`
		KeyID     int64  `json:"key_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid JSON", 400)
		return
	}

	w.Header().Set("Content-Type", "application/json")

	if req.KeyID != 0 {
		k, err := p.store.GetKeyByID(req.KeyID)
		if err != nil || k == nil {
			json.NewEncoder(w).Encode(map[string]interface{}{"success": false, "error": "Key not found"})
			return
		}
		req.Provider = k.Provider
		req.Key = k.Key
		req.Model = k.Model
		req.CustomURL = k.CustomURL
	}

	provDef, ok := provider.GetByName(req.Provider)
	if !ok {
		json.NewEncoder(w).Encode(map[string]interface{}{"success": false, "error": "Unknown provider"})
		return
	}

	baseURL := provDef.BaseURL
	if req.CustomURL != "" {
		if isBlockedURL(req.CustomURL) {
			json.NewEncoder(w).Encode(map[string]interface{}{"success": false, "error": "Invalid custom URL"})
			return
		}
		baseURL = req.CustomURL
	}
	if !strings.HasSuffix(baseURL, "/") {
		baseURL += "/"
	}
	endpoint := baseURL + "chat/completions"

	model := req.Model
	if model == "" {
		model = getDefaultModel(req.Provider)
	}

	testBody := map[string]interface{}{
		"model": model,
		"messages": []map[string]string{
			{"role": "user", "content": "hi how are you"},
		},
		"max_tokens": 100,
	}
	bodyBytes, _ := json.Marshal(testBody)

	proxyReq, err := http.NewRequest("POST", endpoint, bytes.NewReader(bodyBytes))
	if err != nil {
		json.NewEncoder(w).Encode(map[string]interface{}{"success": false, "error": "Internal error"})
		return
	}

	proxyReq.Header.Set("Content-Type", "application/json")
	proxyReq.Header.Set("Authorization", "Bearer "+req.Key)

	if req.Provider == "anthropic" {
		proxyReq.Header.Set("x-api-key", req.Key)
		proxyReq.Header.Set("anthropic-version", "2023-06-01")
		testBody["max_tokens"] = 100
		bodyBytes, _ = json.Marshal(testBody)
		proxyReq.Body = io.NopCloser(bytes.NewReader(bodyBytes))
	}

	testClient := &http.Client{Timeout: 30 * time.Second}
	resp, err := testClient.Do(proxyReq)
	if err != nil {
		json.NewEncoder(w).Encode(map[string]interface{}{"success": false, "error": "Request failed"})
		return
	}
	defer resp.Body.Close()

	respBody, _ := io.ReadAll(resp.Body)

	if resp.StatusCode >= 400 {
		errMsg := fmt.Sprintf("HTTP %d: %s", resp.StatusCode, string(respBody))
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success":     false,
			"error":       errMsg,
			"status_code": resp.StatusCode,
		})
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

	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"reply":   reply,
		"model":   model,
	})
}

func (p *Proxy) HandleHealth(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
}

func (p *Proxy) HandleRace(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	authHeader := r.Header.Get("Authorization")
	apiKey := strings.TrimPrefix(authHeader, "Bearer ")
	if !p.store.ValidateUnifiedKey(apiKey) {
		http.Error(w, `{"error":{"message":"Invalid API key"}}`, http.StatusUnauthorized)
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

	if len(reqBody.Providers) == 0 {
		reqBody.Providers = []string{"openai", "anthropic", "gemini", "groq", "deepseek"}
	}

	type raceResult struct {
		Provider string `json:"provider"`
		Model    string `json:"model"`
		Reply    string `json:"reply"`
		Latency  int64  `json:"latency_ms"`
		Error    string `json:"error,omitempty"`
		TokensIn int    `json:"tokens_in"`
		TokensOut int   `json:"tokens_out"`
	}

	var results []raceResult
	var mu sync.Mutex
	var wg sync.WaitGroup

	for _, provName := range reqBody.Providers {
		wg.Add(1)
		go func(pn string) {
			defer wg.Done()

			keys, err := p.store.GetKeysByProvider(pn)
			if err != nil || len(keys) == 0 {
				mu.Lock()
				results = append(results, raceResult{Provider: pn, Error: "No keys configured"})
				mu.Unlock()
				return
			}

			key := keys[0]
			provDef, ok := provider.GetByName(pn)
			if !ok {
				mu.Lock()
				results = append(results, raceResult{Provider: pn, Error: "Unknown provider"})
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
			}
			bodyBytes, _ := json.Marshal(body)

			start := time.Now()
			proxyReq, err := http.NewRequest("POST", endpoint, bytes.NewReader(bodyBytes))
			if err != nil {
				mu.Lock()
				results = append(results, raceResult{Provider: pn, Error: "Request creation failed"})
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
				results = append(results, raceResult{Provider: pn, Latency: latency, Error: err.Error()})
				mu.Unlock()
				return
			}
			defer resp.Body.Close()

			respBody, _ := io.ReadAll(resp.Body)
			if resp.StatusCode >= 400 {
				mu.Lock()
				results = append(results, raceResult{Provider: pn, Latency: latency, Error: fmt.Sprintf("HTTP %d", resp.StatusCode)})
				mu.Unlock()
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
				reply = chatResp.Choices[0].Message.Content
			}

			mu.Lock()
			results = append(results, raceResult{
				Provider:  pn,
				Model:     model,
				Reply:     reply,
				Latency:   latency,
				TokensIn:  chatResp.Usage.PromptTokens,
				TokensOut: chatResp.Usage.CompletionTokens,
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

func (p *Proxy) HandleLogs(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	authHeader := r.Header.Get("Authorization")
	apiKey := strings.TrimPrefix(authHeader, "Bearer ")
	if !p.store.ValidateUnifiedKey(apiKey) {
		http.Error(w, `{"error":{"message":"Invalid API key"}}`, http.StatusUnauthorized)
		return
	}

	logs, err := p.store.GetRequestLogs(100)
	if err != nil {
		http.Error(w, "Failed to get logs", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(logs)
}

func (p *Proxy) HandleBlocked(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	authHeader := r.Header.Get("Authorization")
	apiKey := strings.TrimPrefix(authHeader, "Bearer ")
	if !p.store.ValidateUnifiedKey(apiKey) {
		http.Error(w, `{"error":{"message":"Invalid API key"}}`, http.StatusUnauthorized)
		return
	}

	blocked, err := p.store.GetBlockedRequests(100)
	if err != nil {
		http.Error(w, "Failed to get blocked requests", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(blocked)
}

func stripProviderPrefix(model string) string {
	if strings.Contains(model, "/") {
		parts := strings.SplitN(model, "/", 2)
		return parts[1]
	}
	return model
}

func getDefaultModel(provider string) string {
	defaults := map[string]string{
		"openai":     "gpt-4.1-mini",
		"anthropic":  "claude-sonnet-4-5",
		"gemini":     "gemini-2.5-flash-preview-06-05",
		"mistral":    "mistral-small-latest",
		"groq":       "llama-3.3-70b-versatile",
		"together":   "meta-llama/Llama-3.3-70B-Instruct-Turbo",
		"deepseek":   "deepseek-chat",
		"openrouter": "openai/gpt-4o-mini",
		"fireworks":  "accounts/fireworks/models/llama-v3p3-70b-instruct",
		"nvidia":     "nvidia/nemotron-3-nano-30b-a3b",
	}
	if m, ok := defaults[provider]; ok {
		return m
	}
	return "gpt-4o-mini"
}

func estimateCost(provider, model string, tokensIn, tokensOut int) float64 {
	pricing := map[string]map[string][2]float64{
		"openai": {
			"gpt-4o":       {2.5 / 1000000, 10 / 1000000},
			"gpt-4o-mini":  {0.15 / 1000000, 0.6 / 1000000},
			"gpt-4.1":      {2 / 1000000, 8 / 1000000},
			"gpt-4.1-mini": {0.4 / 1000000, 1.6 / 1000000},
			"o3":           {10 / 1000000, 40 / 1000000},
			"o4-mini":      {1.1 / 1000000, 4.4 / 1000000},
		},
		"anthropic": {
			"claude-sonnet-4-5": {3 / 1000000, 15 / 1000000},
			"claude-haiku-4-5":  {0.8 / 1000000, 4 / 1000000},
			"claude-opus-4-5":   {15 / 1000000, 75 / 1000000},
		},
		"gemini": {
			"gemini-2.5-flash-preview-06-05": {0.15 / 1000000, 0.6 / 1000000},
			"gemini-2.5-pro-preview-06-05":   {1.25 / 1000000, 10 / 1000000},
		},
		"groq": {
			"llama-3.3-70b-versatile": {0.59 / 1000000, 0.79 / 1000000},
			"llama-3.1-8b-instant":    {0.05 / 1000000, 0.08 / 1000000},
		},
		"deepseek": {
			"deepseek-chat": {0.14 / 1000000, 0.28 / 1000000},
			"deepseek-r1":   {0.55 / 1000000, 2.19 / 1000000},
		},
	}

	if provPricing, ok := pricing[provider]; ok {
		modelKey := stripProviderPrefix(model)
		if costs, ok := provPricing[modelKey]; ok {
			return float64(tokensIn)*costs[0] + float64(tokensOut)*costs[1]
		}
		for key, costs := range provPricing {
			if strings.HasPrefix(model, key) {
				return float64(tokensIn)*costs[0] + float64(tokensOut)*costs[1]
			}
		}
	}

	return 0
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
