// Package repository implements the service's persistence on Postgres with
// the sqlc queries in internal/repository/db.
package repository

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/YaelBadillo/product-catalog/api/internal/domain"
	"github.com/YaelBadillo/product-catalog/api/internal/repository/db"
)

const (
	pgUniqueViolation = "23505"
	skuActiveKey      = "products_sku_active_key"
)

// Products stores products in Postgres.
type Products struct {
	pool    *pgxpool.Pool
	queries *db.Queries
}

// NewProducts returns a repository on pool.
func NewProducts(pool *pgxpool.Pool) *Products {
	return &Products{pool: pool, queries: db.New(pool)}
}

// Create inserts a normalized, validated product.
func (r *Products) Create(ctx context.Context, in domain.ProductInput) (domain.Product, error) {
	stock, err := toInt32(in.Stock)
	if err != nil {
		return domain.Product{}, err
	}
	row, err := r.queries.CreateProduct(ctx, db.CreateProductParams{
		Sku:         in.SKU,
		Name:        in.Name,
		Description: in.Description,
		PriceCents:  in.PriceCents,
		Currency:    string(in.Currency),
		Stock:       stock,
		Category:    string(in.Category),
		IsActive:    in.IsActive == nil || *in.IsActive,
	})
	if err != nil {
		if isSKUConflict(err) {
			return domain.Product{}, domain.ErrSKUAlreadyExists
		}
		return domain.Product{}, fmt.Errorf("create product: %w", err)
	}
	return toDomain(row), nil
}

// Get returns the non-deleted product with id.
func (r *Products) Get(ctx context.Context, id string) (domain.Product, error) {
	row, err := r.queries.GetProduct(ctx, id)
	if err != nil {
		return domain.Product{}, notFound(err, "get product")
	}
	return toDomain(row), nil
}

// notFound turns "no rows" into domain.ErrNotFound and wraps anything else.
func notFound(err error, op string) error {
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.ErrNotFound
	}
	return fmt.Errorf("%s: %w", op, err)
}

func isSKUConflict(err error) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == pgUniqueViolation && pgErr.ConstraintName == skuActiveKey
}

var errStockRange = errors.New("stock out of range")

// toInt32 converts a stock already validated by the domain.
func toInt32(v int64) (int32, error) {
	if v < 0 || v > domain.MaxStock {
		return 0, errStockRange
	}
	return int32(v), nil
}

func toDomain(row db.Product) domain.Product {
	return domain.Product{
		ID:          row.ID,
		SKU:         row.Sku,
		Name:        row.Name,
		Description: row.Description,
		PriceCents:  row.PriceCents,
		Currency:    domain.Currency(row.Currency),
		Stock:       row.Stock,
		Category:    domain.Category(row.Category),
		IsActive:    row.IsActive,
		Version:     row.Version,
		CreatedAt:   row.CreatedAt,
		UpdatedAt:   row.UpdatedAt,
	}
}
