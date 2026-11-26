package repository

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"

	"github.com/YaelBadillo/product-catalog/api/internal/domain"
	"github.com/YaelBadillo/product-catalog/api/internal/repository/db"
)

// Update applies a normalized, validated patch if the version matches. An
// empty Description removes it.
func (r *Products) Update(ctx context.Context, id string, p domain.ProductPatch) (domain.Product, error) {
	params := db.UpdateProductParams{
		ID:         id,
		Version:    p.Version,
		Name:       p.Name,
		PriceCents: p.PriceCents,
		IsActive:   p.IsActive,
		Currency:   stringPtr(p.Currency),
		Category:   stringPtr(p.Category),
	}
	if p.Description != nil {
		params.DescriptionSet = true
		if *p.Description != "" {
			params.Description = p.Description
		}
	}
	if p.Stock != nil {
		stock, err := toInt32(*p.Stock)
		if err != nil {
			return domain.Product{}, err
		}
		params.Stock = &stock
	}

	row, err := r.queries.UpdateProduct(ctx, params)
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.Product{}, r.missingOrConflict(ctx, id)
	}
	if err != nil {
		return domain.Product{}, fmt.Errorf("update product: %w", err)
	}
	return toDomain(row), nil
}

// missingOrConflict explains why a versioned write changed no row: the
// product is gone, or someone else changed it first.
func (r *Products) missingOrConflict(ctx context.Context, id string) error {
	exists, err := r.queries.ProductExists(ctx, id)
	if err != nil {
		return fmt.Errorf("check product exists: %w", err)
	}
	if !exists {
		return domain.ErrNotFound
	}
	return domain.ErrVersionConflict
}

func stringPtr[T ~string](v *T) *string {
	if v == nil {
		return nil
	}
	s := string(*v)
	return &s
}
