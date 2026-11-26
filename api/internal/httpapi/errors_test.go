package httpapi_test

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/YaelBadillo/product-catalog/api/internal/domain"
	"github.com/YaelBadillo/product-catalog/api/internal/httpapi"
	"github.com/YaelBadillo/product-catalog/api/internal/httpapi/handler"
	"github.com/YaelBadillo/product-catalog/api/internal/httpapi/middleware"
)

type envelope struct {
	Error struct {
		Code    string `json:"code"`
		Message string `json:"message"`
		Details []struct {
			Field   string `json:"field"`
			Message string `json:"message"`
		} `json:"details"`
	} `json:"error"`
}

func decodeEnvelope(t *testing.T, rec *httptest.ResponseRecorder) envelope {
	t.Helper()
	var body envelope
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &body), rec.Body.String())
	require.NotNil(t, body.Error.Details, "details is always an array")
	assert.NotEmpty(t, body.Error.Message)
	return body
}

func TestErrorHandler(t *testing.T) {
	tests := []struct {
		name       string
		err        error
		wantStatus int
		wantCode   string
		wantFields []string
	}{
		{name: "not found", err: fmt.Errorf("get: %w", domain.ErrNotFound), wantStatus: http.StatusNotFound, wantCode: "NOT_FOUND"},
		{name: "sku already exists", err: domain.ErrSKUAlreadyExists, wantStatus: http.StatusConflict, wantCode: "SKU_ALREADY_EXISTS", wantFields: []string{"sku"}},
		{name: "version conflict", err: domain.ErrVersionConflict, wantStatus: http.StatusConflict, wantCode: "VERSION_CONFLICT"},
		{
			name: "validation error",
			err: &domain.ValidationError{Fields: []domain.FieldError{
				{Field: "name", Message: "must be 2 to 120 characters"},
				{Field: "price_cents", Message: "must be between 1 and 99999999999 cents"},
			}},
			wantStatus: http.StatusUnprocessableEntity,
			wantCode:   "VALIDATION_ERROR",
			wantFields: []string{"name", "price_cents"},
		},
		{
			name:       "request error",
			err:        &handler.RequestError{Field: "color", Message: "unknown field"},
			wantStatus: http.StatusBadRequest,
			wantCode:   "VALIDATION_ERROR",
			wantFields: []string{"color"},
		},
		{name: "body too large", err: fmt.Errorf("decode: %w", &http.MaxBytesError{Limit: 64 << 10}), wantStatus: http.StatusRequestEntityTooLarge, wantCode: "PAYLOAD_TOO_LARGE"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rec, _ := serveError(t, tt.err)

			assert.Equal(t, tt.wantStatus, rec.Code)
			body := decodeEnvelope(t, rec)
			assert.Equal(t, tt.wantCode, body.Error.Code)
			fields := []string{}
			for _, d := range body.Error.Details {
				fields = append(fields, d.Field)
				assert.NotEmpty(t, d.Message)
			}
			if tt.wantFields == nil {
				tt.wantFields = []string{}
			}
			assert.Equal(t, tt.wantFields, fields)
		})
	}
}

func TestErrorHandlerRateLimited(t *testing.T) {
	rec, _ := serveError(t, &middleware.RateLimitedError{RetryAfter: 1500 * time.Millisecond})

	assert.Equal(t, http.StatusTooManyRequests, rec.Code)
	assert.Equal(t, "2", rec.Header().Get("Retry-After"), "seconds, rounded up")
	assert.Equal(t, "RATE_LIMITED", decodeEnvelope(t, rec).Error.Code)
}

func TestErrorHandlerUnexpectedError(t *testing.T) {
	rec, logs := serveError(t, errors.New("pq: secret connection detail"))

	assert.Equal(t, http.StatusInternalServerError, rec.Code)
	body := decodeEnvelope(t, rec)
	requestID := rec.Header().Get(middleware.RequestIDHeader)
	require.NotEmpty(t, requestID)
	assert.Equal(t, "INTERNAL_ERROR", body.Error.Code)
	assert.Contains(t, body.Error.Message, requestID)
	assert.Empty(t, body.Error.Details)
	assert.NotContains(t, rec.Body.String(), "secret")
	assert.Contains(t, logs.String(), "secret connection detail", "the cause goes to the logs")
	assert.Contains(t, logs.String(), requestID, "with the request id")
}

func TestErrorHandlerLeavesSuccessAlone(t *testing.T) {
	engine := gin.New()
	engine.Use(middleware.RequestID(), httpapi.ErrorHandler(slog.New(slog.DiscardHandler)))
	engine.GET("/ok", func(c *gin.Context) { c.Status(http.StatusNoContent) })
	rec := httptest.NewRecorder()

	engine.ServeHTTP(rec, httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/ok", http.NoBody))

	assert.Equal(t, http.StatusNoContent, rec.Code)
	assert.Empty(t, rec.Body.String())
}

// serveError runs a handler that reports err with c.Error.
func serveError(t *testing.T, err error) (*httptest.ResponseRecorder, *bytes.Buffer) {
	t.Helper()
	var logs bytes.Buffer
	engine := gin.New()
	engine.Use(middleware.RequestID(), httpapi.ErrorHandler(slog.New(slog.NewJSONHandler(&logs, nil))))
	engine.GET("/fail", func(c *gin.Context) { _ = c.Error(err) })
	rec := httptest.NewRecorder()
	engine.ServeHTTP(rec, httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/fail", http.NoBody))
	return rec, &logs
}
