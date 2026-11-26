// Package httpapi wires the HTTP router, its middleware and the translation
// of errors to HTTP responses.
package httpapi

import (
	"log/slog"
	"net/http"
	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"

	"github.com/YaelBadillo/product-catalog/api/internal/httpapi/handler"
	"github.com/YaelBadillo/product-catalog/api/internal/httpapi/middleware"
)

// NewRouter builds the Gin engine with the base middleware and every route.
// The health probes live at the root, outside /api/v1.
func NewRouter(logger *slog.Logger, allowedOrigins []string, health *handler.Health) *gin.Engine {
	router := gin.New()
	router.Use(
		middleware.RequestID(),
		middleware.Logger(logger),
		recovery(logger),
		cors.New(cors.Config{
			AllowOrigins:  allowedOrigins,
			AllowMethods:  []string{http.MethodGet, http.MethodPost, http.MethodPatch, http.MethodDelete},
			AllowHeaders:  []string{"Content-Type", middleware.RequestIDHeader},
			ExposeHeaders: []string{middleware.RequestIDHeader, "Location"},
			MaxAge:        12 * time.Hour,
		}),
	)

	router.GET("/healthz", health.Live)
	router.GET("/readyz", health.Ready)

	return router
}
