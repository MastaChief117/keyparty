package proxy

import (
	"bytes"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sort"
	"strings"
	"sync"
	"time"

	"ai-gateway/provider"
	"ai-gateway/store"
)

// ── HEALTH CHECK ──────────────────────────────────────────────────────────

func (p *Proxy) HandleHealthCheck(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	keys, err := p.store.GetKeys()
	if err != nil || len(keys) == 0 {
		http.Error(w, `{"error":"No API keys configured"}`, 400)
		return
	}

	type providerStatus struct {
		Provider  string `json:"provider"`
		Status    string `json:"status"`
		LatencyMs int64  `json:"latency_ms"`
		Error     string `json:"error,omitempty"`
		Model     string `json:"model"`
		KeyCount  int    `json:"key_count"`
	}

	providerKeys := make(map[string][]store.APIKey)
	for _, k := range keys {
		if k.Enabled && k.Provider != "custom" {
			providerKeys[k.Provider] = append(providerKeys[k.Provider], k)
		}
	}

	var results []providerStatus
	var mu sync.Mutex
	var wg sync.WaitGroup

	for pname, pkeys := range providerKeys {
		wg.Add(1)
		go func(pname string, pkeys []store.APIKey) {
			defer wg.Done()

			provDef, ok := provider.GetByName(pname)
			if !ok {
				mu.Lock()
				results = append(results, providerStatus{
					Provider: pname,
					Status:   "unknown",
					Error:    "Provider not found",
					KeyCount: len(pkeys),
				})
				mu.Unlock()
				return
			}

			baseURL := provDef.BaseURL
			if !strings.HasSuffix(baseURL, "/") {
				baseURL += "/"
			}

			model := getDefaultModel(pname)
			body := map[string]interface{}{
				"model":      model,
				"messages":   []map[string]string{{"role": "user", "content": "ping"}},
				"max_tokens": 5,
			}
			bodyBytes, _ := json.Marshal(body)

			proxyReq, _ := http.NewRequest("POST", baseURL+"chat/completions", bytes.NewReader(bodyBytes))
			proxyReq.Header.Set("Content-Type", "application/json")
			proxyReq.Header.Set("Authorization", "Bearer "+pkeys[0].Key)
			if pname == "anthropic" {
				proxyReq.Header.Set("x-api-key", pkeys[0].Key)
				proxyReq.Header.Set("anthropic-version", "2023-06-01")
			}

			start := time.Now()
			resp, err := p.client.Do(proxyReq)
			latency := time.Since(start).Milliseconds()

			status := "healthy"
			errMsg := ""
			if err != nil {
				status = "error"
				errMsg = err.Error()
			} else {
				if resp.StatusCode >= 400 {
					status = "degraded"
					respBody, _ := io.ReadAll(resp.Body)
					resp.Body.Close()
					errMsg = fmt.Sprintf("HTTP %d: %s", resp.StatusCode, string(respBody[:min(len(respBody), 200)]))
				} else {
					resp.Body.Close()
				}
			}

			mu.Lock()
			results = append(results, providerStatus{
				Provider:  pname,
				Status:    status,
				LatencyMs: latency,
				Error:     errMsg,
				Model:     model,
				KeyCount:  len(pkeys),
			})
			mu.Unlock()
		}(pname, pkeys)
	}
	wg.Wait()

	sort.Slice(results, func(i, j int) bool {
		return results[i].Provider < results[j].Provider
	})

	healthyCount := 0
	for _, r := range results {
		if r.Status == "healthy" {
			healthyCount++
		}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"providers": results,
		"total":     len(results),
		"healthy":   healthyCount,
		"timestamp": time.Now().Format(time.RFC3339),
	})
}

// ── EXPORT LOGS ───────────────────────────────────────────────────────────

func (p *Proxy) HandleExportLogs(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	format := r.URL.Query().Get("format")
	if format == "" {
		format = "json"
	}

	limit := 1000
	if l := r.URL.Query().Get("limit"); l != "" {
		fmt.Sscanf(l, "%d", &limit)
	}

	logs, err := p.store.GetRequestLogs(limit)
	if err != nil {
		http.Error(w, `{"error":"Failed to get logs"}`, 500)
		return
	}

	switch format {
	case "csv":
		w.Header().Set("Content-Type", "text/csv")
		w.Header().Set("Content-Disposition", "attachment; filename=keyparty-logs.csv")
		writer := csv.NewWriter(w)
		writer.Write([]string{"ID", "Virtual Key", "Provider", "Model", "Status", "Tokens In", "Tokens Out", "Cost", "Latency (ms)", "Cache Hit", "Timestamp"})
		for _, l := range logs {
			cacheHit := "false"
			if l.CacheHit {
				cacheHit = "true"
			}
			writer.Write([]string{
				fmt.Sprintf("%d", l.ID),
				l.VirtualKey,
				l.Provider,
				l.Model,
				fmt.Sprintf("%d", l.StatusCode),
				fmt.Sprintf("%d", l.TokensIn),
				fmt.Sprintf("%d", l.TokensOut),
				fmt.Sprintf("%.6f", l.Cost),
				fmt.Sprintf("%d", l.Latency),
				cacheHit,
				l.Timestamp.Format(time.RFC3339),
			})
		}
		writer.Flush()
	default:
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("Content-Disposition", "attachment; filename=keyparty-logs.json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"logs":  logs,
			"total": len(logs),
		})
	}
}

// ── PROMPT BUILDER ────────────────────────────────────────────────────────

func (p *Proxy) HandlePromptBuilder(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var reqBody struct {
		Template  string            `json:"template"`
		Variables map[string]string `json:"variables"`
		Provider  string            `json:"provider"`
	}
	if err := json.NewDecoder(r.Body).Decode(&reqBody); err != nil {
		http.Error(w, `{"error":"Invalid JSON"}`, 400)
		return
	}

	if reqBody.Template == "" {
		http.Error(w, `{"error":"template required"}`, 400)
		return
	}

	built := reqBody.Template
	for key, value := range reqBody.Variables {
		built = strings.ReplaceAll(built, "{{"+key+"}}", value)
	}

	if reqBody.Provider != "" {
		keys, err := p.store.GetKeysByProvider(reqBody.Provider)
		if err != nil || len(keys) == 0 {
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(map[string]interface{}{
				"built_prompt": built,
				"enhanced":     false,
			})
			return
		}

		provDef, ok := provider.GetByName(reqBody.Provider)
		if !ok {
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(map[string]interface{}{
				"built_prompt": built,
				"enhanced":     false,
			})
			return
		}

		baseURL := provDef.BaseURL
		if !strings.HasSuffix(baseURL, "/") {
			baseURL += "/"
		}

		body := map[string]interface{}{
			"model":       getDefaultModel(reqBody.Provider),
			"messages":    []map[string]string{{"role": "user", "content": "Enhance this prompt to be more effective and clear:\n\n" + built}},
			"max_tokens":  500,
			"temperature": 0.7,
		}
		bodyBytes, _ := json.Marshal(body)

		proxyReq, _ := http.NewRequest("POST", baseURL+"chat/completions", bytes.NewReader(bodyBytes))
		proxyReq.Header.Set("Content-Type", "application/json")
		proxyReq.Header.Set("Authorization", "Bearer "+keys[0].Key)
		if reqBody.Provider == "anthropic" {
			proxyReq.Header.Set("x-api-key", keys[0].Key)
			proxyReq.Header.Set("anthropic-version", "2023-06-01")
		}

		resp, err := p.client.Do(proxyReq)
		if err == nil {
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
			if len(chatResp.Choices) > 0 {
				w.Header().Set("Content-Type", "application/json")
				json.NewEncoder(w).Encode(map[string]interface{}{
					"built_prompt":    built,
					"enhanced_prompt": stripThinking(chatResp.Choices[0].Message.Content),
					"enhanced":        true,
					"provider":        reqBody.Provider,
				})
				return
			}
		}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"built_prompt": built,
		"enhanced":     false,
	})
}
