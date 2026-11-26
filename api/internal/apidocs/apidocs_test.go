package apidocs_test

import (
	"bytes"
	"context"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"regexp"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/YaelBadillo/product-catalog/api/internal/httpapi"
	"github.com/YaelBadillo/product-catalog/api/internal/httpapi/handler"
	"github.com/YaelBadillo/product-catalog/api/internal/testutil"
)

type pinger struct{}

func (pinger) Ping(context.Context) error { return nil }

func TestMain(m *testing.M) {
	gin.SetMode(gin.TestMode)
	os.Exit(m.Run())
}

func newRouter() *gin.Engine {
	logger := slog.New(slog.DiscardHandler)
	health := handler.NewHealth(pinger{}, 0, logger)
	return httpapi.NewRouter(logger, httpapi.RouterConfig{AllowedOrigins: []string{"http://localhost:3000"}}, health)
}

func get(t *testing.T, router http.Handler, path string) (*http.Request, *httptest.ResponseRecorder) {
	t.Helper()
	req := httptest.NewRequestWithContext(t.Context(), http.MethodGet, path, http.NoBody)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	return req, rec
}

var pinnedAsset = regexp.MustCompile(`https://cdn\.jsdelivr\.net/npm/swagger-ui-dist@\d+\.\d+\.\d+/[^"]+"\s+integrity="sha384-[A-Za-z0-9+/=]+"\s+crossorigin="anonymous"`)

func TestDocsPage(t *testing.T) {
	req, rec := get(t, newRouter(), "/docs")

	require.Equal(t, http.StatusOK, rec.Code)
	assert.Contains(t, rec.Header().Get("Content-Type"), "text/html")
	csp := rec.Header().Get("Content-Security-Policy")
	assert.Contains(t, csp, "script-src https://cdn.jsdelivr.net 'sha256-")
	assert.Contains(t, csp, "style-src https://cdn.jsdelivr.net")
	assert.Contains(t, csp, "connect-src 'self'")
	assert.Contains(t, csp, "frame-ancestors 'none'")
	assert.NotContains(t, csp, "'unsafe-eval'")
	assert.Len(t, pinnedAsset.FindAllString(rec.Body.String(), -1), 2, "the CSS and the bundle are pinned with SRI")
	assert.Contains(t, rec.Body.String(), "/docs/openapi.yaml")
	assert.Equal(t, "nosniff", rec.Header().Get("X-Content-Type-Options"))
	testutil.AssertMatchesContract(t, req, rec)
}

func TestDocsContractIsTheRootOne(t *testing.T) {
	req, rec := get(t, newRouter(), "/docs/openapi.yaml")

	require.Equal(t, http.StatusOK, rec.Code)
	assert.Contains(t, rec.Header().Get("Content-Type"), "application/yaml")
	path, err := testutil.ContractPath()
	require.NoError(t, err)
	root, err := os.ReadFile(path) //nolint:gosec // G304: the repository's own contract, located by testutil.
	require.NoError(t, err)
	assert.True(t, bytes.Equal(root, rec.Body.Bytes()), "the embedded contract must match openapi.yaml: run make sync-openapi")
	testutil.AssertMatchesContract(t, req, rec)
}

func TestDocsAreNotRateLimited(t *testing.T) {
	router := newRouter()

	for i := range 400 {
		_, rec := get(t, router, "/docs/openapi.yaml")
		require.Equal(t, http.StatusOK, rec.Code, "request %d", i+1)
	}
}
