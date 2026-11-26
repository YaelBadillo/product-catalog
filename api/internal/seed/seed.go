// Package seed loads the sample products behind `make seed`.
package seed

import (
	"context"
	"errors"
	"fmt"

	"github.com/YaelBadillo/product-catalog/api/internal/domain"
)

// Creator creates a product applying every rule (the service).
type Creator interface {
	Create(ctx context.Context, in domain.ProductInput) (domain.Product, error)
}

// Run creates the sample products and returns how many it created. A SKU
// already in use is skipped, so running it again creates no duplicates.
func Run(ctx context.Context, creator Creator) (int, error) {
	created := 0
	for _, in := range Products() {
		_, err := creator.Create(ctx, in)
		switch {
		case err == nil:
			created++
		case errors.Is(err, domain.ErrSKUAlreadyExists):
		default:
			return created, fmt.Errorf("seed %s: %w", in.SKU, err)
		}
	}
	return created, nil
}
