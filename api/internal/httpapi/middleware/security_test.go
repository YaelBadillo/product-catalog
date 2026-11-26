package middleware_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"

	"github.com/YaelBadillo/product-catalog/api/internal/httpapi/middleware"
)

func TestSecurityHeaders(t *testing.T) {
	for _, status := range []int{http.StatusOK, http.StatusNotFound, http.StatusInternalServerError} {
		engine := gin.New()
		engine.Use(middleware.SecurityHeaders())
		engine.GET("/", func(c *gin.Context) { c.Status(status) })
		rec := httptest.NewRecorder()

		engine.ServeHTTP(rec, httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/", http.NoBody))

		h := rec.Header()
		assert.Equal(t, "nosniff", h.Get("X-Content-Type-Options"))
		assert.Equal(t, "DENY", h.Get("X-Frame-Options"))
		assert.Equal(t, "no-referrer", h.Get("Referrer-Policy"))
		assert.Equal(t, "max-age=31536000", h.Get("Strict-Transport-Security"))
		assert.Equal(t, "default-src 'none'; frame-ancestors 'none'", h.Get("Content-Security-Policy"))
	}
}
