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

package auth

import (
	"crypto/subtle"
	"log"
	"net/http"
	"strings"
	"sync"
	"time"
)

type attemptTracker struct {
	mu       sync.Mutex
	attempts map[string]int64 // IP -> first failed attempt timestamp
	counts   map[string]int   // IP -> count of failures in window
}

var tracker = &attemptTracker{
	attempts: make(map[string]int64),
	counts:   make(map[string]int),
}

const (
	maxAttempts    = 5
	windowDuration = 1 * time.Minute
)

func (t *attemptTracker) recordFailure(key string) bool {
	t.mu.Lock()
	defer t.mu.Unlock()

	now := time.Now().Unix()
	first, exists := t.attempts[key]

	if !exists || now-first > int64(windowDuration.Seconds()) {
		t.attempts[key] = now
		t.counts[key] = 1
		return false
	}

	t.counts[key]++
	return t.counts[key] >= maxAttempts
}

func (t *attemptTracker) isLockedOut(key string) bool {
	t.mu.Lock()
	defer t.mu.Unlock()

	now := time.Now().Unix()
	first, exists := t.attempts[key]
	if !exists {
		return false
	}

	if now-first > int64(windowDuration.Seconds()) {
		delete(t.attempts, key)
		delete(t.counts, key)
		return false
	}

	return t.counts[key] >= maxAttempts
}

func (t *attemptTracker) clear(key string) {
	t.mu.Lock()
	defer t.mu.Unlock()
	delete(t.attempts, key)
	delete(t.counts, key)
}

func AdminAuth(adminPassword string, next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if adminPassword == "" {
			next(w, r)
			return
		}

		ip := r.RemoteAddr
		if idx := strings.LastIndex(ip, ":"); idx != -1 {
			ip = ip[:idx]
		}

		if tracker.isLockedOut(ip) {
			log.Printf("AUDIT: Rate limited admin auth attempt from %s", r.RemoteAddr)
			http.Error(w, `{"error":"Too many failed attempts. Try again later."}`, http.StatusTooManyRequests)
			return
		}

		authHeader := r.Header.Get("Authorization")
		token := strings.TrimPrefix(authHeader, "Bearer ")
		if token == "" {
			token = r.Header.Get("X-Admin-Key")
		}
		if subtle.ConstantTimeCompare([]byte(token), []byte(adminPassword)) != 1 {
			tracker.recordFailure(ip)
			log.Printf("AUDIT: Failed admin auth attempt - %s %s from %s", r.Method, r.URL.Path, r.RemoteAddr)
			http.Error(w, `{"error":"Unauthorized"}`, http.StatusUnauthorized)
			return
		}

		tracker.clear(ip)
		next(w, r)
	}
}

func WrapAdmin(adminPassword string, mux *http.ServeMux) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if strings.HasPrefix(r.URL.Path, "/admin/") {
			AdminAuth(adminPassword, mux.ServeHTTP)(w, r)
			return
		}
		mux.ServeHTTP(w, r)
	}
}
