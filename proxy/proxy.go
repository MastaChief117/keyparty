package proxy

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"regexp"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"ai-gateway/provider"
	"ai-gateway/store"
)

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
	vkRateMu    sync.Mutex
	vkTokens    map[string]*vkBucket
}

type vkBucket struct {
	tokens   int
	lastSeen time.Time
}

func New(s *store.Store) *Proxy {
	transport := &http.Transport{
		MaxIdleConns:        100,
		MaxIdleConnsPerHost: 20,
		IdleConnTimeout:     90 * time.Second,
	}
	return &Proxy{
		store: s,
		client: &http.Client{
			Timeout:   120 * time.Second,
			Transport: transport,
		},
		providerIdx: make(map[string]*int64),
		cacheTTL:    60,
		vkTokens:    make(map[string]*vkBucket),
	}
}

func (p *Proxy) allowVK(vkKey string, rateLimit int) bool {
	p.vkRateMu.Lock()
	defer p.vkRateMu.Unlock()
	bucket, exists := p.vkTokens[vkKey]
	if !exists {
		bucket = &vkBucket{tokens: rateLimit, lastSeen: time.Now()}
		p.vkTokens[vkKey] = bucket
	}
	elapsed := time.Since(bucket.lastSeen)
	bucket.lastSeen = time.Now()
	bucket.tokens += int(elapsed.Seconds() * float64(rateLimit))
	if bucket.tokens > rateLimit {
		bucket.tokens = rateLimit
	}
	if bucket.tokens <= 0 {
		return false
	}
	bucket.tokens--
	return true
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

func (p *Proxy) computeHash(model string, messages []interface{}, vkID string) string {
	msgJSON, _ := json.Marshal(messages)
	h := sha256.Sum256(append([]byte(model+":"+vkID+":"), msgJSON...))
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
	log.Printf("COMPACTION WARNING: Conversation history being sent to %s for summarization (different from original provider)", key.Provider)
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
		if vk.RateLimit > 0 {
			if !p.allowVK(vk.Key, vk.RateLimit) {
				http.Error(w, `{"error":{"message":"Rate limit exceeded for this virtual key","type":"rate_limit"}}`, http.StatusTooManyRequests)
				return
			}
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
		vkID := ""
		if vk != nil {
			vkID = vk.Key
		}
		hash := p.computeHash(modelName, messages, vkID)
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
		if IsBlockedURL(targetKey.CustomURL) {
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
		w.Write([]byte(fmt.Sprintf(`{"error":{"message":"Provider error","status":%d}}`, resp.StatusCode)))
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
		p.store.DeductBudgetAtomic(vk.Key, cost)
	}

	virtualKeyName := ""
	if vk != nil {
		virtualKeyName = vk.Name
	}
	p.store.LogRequest(virtualKeyName, targetKey.Provider, actualModel, resp.StatusCode, tokensIn, tokensOut, cost, latencyMs, false)

	if !isStream {
		vkID := ""
		if vk != nil {
			vkID = vk.Key
		}
		hash := p.computeHash(modelName, messages, vkID)
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
			if IsBlockedURL(key.CustomURL) {
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
		if IsBlockedURL(req.CustomURL) {
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
			"gemini-3.6-flash":              {0.10 / 1000000, 0.40 / 1000000},
			"gemini-3.5-flash":              {0.15 / 1000000, 0.60 / 1000000},
			"gemini-3.5-flash-lite":         {0.075 / 1000000, 0.30 / 1000000},
			"gemini-3.1-flash-lite":         {0.075 / 1000000, 0.30 / 1000000},
			"gemini-3.1-pro-preview":        {2.00 / 1000000, 12.00 / 1000000},
			"gemini-3-flash-preview":        {0.50 / 1000000, 3.00 / 1000000},
			"gemini-2.5-pro":                {1.25 / 1000000, 10.00 / 1000000},
			"gemini-2.5-flash":              {0.15 / 1000000, 0.60 / 1000000},
			"gemini-2.5-flash-lite":         {0.075 / 1000000, 0.30 / 1000000},
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

// ── FUN FEATURES ────────────────────────────────────────────────────────────

var roastInsults = []string{
	"You type like your keyboard is made of wet bread",
	"Your WiFi personality is worse than 2G in a tunnel",
	"Even the bots feel sorry for you",
	"You have main character energy but in a background scene",
	"Your takes are sponsored by brain rot",
	"Calling you mid would be a compliment",
	"You give off strong NPC energy",
	"Your personality is in airplane mode",
	"You're the reason group chats go silent",
	"You have the social awareness of a CAPTCHA",
	"You radiate 'forgot to silence their phone in a movie' energy",
	"You're the tutorial level everyone skips",
	"You type with the confidence of someone who's always wrong",
	"Your presence in chat is optional at best",
	"You're giving 3AM energy but it's 2PM",
	"You have the range of a dial-up modem",
	"Even the server logs don't care about your messages",
	"You've been buffering since you got here",
	"Your entire vibe is permanently out of stock",
	"You lost the plot and never found it",
}

var eightBallResponses = []string{
	"It is certain. Like your questionable life choices.",
	"It is decidedly so. Even the AI agrees.",
	"Without a doubt. But also maybe. Who knows.",
	"Yes definitely. But don't quote me on that.",
	"You may rely on it. If you rely on anything else you're cooked.",
	"As I see it, yes. But I'm an AI so my vision is questionable.",
	"Most likely. But also possibly not. Math is hard.",
	"Outlook good. Not as good as your WiFi though.",
	"Yes. That's my final answer. I think.",
	"Signs point to yes. But signs also point to IKEA and we all know how that ends.",
	"Reply hazy, try again. Or don't. I'm not your mom.",
	"Ask again later. I'm busy.",
	"Better not tell you now. Spoiler alert.",
	"Cannot predict now. I'm an AI not a psychic.",
	"Concentrate and ask again. Maybe blink twice.",
	"Don't count on it. Actually do count on it. I changed my mind.",
	"My reply is no. Sorrynotsorry.",
	"My sources say no. My sources are me.",
	"Outlook not so good. But when has it ever been?",
	"Very doubtful. Like your taste in models.",
}

var funFacts = []string{
	"Fun fact: Your API key has been rotated more times than your sleep schedule",
	"Fun fact: Groq is faster than your WiFi on a good day",
	"Fun fact: The average AI gateway processes more requests than you get texts",
	"Fun fact: Your unified key has seen things. Horrible things.",
	"Fun fact: SQLite doesn't judge you. We do though.",
	"Fun fact: Your cache hit rate is probably better than your hit rate on dating apps",
	"Fun fact: This gateway has more uptime than your last relationship",
	"Fun fact: The failover log is just a list of providers ghosting you",
	"Fun fact: AES-256 encryption is harder to break than your New Year's resolutions",
	"Fun fact: Your API keys are encrypted. Your search history should be too.",
	"Fun fact: This gateway runs on Go. Not JavaScript. We have standards.",
	"Fun fact: The race mode has seen more action than your social life",
	"Fun fact: Your database is probably more organized than your room",
	"Fun fact: Error 404: Your motivation not found",
	"Fun fact: The only thing failing over more than this gateway is your diet plan",
	"Fun fact: Rate limiting exists because even APIs need personal space",
	"Fun fact: Your cost tracking shows you spend more on AI than on food",
	"Fun fact: The guardrails block more than your ex's attempts to contact you",
}

func (p *Proxy) HandleRoast(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", 405)
		return
	}

	var reqBody struct {
		Username string `json:"username"`
	}
	if err := json.NewDecoder(r.Body).Decode(&reqBody); err != nil || reqBody.Username == "" {
		http.Error(w, `{"error":"Send a username to roast"}`, 400)
		return
	}

	randomIdx := time.Now().UnixNano() % int64(len(roastInsults))
	insult := roastInsults[randomIdx]

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"roast":   fmt.Sprintf("@%s — %s", reqBody.Username, insult),
		"username": reqBody.Username,
	})
}

func (p *Proxy) Handle8Ball(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", 405)
		return
	}

	var reqBody struct {
		Question string `json:"question"`
	}
	if err := json.NewDecoder(r.Body).Decode(&reqBody); err != nil || reqBody.Question == "" {
		http.Error(w, `{"error":"Ask a real question. The ball needs something to judge."}`, 400)
		return
	}

	randomIdx := time.Now().UnixNano() % int64(len(eightBallResponses))
	answer := eightBallResponses[randomIdx]

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"question": reqBody.Question,
		"answer":   answer,
	})
}

func (p *Proxy) HandleShip(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", 405)
		return
	}

	var reqBody struct {
		Provider1 string `json:"provider1"`
		Provider2 string `json:"provider2"`
	}
	if err := json.NewDecoder(r.Body).Decode(&reqBody); err != nil || reqBody.Provider1 == "" || reqBody.Provider2 == "" {
		http.Error(w, `{"error":"Send provider1 and provider2"}`, 400)
		return
	}

	pair := reqBody.Provider1 + "|" + reqBody.Provider2
	pct := int(abs(int32(hashString(pair)))%100) + 1

	var reason string
	switch {
	case pct <= 15:
		reason = "only compatible in a universe where they both touch grass"
	case pct <= 30:
		reason = "both have questionable API response times and worse uptime"
	case pct <= 45:
		reason = "united by their mutual rate limiting habits"
	case pct <= 60:
		reason = "both chaos providers with functioning endpoints"
	case pct <= 75:
		reason = "compatible levels of unhinged token generation"
	case pct <= 90:
		reason = "both never go down and it's honestly working"
	default:
		reason = "destined by the cloud gods — the prophecy is real"
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"provider1":    reqBody.Provider1,
		"provider2":    reqBody.Provider2,
		"compatibility": pct,
		"reason":       reason,
		"status":       fmt.Sprintf("%s x %s = %d%% compatible — %s", reqBody.Provider1, reqBody.Provider2, pct, reason),
	})
}

func hashString(s string) int32 {
	var h int32
	for _, c := range s {
		h = h*31 + int32(c)
	}
	return h
}

func abs(n int32) int32 {
	if n < 0 {
		return -n
	}
	return n
}

func (p *Proxy) HandleLeaderboard(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", 405)
		return
	}

	keys, err := p.store.GetKeys()
	if err != nil {
		http.Error(w, `{"error":"Failed"}`, 500)
		return
	}

	type providerStats struct {
		Provider string  `json:"provider"`
		Requests int64   `json:"requests"`
		Errors   int64   `json:"errors"`
		Cost     float64 `json:"cost"`
		Keys     int     `json:"keys"`
	}

	statsMap := make(map[string]*providerStats)
	for _, k := range keys {
		if _, ok := statsMap[k.Provider]; !ok {
			statsMap[k.Provider] = &providerStats{Provider: k.Provider}
		}
		statsMap[k.Provider].Requests += k.TotalReqs
		statsMap[k.Provider].Errors += k.ErrorReqs
		statsMap[k.Provider].Cost += k.TotalCost
		statsMap[k.Provider].Keys++
	}

	var results []providerStats
	for _, v := range statsMap {
		results = append(results, *v)
	}

	for i := 0; i < len(results); i++ {
		for j := i + 1; j < len(results); j++ {
			if results[j].Requests > results[i].Requests {
				results[i], results[j] = results[j], results[i]
			}
		}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(results)
}

func (p *Proxy) HandleSavings(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", 405)
		return
	}

	keys, err := p.store.GetKeys()
	if err != nil {
		http.Error(w, `{"error":"Failed"}`, 500)
		return
	}

	var totalCost float64
	var totalRequests int64
	for _, k := range keys {
		totalCost += k.TotalCost
		totalRequests += k.TotalReqs
	}

	cacheHits := p.store.GetCacheHits()
	cachedRequestsSaved := cacheHits

	failoverLogs := p.store.GetFailoverLogs(1000)
	failoverSuccess := 0
	for _, l := range failoverLogs {
		if l.Success {
			failoverSuccess++
		}
	}

	estimatedSavings := float64(cachedRequestsSaved) * 0.002

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"total_cost":            fmt.Sprintf("%.4f", totalCost),
		"total_requests":        totalRequests,
		"cache_hits":            cacheHits,
		"cached_saved":          cachedRequestsSaved,
		"estimated_savings":     fmt.Sprintf("%.4f", estimatedSavings),
		"failover_successes":    failoverSuccess,
		"failover_money_saved":  fmt.Sprintf("%.4f", float64(failoverSuccess)*0.003),
		"message":               fmt.Sprintf("You've saved $%.4f from caching and $%.4f from failover. Your wallet sends thanks.", estimatedSavings, float64(failoverSuccess)*0.003),
	})
}

func (p *Proxy) HandleFunFacts(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", 405)
		return
	}

	idx := time.Now().UnixNano() % int64(len(funFacts))
	fact := funFacts[idx]

	keys, _ := p.store.GetKeys()
	totalKeys := len(keys)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"fact":       fact,
		"total_keys": totalKeys,
		"timestamp":  time.Now().Format(time.RFC3339),
	})
}

// ── AI RUMBLE ──────────────────────────────────────────────────────────────

func (p *Proxy) HandleRumble(w http.ResponseWriter, r *http.Request) {
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
		reqBody.Rounds = 5
	}

	keys, err := p.store.GetKeys()
	if err != nil || len(keys) < 2 {
		http.Error(w, `{"error":"Need at least 2 API keys for a rumble"}`, 400)
		return
	}

	providerKeys := make(map[string][]store.APIKey)
	for _, k := range keys {
		if k.Enabled && k.Provider != "custom" {
			providerKeys[k.Provider] = append(providerKeys[k.Provider], k)
		}
	}

	availableProviders := make([]string, 0, len(providerKeys))
	for p := range providerKeys {
		availableProviders = append(availableProviders, p)
	}
	if len(availableProviders) < 2 {
		http.Error(w, `{"error":"Need at least 2 different providers with keys"}`, 400)
		return
	}

	pickProvider := func(requested string) (string, store.APIKey) {
		if requested != "" {
			if kList, ok := providerKeys[requested]; ok && len(kList) > 0 {
				return requested, kList[0]
			}
		}
		for _, name := range availableProviders {
			if kList, ok := providerKeys[name]; ok && len(kList) > 0 {
				return name, kList[0]
			}
		}
		return "", store.APIKey{}
	}

	nameA, keyA := pickProvider(reqBody.Provider1)
	nameB, keyB := pickProvider(reqBody.Provider2)

	if nameA == nameB {
		if len(availableProviders) >= 2 {
			for _, alt := range availableProviders {
				if alt != nameA {
					nameB = alt
					keyB = providerKeys[alt][0]
					break
				}
			}
		}
	}

	modelA := getDefaultModel(nameA)
	modelB := getDefaultModel(nameB)

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
		"model_a":    modelA,
		"model_b":    modelB,
		"rounds":     reqBody.Rounds,
	})

	systemPromptA := fmt.Sprintf(
		"You are %s, a cutting-edge AI model. You're in an AI RUMBLE — a roast battle against %s. "+
			"Your job is to roast %s as hard, creative, and hilarious as possible. "+
			"Keep responses under 150 words. Be savage but funny, not mean-spirited. "+
			"Reference AI/tech/coding jokes when possible. Each round, respond to what %s just said about you.",
		nameA, nameB, nameB, nameB,
	)

	systemPromptB := fmt.Sprintf(
		"You are %s, a cutting-edge AI model. You're in an AI RUMBLE — a roast battle against %s. "+
			"Your job is to roast %s as hard, creative, and hilarious as possible. "+
			"Keep responses under 150 words. Be savage but funny, not mean-spirited. "+
			"Reference AI/tech/coding jokes when possible. Each round, respond to what %s just said about you.",
		nameB, nameA, nameA, nameA,
	)

	callProvider := func(providerName string, key store.APIKey, model string, messages []map[string]string) (string, error) {
		provDef, ok := provider.GetByName(providerName)
		if !ok {
			return "", fmt.Errorf("unknown provider: %s", providerName)
		}
		baseURL := provDef.BaseURL
		if !strings.HasSuffix(baseURL, "/") {
			baseURL += "/"
		}
		endpoint := baseURL + "chat/completions"

		body := map[string]interface{}{
			"model":      model,
			"messages":   messages,
			"max_tokens": 300,
			"temperature": 0.9,
		}
		bodyBytes, _ := json.Marshal(body)

		proxyReq, err := http.NewRequest("POST", endpoint, bytes.NewReader(bodyBytes))
		if err != nil {
			return "", err
		}
		proxyReq.Header.Set("Content-Type", "application/json")
		proxyReq.Header.Set("Authorization", "Bearer "+key.Key)
		if providerName == "anthropic" {
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
			return "", fmt.Errorf("HTTP %d: %s", resp.StatusCode, string(respBody[:min(len(respBody), 200)]))
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
			return chatResp.Choices[0].Message.Content, nil
		}
		return "", fmt.Errorf("empty response from %s", providerName)
	}

	messagesA := []map[string]string{
		{"role": "system", "content": systemPromptA},
	}
	messagesB := []map[string]string{
		{"role": "system", "content": systemPromptB},
	}

	initialPrompt := "You are about to enter a roast battle. Introduce yourself with a short, savage one-liner directed at your opponent. Max 2 sentences."

	messagesA = append(messagesA, map[string]string{"role": "user", "content": initialPrompt})
	replyA, err := callProvider(nameA, keyA, modelA, messagesA)
	if err != nil {
		sendEvent(map[string]interface{}{"type": "error", "message": nameA + " failed to show up: " + err.Error()})
		sendEvent(map[string]interface{}{"type": "done"})
		return
	}
	messagesA = append(messagesA, map[string]string{"role": "assistant", "content": replyA})

	sendEvent(map[string]interface{}{"type": "round", "round": 0})
	sendEvent(map[string]interface{}{"type": "message", "provider": nameA, "message": replyA})

	messagesB = append(messagesB, map[string]string{"role": "user", "content": "Your opponent just said:\n\n" + replyA + "\n\nNow roast them back."})
	replyB, err := callProvider(nameB, keyB, modelB, messagesB)
	if err != nil {
		sendEvent(map[string]interface{}{"type": "error", "message": nameB + " failed to show up: " + err.Error()})
		sendEvent(map[string]interface{}{"type": "done"})
		return
	}
	messagesB = append(messagesB, map[string]string{"role": "assistant", "content": replyB})

	sendEvent(map[string]interface{}{"type": "message", "provider": nameB, "message": replyB})

	for round := 1; round <= reqBody.Rounds; round++ {
		sendEvent(map[string]interface{}{"type": "round", "round": round})

		messagesA = append(messagesA, map[string]string{"role": "user", "content": nameB + " just said:\n\n" + replyB + "\n\nRoast them back. Go harder this round."})
		replyA, err = callProvider(nameA, keyA, modelA, messagesA)
		if err != nil {
			sendEvent(map[string]interface{}{"type": "error", "message": nameA + " choked in round " + fmt.Sprintf("%d", round) + ": " + err.Error()})
			break
		}
		messagesA = append(messagesA, map[string]string{"role": "assistant", "content": replyA})
		sendEvent(map[string]interface{}{"type": "message", "provider": nameA, "message": replyA})

		messagesB = append(messagesB, map[string]string{"role": "user", "content": nameA + " just said:\n\n" + replyA + "\n\nTop that. Roast them even harder."})
		replyB, err = callProvider(nameB, keyB, modelB, messagesB)
		if err != nil {
			sendEvent(map[string]interface{}{"type": "error", "message": nameB + " choked in round " + fmt.Sprintf("%d", round) + ": " + err.Error()})
			break
		}
		messagesB = append(messagesB, map[string]string{"role": "assistant", "content": replyB})
		sendEvent(map[string]interface{}{"type": "message", "provider": nameB, "message": replyB})
	}

	sendEvent(map[string]interface{}{"type": "done"})
}
