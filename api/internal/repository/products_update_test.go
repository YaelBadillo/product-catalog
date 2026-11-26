package repository_test

import (
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/YaelBadillo/product-catalog/api/internal/domain"
)

func TestUpdateChangesOnlyTheGivenFields(t *testing.T) {
	repo := newRepo(t)
	in := input("ABC-123")
	in.Description = ptr("Tostado")
	created, err := repo.Create(t.Context(), in)
	require.NoError(t, err)

	got, err := repo.Update(t.Context(), created.ID, domain.ProductPatch{
		Version: 1, Stock: ptr[int64](7), Currency: ptr(domain.CurrencyUSD),
	})

	require.NoError(t, err)
	assert.Equal(t, int32(7), got.Stock)
	assert.Equal(t, domain.CurrencyUSD, got.Currency)
	assert.Equal(t, created.PriceCents, got.PriceCents, "the amount is kept, not converted")
	assert.Equal(t, created.Name, got.Name)
	assert.Equal(t, "Tostado", *got.Description)
	assert.Equal(t, created.Category, got.Category)
	assert.Equal(t, created.IsActive, got.IsActive)
	assert.Equal(t, created.SKU, got.SKU)
	assert.Equal(t, int32(2), got.Version)
	assert.Equal(t, created.CreatedAt, got.CreatedAt)
	assert.True(t, got.UpdatedAt.After(created.UpdatedAt), "updated_at moves forward")

	stored, err := repo.Get(t.Context(), created.ID)
	require.NoError(t, err)
	assert.Equal(t, got, stored)
}

func TestUpdateEveryField(t *testing.T) {
	repo := newRepo(t)
	created, err := repo.Create(t.Context(), input("ABC-123"))
	require.NoError(t, err)

	got, err := repo.Update(t.Context(), created.ID, domain.ProductPatch{
		Version: 1, Name: ptr("Té verde"), Description: ptr("Hojas"), PriceCents: ptr[int64](500),
		Currency: ptr(domain.CurrencyUSD), Stock: ptr[int64](0), Category: ptr(domain.CategoryOther),
		IsActive: ptr(false),
	})

	require.NoError(t, err)
	assert.Equal(t, "Té verde", got.Name)
	assert.Equal(t, "Hojas", *got.Description)
	assert.Equal(t, int64(500), got.PriceCents)
	assert.Zero(t, got.Stock)
	assert.Equal(t, domain.CategoryOther, got.Category)
	assert.False(t, got.IsActive)
}

func TestUpdateEmptyDescriptionRemovesIt(t *testing.T) {
	repo := newRepo(t)
	in := input("ABC-123")
	in.Description = ptr("Tostado")
	created, err := repo.Create(t.Context(), in)
	require.NoError(t, err)

	got, err := repo.Update(t.Context(), created.ID, domain.ProductPatch{Version: 1, Description: ptr("")})

	require.NoError(t, err)
	assert.Nil(t, got.Description)
}

func TestUpdateVersionConflict(t *testing.T) {
	repo := newRepo(t)
	created, err := repo.Create(t.Context(), input("ABC-123"))
	require.NoError(t, err)
	_, err = repo.Update(t.Context(), created.ID, domain.ProductPatch{Version: 1, Stock: ptr[int64](7)})
	require.NoError(t, err)

	_, err = repo.Update(t.Context(), created.ID, domain.ProductPatch{Version: 1, Stock: ptr[int64](9)})

	require.ErrorIs(t, err, domain.ErrVersionConflict)
	stored, err := repo.Get(t.Context(), created.ID)
	require.NoError(t, err)
	assert.Equal(t, int32(7), stored.Stock, "the row is unchanged")
	assert.Equal(t, int32(2), stored.Version)
}

func TestUpdateNotFound(t *testing.T) {
	repo := newRepo(t)
	created, err := repo.Create(t.Context(), input("ABC-123"))
	require.NoError(t, err)
	_, err = pool.Exec(t.Context(), "UPDATE products SET deleted_at = now() WHERE id = $1", created.ID)
	require.NoError(t, err)

	for name, id := range map[string]string{"missing": "00000000-0000-7000-8000-000000000000", "deleted": created.ID} {
		t.Run(name, func(t *testing.T) {
			_, err := repo.Update(t.Context(), id, domain.ProductPatch{Version: 1, Stock: ptr[int64](1)})

			assert.ErrorIs(t, err, domain.ErrNotFound)
		})
	}
}

func TestUpdateConcurrentlyWithTheSameVersion(t *testing.T) {
	repo := newRepo(t)
	created, err := repo.Create(t.Context(), input("ABC-123"))
	require.NoError(t, err)

	var wg sync.WaitGroup
	errs := make([]error, 2)
	for i := range errs {
		wg.Go(func() {
			_, errs[i] = repo.Update(t.Context(), created.ID, domain.ProductPatch{Version: 1, Stock: ptr(int64(10 + i))})
		})
	}
	wg.Wait()

	succeeded := 0
	for _, err := range errs {
		if err == nil {
			succeeded++
			continue
		}
		assert.ErrorIs(t, err, domain.ErrVersionConflict)
	}
	assert.Equal(t, 1, succeeded, "exactly one update wins")
}
