package httpapi_test

import (
	"bytes"
	"context"
	"encoding/json"
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

func newRouter(logs *bytes.Buffer) *gin.Engine {
	logger := slog.New(slog.NewJSONHandler(logs, nil))
	health := handler.NewHealth(okPinger{}, time.Second, logger)
	return httpapi.NewRouter(logger, []string{allowedOrigin}, health)
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
			router := newRouter(&bytes.Buffer{})
			req := httptest.NewRequestWithContext(t.Context(), http.MethodGet, tt.path, http.NoBody)

			rec := do(t, router, req)

			assert.Equal(t, tt.wantStatus, rec.Code)
			assert.NotEmpty(t, rec.Header().Get(middleware.RequestIDHeader), "every response carries a request id")
		})
	}
}

func TestRouterRecoversFromPanics(t *testing.T) {
	var logs bytes.Buffer
	router := newRouter(&logs)
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
			router := newRouter(&bytes.Buffer{})
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
