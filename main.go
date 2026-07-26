package main

import (
	_ "embed"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"os"
	"strings"

	"ai-gateway/auth"
	"ai-gateway/middleware"
	"ai-gateway/proxy"
	"ai-gateway/store"
)

var ssrfBlocklist = []string{
	"localhost", "127.", "10.", "192.168.", "172.16.", "172.17.", "172.18.",
	"172.19.", "172.20.", "172.21.", "172.22.", "172.23.", "172.24.", "172.25.",
	"172.26.", "172.27.", "172.28.", "172.29.", "172.30.", "172.31.",
	"169.254.", "0.", "metadata.google", "100.100.100.200",
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
	return host == ""
}

func main() {
	port := flag.Int("port", 8080, "Server port")
	dbPath := flag.String("db", "gateway.db", "SQLite database path")
	adminPass := flag.String("admin-pass", "", "Admin password (or set ADMIN_PASSWORD env)")
	corsOrigin := flag.String("cors-origin", "", "Allowed CORS origin (empty=allow all)")
	cacheTTL := flag.Int("cache-ttl", 60, "Cache TTL in minutes")
	flag.Parse()

	if *adminPass == "" {
		*adminPass = os.Getenv("ADMIN_PASSWORD")
	}

	s, err := store.New(*dbPath)
	if err != nil {
		log.Fatalf("Failed to initialize store: %v", err)
	}
	defer s.Close()

	p := proxy.New(s)
	p.SetCacheTTL(*cacheTTL)

	mux := http.NewServeMux()

	mux.HandleFunc("/v1/chat/completions", p.HandleChatCompletions)
	mux.HandleFunc("/v1/models", p.HandleModels)
	mux.HandleFunc("/health", p.HandleHealth)
	mux.HandleFunc("/admin/test-key", p.HandleTestKey)
	mux.HandleFunc("/admin/race", p.HandleRace)
	mux.HandleFunc("/admin/logs", p.HandleLogs)
	mux.HandleFunc("/admin/blocked", p.HandleBlocked)
	mux.HandleFunc("/admin/roast", p.HandleRoast)
	mux.HandleFunc("/admin/8ball", p.Handle8Ball)
	mux.HandleFunc("/admin/ship", p.HandleShip)
	mux.HandleFunc("/admin/leaderboard", p.HandleLeaderboard)
	mux.HandleFunc("/admin/savings", p.HandleSavings)
	mux.HandleFunc("/admin/fun-facts", p.HandleFunFacts)

	mux.HandleFunc("/admin/failover", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.Method {
		case "GET":
			config := s.GetFailoverConfig()
			json.NewEncoder(w).Encode(config)
		case "POST":
			var req struct {
				Key   string `json:"key"`
				Value string `json:"value"`
			}
			if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
				http.Error(w, `{"error":"Invalid JSON"}`, 400)
				return
			}
			if err := s.SetFailoverConfig(req.Key, req.Value); err != nil {
				http.Error(w, `{"error":"Failed to save"}`, 500)
				return
			}
			json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
		default:
			http.Error(w, "Method not allowed", 405)
		}
	})

	mux.HandleFunc("/admin/failover/logs", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "GET" {
			http.Error(w, "Method not allowed", 405)
			return
		}
		logs := s.GetFailoverLogs(50)
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(logs)
	})

	mux.HandleFunc("/admin/unified-key", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.Method {
		case "GET":
			key, err := s.GetUnifiedKey()
			if err != nil {
				http.Error(w, `{"error":"Internal error"}`, 500)
				return
			}
			json.NewEncoder(w).Encode(map[string]string{"unified_api_key": key})
		case "POST":
			key, err := s.RegenerateUnifiedKey()
			if err != nil {
				http.Error(w, `{"error":"Internal error"}`, 500)
				return
			}
			json.NewEncoder(w).Encode(map[string]string{"unified_api_key": key})
		default:
			http.Error(w, "Method not allowed", 405)
		}
	})

	mux.HandleFunc("/admin/keys", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.Method {
		case "GET":
			keys, err := s.GetKeysMasked()
			if err != nil {
				http.Error(w, `{"error":"Internal error"}`, 500)
				return
			}
			json.NewEncoder(w).Encode(keys)
		case "POST":
			var req struct {
				Name      string `json:"name"`
				Provider  string `json:"provider"`
				Key       string `json:"key"`
				Model     string `json:"model"`
				CustomURL string `json:"custom_url"`
				Priority  int    `json:"priority"`
			}
			if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
				http.Error(w, "Invalid JSON", 400)
				return
			}
			if req.Provider == "" || req.Key == "" {
				http.Error(w, "provider and key required", 400)
				return
			}
			if !store.ValidateProvider(req.Provider) {
				http.Error(w, "Invalid provider", 400)
				return
			}
			if req.CustomURL != "" && isBlockedURL(req.CustomURL) {
				http.Error(w, "Invalid custom URL", 400)
				return
			}
			if req.Name == "" {
				req.Name = req.Provider + " Key"
			}
			id, err := s.AddKey(req.Name, req.Provider, req.Key, req.Model, req.CustomURL, req.Priority)
			if err != nil {
				http.Error(w, `{"error":"Internal error"}`, 500)
				return
			}
			json.NewEncoder(w).Encode(map[string]interface{}{"id": id, "status": "created"})
		default:
			http.Error(w, "Method not allowed", 405)
		}
	})

	mux.HandleFunc("/admin/keys/", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		path := r.URL.Path
		var id int64
		var action string
		n, _ := fmt.Sscanf(path, "/admin/keys/%d/%s", &id, &action)
		if n == 1 {
			action = ""
		}

		switch {
		case r.Method == "DELETE":
			if err := s.DeleteKey(id); err != nil {
				http.Error(w, `{"error":"Internal error"}`, 500)
				return
			}
			json.NewEncoder(w).Encode(map[string]string{"status": "deleted"})

		case r.Method == "POST" && action == "toggle":
			var req struct {
				Enabled bool `json:"enabled"`
			}
			json.NewDecoder(r.Body).Decode(&req)
			if err := s.ToggleKey(id, req.Enabled); err != nil {
				http.Error(w, `{"error":"Internal error"}`, 500)
				return
			}
			json.NewEncoder(w).Encode(map[string]string{"status": "toggled"})

		case r.Method == "PUT":
			var req struct {
				Name      string `json:"name"`
				Provider  string `json:"provider"`
				Key       string `json:"key"`
				Model     string `json:"model"`
				CustomURL string `json:"custom_url"`
				Priority  int    `json:"priority"`
			}
			if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
				http.Error(w, "Invalid JSON", 400)
				return
			}
			if !store.ValidateProvider(req.Provider) {
				http.Error(w, "Invalid provider", 400)
				return
			}
			if req.CustomURL != "" && isBlockedURL(req.CustomURL) {
				http.Error(w, "Invalid custom URL", 400)
				return
			}
			if err := s.UpdateKey(id, req.Name, req.Provider, req.Key, req.Model, req.CustomURL, req.Priority); err != nil {
				http.Error(w, `{"error":"Internal error"}`, 500)
				return
			}
			json.NewEncoder(w).Encode(map[string]string{"status": "updated"})

		default:
			http.Error(w, "Method not allowed", 405)
		}
	})

	mux.HandleFunc("/admin/stats", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		stats, err := s.GetStats()
		if err != nil {
			http.Error(w, `{"error":"Internal error"}`, 500)
			return
		}
		json.NewEncoder(w).Encode(stats)
	})

	mux.HandleFunc("/admin/export", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		keys, err := s.GetKeysMasked()
		if err != nil {
			http.Error(w, `{"error":"Internal error"}`, 500)
			return
		}
		json.NewEncoder(w).Encode(keys)
	})

	mux.HandleFunc("/admin/import", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		var keys []struct {
			Name      string `json:"name"`
			Provider  string `json:"provider"`
			Key       string `json:"key"`
			Model     string `json:"model"`
			CustomURL string `json:"custom_url"`
			Priority  int    `json:"priority"`
		}
		if err := json.NewDecoder(r.Body).Decode(&keys); err != nil {
			http.Error(w, "Invalid JSON", 400)
			return
		}
		count := 0
		for _, k := range keys {
			if k.Provider == "" || k.Key == "" {
				continue
			}
			if !store.ValidateProvider(k.Provider) {
				continue
			}
			_, err := s.AddKey(k.Name, k.Provider, k.Key, k.Model, k.CustomURL, k.Priority)
			if err != nil {
				continue
			}
			count++
		}
		json.NewEncoder(w).Encode(map[string]interface{}{"imported": count})
	})

	mux.HandleFunc("/admin/reset", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			http.Error(w, "Method not allowed", 405)
			return
		}
		if err := s.ResetStats(); err != nil {
			http.Error(w, `{"error":"Internal error"}`, 500)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{"status": "reset"})
	})

	mux.HandleFunc("/admin/virtual-keys", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.Method {
		case "GET":
			keys, err := s.GetVirtualKeysMasked()
			if err != nil {
				http.Error(w, `{"error":"Internal error"}`, 500)
				return
			}
			json.NewEncoder(w).Encode(keys)
		case "POST":
			var req struct {
				Name          string  `json:"name"`
				OwnerID       string  `json:"owner_id"`
				MonthlyBudget float64 `json:"monthly_budget"`
				AllowedModels string  `json:"allowed_models"`
				RateLimit     int     `json:"rate_limit"`
			}
			if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
				http.Error(w, "Invalid JSON", 400)
				return
			}
			if req.Name == "" {
				http.Error(w, "name required", 400)
				return
			}
			if req.AllowedModels == "" {
				req.AllowedModels = "*"
			}
			key, err := s.AddVirtualKey(req.Name, req.OwnerID, req.MonthlyBudget, req.AllowedModels, req.RateLimit)
			if err != nil {
				http.Error(w, `{"error":"Internal error"}`, 500)
				return
			}
			json.NewEncoder(w).Encode(map[string]interface{}{"key": key, "status": "created"})
		default:
			http.Error(w, "Method not allowed", 405)
		}
	})

	mux.HandleFunc("/admin/virtual-keys/", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		path := r.URL.Path
		var id int64
		var action string
		n, _ := fmt.Sscanf(path, "/admin/virtual-keys/%d/%s", &id, &action)
		if n == 1 {
			action = ""
		}

		switch {
		case r.Method == "DELETE":
			if err := s.DeleteVirtualKey(id); err != nil {
				http.Error(w, `{"error":"Internal error"}`, 500)
				return
			}
			json.NewEncoder(w).Encode(map[string]string{"status": "deleted"})

		case r.Method == "POST" && action == "toggle":
			var req struct {
				Enabled bool `json:"enabled"`
			}
			json.NewDecoder(r.Body).Decode(&req)
			if err := s.ToggleVirtualKey(id, req.Enabled); err != nil {
				http.Error(w, `{"error":"Internal error"}`, 500)
				return
			}
			json.NewEncoder(w).Encode(map[string]string{"status": "toggled"})

		default:
			http.Error(w, "Method not allowed", 405)
		}
	})

	mux.HandleFunc("/admin/aliases", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.Method {
		case "GET":
			aliases, err := s.GetAliases()
			if err != nil {
				http.Error(w, `{"error":"Internal error"}`, 500)
				return
			}
			json.NewEncoder(w).Encode(aliases)
		case "POST":
			var req struct {
				Name         string `json:"name"`
				TargetModel  string `json:"target_model"`
			}
			if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
				http.Error(w, "Invalid JSON", 400)
				return
			}
			if err := s.AddAlias(req.Name, req.TargetModel); err != nil {
				http.Error(w, `{"error":"Internal error"}`, 500)
				return
			}
			json.NewEncoder(w).Encode(map[string]string{"status": "created"})
		case "DELETE":
			var req struct {
				Name string `json:"name"`
			}
			json.NewDecoder(r.Body).Decode(&req)
			if err := s.DeleteAlias(req.Name); err != nil {
				http.Error(w, `{"error":"Internal error"}`, 500)
				return
			}
			json.NewEncoder(w).Encode(map[string]string{"status": "deleted"})
		default:
			http.Error(w, "Method not allowed", 405)
		}
	})

	mux.HandleFunc("/admin/guardrails", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.Method {
		case "GET":
			guardrails, err := s.GetGuardrails()
			if err != nil {
				http.Error(w, `{"error":"Internal error"}`, 500)
				return
			}
			json.NewEncoder(w).Encode(guardrails)
		case "POST":
			var req struct {
				Name    string `json:"name"`
				Pattern string `json:"pattern"`
				Action  string `json:"action"`
			}
			if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
				http.Error(w, "Invalid JSON", 400)
				return
			}
			if req.Action == "" {
				req.Action = "block"
			}
			if err := s.AddGuardrail(req.Name, req.Pattern, req.Action); err != nil {
				http.Error(w, `{"error":"Internal error"}`, 500)
				return
			}
			json.NewEncoder(w).Encode(map[string]string{"status": "created"})
		case "DELETE":
			var req struct {
				Name string `json:"name"`
			}
			json.NewDecoder(r.Body).Decode(&req)
			if err := s.DeleteGuardrail(req.Name); err != nil {
				http.Error(w, `{"error":"Internal error"}`, 500)
				return
			}
			json.NewEncoder(w).Encode(map[string]string{"status": "deleted"})
		default:
			http.Error(w, "Method not allowed", 405)
		}
	})

	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/" {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", "text/html")
		fmt.Fprint(w, indexHTML)
	})

	handler := middleware.CORS(*corsOrigin,
		middleware.SecurityHeaders(
			middleware.RateLimit(
				auth.WrapAdmin(*adminPass, mux),
			),
		),
	)

	limitedHandler := middleware.BodyLimit(1<<20, handler)

	addr := fmt.Sprintf(":%d", *port)
	scheme := "http"
	if strings.Contains(addr, "443") {
		scheme = "https"
	}
	fmt.Printf("AI Gateway running on %s://localhost%s\n", scheme, addr)
	fmt.Printf("Proxy endpoint: %s://localhost%s/v1/chat/completions\n", scheme, addr)
	fmt.Printf("Dashboard: %s://localhost%s\n", scheme, addr)
	if *adminPass != "" {
		fmt.Println("Admin auth: enabled")
	} else {
		fmt.Println("Admin auth: DISABLED (set -admin-pass or ADMIN_PASSWORD)")
	}

	if err := http.ListenAndServe(addr, limitedHandler); err != nil {
		log.Fatalf("Server failed: %v", err)
	}
}

//go:embed web/index.html
var indexHTML string
