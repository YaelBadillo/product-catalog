package service_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/YaelBadillo/product-catalog/api/internal/domain"
	"github.com/YaelBadillo/product-catalog/api/internal/service"
)

func TestUpdateNormalizesAndValidates(t *testing.T) {
	var gotID string
	var gotPatch domain.ProductPatch
	repo := &fakeRepo{update: func(id string, p domain.ProductPatch) (domain.Product, error) {
		gotID, gotPatch = id, p
		return domain.Product{ID: id, Version: 2}, nil
	}}

	got, err := service.NewProducts(repo).Update(t.Context(), "id-1", domain.ProductPatch{
		Version: 1, Name: ptr("  Té verde  "), Description: ptr("   "),
	})

	require.NoError(t, err)
	assert.Equal(t, int32(2), got.Version)
	assert.Equal(t, "id-1", gotID)
	assert.Equal(t, "Té verde", *gotPatch.Name)
	require.NotNil(t, gotPatch.Description, "a blank description is a removal")
	assert.Empty(t, *gotPatch.Description)
}

func TestUpdateRejectsInvalidPatches(t *testing.T) {
	tests := map[string]domain.ProductPatch{
		"empty patch":   {Version: 1},
		"invalid field": {Version: 1, PriceCents: ptr[int64](0)},
		"no version":    {Stock: ptr[int64](1)},
	}
	for name, patch := range tests {
		t.Run(name, func(t *testing.T) {
			repo := &fakeRepo{}

			_, err := service.NewProducts(repo).Update(t.Context(), "id-1", patch)

			var verr *domain.ValidationError
			assert.ErrorAs(t, err, &verr)
		})
	}
}

func TestUpdatePropagatesRepositoryErrors(t *testing.T) {
	for _, want := range []error{domain.ErrVersionConflict, domain.ErrNotFound} {
		repo := &fakeRepo{update: func(string, domain.ProductPatch) (domain.Product, error) {
			return domain.Product{}, want
		}}

		_, err := service.NewProducts(repo).Update(t.Context(), "id-1", domain.ProductPatch{Version: 1, Stock: ptr[int64](1)})

		assert.ErrorIs(t, err, want)
	}
}
