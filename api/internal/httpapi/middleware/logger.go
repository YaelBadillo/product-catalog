package middleware

import (
	"log/slog"
	"time"

	"github.com/gin-gonic/gin"
)

// Logger logs one structured line per request. It logs the path without the
// query string, which may carry sensitive data.
func Logger(logger *slog.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		c.Next()
		ctx := c.Request.Context()
		logger.LogAttrs(ctx, slog.LevelInfo, "request",
			slog.String("method", c.Request.Method),
			slog.String("path", c.Request.URL.Path),
			slog.Int("status", c.Writer.Status()),
			slog.Duration("latency", time.Since(start)),
			slog.String("request_id", RequestIDFrom(ctx)),
		)
	}
}
