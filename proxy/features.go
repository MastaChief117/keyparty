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
	"net/http"
	"sort"
	"strings"
	"sync"
	"time"

	"ai-gateway/provider"
	"ai-gateway/store"
)

// ── Token Calculator ────────────────────────────────────────────────────

func (p *Proxy) HandleTokenCalculator(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		TokensIn  int    `json:"tokens_in"`
		TokensOut int    `json:"tokens_out"`
		Model     string `json:"model"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, `{"error":"Invalid JSON"}`, 400)
		return
	}
	if req.TokensIn == 0 && req.TokensOut == 0 {
		req.TokensIn = 1000
		req.TokensOut = 500
	}

	pricing := map[string]map[string][2]float64{
		"openai": {
			"gpt-4o":            {2.5 / 1000000, 10 / 1000000},
			"gpt-4o-mini":       {0.15 / 1000000, 0.6 / 1000000},
			"gpt-4.1":           {2 / 1000000, 8 / 1000000},
			"gpt-4.1-mini":      {0.4 / 1000000, 1.6 / 1000000},
			"o3":                {10 / 1000000, 40 / 1000000},
			"o4-mini":           {1.1 / 1000000, 4.4 / 1000000},
			"gpt-4.1-nano":      {0.1 / 1000000, 0.4 / 1000000},
		},
		"anthropic": {
			"claude-sonnet-4-5": {3 / 1000000, 15 / 1000000},
			"claude-haiku-4-5":  {0.8 / 1000000, 4 / 1000000},
			"claude-opus-4-5":   {15 / 1000000, 75 / 1000000},
		},
		"gemini": {
			"gemini-3.6-flash":         {0.10 / 1000000, 0.40 / 1000000},
			"gemini-3.5-flash":         {0.15 / 1000000, 0.60 / 1000000},
			"gemini-3.5-flash-lite":    {0.075 / 1000000, 0.30 / 1000000},
			"gemini-3.1-flash-lite":    {0.075 / 1000000, 0.30 / 1000000},
			"gemini-3.1-pro-preview":   {2.00 / 1000000, 12.00 / 1000000},
			"gemini-3-flash-preview":   {0.50 / 1000000, 3.00 / 1000000},
			"gemini-2.5-pro":           {1.25 / 1000000, 10.00 / 1000000},
			"gemini-2.5-flash":         {0.15 / 1000000, 0.60 / 1000000},
			"gemini-2.5-flash-lite":    {0.075 / 1000000, 0.30 / 1000000},
		},
		"groq": {
			"llama-3.3-70b-versatile": {0.59 / 1000000, 0.79 / 1000000},
			"llama-3.1-8b-instant":    {0.05 / 1000000, 0.08 / 1000000},
			"mixtral-8x7b-32768":      {0.24 / 1000000, 0.24 / 1000000},
		},
		"deepseek": {
			"deepseek-chat": {0.14 / 1000000, 0.28 / 1000000},
			"deepseek-r1":   {0.55 / 1000000, 2.19 / 1000000},
		},
		"together": {
			"meta-llama/Llama-3.3-70B-Instruct-Turbo": {0.88 / 1000000, 0.88 / 1000000},
			"Qwen/Qwen2.5-72B-Instruct-Turbo":         {1.2 / 1000000, 1.2 / 1000000},
		},
		"mistral": {
			"mistral-large-latest": {2 / 1000000, 6 / 1000000},
			"codestral-latest":     {0.3 / 1000000, 0.9 / 1000000},
		},
		"fireworks": {
			"accounts/fireworks/models/llama-v3p3-70b-instruct": {0.9 / 1000000, 0.9 / 1000000},
		},
		"nvidia": {
			"nvidia/llama-3.1-nemotron-70b-instruct": {0.36 / 1000000, 0.36 / 1000000},
		},
	}

	type modelResult struct {
		Provider    string  `json:"provider"`
		Model       string  `json:"model"`
		InputCost   float64 `json:"input_cost"`
		OutputCost  float64 `json:"output_cost"`
		TotalCost   float64 `json:"total_cost"`
		CostPer1K   float64 `json:"cost_per_1k"`
		CostPer1M   float64 `json:"cost_per_1m"`
	}

	var results []modelResult

	for provider, models := range pricing {
		for model, costs := range models {
			inputCost := float64(req.TokensIn) * costs[0]
			outputCost := float64(req.TokensOut) * costs[1]
			total := inputCost + outputCost
			results = append(results, modelResult{
				Provider:   provider,
				Model:      model,
				InputCost:  inputCost,
				OutputCost: outputCost,
				TotalCost:  total,
				CostPer1K:  total / float64(req.TokensIn+req.TokensOut) * 1000,
				CostPer1M:  total / float64(req.TokensIn+req.TokensOut) * 1000000,
			})
		}
	}

	sort.Slice(results, func(i, j int) bool {
		return results[i].TotalCost < results[j].TotalCost
	})

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"tokens_in":  req.TokensIn,
		"tokens_out": req.TokensOut,
		"results":    results,
		"cheapest":   results[0],
		"most_expensive": results[len(results)-1],
	})
}

// ── Model Explorer ──────────────────────────────────────────────────────

type ModelInfo struct {
	Name           string   `json:"name"`
	Provider       string   `json:"provider"`
	FullName       string   `json:"full_name"`
	ContextWindow  int      `json:"context_window"`
	InputPrice     float64  `json:"input_price_per_1m"`
	OutputPrice    float64  `json:"output_price_per_1m"`
	Capabilities   []string `json:"capabilities"`
	RecommendedFor []string `json:"recommended_for"`
	Speed          string   `json:"speed"`
	MaxOutput      int      `json:"max_output_tokens"`
}

func (p *Proxy) HandleModelExplorer(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	models := []ModelInfo{
		{Name: "gpt-4o", Provider: "openai", FullName: "GPT-4o", ContextWindow: 128000, InputPrice: 2.50, OutputPrice: 10.00, Capabilities: []string{"chat", "vision", "function_calling"}, RecommendedFor: []string{"general", "coding", "analysis"}, Speed: "fast", MaxOutput: 16384},
		{Name: "gpt-4o-mini", Provider: "openai", FullName: "GPT-4o Mini", ContextWindow: 128000, InputPrice: 0.15, OutputPrice: 0.60, Capabilities: []string{"chat", "vision", "function_calling"}, RecommendedFor: []string{"quick tasks", "high volume", "cost-sensitive"}, Speed: "very fast", MaxOutput: 16384},
		{Name: "gpt-4.1", Provider: "openai", FullName: "GPT-4.1", ContextWindow: 1047576, InputPrice: 2.00, OutputPrice: 8.00, Capabilities: []string{"chat", "vision", "function_calling", "long context"}, RecommendedFor: []string{"coding", "analysis", "long documents"}, Speed: "fast", MaxOutput: 32768},
		{Name: "gpt-4.1-mini", Provider: "openai", FullName: "GPT-4.1 Mini", ContextWindow: 1047576, InputPrice: 0.40, OutputPrice: 1.60, Capabilities: []string{"chat", "vision", "function_calling"}, RecommendedFor: []string{"coding", "balanced cost/quality"}, Speed: "fast", MaxOutput: 32768},
		{Name: "gpt-4.1-nano", Provider: "openai", FullName: "GPT-4.1 Nano", ContextWindow: 1047576, InputPrice: 0.10, OutputPrice: 0.40, Capabilities: []string{"chat", "function_calling"}, RecommendedFor: []string{"ultra cheap", "simple tasks"}, Speed: "very fast", MaxOutput: 32768},
		{Name: "o3", Provider: "openai", FullName: "o3", ContextWindow: 200000, InputPrice: 10.00, OutputPrice: 40.00, Capabilities: []string{"chat", "reasoning", "coding"}, RecommendedFor: []string{"complex reasoning", "math", "science"}, Speed: "slow", MaxOutput: 100000},
		{Name: "o4-mini", Provider: "openai", FullName: "o4-mini", ContextWindow: 200000, InputPrice: 1.10, OutputPrice: 4.40, Capabilities: []string{"chat", "reasoning", "coding"}, RecommendedFor: []string{"reasoning on a budget"}, Speed: "medium", MaxOutput: 100000},
		{Name: "claude-sonnet-4-5", Provider: "anthropic", FullName: "Claude Sonnet 4.5", ContextWindow: 200000, InputPrice: 3.00, OutputPrice: 15.00, Capabilities: []string{"chat", "vision", "coding", "analysis"}, RecommendedFor: []string{"coding", "analysis", "writing"}, Speed: "fast", MaxOutput: 16384},
		{Name: "claude-haiku-4-5", Provider: "anthropic", FullName: "Claude Haiku 4.5", ContextWindow: 200000, InputPrice: 0.80, OutputPrice: 4.00, Capabilities: []string{"chat", "vision", "coding"}, RecommendedFor: []string{"quick tasks", "cost-sensitive"}, Speed: "very fast", MaxOutput: 8192},
		{Name: "claude-opus-4-5", Provider: "anthropic", FullName: "Claude Opus 4.5", ContextWindow: 200000, InputPrice: 15.00, OutputPrice: 75.00, Capabilities: []string{"chat", "vision", "coding", "analysis", "reasoning"}, RecommendedFor: []string{"highest quality", "complex tasks"}, Speed: "slow", MaxOutput: 32768},
		{Name: "gemini-3.6-flash", Provider: "gemini", FullName: "Gemini 3.6 Flash", ContextWindow: 1048576, InputPrice: 0.10, OutputPrice: 0.40, Capabilities: []string{"chat", "vision", "long context"}, RecommendedFor: []string{"fastest, latest stable"}, Speed: "very fast", MaxOutput: 65536},
		{Name: "gemini-3.5-flash", Provider: "gemini", FullName: "Gemini 3.5 Flash", ContextWindow: 1048576, InputPrice: 0.15, OutputPrice: 0.60, Capabilities: []string{"chat", "vision", "long context"}, RecommendedFor: []string{"balanced speed/quality"}, Speed: "very fast", MaxOutput: 65536},
		{Name: "gemini-3.5-flash-lite", Provider: "gemini", FullName: "Gemini 3.5 Flash Lite", ContextWindow: 1048576, InputPrice: 0.075, OutputPrice: 0.30, Capabilities: []string{"chat", "vision", "long context"}, RecommendedFor: []string{"ultra cheap", "high volume"}, Speed: "very fast", MaxOutput: 65536},
		{Name: "gemini-3.1-flash-lite", Provider: "gemini", FullName: "Gemini 3.1 Flash Lite", ContextWindow: 1048576, InputPrice: 0.075, OutputPrice: 0.30, Capabilities: []string{"chat", "vision", "long context"}, RecommendedFor: []string{"cost-effective stable"}, Speed: "very fast", MaxOutput: 65536},
		{Name: "gemini-3.1-pro-preview", Provider: "gemini", FullName: "Gemini 3.1 Pro Preview", ContextWindow: 1048576, InputPrice: 2.00, OutputPrice: 12.00, Capabilities: []string{"chat", "vision", "reasoning", "long context", "coding"}, RecommendedFor: []string{"complex reasoning", "analysis"}, Speed: "medium", MaxOutput: 65536},
		{Name: "gemini-3-flash-preview", Provider: "gemini", FullName: "Gemini 3 Flash Preview", ContextWindow: 1048576, InputPrice: 0.50, OutputPrice: 3.00, Capabilities: []string{"chat", "vision", "long context"}, RecommendedFor: []string{"fast Gemini 3 preview"}, Speed: "fast", MaxOutput: 65536},
		{Name: "gemini-2.5-pro", Provider: "gemini", FullName: "Gemini 2.5 Pro", ContextWindow: 1048576, InputPrice: 1.25, OutputPrice: 10.00, Capabilities: []string{"chat", "vision", "long context", "coding", "reasoning"}, RecommendedFor: []string{"long documents", "analysis"}, Speed: "fast", MaxOutput: 65536},
		{Name: "gemini-2.5-flash", Provider: "gemini", FullName: "Gemini 2.5 Flash", ContextWindow: 1048576, InputPrice: 0.15, OutputPrice: 0.60, Capabilities: []string{"chat", "vision", "long context"}, RecommendedFor: []string{"balanced, production stable"}, Speed: "very fast", MaxOutput: 65536},
		{Name: "llama-3.3-70b-versatile", Provider: "groq", FullName: "Llama 3.3 70B (Groq)", ContextWindow: 128000, InputPrice: 0.59, OutputPrice: 0.79, Capabilities: []string{"chat", "function_calling"}, RecommendedFor: []string{"fast inference", "cost-effective"}, Speed: "very fast", MaxOutput: 32768},
		{Name: "llama-3.1-8b-instant", Provider: "groq", FullName: "Llama 3.1 8B (Groq)", ContextWindow: 128000, InputPrice: 0.05, OutputPrice: 0.08, Capabilities: []string{"chat"}, RecommendedFor: []string{"ultra fast", "simple tasks"}, Speed: "blazing fast", MaxOutput: 8192},
		{Name: "deepseek-chat", Provider: "deepseek", FullName: "DeepSeek V3", ContextWindow: 128000, InputPrice: 0.14, OutputPrice: 0.28, Capabilities: []string{"chat", "coding"}, RecommendedFor: []string{"coding", "cheap quality"}, Speed: "fast", MaxOutput: 8192},
		{Name: "deepseek-r1", Provider: "deepseek", FullName: "DeepSeek R1", ContextWindow: 128000, InputPrice: 0.55, OutputPrice: 2.19, Capabilities: []string{"chat", "reasoning", "coding"}, RecommendedFor: []string{"reasoning", "math", "coding"}, Speed: "medium", MaxOutput: 8192},
		{Name: "mistral-large-latest", Provider: "mistral", FullName: "Mistral Large", ContextWindow: 128000, InputPrice: 2.00, OutputPrice: 6.00, Capabilities: []string{"chat", "coding", "function_calling"}, RecommendedFor: []string{"multilingual", "coding"}, Speed: "fast", MaxOutput: 32768},
		{Name: "codestral-latest", Provider: "mistral", FullName: "Codestral", ContextWindow: 32000, InputPrice: 0.30, OutputPrice: 0.90, Capabilities: []string{"coding", "fill_in_middle"}, RecommendedFor: []string{"code completion", "coding"}, Speed: "fast", MaxOutput: 8192},
	}

	capability := r.URL.Query().Get("capability")
	provider := r.URL.Query().Get("provider")
	search := strings.ToLower(r.URL.Query().Get("q"))

	var filtered []ModelInfo
	for _, m := range models {
		if capability != "" {
			hasCap := false
			for _, c := range m.Capabilities {
				if strings.EqualFold(c, capability) {
					hasCap = true
					break
				}
			}
			if !hasCap {
				continue
			}
		}
		if provider != "" && !strings.EqualFold(m.Provider, provider) {
			continue
		}
		if search != "" && !strings.Contains(strings.ToLower(m.Name), search) && !strings.Contains(strings.ToLower(m.FullName), search) {
			continue
		}
		filtered = append(filtered, m)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"models":  filtered,
		"total":   len(filtered),
		"filters": map[string]string{"capability": capability, "provider": provider, "q": search},
	})
}

// ── Model Comparison ────────────────────────────────────────────────────

func (p *Proxy) HandleModelCompare(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		Message     string   `json:"message"`
		Providers   []string `json:"providers"`
		MaxTokens   int      `json:"max_tokens"`
		Temperature float64  `json:"temperature"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, `{"error":"Invalid JSON"}`, 400)
		return
	}
	if req.Message == "" {
		http.Error(w, `{"error":"message required"}`, 400)
		return
	}
	if len(req.Providers) < 2 {
		req.Providers = []string{"openai", "anthropic", "gemini"}
	}
	if req.MaxTokens == 0 {
		req.MaxTokens = 256
	}
	if req.Temperature == 0 {
		req.Temperature = 0.7
	}

	messages := []map[string]string{{"role": "user", "content": req.Message}}

	type compareResult struct {
		Provider string `json:"provider"`
		Reply    string `json:"reply"`
		Tokens   int    `json:"tokens"`
		Latency  int    `json:"latency_ms"`
		Error    string `json:"error,omitempty"`
	}

	type raceResult struct {
		Results []compareResult `json:"results"`
	}

	keys, err := p.store.GetKeys()
	if err != nil {
		http.Error(w, `{"error":"Failed to get keys"}`, 500)
		return
	}

	type providerKey struct {
		provider string
		key      interface{ GetKey() string }
	}

	var enabledKeys []store.APIKey
	for _, k := range keys {
		if k.Enabled {
			for _, rp := range req.Providers {
				if strings.EqualFold(k.Provider, rp) {
					enabledKeys = append(enabledKeys, k)
				}
			}
		}
	}

	if len(enabledKeys) == 0 {
		http.Error(w, `{"error":"No enabled keys for requested providers"}`, 400)
		return
	}

	var results []compareResult
	var wg sync.WaitGroup
	var mu sync.Mutex

	for _, k := range enabledKeys {
		wg.Add(1)
		go func(key store.APIKey) {
			defer wg.Done()
			start := time.Now()
			reply, tokensIn, tokensOut, err := p.callProviderForPro(key.Provider, key, key.Model, messages)
			latency := int(time.Since(start).Milliseconds())

			mu.Lock()
			defer mu.Unlock()
			if err != nil {
				results = append(results, compareResult{
					Provider: key.Provider + "/" + key.Model,
					Error:    err.Error(),
					Latency:  latency,
				})
			} else {
				results = append(results, compareResult{
					Provider: key.Provider + "/" + key.Model,
					Reply:    reply,
					Tokens:   tokensIn + tokensOut,
					Latency:  latency,
				})
			}
		}(k)
	}
	wg.Wait()

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(raceResult{Results: results})
}

// ── Prompt Library Enhancements ─────────────────────────────────────────

func (p *Proxy) HandleTemplateSearch(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	q := strings.ToLower(r.URL.Query().Get("q"))

	templates, err := p.store.GetTemplates()
	if err != nil {
		http.Error(w, `{"error":"Failed to get templates"}`, 500)
		return
	}

	type templateResult struct {
		Name      string    `json:"name"`
		Content   string    `json:"content"`
		Category  string    `json:"category"`
		CreatedAt time.Time `json:"created_at"`
		UpdatedAt time.Time `json:"updated_at"`
	}

	var results []templateResult
	for _, t := range templates {
		if q != "" && !strings.Contains(strings.ToLower(t.Name), q) && !strings.Contains(strings.ToLower(t.SystemPrompt), q) {
			continue
		}
		results = append(results, templateResult{
			Name:      t.Name,
			Content:   t.SystemPrompt,
			CreatedAt: t.CreatedAt,
			UpdatedAt: t.UpdatedAt,
		})
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"templates": results,
		"total":     len(results),
	})
}

// ── Shareable Links ─────────────────────────────────────────────────────

func (p *Proxy) HandleShareLink(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		Type    string `json:"type"`    // "rumble", "rap", "compare", "poll"
		Data    string `json:"data"`    // JSON-encoded result data
		Expiry  int    `json:"expiry"`  // minutes, 0 = never
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, `{"error":"Invalid JSON"}`, 400)
		return
	}

	if req.Expiry == 0 {
		req.Expiry = 60
	}

	w.Header().Set("Content-Type", "application/json")
	shareID := fmt.Sprintf("%s_%d", req.Type, time.Now().UnixNano())
	json.NewEncoder(w).Encode(map[string]interface{}{
		"share_id": shareID,
		"expiry":   req.Expiry,
		"url":      fmt.Sprintf("/share/%s", shareID),
	})
}

// ── Smart Routing Profiles ──────────────────────────────────────────────

func (p *Proxy) HandleRoutingProfiles(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	profiles := []map[string]interface{}{
		{"name": "fastest", "description": "Route to the fastest provider", "providers": []string{"groq", "nvidia"}, "strategy": "latency"},
		{"name": "cheapest", "description": "Route to the cheapest provider", "providers": []string{"groq", "deepseek"}, "strategy": "cost"},
		{"name": "quality", "description": "Route to the highest quality provider", "providers": []string{"anthropic", "openai"}, "strategy": "quality"},
		{"name": "coding", "description": "Best models for code generation", "providers": []string{"anthropic", "deepseek", "mistral"}, "strategy": "capability"},
		{"name": "vision", "description": "Models with vision/multimodal support", "providers": []string{"openai", "anthropic", "gemini"}, "strategy": "capability"},
		{"name": "creative", "description": "Best for creative writing and brainstorming", "providers": []string{"anthropic", "openai"}, "strategy": "capability"},
		{"name": "long_context", "description": "Models with 1M+ context windows", "providers": []string{"gemini", "openai"}, "strategy": "capability"},
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"profiles": profiles,
	})
}

// ── Provider Profiles ───────────────────────────────────────────────────

func (p *Proxy) HandleProviderProfiles(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	profiles := []map[string]interface{}{
		{
			"name": "openai", "display_name": "OpenAI",
			"strengths": []string{"Best overall quality", "Huge model variety", "Strong reasoning (o3)"},
			"weaknesses": []string{"Expensive for high volume", "Rate limits on free tier"},
			"context_window": "128K-1M", "pricing_tier": "premium",
			"best_for": []string{"General purpose", "Complex reasoning", "Vision tasks"},
			"url": "https://platform.openai.com",
		},
		{
			"name": "anthropic", "display_name": "Anthropic",
			"strengths": []string{"Best for coding", "Excellent analysis", "Long context"},
			"weaknesses": []string{"No free tier", "Can be pricey"},
			"context_window": "200K", "pricing_tier": "premium",
			"best_for": []string{"Coding", "Analysis", "Writing"},
			"url": "https://console.anthropic.com",
		},
		{
			"name": "gemini", "display_name": "Google Gemini",
			"strengths": []string{"Massive context (1M)", "Generous free tier", "Fast inference"},
			"weaknesses": []string{"Less consistent quality", "Can be verbose"},
			"context_window": "1M+", "pricing_tier": "budget",
			"best_for": []string{"Long documents", "High volume", "Cost-sensitive"},
			"url": "https://aistudio.google.com",
		},
		{
			"name": "groq", "display_name": "Groq",
			"strengths": []string{"Blazing fast inference", "Very generous free tier", "Low cost"},
			"weaknesses": []string{"Limited model selection", "Smaller context windows"},
			"context_window": "128K", "pricing_tier": "budget",
			"best_for": []string{"Speed-critical tasks", "Prototyping", "High throughput"},
			"url": "https://console.groq.com",
		},
		{
			"name": "deepseek", "display_name": "DeepSeek",
			"strengths": []string{"Cheapest option", "Strong coding", "Good reasoning"},
			"weaknesses": []string{"Chinese company", "Occasional latency"},
			"context_window": "128K", "pricing_tier": "ultra budget",
			"best_for": []string{"Budget coding", "Reasoning tasks", "Cost optimization"},
			"url": "https://platform.deepseek.com",
		},
		{
			"name": "mistral", "display_name": "Mistral AI",
			"strengths": []string{"Strong multilingual", "Good coding", "European company"},
			"weaknesses": []string{"Smaller ecosystem", "Limited free tier"},
			"context_window": "32K-128K", "pricing_tier": "mid-range",
			"best_for": []string{"Multilingual tasks", "Code completion", "EU data residency"},
			"url": "https://console.mistral.ai",
		},
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"providers": profiles,
	})
}

// ── Tournament: Get Available Models from Saved Keys ───────────────────

func (p *Proxy) HandleTournamentModels(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	keys, err := p.store.GetKeys()
	if err != nil {
		http.Error(w, `{"error":"Failed to get keys"}`, 500)
		return
	}

	type modelOption struct {
		Value    string `json:"value"`
		Label    string `json:"label"`
		Provider string `json:"provider"`
		Model    string `json:"model"`
	}

	seen := map[string]bool{}
	var options []modelOption
	for _, k := range keys {
		if !k.Enabled || k.Provider == "custom" {
			continue
		}
		model := k.Model
		if model == "" {
			model = "default"
		}
		key := k.Provider + "/" + model
		if seen[key] {
			continue
		}
		seen[key] = true
		options = append(options, modelOption{
			Value:    key,
			Label:    k.Name + " (" + k.Provider + "/" + model + ")",
			Provider: k.Provider,
			Model:    model,
		})
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"models": options,
		"total":  len(options),
	})
}

// ── AI Tournament (Bracket-Style) ──────────────────────────────────────

func (p *Proxy) HandleTournament(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		Message string   `json:"message"`
		Models  []string `json:"models"` // e.g. ["openai/gpt-4o", "anthropic/claude-sonnet-4-5", "groq/llama-3.3-70b-versatile"]
		Rounds  int      `json:"rounds"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, `{"error":"Invalid JSON"}`, 400)
		return
	}
	if req.Message == "" {
		req.Message = "Write a short poem about the future of AI."
	}
	if len(req.Models) < 2 {
		req.Models = []string{"openai/gpt-4o", "anthropic/claude-sonnet-4-5", "groq/llama-3.3-70b-versatile", "deepseek/deepseek-chat"}
	}
	if req.Rounds == 0 {
		req.Rounds = 3
	}

	type MatchResult struct {
		Round    int    `json:"round"`
		Model    string `json:"model"`
		Reply    string `json:"reply"`
		Tokens   int    `json:"tokens"`
		Latency  int    `json:"latency_ms"`
		Winner   bool   `json:"winner"`
		Error    string `json:"error,omitempty"`
	}

	type Bracket struct {
		Round    int           `json:"round"`
		Matches  []MatchResult `json:"matches"`
	}

	keys, err := p.store.GetKeys()
	if err != nil || len(keys) == 0 {
		http.Error(w, `{"error":"No API keys configured"}`, 400)
		return
	}

	type modelSpec struct {
		provider string
		model    string
	}
	var specs []modelSpec
	for _, m := range req.Models {
		parts := strings.SplitN(m, "/", 2)
		if len(parts) == 2 {
			specs = append(specs, modelSpec{provider: parts[0], model: parts[1]})
		}
	}

	var brackets []Bracket
	remaining := specs

	for round := 1; round <= req.Rounds && len(remaining) > 1; round++ {
		var matches []MatchResult
		var next []modelSpec

		for i := 0; i < len(remaining); i += 2 {
			if i+1 >= len(remaining) {
				next = append(next, remaining[i])
				continue
			}
			a, b := remaining[i], remaining[i+1]

			var resultA, resultB MatchResult
			var wg sync.WaitGroup
			var mu sync.Mutex

			// Find the actual API keys for these models
			var keyA, keyB store.APIKey
			for _, k := range keys {
				if k.Provider == a.provider && k.Model == a.model && k.Enabled && keyA.Key == "" {
					keyA = k
				}
				if k.Provider == b.provider && k.Model == b.model && k.Enabled && keyB.Key == "" {
					keyB = k
				}
			}

			if keyA.Key == "" {
				resultA = MatchResult{Round: round, Model: a.provider + "/" + a.model, Error: "no enabled key found for model"}
			}
			if keyB.Key == "" {
				resultB = MatchResult{Round: round, Model: b.provider + "/" + b.model, Error: "no enabled key found for model"}
			}

			wg.Add(2)
			go func(spec modelSpec, apiKey store.APIKey) {
				defer wg.Done()
				if apiKey.Key == "" {
					return
				}
				start := time.Now()
				reply, tokens, _, err := p.callProviderForPro(spec.provider, apiKey, spec.model, []map[string]string{{"role": "user", "content": req.Message}})
				latency := int(time.Since(start).Milliseconds())
				mu.Lock()
				defer mu.Unlock()
				if err != nil {
					resultA = MatchResult{Round: round, Model: spec.provider + "/" + spec.model, Latency: latency, Error: err.Error()}
				} else {
					resultA = MatchResult{Round: round, Model: spec.provider + "/" + spec.model, Reply: reply, Tokens: tokens, Latency: latency}
				}
			}(a, keyA)
			go func(spec modelSpec, apiKey store.APIKey) {
				defer wg.Done()
				if apiKey.Key == "" {
					return
				}
				start := time.Now()
				reply, tokens, _, err := p.callProviderForPro(spec.provider, apiKey, spec.model, []map[string]string{{"role": "user", "content": req.Message}})
				latency := int(time.Since(start).Milliseconds())
				mu.Lock()
				defer mu.Unlock()
				if err != nil {
					resultB = MatchResult{Round: round, Model: spec.provider + "/" + spec.model, Latency: latency, Error: err.Error()}
				} else {
					resultB = MatchResult{Round: round, Model: spec.provider + "/" + spec.model, Reply: reply, Tokens: tokens, Latency: latency}
				}
			}(b, keyB)
			wg.Wait()

			if resultA.Error != "" && resultB.Error == "" {
				resultB.Winner = true
				next = append(next, b)
			} else if resultB.Error != "" && resultA.Error == "" {
				resultA.Winner = true
				next = append(next, a)
			} else if resultA.Error == "" && resultB.Error == "" {
				if len(resultA.Reply) > len(resultB.Reply) {
					resultA.Winner = true
					next = append(next, a)
				} else {
					resultB.Winner = true
					next = append(next, b)
				}
			}

			matches = append(matches, resultA, resultB)
		}
		brackets = append(brackets, Bracket{Round: round, Matches: matches})
		remaining = next
	}

	champion := ""
	if len(remaining) > 0 {
		champion = remaining[0].provider + "/" + remaining[0].model
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"brackets":   brackets,
		"champion":   champion,
		"total_rounds": len(brackets),
		"message":    req.Message,
	})
}

// ── Key Expiration ─────────────────────────────────────────────────────

func (p *Proxy) HandleSetKeyExpiry(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	http.Error(w, `{"error":"Key expiry not yet implemented"}`, http.StatusNotImplemented)
}

// ── IP Allowlist ───────────────────────────────────────────────────────

func (p *Proxy) HandleIPAllowlist(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	switch r.Method {
	case "GET":
		json.NewEncoder(w).Encode(map[string]interface{}{
			"allowed_ips": []string{},
			"enabled":     false,
		})
	case "POST":
		var req struct {
			KeyID      int64    `json:"key_id"`
			AllowedIPs []string `json:"allowed_ips"`
			Enabled    bool     `json:"enabled"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, `{"error":"Invalid JSON"}`, 400)
			return
		}
		http.Error(w, `{"error":"IP allowlist not yet implemented"}`, http.StatusNotImplemented)
	default:
		http.Error(w, "Method not allowed", 405)
	}
}

// ── Prompt Versioning ──────────────────────────────────────────────────

func (p *Proxy) HandleTemplateVersions(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	switch r.Method {
	case "GET":
		name := r.URL.Query().Get("name")
		if name == "" {
			http.Error(w, `{"error":"name required"}`, 400)
			return
		}
		templates, err := p.store.GetTemplates()
		if err != nil {
			http.Error(w, `{"error":"Failed"}`, 500)
			return
		}
		for _, t := range templates {
			if t.Name == name {
				json.NewEncoder(w).Encode(map[string]interface{}{
					"name":     t.Name,
					"content":  t.SystemPrompt,
					"versions": []map[string]interface{}{
						{"version": 1, "content": t.SystemPrompt, "created_at": t.CreatedAt},
					},
				})
				return
			}
		}
		http.Error(w, `{"error":"Template not found"}`, 404)
	case "POST":
		var req struct {
			Name    string `json:"name"`
			Content string `json:"content"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, `{"error":"Invalid JSON"}`, 400)
			return
		}
		http.Error(w, `{"error":"Template versioning not yet implemented"}`, http.StatusNotImplemented)
	default:
		http.Error(w, "Method not allowed", 405)
	}
}

// ── Request Replay ─────────────────────────────────────────────────────

func (p *Proxy) HandleRequestReplay(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	var req struct {
		Provider string            `json:"provider"`
		Model    string            `json:"model"`
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

	keys, err := p.store.GetKeys()
	if err != nil {
		http.Error(w, `{"error":"No keys"}`, 500)
		return
	}
	var target store.APIKey
	for _, k := range keys {
		if k.Enabled && strings.EqualFold(k.Provider, req.Provider) {
			target = k
			break
		}
	}
	if target.Key == "" {
		http.Error(w, `{"error":"No enabled key for provider"}`, 400)
		return
	}

	start := time.Now()
	reply, tokensIn, tokensOut, err := p.callProviderForPro(req.Provider, target, req.Model, req.Messages)
	latency := int(time.Since(start).Milliseconds())

	w.Header().Set("Content-Type", "application/json")
	if err != nil {
		json.NewEncoder(w).Encode(map[string]interface{}{"error": err.Error(), "latency_ms": latency})
	} else {
		json.NewEncoder(w).Encode(map[string]interface{}{
			"reply":      reply,
			"tokens_in":  tokensIn,
			"tokens_out": tokensOut,
			"latency_ms": latency,
			"provider":   req.Provider,
			"model":      req.Model,
		})
	}
}

// ── Streaming Inspector ────────────────────────────────────────────────

func (p *Proxy) HandleStreamingInspect(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	var req struct {
		Provider string            `json:"provider"`
		Model    string            `json:"model"`
		Messages []map[string]string `json:"messages"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, `{"error":"Invalid JSON"}`, 400)
		return
	}

	keys, err := p.store.GetKeys()
	if err != nil {
		http.Error(w, `{"error":"No keys"}`, 500)
		return
	}
	var target store.APIKey
	for _, k := range keys {
		if k.Enabled && strings.EqualFold(k.Provider, req.Provider) {
			target = k
			break
		}
	}
	if target.Key == "" {
		http.Error(w, `{"error":"No enabled key for provider"}`, 400)
		return
	}

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")

	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "Streaming not supported", 500)
		return
	}

	provDef, ok := provider.GetByName(req.Provider)
	if !ok {
		http.Error(w, `{"error":"Unknown provider"}`, 400)
		return
	}

	body := map[string]interface{}{
		"model":       req.Model,
		"messages":    req.Messages,
		"max_tokens":  256,
		"temperature": 0.7,
		"stream":      true,
	}
	bodyBytes, _ := json.Marshal(body)

	proxyReq, err := http.NewRequest("POST", provDef.BaseURL+"/chat/completions", strings.NewReader(string(bodyBytes)))
	if err != nil {
		http.Error(w, `{"error":"Failed to create request"}`, 500)
		return
	}
	proxyReq.Header.Set("Content-Type", "application/json")
	proxyReq.Header.Set("Authorization", "Bearer "+target.Key)
	if req.Provider == "anthropic" {
		proxyReq.Header.Set("x-api-key", target.Key)
		proxyReq.Header.Set("anthropic-version", "2023-06-01")
	}

	resp, err := p.client.Do(proxyReq)
	if err != nil {
		fmt.Fprintf(w, "data: {\"error\":\"%s\"}\n\n", err.Error())
		flusher.Flush()
		return
	}
	defer resp.Body.Close()

	start := time.Now()
	chunkNum := 0
	buf := make([]byte, 4096)
	for {
		n, err := resp.Body.Read(buf)
		if n > 0 {
			chunkNum++
			chunk := string(buf[:n])
			elapsed := int(time.Since(start).Milliseconds())
			frame := map[string]interface{}{
				"chunk_num":  chunkNum,
				"elapsed_ms": elapsed,
				"size_bytes": n,
				"raw":        chunk,
			}
			frameBytes, _ := json.Marshal(frame)
			fmt.Fprintf(w, "data: %s\n\n", string(frameBytes))
			flusher.Flush()
		}
		if err != nil {
			break
		}
	}
	fmt.Fprintf(w, "data: [DONE]\n\n")
	flusher.Flush()
}

func (p *Proxy) HandleGetProviders(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	providers, err := p.store.GetAvailableProviders()
	if err != nil {
		http.Error(w, `{"error":"failed to get providers"}`, http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(providers)
}
