package middleware_test

import (
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"

	"github.com/YaelBadillo/product-catalog/api/internal/httpapi/middleware"
)

func TestBodyLimit(t *testing.T) {
	const limit = 64 << 10
	tests := []struct {
		name    string
		size    int
		wantErr bool
	}{
		{name: "at the limit", size: limit},
		{name: "one byte over", size: limit + 1, wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var readErr error
			engine := gin.New()
			engine.Use(middleware.BodyLimit(limit))
			engine.POST("/", func(c *gin.Context) {
				_, readErr = io.ReadAll(c.Request.Body)
				c.Status(http.StatusNoContent)
			})
			req := httptest.NewRequestWithContext(t.Context(), http.MethodPost, "/", strings.NewReader(strings.Repeat("a", tt.size)))

			engine.ServeHTTP(httptest.NewRecorder(), req)

			var maxBytes *http.MaxBytesError
			assert.Equal(t, tt.wantErr, errors.As(readErr, &maxBytes), "read error: %v", readErr)
		})
	}
}
