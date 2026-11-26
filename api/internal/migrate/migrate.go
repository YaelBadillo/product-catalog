// Package migrate applies the embedded database migrations.
package migrate

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/jackc/pgx/v5/stdlib"
	"github.com/pressly/goose/v3"
	"github.com/pressly/goose/v3/lock"

	"github.com/YaelBadillo/product-catalog/api/migrations"
)

const (
	defaultRetryFor = 30 * time.Second
	initialBackoff  = 100 * time.Millisecond
	maxBackoff      = 2 * time.Second
)

// Options tunes Up. The zero value uses the defaults.
type Options struct {
	// RetryFor bounds how long Up waits for the database to become
	// reachable. Defaults to 30 s.
	RetryFor time.Duration
}

// Up applies every pending migration. It waits, with backoff, up to
// opts.RetryFor for the database to answer, and holds a Postgres session lock
// so concurrent instances never migrate at the same time.
func Up(ctx context.Context, pool *pgxpool.Pool, opts Options) error {
	if opts.RetryFor <= 0 {
		opts.RetryFor = defaultRetryFor
	}
	if err := waitForDatabase(ctx, pool, opts.RetryFor); err != nil {
		return err
	}

	locker, err := lock.NewPostgresSessionLocker(lock.WithLockTimeout(1, 60))
	if err != nil {
		return fmt.Errorf("create migration locker: %w", err)
	}
	db := stdlib.OpenDBFromPool(pool)
	defer func() { _ = db.Close() }()

	provider, err := goose.NewProvider(goose.DialectPostgres, db, migrations.FS,
		goose.WithSessionLocker(locker),
		goose.WithDisableGlobalRegistry(true),
	)
	if err != nil {
		return fmt.Errorf("create migration provider: %w", err)
	}
	if _, err := provider.Up(ctx); err != nil {
		return fmt.Errorf("apply migrations: %w", err)
	}
	return nil
}

// waitForDatabase pings until the database answers or retryFor expires.
func waitForDatabase(ctx context.Context, pool *pgxpool.Pool, retryFor time.Duration) error {
	deadline := time.Now().Add(retryFor)
	backoff := initialBackoff
	for {
		err := pool.Ping(ctx)
		if err == nil {
			return nil
		}
		wait := min(backoff, time.Until(deadline))
		if wait <= 0 {
			return fmt.Errorf("database unreachable after %s: %w", retryFor, err)
		}
		select {
		case <-ctx.Done():
			return errors.Join(fmt.Errorf("wait for database: %w", ctx.Err()), err)
		case <-time.After(wait):
		}
		backoff = min(backoff*2, maxBackoff)
	}
}
