package middleware

import (
	"fmt"
	"log"
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
	return rl.allowLocked(key)
}

func (rl *rateLimiter) allowLocked(key string) bool {
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
		if idx := strings.LastIndex(ip, ":"); idx != -1 {
			ip = ip[:idx]
		}
		limiter.mu.Lock()
		if len(limiter.visitors) > 10000 {
			// Evict oldest 10%
			toEvict := len(limiter.visitors) / 10
			for k := range limiter.visitors {
				if toEvict <= 0 {
					break
				}
				delete(limiter.visitors, k)
				toEvict--
			}
		}
		remaining := limiter.burst
		allowed := limiter.allowLocked(ip)
		if v, ok := limiter.visitors[ip]; ok {
			remaining = v.tokens
		}
		limiter.mu.Unlock()
		w.Header().Set("X-RateLimit-Limit", fmt.Sprintf("%d", limiter.burst))
		w.Header().Set("X-RateLimit-Remaining", fmt.Sprintf("%d", remaining))
		if !allowed {
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
		w.Header().Set("Referrer-Policy", "strict-origin-when-cross-origin")
		w.Header().Set("Content-Security-Policy", "default-src 'self'; script-src 'self' 'unsafe-inline'; style-src 'self' 'unsafe-inline'")
		next(w, r)
	}
}

func Recover(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if err := recover(); err != nil {
				log.Printf("panic recovered: %v", err)
				http.Error(w, `{"error":"Internal server error"}`, http.StatusInternalServerError)
			}
		}()
		next(w, r)
	}
}

func CORS(allowOrigin string, next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		origin := r.Header.Get("Origin")
		if allowOrigin != "" && allowOrigin != "*" {
			if origin == allowOrigin {
				w.Header().Set("Access-Control-Allow-Origin", origin)
				w.Header().Set("Vary", "Origin")
			}
		} else if allowOrigin == "*" {
			w.Header().Set("Access-Control-Allow-Origin", "*")
			w.Header().Set("Vary", "Origin")
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
