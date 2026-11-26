package migrate_test

import (
	"errors"
	"strings"
	"testing"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/YaelBadillo/product-catalog/api/internal/migrate"
)

func migratedDB(t *testing.T) *pgxpool.Pool {
	t.Helper()
	pool, _ := newDatabase(t)
	require.NoError(t, migrate.Up(t.Context(), pool, migrate.Options{}))
	return pool
}

// row is a valid product; each case overrides one column.
type row struct {
	sku, name, currency, category string
	description                   *string
	price                         int64
	stock                         int64
	version                       int64
}

func validRow() row {
	return row{sku: "ABC-123", name: "Café", currency: "MXN", category: "food", price: 100, stock: 0, version: 1}
}

func insert(t *testing.T, pool *pgxpool.Pool, r row) error {
	t.Helper()
	_, err := pool.Exec(t.Context(), `INSERT INTO products
		(sku, name, description, price_cents, currency, stock, category, version)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)`,
		r.sku, r.name, r.description, r.price, r.currency, r.stock, r.category, r.version)
	return err
}

func pgCode(err error) (code, constraint string) {
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		return pgErr.Code, pgErr.ConstraintName
	}
	return "", ""
}

func ptr(s string) *string { return &s }

func TestSchemaChecks(t *testing.T) {
	pool := migratedDB(t)

	tests := []struct {
		name   string
		mutate func(*row)
	}{
		{name: "lowercase sku", mutate: func(r *row) { r.sku = "abc-123" }},
		{name: "short sku", mutate: func(r *row) { r.sku = "ABC" }},
		{name: "long sku", mutate: func(r *row) { r.sku = strings.Repeat("A", 33) }},
		{name: "sku with a space", mutate: func(r *row) { r.sku = "AB C1" }},
		{name: "sku with Ñ", mutate: func(r *row) { r.sku = "AÑO-1" }},
		{name: "short name", mutate: func(r *row) { r.name = "a" }},
		{name: "long name", mutate: func(r *row) { r.name = strings.Repeat("a", 121) }},
		{name: "empty description", mutate: func(r *row) { r.description = ptr("") }},
		{name: "long description", mutate: func(r *row) { r.description = ptr(strings.Repeat("a", 2001)) }},
		{name: "zero price", mutate: func(r *row) { r.price = 0 }},
		{name: "price above the maximum", mutate: func(r *row) { r.price = 100_000_000_000 }},
		{name: "negative stock", mutate: func(r *row) { r.stock = -1 }},
		{name: "unknown currency", mutate: func(r *row) { r.currency = "EUR" }},
		{name: "unknown category", mutate: func(r *row) { r.category = "toys" }},
		{name: "version zero", mutate: func(r *row) { r.version = 0 }},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := validRow()
			tt.mutate(&r)

			err := insert(t, pool, r)

			code, _ := pgCode(err)
			assert.Equal(t, "23514", code, "expected a CHECK violation, got %v", err)
		})
	}

	t.Run("bounds are inclusive", func(t *testing.T) {
		r := validRow()
		r.sku = strings.Repeat("Z", 32)
		r.name = strings.Repeat("n", 120)
		r.description = ptr(strings.Repeat("d", 2000))
		r.price = 99_999_999_999
		r.stock = 2_147_483_647
		require.NoError(t, insert(t, pool, r))
	})
}

func TestSchemaDefaults(t *testing.T) {
	pool := migratedDB(t)
	mustExec(t, pool, `INSERT INTO products (sku, name, price_cents, stock, category)
		VALUES ('DEF-1', 'Defaults', 100, 1, 'home'), ('DEF-2', 'Defaults', 100, 1, 'home')`)

	rows, err := pool.Query(t.Context(), `SELECT currency, is_active, version, uuid_extract_version(id),
		created_at IS NOT NULL, updated_at IS NOT NULL, deleted_at IS NULL, id::text
		FROM products ORDER BY sku`)
	require.NoError(t, err)
	type defaults struct {
		Currency                       string
		IsActive                       bool
		Version, UUIDVersion           int
		HasCreated, HasUpdated, NotDel bool
		ID                             string
	}
	got, err := pgx.CollectRows(rows, pgx.RowToStructByPos[defaults])
	require.NoError(t, err)
	require.Len(t, got, 2)

	for _, d := range got {
		assert.Equal(t, "MXN", d.Currency)
		assert.True(t, d.IsActive)
		assert.Equal(t, 1, d.Version)
		assert.Equal(t, 7, d.UUIDVersion, "ids are uuidv7")
		assert.True(t, d.HasCreated)
		assert.True(t, d.HasUpdated)
		assert.True(t, d.NotDel)
	}
	assert.Less(t, got[0].ID, got[1].ID, "uuidv7 ids are time-ordered")
}

func TestSchemaSKUUniqueAmongActive(t *testing.T) {
	pool := migratedDB(t)
	require.NoError(t, insert(t, pool, validRow()))

	err := insert(t, pool, validRow())

	code, constraint := pgCode(err)
	assert.Equal(t, "23505", code)
	assert.Equal(t, "products_sku_active_key", constraint)

	mustExec(t, pool, "UPDATE products SET deleted_at = now() WHERE sku = 'ABC-123'")
	assert.NoError(t, insert(t, pool, validRow()), "a deleted product frees its SKU")
}

func TestSchemaSpanishOrder(t *testing.T) {
	pool := migratedDB(t)
	for i, name := range []string{"Zapato", "árbol", "Arroz", "ñame", "nuez"} {
		r := validRow()
		r.sku = "ORD-" + string(rune('A'+i))
		r.name = name
		require.NoError(t, insert(t, pool, r))
	}

	rows, err := pool.Query(t.Context(), "SELECT name FROM products ORDER BY name")
	require.NoError(t, err)
	names, err := pgx.CollectRows(rows, pgx.RowTo[string])
	require.NoError(t, err)

	assert.Equal(t, []string{"árbol", "Arroz", "nuez", "ñame", "Zapato"}, names)
}

func TestSchemaCatalogFold(t *testing.T) {
	pool := migratedDB(t)

	for in, want := range map[string]string{"Café": "cafe", "Ñandú": "nandu", "ABC-123": "abc-123"} {
		var got string
		require.NoError(t, pool.QueryRow(t.Context(), "SELECT catalog_fold($1)", in).Scan(&got))
		assert.Equal(t, want, got, in)
	}
}

func TestSchemaIndexes(t *testing.T) {
	pool := migratedDB(t)

	rows, err := pool.Query(t.Context(),
		"SELECT indexname FROM pg_indexes WHERE tablename = 'products' AND indexdef LIKE '%WHERE (deleted_at IS NULL)%'")
	require.NoError(t, err)
	names, err := pgx.CollectRows(rows, pgx.RowTo[string])
	require.NoError(t, err)

	assert.ElementsMatch(t, []string{
		"products_sku_active_key", "products_category_idx",
		"products_created_at_idx", "products_currency_price_idx",
	}, names)
}
