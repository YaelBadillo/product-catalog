package repository_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/YaelBadillo/product-catalog/api/internal/domain"
)

func TestDeleteIsLogical(t *testing.T) {
	repo := newRepo(t)
	created, err := repo.Create(t.Context(), input("ABC-123"))
	require.NoError(t, err)

	require.NoError(t, repo.Delete(t.Context(), created.ID, 1))

	var version int32
	var deleted bool
	require.NoError(t, pool.QueryRow(t.Context(),
		"SELECT version, deleted_at IS NOT NULL FROM products WHERE id = $1", created.ID).Scan(&version, &deleted))
	assert.True(t, deleted, "the row stays, marked deleted")
	assert.Equal(t, int32(2), version)

	_, err = repo.Get(t.Context(), created.ID)
	require.ErrorIs(t, err, domain.ErrNotFound)

	page, err := repo.List(t.Context(), params(nil))
	require.NoError(t, err)
	assert.Empty(t, page.Items)
	assert.Zero(t, page.Total)

	_, err = repo.Create(t.Context(), input("ABC-123"))
	assert.NoError(t, err, "the SKU is free again")
}

func TestDeleteVersionConflict(t *testing.T) {
	repo := newRepo(t)
	created, err := repo.Create(t.Context(), input("ABC-123"))
	require.NoError(t, err)
	_, err = repo.Update(t.Context(), created.ID, domain.ProductPatch{Version: 1, Stock: ptr[int64](2)})
	require.NoError(t, err)

	err = repo.Delete(t.Context(), created.ID, 1)

	require.ErrorIs(t, err, domain.ErrVersionConflict)
	_, err = repo.Get(t.Context(), created.ID)
	assert.NoError(t, err, "the product is still there")
}

func TestDeleteNotFound(t *testing.T) {
	repo := newRepo(t)
	created, err := repo.Create(t.Context(), input("ABC-123"))
	require.NoError(t, err)
	require.NoError(t, repo.Delete(t.Context(), created.ID, 1))

	for name, id := range map[string]string{"already deleted": created.ID, "missing": "00000000-0000-7000-8000-000000000000"} {
		t.Run(name, func(t *testing.T) {
			assert.ErrorIs(t, repo.Delete(t.Context(), id, 2), domain.ErrNotFound)
		})
	}
}
