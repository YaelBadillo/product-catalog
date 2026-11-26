package testutil_test

import (
	"net/http"
	"net/http/httptest"
	"runtime"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/YaelBadillo/product-catalog/api/internal/testutil"
)

// fakeT records failures instead of failing the real test.
type fakeT struct{ failed bool }

func (f *fakeT) Errorf(string, ...any) { f.failed = true }
func (f *fakeT) Helper()               {}
func (f *fakeT) FailNow() {
	f.failed = true
	runtime.Goexit()
}

// runFake runs fn with a fakeT in its own goroutine, so FailNow can stop it.
func runFake(fn func(t *fakeT)) *fakeT {
	f := &fakeT{}
	done := make(chan struct{})
	go func() {
		defer close(done)
		fn(f)
	}()
	<-done
	return f
}

func TestAssertMatchesContract(t *testing.T) {
	tests := []struct {
		name       string
		path       string
		status     int
		body       string
		wantFailed bool
	}{
		{name: "documented response", path: "/healthz", status: http.StatusOK, body: `{"status":"ok"}`},
		{name: "body missing a required field", path: "/healthz", status: http.StatusOK, body: `{}`, wantFailed: true},
		{name: "body with an unknown field", path: "/healthz", status: http.StatusOK, body: `{"status":"ok","extra":1}`, wantFailed: true},
		{name: "undocumented status", path: "/healthz", status: http.StatusInternalServerError, body: `{"status":"ok"}`, wantFailed: true},
		{name: "undocumented path", path: "/nope", status: http.StatusOK, body: `{}`, wantFailed: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequestWithContext(t.Context(), http.MethodGet, tt.path, http.NoBody)
			rec := httptest.NewRecorder()
			rec.Header().Set("Content-Type", "application/json; charset=utf-8")
			rec.WriteHeader(tt.status)
			_, _ = rec.WriteString(tt.body)

			fake := runFake(func(ft *fakeT) { testutil.AssertMatchesContract(ft, req, rec) })

			assert.Equal(t, tt.wantFailed, fake.failed)
		})
	}
}

func TestAssertResponseMatchesContract(t *testing.T) {
	const product = `{
		"id": "0193a5b0-7c1e-7000-8000-000000000001",
		"sku": "ABC-123",
		"name": "Café molido",
		"description": null,
		"price_cents": 19990,
		"currency": "MXN",
		"stock": 5,
		"category": "food",
		"is_active": true,
		"version": 1,
		"created_at": "2026-11-25T14:30:00Z",
		"updated_at": "2026-11-25T14:30:00Z"
	}`
	const badRequest = `{"error":{"code":"VALIDATION_ERROR","message":"invalid id","details":[{"field":"id","message":"must be a UUID"}]}}`

	tests := []struct {
		name       string
		path       string
		status     int
		body       string
		wantFailed bool
	}{
		{
			name:   "request outside the contract with a documented error",
			path:   "/api/v1/products/abc",
			status: http.StatusBadRequest,
			body:   badRequest,
		},
		{
			name:   "nullable description",
			path:   "/api/v1/products/0193a5b0-7c1e-7000-8000-000000000001",
			status: http.StatusOK,
			body:   product,
		},
		{
			name:       "undocumented status",
			path:       "/api/v1/products/abc",
			status:     http.StatusTeapot,
			body:       badRequest,
			wantFailed: true,
		},
		{
			name:       "invalid body",
			path:       "/api/v1/products/abc",
			status:     http.StatusBadRequest,
			body:       `{"error":{"code":"NOPE","message":"x","details":[]}}`,
			wantFailed: true,
		},
		{name: "undocumented path", path: "/nope", status: http.StatusOK, body: `{}`, wantFailed: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequestWithContext(t.Context(), http.MethodGet, tt.path, http.NoBody)
			rec := httptest.NewRecorder()
			rec.Header().Set("Content-Type", "application/json; charset=utf-8")
			rec.WriteHeader(tt.status)
			_, _ = rec.WriteString(tt.body)

			fake := runFake(func(ft *fakeT) { testutil.AssertResponseMatchesContract(ft, req, rec) })

			assert.Equal(t, tt.wantFailed, fake.failed)
		})
	}
}
