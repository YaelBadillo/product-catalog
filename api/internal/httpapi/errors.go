package httpapi

import (
	"errors"
	"fmt"
	"io"
	"log/slog"
	"math"
	"net/http"
	"runtime/debug"
	"strconv"

	"github.com/gin-gonic/gin"

	"github.com/YaelBadillo/product-catalog/api/internal/domain"
	"github.com/YaelBadillo/product-catalog/api/internal/httpapi/handler"
	"github.com/YaelBadillo/product-catalog/api/internal/httpapi/middleware"
)

// Error codes of the single error shape.
const (
	codeValidationError  = "VALIDATION_ERROR"
	codeNotFound         = "NOT_FOUND"
	codeSKUAlreadyExists = "SKU_ALREADY_EXISTS"
	codeVersionConflict  = "VERSION_CONFLICT"
	codePayloadTooLarge  = "PAYLOAD_TOO_LARGE"
	codeRateLimited      = "RATE_LIMITED"
	codeInternalError    = "INTERNAL_ERROR"
)

// errorResponse is the single error shape of the API:
// { "error": { "code", "message", "details": [{ "field", "message" }] } }.
type errorResponse struct {
	Error errorBody `json:"error"`
}

type errorBody struct {
	Code    string        `json:"code"`
	Message string        `json:"message"`
	Details []errorDetail `json:"details"`
}

type errorDetail struct {
	Field   string `json:"field"`
	Message string `json:"message"`
}

func abortWithError(c *gin.Context, status int, code, message string, details ...errorDetail) {
	if details == nil {
		details = []errorDetail{}
	}
	c.AbortWithStatusJSON(status, errorResponse{Error: errorBody{
		Code:    code,
		Message: message,
		Details: details,
	}})
}

// ErrorHandler is the single translation of errors to HTTP. After the chain
// runs, it answers the last error reported with c.Error, unless a response
// was already written. Unexpected errors become a generic 500 with the
// request ID; their cause goes only to the logs.
func ErrorHandler(logger *slog.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Next()
		last := c.Errors.Last()
		if last == nil || c.Writer.Written() {
			return
		}
		writeError(c, logger, last.Err)
	}
}

func writeError(c *gin.Context, logger *slog.Logger, err error) {
	var (
		validation *domain.ValidationError
		request    *handler.RequestError
		maxBytes   *http.MaxBytesError
		limited    *middleware.RateLimitedError
	)
	switch {
	case errors.As(err, &validation):
		details := make([]errorDetail, 0, len(validation.Fields))
		for _, f := range validation.Fields {
			details = append(details, errorDetail(f))
		}
		abortWithError(c, http.StatusUnprocessableEntity, codeValidationError, "the request breaks a business rule", details...)
	case errors.As(err, &request):
		abortWithError(c, http.StatusBadRequest, codeValidationError, "invalid request",
			errorDetail{Field: request.Field, Message: request.Message})
	case errors.As(err, &maxBytes):
		abortWithError(c, http.StatusRequestEntityTooLarge, codePayloadTooLarge,
			fmt.Sprintf("request body exceeds %d bytes", maxBytes.Limit))
	case errors.As(err, &limited):
		logger.InfoContext(c.Request.Context(), "rate limited", "ip", c.ClientIP(),
			"request_id", middleware.RequestIDFrom(c.Request.Context()))
		c.Header("Retry-After", strconv.Itoa(retryAfterSeconds(limited)))
		abortWithError(c, http.StatusTooManyRequests, codeRateLimited, "too many requests, retry later")
	case errors.Is(err, domain.ErrNotFound):
		abortWithError(c, http.StatusNotFound, codeNotFound, "product not found")
	case errors.Is(err, domain.ErrSKUAlreadyExists):
		abortWithError(c, http.StatusConflict, codeSKUAlreadyExists, "another product already uses this sku",
			errorDetail{Field: "sku", Message: "already exists"})
	case errors.Is(err, domain.ErrVersionConflict):
		abortWithError(c, http.StatusConflict, codeVersionConflict, "the product changed since it was loaded")
	default:
		ctx := c.Request.Context()
		requestID := middleware.RequestIDFrom(ctx)
		logger.ErrorContext(ctx, "unexpected error", "error", err.Error(), "request_id", requestID)
		abortWithError(c, http.StatusInternalServerError, codeInternalError,
			"Internal server error. Request ID: "+requestID)
	}
}

func retryAfterSeconds(e *middleware.RateLimitedError) int {
	return max(1, int(math.Ceil(e.RetryAfter.Seconds())))
}

// recovery turns a panic into a generic 500 with the request ID; the panic
// value and stack trace go only to the logs.
func recovery(logger *slog.Logger) gin.HandlerFunc {
	return gin.CustomRecoveryWithWriter(io.Discard, func(c *gin.Context, recovered any) {
		ctx := c.Request.Context()
		requestID := middleware.RequestIDFrom(ctx)
		logger.ErrorContext(ctx, "panic recovered",
			"panic", fmt.Sprint(recovered),
			"stack", string(debug.Stack()),
			"request_id", requestID,
		)
		abortWithError(c, http.StatusInternalServerError, codeInternalError,
			"Internal server error. Request ID: "+requestID)
	})
}
