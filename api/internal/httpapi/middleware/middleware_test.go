package middleware_test

import (
	"bytes"
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"regexp"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/YaelBadillo/product-catalog/api/internal/httpapi/middleware"
)

func TestMain(m *testing.M) {
	gin.SetMode(gin.TestMode)
	os.Exit(m.Run())
}

var generatedID = regexp.MustCompile(`^[A-Za-z0-9-]{1,64}$`)

func TestRequestID(t *testing.T) {
	tests := []struct {
		name     string
		incoming string
		wantSame bool
	}{
		{name: "generates an id when none is sent", incoming: ""},
		{name: "reuses a valid incoming id", incoming: "abc-123", wantSame: true},
		{name: "replaces an id with unsafe characters", incoming: "bad id\ninjected"},
		{name: "replaces an id longer than 64 characters", incoming: strings.Repeat("a", 65)},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var seen string
			engine := gin.New()
			engine.Use(middleware.RequestID())
			engine.GET("/", func(c *gin.Context) {
				seen = middleware.RequestIDFrom(c.Request.Context())
				c.Status(http.StatusNoContent)
			})
			req := httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/", http.NoBody)
			if tt.incoming != "" {
				req.Header.Set(middleware.RequestIDHeader, tt.incoming)
			}
			rec := httptest.NewRecorder()

			engine.ServeHTTP(rec, req)

			got := rec.Header().Get(middleware.RequestIDHeader)
			assert.Equal(t, got, seen, "the handler must see the same id as the response header")
			if tt.wantSame {
				assert.Equal(t, tt.incoming, got)
				return
			}
			assert.NotEqual(t, tt.incoming, got)
			assert.Regexp(t, generatedID, got)
		})
	}
}

func TestRequestIDFromWithoutMiddleware(t *testing.T) {
	assert.Empty(t, middleware.RequestIDFrom(t.Context()))
}

func TestLogger(t *testing.T) {
	var buf bytes.Buffer
	logger := slog.New(slog.NewJSONHandler(&buf, nil))
	engine := gin.New()
	engine.Use(middleware.RequestID(), middleware.Logger(logger))
	engine.GET("/things", func(c *gin.Context) { c.Status(http.StatusCreated) })
	req := httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/things?token=secret", http.NoBody)
	req.Header.Set(middleware.RequestIDHeader, "abc-123")
	rec := httptest.NewRecorder()

	engine.ServeHTTP(rec, req)

	var entry map[string]any
	require.NoError(t, json.Unmarshal(buf.Bytes(), &entry), "expected exactly one JSON log line, got %q", buf.String())
	assert.Equal(t, "request", entry["msg"])
	assert.Equal(t, http.MethodGet, entry["method"])
	assert.Equal(t, "/things", entry["path"])
	assert.InDelta(t, http.StatusCreated, entry["status"], 0)
	assert.Equal(t, "abc-123", entry["request_id"])
	assert.Contains(t, entry, "latency")
	assert.NotContains(t, buf.String(), "secret", "query strings must not be logged")
}
