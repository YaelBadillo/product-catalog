package repository

import (
	"context"
	"fmt"

	"github.com/YaelBadillo/product-catalog/api/internal/repository/db"
)

// Delete marks the product deleted if the version matches.
func (r *Products) Delete(ctx context.Context, id string, version int32) error {
	rows, err := r.queries.SoftDeleteProduct(ctx, db.SoftDeleteProductParams{ID: id, Version: version})
	if err != nil {
		return fmt.Errorf("delete product: %w", err)
	}
	if rows == 0 {
		return r.missingOrConflict(ctx, id)
	}
	return nil
}
