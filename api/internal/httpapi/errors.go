package httpapi

import (
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"runtime/debug"

	"github.com/gin-gonic/gin"

	"github.com/YaelBadillo/product-catalog/api/internal/httpapi/middleware"
)

// Error codes of the single error shape.
const codeInternalError = "INTERNAL_ERROR"

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

func abortWithError(c *gin.Context, status int, code, message string) {
	c.AbortWithStatusJSON(status, errorResponse{Error: errorBody{
		Code:    code,
		Message: message,
		Details: []errorDetail{},
	}})
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
