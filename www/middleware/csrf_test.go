package middleware

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestCSRFProtection(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("success"))
	})

	protected := CSRFProtection(handler)

	t.Run("allows same-origin GET requests", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "https://example.com/api/data", nil)
		req.Header.Set("Sec-Fetch-Site", "same-origin")
		w := httptest.NewRecorder()

		protected.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		assert.Equal(t, "success", w.Body.String())
	})

	t.Run("allows same-origin POST requests", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "https://example.com/api/submit", strings.NewReader("data"))
		req.Header.Set("Sec-Fetch-Site", "same-origin")
		w := httptest.NewRecorder()

		protected.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
	})

	t.Run("blocks cross-origin POST requests", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "https://example.com/api/submit", strings.NewReader("data"))
		req.Header.Set("Sec-Fetch-Site", "cross-site")
		w := httptest.NewRecorder()

		protected.ServeHTTP(w, req)

		assert.Equal(t, http.StatusForbidden, w.Code)
		assert.Contains(t, w.Body.String(), "Cross-origin request blocked")
	})

	t.Run("allows cross-origin GET requests", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "https://example.com/api/data", nil)
		req.Header.Set("Sec-Fetch-Site", "cross-site")
		w := httptest.NewRecorder()

		protected.ServeHTTP(w, req)

		// GET is safe, should be allowed even cross-origin
		assert.Equal(t, http.StatusOK, w.Code)
	})

	t.Run("blocks cross-origin PUT requests", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPut, "https://example.com/api/update", strings.NewReader("data"))
		req.Header.Set("Sec-Fetch-Site", "cross-site")
		w := httptest.NewRecorder()

		protected.ServeHTTP(w, req)

		assert.Equal(t, http.StatusForbidden, w.Code)
	})

	t.Run("blocks cross-origin DELETE requests", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodDelete, "https://example.com/api/delete", nil)
		req.Header.Set("Sec-Fetch-Site", "cross-site")
		w := httptest.NewRecorder()

		protected.ServeHTTP(w, req)

		assert.Equal(t, http.StatusForbidden, w.Code)
	})

	t.Run("fallback to Origin header validation", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "https://example.com/api/submit", strings.NewReader("data"))
		req.Host = "example.com"
		req.Header.Set("Origin", "https://example.com")
		// No Sec-Fetch-Site - will use Origin/Host comparison
		w := httptest.NewRecorder()

		protected.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
	})

	t.Run("blocks when Origin mismatches Host", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "https://example.com/api/submit", strings.NewReader("data"))
		req.Host = "example.com"
		req.Header.Set("Origin", "https://evil.com")
		w := httptest.NewRecorder()

		protected.ServeHTTP(w, req)

		assert.Equal(t, http.StatusForbidden, w.Code)
	})
}

func TestCSRFProtectionFunc(t *testing.T) {
	handler := func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("success"))
	}

	protected := CSRFProtectionFunc(handler)

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Sec-Fetch-Site", "same-origin")
	w := httptest.NewRecorder()

	protected(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}
