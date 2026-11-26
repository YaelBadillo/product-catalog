// Package handler contains the HTTP handlers and their DTOs.
package handler

import (
	"context"
	"log/slog"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
)

const (
	checkOK          = "ok"
	checkUnavailable = "unavailable"
)

// Pinger reports whether a dependency is reachable.
type Pinger interface {
	Ping(ctx context.Context) error
}

// HealthResponse is the body of GET /healthz.
type HealthResponse struct {
	Status string `json:"status"`
}

// ReadinessResponse is the body of GET /readyz.
type ReadinessResponse struct {
	Status string          `json:"status"`
	Checks ReadinessChecks `json:"checks"`
}

// ReadinessChecks holds the result of each dependency check.
type ReadinessChecks struct {
	Database string `json:"database"`
}

// Health serves the liveness and readiness probes.
type Health struct {
	db      Pinger
	timeout time.Duration
	logger  *slog.Logger
}

// NewHealth returns the health handlers. timeout bounds the database ping.
func NewHealth(db Pinger, timeout time.Duration, logger *slog.Logger) *Health {
	return &Health{db: db, timeout: timeout, logger: logger}
}

// Live handles GET /healthz: the process is alive. It checks no dependency.
func (h *Health) Live(c *gin.Context) {
	c.JSON(http.StatusOK, HealthResponse{Status: checkOK})
}

// Ready handles GET /readyz: it pings the database within the timeout.
func (h *Health) Ready(c *gin.Context) {
	ctx, cancel := context.WithTimeout(c.Request.Context(), h.timeout)
	defer cancel()

	if err := h.db.Ping(ctx); err != nil {
		h.logger.WarnContext(ctx, "readiness check failed", "check", "database", "error", err)
		c.JSON(http.StatusServiceUnavailable, ReadinessResponse{
			Status: checkUnavailable,
			Checks: ReadinessChecks{Database: checkUnavailable},
		})
		return
	}

	c.JSON(http.StatusOK, ReadinessResponse{
		Status: checkOK,
		Checks: ReadinessChecks{Database: checkOK},
	})
}
