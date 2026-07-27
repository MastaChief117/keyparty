package auth

import (
	"crypto/subtle"
	"log"
	"net/http"
	"strings"
)

func AdminAuth(adminPassword string, next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if adminPassword == "" {
			log.Printf("AUDIT: Admin access attempt with no password configured - %s %s from %s", r.Method, r.URL.Path, r.RemoteAddr)
			http.Error(w, `{"error":"Admin password not configured. Server operator must set -admin-pass or ADMIN_PASSWORD."}`, http.StatusServiceUnavailable)
			return
		}
		authHeader := r.Header.Get("Authorization")
		token := strings.TrimPrefix(authHeader, "Bearer ")
		if token == "" {
			token = r.Header.Get("X-Admin-Key")
		}
		if subtle.ConstantTimeCompare([]byte(token), []byte(adminPassword)) != 1 {
			log.Printf("AUDIT: Failed admin auth attempt - %s %s from %s", r.Method, r.URL.Path, r.RemoteAddr)
			http.Error(w, `{"error":"Unauthorized"}`, http.StatusUnauthorized)
			return
		}
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
