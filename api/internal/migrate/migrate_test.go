package migrate_test

import (
	"sync"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/YaelBadillo/product-catalog/api/internal/migrate"
)

func tableExists(t *testing.T, pool *pgxpool.Pool, name string) bool {
	t.Helper()
	var exists bool
	err := pool.QueryRow(t.Context(), "SELECT to_regclass($1) IS NOT NULL", name).Scan(&exists)
	require.NoError(t, err)
	return exists
}

func TestUpCreatesProducts(t *testing.T) {
	pool, _ := newDatabase(t)

	require.NoError(t, migrate.Up(t.Context(), pool, migrate.Options{}))

	assert.True(t, tableExists(t, pool, "products"))
}

func TestUpTwiceIsANoOp(t *testing.T) {
	pool, _ := newDatabase(t)
	require.NoError(t, migrate.Up(t.Context(), pool, migrate.Options{}))
	mustExec(t, pool, `INSERT INTO products (sku, name, price_cents, stock, category)
		VALUES ('KEEP-1', 'Keep me', 100, 1, 'other')`)

	require.NoError(t, migrate.Up(t.Context(), pool, migrate.Options{}))

	var count int
	require.NoError(t, pool.QueryRow(t.Context(), "SELECT count(*) FROM products").Scan(&count))
	assert.Equal(t, 1, count, "a second Up must not touch existing data")
}

func TestUpConcurrently(t *testing.T) {
	pool, _ := newDatabase(t)

	var wg sync.WaitGroup
	errs := make([]error, 2)
	for i := range errs {
		wg.Go(func() { errs[i] = migrate.Up(t.Context(), pool, migrate.Options{}) })
	}
	wg.Wait()

	for _, err := range errs {
		require.NoError(t, err)
	}
	assert.True(t, tableExists(t, pool, "products"))
}

func TestUpRetriesThenFailsWhenTheDatabaseIsUnreachable(t *testing.T) {
	pool, err := pgxpool.New(t.Context(),
		"postgres://catalog:catalog@127.0.0.1:1/catalog?sslmode=disable&connect_timeout=1")
	require.NoError(t, err)
	t.Cleanup(pool.Close)
	start := time.Now()

	err = migrate.Up(t.Context(), pool, migrate.Options{RetryFor: 300 * time.Millisecond})

	require.Error(t, err)
	assert.GreaterOrEqual(t, time.Since(start), 300*time.Millisecond, "it retries until the window expires")
	assert.Less(t, time.Since(start), 10*time.Second)
}
