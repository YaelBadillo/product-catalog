// Package httpapi wires the HTTP router, its middleware and the translation
// of errors to HTTP responses.
package httpapi

import (
	"log/slog"
	"net/http"
	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"

	"github.com/YaelBadillo/product-catalog/api/internal/apidocs"
	"github.com/YaelBadillo/product-catalog/api/internal/httpapi/handler"
	"github.com/YaelBadillo/product-catalog/api/internal/httpapi/middleware"
)

// maxBodyBytes caps request bodies on /api/v1; the largest valid product is
// about 3 KiB.
const maxBodyBytes = 64 << 10

// RouterConfig is what the router needs from the configuration.
type RouterConfig struct {
	AllowedOrigins []string
	// ClientIPHeader, when set, is the header the platform's edge fills with
	// the client IP (X-Real-IP on Railway). Empty: the connection's address.
	ClientIPHeader string
	// RateLimiter guards /api/v1. The caller owns it and runs its janitor;
	// nil uses a default limiter without a janitor (tests).
	RateLimiter *middleware.RateLimiter
}

// NewRouter builds the Gin engine with the middleware and every route. The
// health probes and /docs live at the root; each api registrar adds its routes to the
// /api/v1 group, which is rate-limited and body-limited.
func NewRouter(logger *slog.Logger, cfg RouterConfig, health *handler.Health, api ...func(*gin.RouterGroup)) *gin.Engine {
	router := gin.New()
	// Never trust X-Forwarded-For: only the platform header, if configured.
	_ = router.SetTrustedProxies(nil) // nil never fails
	router.TrustedPlatform = cfg.ClientIPHeader

	router.Use(
		middleware.RequestID(),
		middleware.Logger(logger),
		recovery(logger),
		ErrorHandler(logger),
		middleware.SecurityHeaders(),
		cors.New(cors.Config{
			AllowOrigins:  cfg.AllowedOrigins,
			AllowMethods:  []string{http.MethodGet, http.MethodPost, http.MethodPatch, http.MethodDelete},
			AllowHeaders:  []string{"Content-Type", middleware.RequestIDHeader},
			ExposeHeaders: []string{middleware.RequestIDHeader, "Location", "Retry-After"},
			MaxAge:        12 * time.Hour,
		}),
	)

	router.GET("/healthz", health.Live)
	router.GET("/readyz", health.Ready)
	apidocs.Register(router)

	limiter := cfg.RateLimiter
	if limiter == nil {
		limiter = middleware.NewRateLimiter(middleware.RateLimitOptions{})
	}
	v1 := router.Group("/api/v1", middleware.BodyLimit(maxBodyBytes), limiter.Middleware())
	for _, register := range api {
		register(v1)
	}

	return router
}
