package middleware

import "net/http"

// CSRFProtection uses Go 1.25's standard library http.CrossOriginProtection
// to prevent CSRF/CORF attacks. Works by validating Sec-Fetch-Site and Origin headers.
//
// Safe methods (GET, HEAD, OPTIONS) are always allowed.
// Unsafe methods (POST, PUT, DELETE) require same-origin validation.
func CSRFProtection(next http.Handler) http.Handler {
	cop := http.NewCrossOriginProtection()

	// Custom error response
	cop.SetDenyHandler(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "Cross-origin request blocked", http.StatusForbidden)
	}))

	return cop.Handler(next)
}

// CSRFProtectionFunc wraps the middleware for http.HandlerFunc
func CSRFProtectionFunc(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		CSRFProtection(next).ServeHTTP(w, r)
	}
}
