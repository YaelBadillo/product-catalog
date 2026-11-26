// Command api starts the Product Catalog HTTP API.
package main

import (
	"log/slog"
	"net/http"
	"os"
	"time"

	"github.com/YaelBadillo/product-catalog/api/internal/httpapi"
)

func main() {
	srv := &http.Server{
		Addr:              ":8080",
		Handler:           httpapi.NewRouter(),
		ReadHeaderTimeout: 5 * time.Second,
	}

	slog.Info("listening", "addr", srv.Addr)
	if err := srv.ListenAndServe(); err != nil {
		slog.Error("server stopped", "error", err)
		os.Exit(1)
	}
}
