package config_test

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/YaelBadillo/product-catalog/api/internal/config"
)

var envKeys = []string{"PORT", "DATABASE_URL", "CORS_ALLOWED_ORIGINS"}

// setEnv sets exactly the given variables and unsets the other known ones,
// restoring everything when the test ends.
func setEnv(t *testing.T, vars map[string]string) {
	t.Helper()
	for _, key := range envKeys {
		t.Setenv(key, "")
		value, ok := vars[key]
		if !ok {
			require.NoError(t, os.Unsetenv(key))
			continue
		}
		t.Setenv(key, value)
	}
}

func TestLoad(t *testing.T) {
	tests := []struct {
		name    string
		env     map[string]string
		want    config.Config
		wantErr bool
	}{
		{
			name: "defaults",
			env: map[string]string{
				"DATABASE_URL":         "postgres://localhost:5432/catalog",
				"CORS_ALLOWED_ORIGINS": "http://localhost:3000",
			},
			want: config.Config{
				HTTP:               config.HTTP{Port: 8080},
				DatabaseURL:        "postgres://localhost:5432/catalog",
				CORSAllowedOrigins: []string{"http://localhost:3000"},
			},
		},
		{
			name: "every variable set",
			env: map[string]string{
				"PORT":                 "9090",
				"DATABASE_URL":         "postgres://db/catalog",
				"CORS_ALLOWED_ORIGINS": "http://localhost:3000,https://catalog.example.com",
			},
			want: config.Config{
				HTTP:               config.HTTP{Port: 9090},
				DatabaseURL:        "postgres://db/catalog",
				CORSAllowedOrigins: []string{"http://localhost:3000", "https://catalog.example.com"},
			},
		},
		{
			name:    "missing database url",
			env:     map[string]string{"CORS_ALLOWED_ORIGINS": "http://localhost:3000"},
			wantErr: true,
		},
		{
			name:    "empty database url",
			env:     map[string]string{"DATABASE_URL": "", "CORS_ALLOWED_ORIGINS": "http://localhost:3000"},
			wantErr: true,
		},
		{
			name:    "missing cors origins",
			env:     map[string]string{"DATABASE_URL": "postgres://db/catalog"},
			wantErr: true,
		},
		{
			name:    "invalid port",
			env:     map[string]string{"PORT": "abc", "DATABASE_URL": "postgres://db/catalog", "CORS_ALLOWED_ORIGINS": "http://localhost:3000"},
			wantErr: true,
		},
		{
			name:    "wildcard cors origin",
			env:     map[string]string{"DATABASE_URL": "postgres://db/catalog", "CORS_ALLOWED_ORIGINS": "*"},
			wantErr: true,
		},
		{
			name:    "cors origin without scheme",
			env:     map[string]string{"DATABASE_URL": "postgres://db/catalog", "CORS_ALLOWED_ORIGINS": "localhost:3000"},
			wantErr: true,
		},
		{
			name:    "cors origin with path",
			env:     map[string]string{"DATABASE_URL": "postgres://db/catalog", "CORS_ALLOWED_ORIGINS": "http://localhost:3000/app"},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			setEnv(t, tt.env)

			got, err := config.Load()

			if tt.wantErr {
				assert.Error(t, err)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestLoadHTTP(t *testing.T) {
	tests := []struct {
		name    string
		env     map[string]string
		want    config.HTTP
		wantErr bool
	}{
		{name: "default port", env: map[string]string{}, want: config.HTTP{Port: 8080}},
		{name: "custom port", env: map[string]string{"PORT": "9090"}, want: config.HTTP{Port: 9090}},
		{name: "invalid port", env: map[string]string{"PORT": "abc"}, wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			setEnv(t, tt.env)

			got, err := config.LoadHTTP()

			if tt.wantErr {
				assert.Error(t, err)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tt.want, got)
		})
	}
}
