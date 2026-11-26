package httpapi_test

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/YaelBadillo/product-catalog/api/internal/httpapi"
	"github.com/YaelBadillo/product-catalog/api/internal/httpapi/handler"
	"github.com/YaelBadillo/product-catalog/api/internal/httpapi/middleware"
)

const allowedOrigin = "http://localhost:3000"

func TestMain(m *testing.M) {
	gin.SetMode(gin.TestMode)
	os.Exit(m.Run())
}

type okPinger struct{}

func (okPinger) Ping(context.Context) error { return nil }

// frozenNow keeps the rate limiter from refilling during a test.
func frozenNow() time.Time { return time.Date(2026, 11, 25, 12, 0, 0, 0, time.UTC) }

func newRouterWith(t *testing.T, logs *bytes.Buffer, clientIPHeader string, api ...func(*gin.RouterGroup)) *gin.Engine {
	t.Helper()
	logger := slog.New(slog.NewJSONHandler(logs, nil))
	health := handler.NewHealth(okPinger{}, time.Second, logger)
	cfg := httpapi.RouterConfig{
		AllowedOrigins: []string{allowedOrigin},
		ClientIPHeader: clientIPHeader,
		RateLimiter:    middleware.NewRateLimiter(middleware.RateLimitOptions{Now: frozenNow}),
	}
	return httpapi.NewRouter(logger, cfg, health, api...)
}

func newRouter(t *testing.T, logs *bytes.Buffer) *gin.Engine {
	t.Helper()
	return newRouterWith(t, logs, "")
}

func do(t *testing.T, engine http.Handler, req *http.Request) *httptest.ResponseRecorder {
	t.Helper()
	rec := httptest.NewRecorder()
	engine.ServeHTTP(rec, req)
	return rec
}

func TestRouterRoutes(t *testing.T) {
	tests := []struct {
		path       string
		wantStatus int
	}{
		{path: "/healthz", wantStatus: http.StatusOK},
		{path: "/readyz", wantStatus: http.StatusOK},
		{path: "/api/v1/healthz", wantStatus: http.StatusNotFound},
		{path: "/api/v1/readyz", wantStatus: http.StatusNotFound},
	}

	for _, tt := range tests {
		t.Run(tt.path, func(t *testing.T) {
			router := newRouter(t, &bytes.Buffer{})
			req := httptest.NewRequestWithContext(t.Context(), http.MethodGet, tt.path, http.NoBody)

			rec := do(t, router, req)

			assert.Equal(t, tt.wantStatus, rec.Code)
			assert.NotEmpty(t, rec.Header().Get(middleware.RequestIDHeader), "every response carries a request id")
		})
	}
}

func TestRouterRecoversFromPanics(t *testing.T) {
	var logs bytes.Buffer
	router := newRouter(t, &logs)
	router.GET("/panic", func(*gin.Context) { panic("secret internal detail") })
	req := httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/panic", http.NoBody)

	rec := do(t, router, req)

	require.Equal(t, http.StatusInternalServerError, rec.Code)
	requestID := rec.Header().Get(middleware.RequestIDHeader)
	require.NotEmpty(t, requestID)

	var body struct {
		Error struct {
			Code    string `json:"code"`
			Message string `json:"message"`
			Details []any  `json:"details"`
		} `json:"error"`
	}
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &body))
	assert.Equal(t, "INTERNAL_ERROR", body.Error.Code)
	assert.Contains(t, body.Error.Message, requestID)
	assert.NotNil(t, body.Error.Details)
	assert.Empty(t, body.Error.Details)
	assert.NotContains(t, rec.Body.String(), "secret internal detail", "internals must not leak")
	assert.Contains(t, logs.String(), "secret internal detail", "the detail goes to the logs")
}

func TestRouterCORS(t *testing.T) {
	tests := []struct {
		name            string
		method          string
		origin          string
		wantAllowOrigin string
	}{
		{name: "allowed origin", method: http.MethodGet, origin: allowedOrigin, wantAllowOrigin: allowedOrigin},
		{name: "allowed origin preflight", method: http.MethodOptions, origin: allowedOrigin, wantAllowOrigin: allowedOrigin},
		{name: "disallowed origin", method: http.MethodGet, origin: "http://evil.example.com", wantAllowOrigin: ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			router := newRouter(t, &bytes.Buffer{})
			req := httptest.NewRequestWithContext(t.Context(), tt.method, "/healthz", http.NoBody)
			req.Header.Set("Origin", tt.origin)
			if tt.method == http.MethodOptions {
				req.Header.Set("Access-Control-Request-Method", http.MethodGet)
			}

			rec := do(t, router, req)

			assert.Equal(t, tt.wantAllowOrigin, rec.Header().Get("Access-Control-Allow-Origin"))
		})
	}
}

// pingRoute is a test route registered on /api/v1.
func pingRoute(g *gin.RouterGroup) {
	g.GET("/ping", func(c *gin.Context) { c.Status(http.StatusNoContent) })
}

func get(t *testing.T, router http.Handler, path string, headers map[string]string) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequestWithContext(t.Context(), http.MethodGet, path, http.NoBody)
	for k, v := range headers {
		req.Header.Set(k, v)
	}
	return do(t, router, req)
}

func TestRouterSecurityHeaders(t *testing.T) {
	router := newRouterWith(t, &bytes.Buffer{}, "", pingRoute)

	for _, path := range []string{"/healthz", "/api/v1/ping", "/nope"} {
		rec := get(t, router, path, nil)

		assert.Equal(t, "nosniff", rec.Header().Get("X-Content-Type-Options"), path)
		assert.Equal(t, "DENY", rec.Header().Get("X-Frame-Options"), path)
		assert.Equal(t, "default-src 'none'; frame-ancestors 'none'", rec.Header().Get("Content-Security-Policy"), path)
	}
}

func TestRouterHealthIsNeverRateLimited(t *testing.T) {
	router := newRouter(t, &bytes.Buffer{})

	for i := range 400 {
		require.Equal(t, http.StatusOK, get(t, router, "/healthz", nil).Code, "request %d", i+1)
	}
}

func TestRouterRateLimitsAPIRoutes(t *testing.T) {
	router := newRouterWith(t, &bytes.Buffer{}, "", pingRoute)
	for i := range 300 {
		require.Equal(t, http.StatusNoContent, get(t, router, "/api/v1/ping", nil).Code, "request %d", i+1)
	}

	rec := get(t, router, "/api/v1/ping", nil)

	assert.Equal(t, http.StatusTooManyRequests, rec.Code)
	assert.NotEmpty(t, rec.Header().Get("Retry-After"))
	assert.Contains(t, rec.Body.String(), "RATE_LIMITED")
}

func TestRouterIgnoresXForwardedFor(t *testing.T) {
	router := newRouterWith(t, &bytes.Buffer{}, "", pingRoute)
	for i := range 300 {
		get(t, router, "/api/v1/ping", map[string]string{"X-Forwarded-For": fmt.Sprintf("203.0.113.%d", i%250)})
	}

	rec := get(t, router, "/api/v1/ping", map[string]string{"X-Forwarded-For": "198.51.100.7"})

	assert.Equal(t, http.StatusTooManyRequests, rec.Code, "a spoofed X-Forwarded-For must not get a new bucket")
}

func TestRouterClientIPHeader(t *testing.T) {
	router := newRouterWith(t, &bytes.Buffer{}, "X-Real-IP", pingRoute)
	first := map[string]string{"X-Real-IP": "203.0.113.1"}
	for range 300 {
		get(t, router, "/api/v1/ping", first)
	}

	assert.Equal(t, http.StatusTooManyRequests, get(t, router, "/api/v1/ping", first).Code)
	assert.Equal(t, http.StatusNoContent,
		get(t, router, "/api/v1/ping", map[string]string{"X-Real-IP": "203.0.113.2"}).Code,
		"each client IP gets its own bucket")
}
