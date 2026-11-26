package migrate_test

import (
	"context"
	"crypto/rand"
	"fmt"
	"log"
	"net/url"
	"os"
	"strings"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/require"

	"github.com/YaelBadillo/product-catalog/api/internal/testutil"
)

// adminDSN points at the package's container; each test creates its own
// database from it, so every test starts from an empty schema.
var adminDSN string

func TestMain(m *testing.M) {
	dsn, terminate, err := testutil.StartPostgres(context.Background())
	if err != nil {
		log.Fatal(err)
	}
	adminDSN = dsn
	code := m.Run()
	terminate()
	os.Exit(code)
}

// newDatabase creates an empty database and returns a pool connected to it.
func newDatabase(t *testing.T) (*pgxpool.Pool, string) {
	t.Helper()
	testutil.RequireDSN(t, adminDSN)
	ctx := t.Context()

	admin, err := pgxpool.New(ctx, adminDSN)
	require.NoError(t, err)
	defer admin.Close()
	name := "test_" + strings.ToLower(rand.Text())
	_, err = admin.Exec(ctx, "CREATE DATABASE "+name)
	require.NoError(t, err)

	u, err := url.Parse(adminDSN)
	require.NoError(t, err)
	u.Path = "/" + name
	dsn := u.String()
	pool, err := pgxpool.New(ctx, dsn)
	require.NoError(t, err)
	t.Cleanup(pool.Close)
	return pool, dsn
}

func mustExec(t *testing.T, pool *pgxpool.Pool, sql string, args ...any) {
	t.Helper()
	_, err := pool.Exec(t.Context(), sql, args...)
	require.NoError(t, err, fmt.Sprintf("exec %q", sql))
}
