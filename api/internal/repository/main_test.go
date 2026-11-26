package repository_test

import (
	"context"
	"log"
	"os"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/YaelBadillo/product-catalog/api/internal/repository"
	"github.com/YaelBadillo/product-catalog/api/internal/testutil"
)

// pool is the package's migrated database; nil under -short.
var pool *pgxpool.Pool

func TestMain(m *testing.M) {
	ctx := context.Background()
	dsn, terminate, err := testutil.StartPostgres(ctx)
	if err != nil {
		log.Fatal(err)
	}
	pool, err = testutil.NewMigratedPool(ctx, dsn)
	if err != nil {
		terminate()
		log.Fatal(err)
	}
	code := m.Run()
	if pool != nil {
		pool.Close()
	}
	terminate()
	os.Exit(code)
}

// newRepo empties the database and returns a repository on it.
func newRepo(t *testing.T) *repository.Products {
	t.Helper()
	testutil.ResetDB(t, pool)
	return repository.NewProducts(pool)
}
