// Package service holds the business rules of the catalog.
package service

import (
	"context"

	"github.com/YaelBadillo/product-catalog/api/internal/domain"
)

// ProductRepository is the persistence the service needs.
type ProductRepository interface {
	Create(ctx context.Context, in domain.ProductInput) (domain.Product, error)
	Get(ctx context.Context, id string) (domain.Product, error)
	List(ctx context.Context, p ListParams) (ProductPage, error)
	Update(ctx context.Context, id string, patch domain.ProductPatch) (domain.Product, error)
}

// Products applies the product rules on top of a repository.
type Products struct {
	repo ProductRepository
}

// NewProducts returns the product service.
func NewProducts(repo ProductRepository) *Products {
	return &Products{repo: repo}
}

// Create normalizes and validates in, then stores it. Every caller (the
// handler, the seed) goes through these rules.
func (s *Products) Create(ctx context.Context, in domain.ProductInput) (domain.Product, error) {
	in = in.Normalize()
	if err := in.Validate(); err != nil {
		return domain.Product{}, err
	}
	return s.repo.Create(ctx, in)
}

// Get returns a product or domain.ErrNotFound.
func (s *Products) Get(ctx context.Context, id string) (domain.Product, error) {
	return s.repo.Get(ctx, id)
}
