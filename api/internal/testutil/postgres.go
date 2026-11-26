package testutil

import (
	"context"
	"flag"
	"fmt"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/modules/postgres"

	"github.com/YaelBadillo/product-catalog/api/internal/migrate"
)

// PostgresImage is the image integration tests run against: the same major
// version as docker-compose.yml and Railway.
const PostgresImage = "postgres:18-alpine"

const skipShort = "integration test: skipped under -short"

// StartPostgres starts a disposable Postgres container for a package's
// TestMain and returns its DSN and a func that removes it. Under -short it
// starts nothing and returns an empty DSN, so the helpers below skip the tests
// that need a database.
func StartPostgres(ctx context.Context) (dsn string, terminate func(), err error) {
	if !flag.Parsed() {
		flag.Parse()
	}
	if testing.Short() {
		return "", func() {}, nil
	}

	container, err := postgres.Run(ctx, PostgresImage,
		postgres.WithDatabase("catalog"),
		postgres.WithUsername("catalog"),
		postgres.WithPassword("catalog"),
		postgres.BasicWaitStrategies(),
	)
	if err != nil {
		return "", func() {}, fmt.Errorf("start postgres container: %w", err)
	}
	terminate = func() { _ = testcontainers.TerminateContainer(container) }

	dsn, err = container.ConnectionString(ctx, "sslmode=disable")
	if err != nil {
		terminate()
		return "", func() {}, fmt.Errorf("postgres connection string: %w", err)
	}
	return dsn, terminate, nil
}

// RequireDSN skips t when dsn is empty, that is, under -short.
func RequireDSN(t testing.TB, dsn string) {
	t.Helper()
	if dsn == "" {
		t.Skip(skipShort)
	}
}

// ResetDB empties every table between tests. A nil pool (under -short) skips
// the test instead.
func ResetDB(t testing.TB, pool *pgxpool.Pool) {
	t.Helper()
	if pool == nil {
		t.Skip(skipShort)
	}
	if _, err := pool.Exec(t.Context(), "TRUNCATE products"); err != nil {
		t.Fatalf("reset database: %v", err)
	}
}

// NewMigratedPool opens a pool on dsn and applies every migration. Repository
// tests call it from TestMain after StartPostgres. An empty dsn (under -short)
// returns a nil pool, which ResetDB turns into a skip.
func NewMigratedPool(ctx context.Context, dsn string) (*pgxpool.Pool, error) {
	if dsn == "" {
		return nil, nil
	}
	pool, err := pgxpool.New(ctx, dsn)
	if err != nil {
		return nil, fmt.Errorf("open pool: %w", err)
	}
	if err := migrate.Up(ctx, pool, migrate.Options{}); err != nil {
		pool.Close()
		return nil, err
	}
	return pool, nil
}
