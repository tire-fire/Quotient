package middleware

import (
	"net/http"
	"strings"
)

// AllowedOrigins defines permitted origins for CORS
// In production, this should be configured via environment or config file
var AllowedOrigins = []string{
	"http://localhost",
	"https://localhost",
	// Add your production domain here
}

func Cors(next http.HandlerFunc) http.HandlerFunc {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		origin := r.Header.Get("Origin")

		// Check if origin is allowed
		allowed := false
		for _, allowedOrigin := range AllowedOrigins {
			// Allow exact match or same origin with different port (for dev)
			if origin == allowedOrigin || strings.HasPrefix(origin, allowedOrigin+":") {
				allowed = true
				break
			}
		}

		// Only set CORS headers if origin is allowed
		if allowed {
			w.Header().Set("Access-Control-Allow-Origin", origin)
			w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE")
			w.Header().Set("Access-Control-Allow-Headers", "Content-Type, X-CSRF-Token")
			w.Header().Set("Access-Control-Allow-Credentials", "true")
		}

		// If preflight request, respond with 200 OK
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusOK)
			return
		}

		// Continue processing the request
		next.ServeHTTP(w, r)
	})
}
