package repository

import (
	"context"
	"fmt"
	"math"
	"strings"

	"github.com/jackc/pgx/v5"

	"github.com/YaelBadillo/product-catalog/api/internal/domain"
	"github.com/YaelBadillo/product-catalog/api/internal/repository/db"
	"github.com/YaelBadillo/product-catalog/api/internal/service"
)

// likeEscaper makes \, % and _ match literally in a LIKE ... ESCAPE '\'.
var likeEscaper = strings.NewReplacer(`\`, `\\`, `%`, `\%`, `_`, `\_`)

// List returns one page and the total from the same snapshot: both queries
// run in one read-only REPEATABLE READ transaction.
func (r *Products) List(ctx context.Context, p service.ListParams) (page service.ProductPage, err error) {
	tx, err := r.pool.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.RepeatableRead, AccessMode: pgx.ReadOnly})
	if err != nil {
		return service.ProductPage{}, fmt.Errorf("begin list: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }() // a no-op after commit

	q := r.queries.WithTx(tx)
	filters := listFilters(p)
	rows, err := q.ListProducts(ctx, db.ListProductsParams{
		Category:   filters.Category,
		IsActive:   filters.IsActive,
		Currency:   filters.Currency,
		MinPrice:   filters.MinPrice,
		MaxPrice:   filters.MaxPrice,
		Pattern:    filters.Pattern,
		Sort:       string(p.Sort),
		SortOrder:  string(p.Order),
		PageLimit:  clampInt32(int64(p.PageSize)),
		PageOffset: clampInt32(int64(p.Page-1) * int64(p.PageSize)),
	})
	if err != nil {
		return service.ProductPage{}, fmt.Errorf("list products: %w", err)
	}
	total, err := q.CountProducts(ctx, filters)
	if err != nil {
		return service.ProductPage{}, fmt.Errorf("count products: %w", err)
	}
	if err := tx.Commit(ctx); err != nil {
		return service.ProductPage{}, fmt.Errorf("commit list: %w", err)
	}

	items := make([]domain.Product, 0, len(rows))
	for _, row := range rows {
		items = append(items, toDomain(row))
	}
	return service.ProductPage{Items: items, Total: total}, nil
}

func listFilters(p service.ListParams) db.CountProductsParams {
	filters := db.CountProductsParams{
		IsActive: p.IsActive,
		MinPrice: p.MinPriceCents,
		MaxPrice: p.MaxPriceCents,
	}
	if p.Category != nil {
		category := string(*p.Category)
		filters.Category = &category
	}
	if p.Currency != nil {
		currency := string(*p.Currency)
		filters.Currency = &currency
	}
	if p.Query != "" {
		pattern := likeEscaper.Replace(p.Query)
		filters.Pattern = &pattern
	}
	return filters
}

// clampInt32 bounds a LIMIT or OFFSET; an offset past every row just returns
// an empty page.
func clampInt32(v int64) int32 {
	return int32(max(0, min(v, math.MaxInt32)))
}
