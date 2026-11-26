package service_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/YaelBadillo/product-catalog/api/internal/domain"
	"github.com/YaelBadillo/product-catalog/api/internal/service"
)

func TestDelete(t *testing.T) {
	var gotID string
	var gotVersion int32
	repo := &fakeRepo{remove: func(id string, version int32) error {
		gotID, gotVersion = id, version
		return nil
	}}

	require.NoError(t, service.NewProducts(repo).Delete(t.Context(), "id-1", 4))

	assert.Equal(t, "id-1", gotID)
	assert.Equal(t, int32(4), gotVersion)
}

func TestDeletePropagatesErrors(t *testing.T) {
	for _, want := range []error{domain.ErrVersionConflict, domain.ErrNotFound} {
		repo := &fakeRepo{remove: func(string, int32) error { return want }}

		err := service.NewProducts(repo).Delete(t.Context(), "id-1", 1)

		assert.ErrorIs(t, err, want)
	}
}
