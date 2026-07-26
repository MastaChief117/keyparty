package auth

import (
	"crypto/subtle"
	"net/http"
	"strings"
)

func AdminAuth(adminPassword string, next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if adminPassword == "" {
			next(w, r)
			return
		}
		authHeader := r.Header.Get("Authorization")
		token := strings.TrimPrefix(authHeader, "Bearer ")
		if token == "" {
			token = r.Header.Get("X-Admin-Key")
		}
		if subtle.ConstantTimeCompare([]byte(token), []byte(adminPassword)) != 1 {
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
