package main_test

import (
	"context"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/YaelBadillo/product-catalog/api/internal/testutil"
)

// dsn is the package's Postgres container; the binary runs against it.
var dsn string

func TestMain(m *testing.M) {
	var terminate func()
	var err error
	dsn, terminate, err = testutil.StartPostgres(context.Background())
	if err != nil {
		log.Fatal(err)
	}
	code := m.Run()
	terminate()
	os.Exit(code)
}

// buildBinary compiles cmd/api into a temporary directory.
func buildBinary(t *testing.T) string {
	t.Helper()
	bin := filepath.Join(t.TempDir(), "api")
	//nolint:gosec // G204: builds this package into the test's own temp dir.
	out, err := exec.CommandContext(t.Context(), "go", "build", "-o", bin, ".").CombinedOutput()
	require.NoError(t, err, string(out))
	return bin
}

func freePort(t *testing.T) int {
	t.Helper()
	l, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	defer func() { _ = l.Close() }()
	return l.Addr().(*net.TCPAddr).Port
}

// command runs the binary with the test environment and extra arguments.
func command(t *testing.T, bin string, port int, args ...string) *exec.Cmd {
	t.Helper()
	//nolint:gosec // G204: runs the binary the test just built.
	cmd := exec.CommandContext(t.Context(), bin, args...)
	cmd.Env = append(os.Environ(),
		"DATABASE_URL="+dsn,
		fmt.Sprintf("PORT=%d", port),
		"CORS_ALLOWED_ORIGINS=http://localhost:3000",
	)
	return cmd
}

func waitHealthy(t *testing.T, port int) {
	t.Helper()
	url := fmt.Sprintf("http://127.0.0.1:%d/healthz", port)
	require.Eventually(t, func() bool {
		req, err := http.NewRequestWithContext(t.Context(), http.MethodGet, url, http.NoBody)
		if err != nil {
			return false
		}
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			return false
		}
		_ = resp.Body.Close()
		return resp.StatusCode == http.StatusOK
	}, 30*time.Second, 100*time.Millisecond, "the API never answered /healthz")
}

func openPool(t *testing.T) *pgxpool.Pool {
	t.Helper()
	pool, err := pgxpool.New(t.Context(), dsn)
	require.NoError(t, err)
	t.Cleanup(pool.Close)
	return pool
}

func TestStartupAppliesMigrations(t *testing.T) {
	testutil.RequireDSN(t, dsn)
	bin := buildBinary(t)
	port := freePort(t)
	cmd := command(t, bin, port)
	require.NoError(t, cmd.Start())
	t.Cleanup(func() {
		_ = cmd.Process.Signal(os.Interrupt)
		_ = cmd.Wait()
	})

	waitHealthy(t, port)

	var exists bool
	require.NoError(t, openPool(t).QueryRow(t.Context(),
		"SELECT to_regclass('products') IS NOT NULL").Scan(&exists))
	assert.True(t, exists, "migrations run at startup")
}
