package service

import (
	"context"

	"github.com/YaelBadillo/product-catalog/api/internal/domain"
)

// Update normalizes and validates the patch, then applies it if the version
// still matches. A patch that changes nothing is a validation error, so an
// empty save never bumps the version.
func (s *Products) Update(ctx context.Context, id string, patch domain.ProductPatch) (domain.Product, error) {
	patch = patch.Normalize()
	if err := patch.Validate(); err != nil {
		return domain.Product{}, err
	}
	return s.repo.Update(ctx, id, patch)
}
