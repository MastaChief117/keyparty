package middleware

import (
	"fmt"
	"net/http"
	"strings"
	"sync"
	"time"
)

type rateLimiter struct {
	mu       sync.Mutex
	visitors map[string]*visitor
	rate     int
	burst    int
}

type visitor struct {
	tokens   int
	lastSeen time.Time
}

var limiter = &rateLimiter{
	visitors: make(map[string]*visitor),
	rate:     100,
	burst:    150,
}

func (rl *rateLimiter) allow(key string) bool {
	rl.mu.Lock()
	defer rl.mu.Unlock()
	v, exists := rl.visitors[key]
	if !exists {
		v = &visitor{tokens: rl.burst, lastSeen: time.Now()}
		rl.visitors[key] = v
	}
	elapsed := time.Since(v.lastSeen)
	v.lastSeen = time.Now()
	v.tokens += int(elapsed.Seconds() * float64(rl.rate))
	if v.tokens > rl.burst {
		v.tokens = rl.burst
	}
	if v.tokens <= 0 {
		return false
	}
	v.tokens--
	return true
}

func init() {
	go func() {
		for {
			time.Sleep(5 * time.Minute)
			limiter.mu.Lock()
			for k, v := range limiter.visitors {
				if time.Since(v.lastSeen) > 10*time.Minute {
					delete(limiter.visitors, k)
				}
			}
			limiter.mu.Unlock()
		}
	}()
}

func RateLimit(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ip := r.RemoteAddr
		if fwd := r.Header.Get("X-Forwarded-For"); fwd != "" {
			ip = strings.Split(fwd, ",")[0]
		}
		limiter.mu.Lock()
		v, exists := limiter.visitors[ip]
		tokens := 0
		if exists {
			tokens = v.tokens
		} else {
			tokens = limiter.burst
		}
		limiter.mu.Unlock()
		w.Header().Set("X-RateLimit-Limit", fmt.Sprintf("%d", limiter.burst))
		w.Header().Set("X-RateLimit-Remaining", fmt.Sprintf("%d", tokens))
		if !limiter.allow(ip) {
			http.Error(w, `{"error":"Rate limit exceeded"}`, http.StatusTooManyRequests)
			return
		}
		next(w, r)
	}
}

func BodyLimit(maxBytes int64, next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		r.Body = http.MaxBytesReader(w, r.Body, maxBytes)
		next(w, r)
	}
}

func SecurityHeaders(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("X-Frame-Options", "DENY")
		w.Header().Set("X-XSS-Protection", "1; mode=block")
		w.Header().Set("Referrer-Policy", "strict-origin-when-cross-origin")
		next(w, r)
	}
}

func CORS(allowOrigin string, next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		origin := r.Header.Get("Origin")
		if allowOrigin == "" || allowOrigin == "*" {
			w.Header().Set("Access-Control-Allow-Origin", "*")
		} else if origin == allowOrigin {
			w.Header().Set("Access-Control-Allow-Origin", origin)
		}
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization, X-API-Key, X-Admin-Key")
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		next(w, r)
	}
}
