// Package config loads the API configuration from environment variables.
package config

import (
	"fmt"
	"net/url"

	"github.com/caarlos0/env/v11"
)

// Config is the full API configuration.
type Config struct {
	HTTP               HTTP
	DatabaseURL        string   `env:"DATABASE_URL,required,notEmpty"`
	CORSAllowedOrigins []string `env:"CORS_ALLOWED_ORIGINS,required,notEmpty" envSeparator:","`
	// ClientIPHeader is the header the platform's edge sets to the client IP
	// (X-Real-IP on Railway). Empty: use the connection's remote address.
	ClientIPHeader string `env:"CLIENT_IP_HEADER"`
}

// HTTP is the HTTP server configuration.
type HTTP struct {
	Port int `env:"PORT" envDefault:"8080"`
}

// Load reads and validates the full configuration.
func Load() (Config, error) {
	cfg, err := env.ParseAs[Config]()
	if err != nil {
		return Config{}, fmt.Errorf("parse config: %w", err)
	}
	for _, origin := range cfg.CORSAllowedOrigins {
		if err := validateOrigin(origin); err != nil {
			return Config{}, fmt.Errorf("CORS_ALLOWED_ORIGINS: %w", err)
		}
	}
	return cfg, nil
}

// LoadHTTP reads only the HTTP server configuration.
func LoadHTTP() (HTTP, error) {
	cfg, err := env.ParseAs[HTTP]()
	if err != nil {
		return HTTP{}, fmt.Errorf("parse http config: %w", err)
	}
	return cfg, nil
}

// validateOrigin accepts only explicit origins: http(s)://host[:port], with no
// wildcard, path, query or fragment.
func validateOrigin(origin string) error {
	u, err := url.Parse(origin)
	if err != nil {
		return fmt.Errorf("invalid origin %q: %w", origin, err)
	}
	if (u.Scheme != "http" && u.Scheme != "https") || u.Host == "" ||
		u.Path != "" || u.RawQuery != "" || u.Fragment != "" || u.User != nil {
		return fmt.Errorf("invalid origin %q: must be http(s)://host[:port]", origin)
	}
	return nil
}
