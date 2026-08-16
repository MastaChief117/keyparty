// Copyright 2026 KeyParty Contributors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package proxy

import (
	"encoding/json"
	"fmt"
	"math"
	"net/http"
	"strings"
	"time"

	"ai-gateway/store"
)

// ── SMART ROUTER ──────────────────────────────────────────────────────────

func (p *Proxy) HandleSmartRoute(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var reqBody struct {
		Message  string `json:"message"`
		Priority string `json:"priority"`
	}
	if err := json.NewDecoder(r.Body).Decode(&reqBody); err != nil {
		http.Error(w, `{"error":"Invalid JSON"}`, 400)
		return
	}
	if reqBody.Message == "" {
		http.Error(w, `{"error":"message required"}`, 400)
		return
	}
	if reqBody.Priority == "" {
		reqBody.Priority = "cost"
	}

	complexity := len(reqBody.Message)
	words := len(strings.Fields(reqBody.Message))

	var recommendedProvider, recommendedModel string
	var reasoning string

	switch reqBody.Priority {
	case "cost":
		if complexity < 50 {
			recommendedProvider = "groq"
			recommendedModel = "llama-3.1-8b-instant"
			reasoning = "Short message + cost priority = cheapest fast model"
		} else if complexity < 200 {
			recommendedProvider = "deepseek"
			recommendedModel = "deepseek-chat"
			reasoning = "Medium message + cost priority = best value"
		} else {
			recommendedProvider = "gemini"
			recommendedModel = "gemini-3.5-flash"
			reasoning = "Long message + cost priority = cheap with large context"
		}
	case "speed":
		if complexity < 100 {
			recommendedProvider = "groq"
			recommendedModel = "llama-3.1-8b-instant"
			reasoning = "Short message + speed priority = blazing fast"
		} else {
			recommendedProvider = "groq"
			recommendedModel = "llama-3.3-70b-versatile"
			reasoning = "Longer message + speed priority = fast with good quality"
		}
	case "quality":
		if words > 500 {
			recommendedProvider = "gemini"
			recommendedModel = "gemini-2.5-pro"
			reasoning = "Very long context needed = Gemini's 1M window"
		} else if complexity < 100 {
			recommendedProvider = "openai"
			recommendedModel = "gpt-4o"
			reasoning = "Short + quality = GPT-4o for best responses"
		} else {
			recommendedProvider = "anthropic"
			recommendedModel = "claude-sonnet-4-5"
			reasoning = "Complex task + quality = Claude's strong analysis"
		}
	}

	keys, _ := p.store.GetKeysByProvider(recommendedProvider)
	if len(keys) == 0 {
		allKeys, _ := p.store.GetKeys()
		if len(allKeys) > 0 {
			recommendedProvider = allKeys[0].Provider
			recommendedModel = getDefaultModel(allKeys[0].Provider)
			reasoning = "Fallback: no keys for recommended provider"
		}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"recommended_provider": recommendedProvider,
		"recommended_model":    recommendedModel,
		"reasoning":            reasoning,
		"message_length":       complexity,
		"word_count":           words,
		"priority":             reqBody.Priority,
	})
}

// ── COST LIMITER ──────────────────────────────────────────────────────────

func (p *Proxy) HandleCostLimiter(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	switch r.Method {
	case "GET":
		limits, _ := p.store.GetCostLimits()
		json.NewEncoder(w).Encode(map[string]interface{}{
			"limits": limits,
			"total":  len(limits),
		})
	case "POST":
		var req struct {
			Name     string  `json:"name"`
			Period   string  `json:"period"`
			MaxCost  float64 `json:"max_cost"`
			Provider string  `json:"provider"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, `{"error":"Invalid JSON"}`, 400)
			return
		}
		if req.Name == "" || req.MaxCost <= 0 {
			http.Error(w, `{"error":"name and max_cost required"}`, 400)
			return
		}
		if req.Period == "" {
			req.Period = "daily"
		}
		id, err := p.store.AddCostLimit(req.Name, req.Period, req.MaxCost, req.Provider)
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
		if err := p.store.DeleteCostLimit(req.ID); err != nil {
			http.Error(w, `{"error":"Failed"}`, 500)
			return
		}
		json.NewEncoder(w).Encode(map[string]string{"status": "deleted"})
	default:
		http.Error(w, "Method not allowed", 405)
	}
}

// ── PROVIDER UPTIME TRACKER ──────────────────────────────────────────────

func (p *Proxy) HandleUptime(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	days := 7
	if d := r.URL.Query().Get("days"); d != "" {
		fmt.Sscanf(d, "%d", &days)
	}

	uptime, err := p.store.GetProviderUptime(days)
	if err != nil {
		uptime = []store.ProviderUptime{}
	}

	type uptimeResult struct {
		Provider   string  `json:"provider"`
		TotalReqs  int64   `json:"total_requests"`
		FailedReqs int64   `json:"failed_requests"`
		Uptime     float64 `json:"uptime_percent"`
		AvgLatency float64 `json:"avg_latency_ms"`
	}

	var results []uptimeResult
	for _, u := range uptime {
		uptimePercent := 100.0
		if u.TotalReqs > 0 {
			uptimePercent = float64(u.TotalReqs-u.FailedReqs) / float64(u.TotalReqs) * 100
		}
		results = append(results, uptimeResult{
			Provider:   u.Provider,
			TotalReqs:  u.TotalReqs,
			FailedReqs: u.FailedReqs,
			Uptime:     math.Round(uptimePercent*100) / 100,
			AvgLatency: u.AvgLatency,
		})
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"providers": results,
		"period":    fmt.Sprintf("%d days", days),
	})
}

// ── REQUEST REPLAY QUEUE ─────────────────────────────────────────────────

func (p *Proxy) HandleReplayQueue(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	switch r.Method {
	case "GET":
		logs, err := p.store.GetRequestLogs(100)
		if err != nil {
			http.Error(w, `{"error":"Failed"}`, 500)
			return
		}
		var failed []store.RequestLog
		for _, l := range logs {
			if l.StatusCode >= 400 {
				failed = append(failed, l)
			}
		}
		json.NewEncoder(w).Encode(map[string]interface{}{
			"failed_requests": failed,
			"total":           len(failed),
		})
	case "POST":
		var req struct {
			Provider string              `json:"provider"`
			Model    string              `json:"model"`
			Messages []map[string]string `json:"messages"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, `{"error":"Invalid JSON"}`, 400)
			return
		}
		if req.Provider == "" || req.Model == "" {
			http.Error(w, `{"error":"provider and model required"}`, 400)
			return
		}

		keys, _ := p.store.GetKeysByProvider(req.Provider)
		if len(keys) == 0 {
			http.Error(w, `{"error":"No keys for provider"}`, 400)
			return
		}

		start := time.Now()
		reply, tokensIn, tokensOut, err := p.callProviderForPro(req.Provider, keys[0], req.Model, req.Messages)
		latency := time.Since(start).Milliseconds()

		if err != nil {
			json.NewEncoder(w).Encode(map[string]interface{}{"error": err.Error(), "latency_ms": latency})
			return
		}
		json.NewEncoder(w).Encode(map[string]interface{}{
			"reply":      reply,
			"tokens_in":  tokensIn,
			"tokens_out": tokensOut,
			"latency_ms": latency,
			"status":     "replayed",
		})
	default:
		http.Error(w, "Method not allowed", 405)
	}
}

// ── CUSTOM MODEL REGISTRY ────────────────────────────────────────────────

func (p *Proxy) HandleCustomModels(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	switch r.Method {
	case "GET":
		models, _ := p.store.GetCustomModels()
		json.NewEncoder(w).Encode(map[string]interface{}{
			"models": models,
			"total":  len(models),
		})
	case "POST":
		var req struct {
			Name        string  `json:"name"`
			Provider    string  `json:"provider"`
			DisplayName string  `json:"display_name"`
			ContextSize int     `json:"context_size"`
			InputPrice  float64 `json:"input_price"`
			OutputPrice float64 `json:"output_price"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, `{"error":"Invalid JSON"}`, 400)
			return
		}
		if req.Name == "" || req.Provider == "" {
			http.Error(w, `{"error":"name and provider required"}`, 400)
			return
		}
		id, err := p.store.AddCustomModel(req.Name, req.Provider, req.DisplayName, req.ContextSize, req.InputPrice, req.OutputPrice)
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
		if err := p.store.DeleteCustomModel(req.ID); err != nil {
			http.Error(w, `{"error":"Failed"}`, 500)
			return
		}
		json.NewEncoder(w).Encode(map[string]string{"status": "deleted"})
	default:
		http.Error(w, "Method not allowed", 405)
	}
}
