package handler_test

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"sync/atomic"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"

	"github.com/YaelBadillo/product-catalog/api/internal/httpapi/handler"
	"github.com/YaelBadillo/product-catalog/api/internal/testutil"
)

func TestMain(m *testing.M) {
	gin.SetMode(gin.TestMode)
	os.Exit(m.Run())
}

// fakePinger runs ping and records how it was called.
type fakePinger struct {
	ping        func(ctx context.Context) error
	calls       atomic.Int32
	hadDeadline atomic.Bool
}

func (f *fakePinger) Ping(ctx context.Context) error {
	f.calls.Add(1)
	_, ok := ctx.Deadline()
	f.hadDeadline.Store(ok)
	return f.ping(ctx)
}

func newEngine(p handler.Pinger, timeout time.Duration) *gin.Engine {
	h := handler.NewHealth(p, timeout, slog.New(slog.NewTextHandler(io.Discard, nil)))
	engine := gin.New()
	engine.GET("/healthz", h.Live)
	engine.GET("/readyz", h.Ready)
	return engine
}

func serve(t *testing.T, engine *gin.Engine, path string) (*http.Request, *httptest.ResponseRecorder) {
	t.Helper()
	req := httptest.NewRequestWithContext(t.Context(), http.MethodGet, path, http.NoBody)
	rec := httptest.NewRecorder()
	engine.ServeHTTP(rec, req)
	return req, rec
}

func TestHealthLive(t *testing.T) {
	pinger := &fakePinger{ping: func(context.Context) error { return errors.New("db down") }}
	engine := newEngine(pinger, time.Second)

	req, rec := serve(t, engine, "/healthz")

	assert.Equal(t, http.StatusOK, rec.Code)
	assert.JSONEq(t, `{"status":"ok"}`, rec.Body.String())
	assert.Zero(t, pinger.calls.Load(), "liveness must not check the database")
	testutil.AssertMatchesContract(t, req, rec)
}

func TestHealthReady(t *testing.T) {
	tests := []struct {
		name       string
		ping       func(ctx context.Context) error
		wantStatus int
		wantBody   string
	}{
		{
			name:       "database available",
			ping:       func(context.Context) error { return nil },
			wantStatus: http.StatusOK,
			wantBody:   `{"status":"ok","checks":{"database":"ok"}}`,
		},
		{
			name:       "database ping fails",
			ping:       func(context.Context) error { return errors.New("connection refused") },
			wantStatus: http.StatusServiceUnavailable,
			wantBody:   `{"status":"unavailable","checks":{"database":"unavailable"}}`,
		},
		{
			name: "database ping exceeds the timeout",
			ping: func(ctx context.Context) error {
				<-ctx.Done()
				return ctx.Err()
			},
			wantStatus: http.StatusServiceUnavailable,
			wantBody:   `{"status":"unavailable","checks":{"database":"unavailable"}}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pinger := &fakePinger{ping: tt.ping}
			engine := newEngine(pinger, 20*time.Millisecond)

			req, rec := serve(t, engine, "/readyz")

			assert.Equal(t, tt.wantStatus, rec.Code)
			assert.JSONEq(t, tt.wantBody, rec.Body.String())
			assert.Equal(t, int32(1), pinger.calls.Load())
			assert.True(t, pinger.hadDeadline.Load(), "the ping must run with a deadline")
			testutil.AssertMatchesContract(t, req, rec)
		})
	}
}
