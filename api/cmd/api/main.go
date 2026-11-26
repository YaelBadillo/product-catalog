// Command api starts the Product Catalog HTTP API. Run as `api healthcheck`,
// it probes the local /healthz instead and exits 0 or 1; as `api seed`, it
// migrates and loads the sample products.
package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/YaelBadillo/product-catalog/api/internal/config"
	"github.com/YaelBadillo/product-catalog/api/internal/healthcheck"
	"github.com/YaelBadillo/product-catalog/api/internal/httpapi"
	"github.com/YaelBadillo/product-catalog/api/internal/httpapi/handler"
	"github.com/YaelBadillo/product-catalog/api/internal/httpapi/middleware"
	"github.com/YaelBadillo/product-catalog/api/internal/migrate"
	"github.com/YaelBadillo/product-catalog/api/internal/repository"
	"github.com/YaelBadillo/product-catalog/api/internal/seed"
	"github.com/YaelBadillo/product-catalog/api/internal/service"
)

const (
	readinessTimeout   = 2 * time.Second
	healthcheckTimeout = 3 * time.Second
	shutdownTimeout    = 10 * time.Second
)

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))

	if len(os.Args) > 1 && os.Args[1] == "healthcheck" {
		if err := runHealthcheck(); err != nil {
			logger.Error("healthcheck failed", "error", err)
			os.Exit(1)
		}
		return
	}

	if len(os.Args) > 1 && os.Args[1] == "seed" {
		if err := runSeed(logger); err != nil {
			logger.Error("seed failed", "error", err)
			os.Exit(1)
		}
		return
	}

	if err := run(logger); err != nil {
		logger.Error("server stopped", "error", err)
		os.Exit(1)
	}
}

func runHealthcheck() error {
	cfg, err := config.LoadHTTP()
	if err != nil {
		return err
	}
	ctx, cancel := context.WithTimeout(context.Background(), healthcheckTimeout)
	defer cancel()
	return healthcheck.Check(ctx, fmt.Sprintf("http://127.0.0.1:%d/healthz", cfg.Port))
}

// runSeed migrates and creates the sample products through the service, so
// they follow every rule. Products whose SKU already exists are skipped.
func runSeed(logger *slog.Logger) error {
	cfg, err := config.Load()
	if err != nil {
		return err
	}
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	pool, err := pgxpool.New(ctx, cfg.DatabaseURL)
	if err != nil {
		return fmt.Errorf("create database pool: %w", err)
	}
	defer pool.Close()
	if err := migrate.Up(ctx, pool, migrate.Options{}); err != nil {
		return fmt.Errorf("migrate: %w", err)
	}
	created, err := seed.Run(ctx, service.NewProducts(repository.NewProducts(pool)))
	if err != nil {
		return err
	}
	logger.Info("seed done", "created", created)
	return nil
}

func run(logger *slog.Logger) error {
	cfg, err := config.Load()
	if err != nil {
		return err
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	pool, err := pgxpool.New(ctx, cfg.DatabaseURL)
	if err != nil {
		return fmt.Errorf("create database pool: %w", err)
	}
	defer pool.Close()

	// Migrate before serving. Up retries for a while; if the database stays
	// unreachable the process exits, so the platform never promotes a
	// deployment with an outdated schema.
	if err := migrate.Up(ctx, pool, migrate.Options{}); err != nil {
		return fmt.Errorf("migrate: %w", err)
	}
	logger.Info("migrations applied")

	limiter := middleware.NewRateLimiter(middleware.RateLimitOptions{})
	go limiter.RunJanitor(ctx, time.Minute)

	health := handler.NewHealth(pool, readinessTimeout, logger)
	products := handler.NewProducts(
		service.NewProducts(repository.NewProducts(pool)),
		handler.NewValidator(),
	)
	srv := &http.Server{
		Addr: fmt.Sprintf(":%d", cfg.HTTP.Port),
		Handler: httpapi.NewRouter(logger, httpapi.RouterConfig{
			AllowedOrigins: cfg.CORSAllowedOrigins,
			ClientIPHeader: cfg.ClientIPHeader,
			RateLimiter:    limiter,
		}, health, products.Register),
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       10 * time.Second,
		WriteTimeout:      15 * time.Second,
		IdleTimeout:       60 * time.Second,
	}

	serveErr := make(chan error, 1)
	go func() {
		logger.Info("listening", "addr", srv.Addr)
		serveErr <- srv.ListenAndServe()
	}()

	select {
	case err := <-serveErr:
		if !errors.Is(err, http.ErrServerClosed) {
			return fmt.Errorf("serve: %w", err)
		}
		return nil
	case <-ctx.Done():
	}

	logger.Info("shutting down")
	shutdownCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), shutdownTimeout)
	defer cancel()
	if err := srv.Shutdown(shutdownCtx); err != nil {
		return fmt.Errorf("shutdown: %w", err)
	}
	return nil
}
