package repository_test

import (
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/YaelBadillo/product-catalog/api/internal/domain"
)

func ptr[T any](v T) *T { return &v }

// input returns a valid, normalized creation input.
func input(sku string) domain.ProductInput {
	return domain.ProductInput{
		SKU:        sku,
		Name:       "Café molido",
		PriceCents: 19990,
		Currency:   domain.CurrencyMXN,
		Stock:      5,
		Category:   domain.CategoryFood,
		IsActive:   ptr(true),
	}
}

func TestCreate(t *testing.T) {
	repo := newRepo(t)
	in := input("ABC-123")

	got, err := repo.Create(t.Context(), in)

	require.NoError(t, err)
	assert.Regexp(t, `^[0-9a-f-]{36}$`, got.ID)
	assert.Equal(t, "ABC-123", got.SKU)
	assert.Equal(t, "Café molido", got.Name)
	assert.Nil(t, got.Description)
	assert.Equal(t, int64(19990), got.PriceCents)
	assert.Equal(t, domain.CurrencyMXN, got.Currency)
	assert.Equal(t, int32(5), got.Stock)
	assert.Equal(t, domain.CategoryFood, got.Category)
	assert.True(t, got.IsActive)
	assert.Equal(t, int32(1), got.Version)
	assert.WithinDuration(t, time.Now(), got.CreatedAt, time.Minute)
	assert.Equal(t, got.CreatedAt, got.UpdatedAt)

	var isNull bool
	require.NoError(t, pool.QueryRow(t.Context(), "SELECT description IS NULL FROM products WHERE id = $1", got.ID).Scan(&isNull))
	assert.True(t, isNull, "no description is stored as NULL")
}

func TestCreateWithDescription(t *testing.T) {
	repo := newRepo(t)
	in := input("ABC-124")
	in.Description = ptr("Tostado medio")
	in.IsActive = ptr(false)
	in.Currency = domain.CurrencyUSD

	got, err := repo.Create(t.Context(), in)

	require.NoError(t, err)
	require.NotNil(t, got.Description)
	assert.Equal(t, "Tostado medio", *got.Description)
	assert.False(t, got.IsActive)
	assert.Equal(t, domain.CurrencyUSD, got.Currency)
}

func TestCreateDuplicateSKU(t *testing.T) {
	repo := newRepo(t)
	_, err := repo.Create(t.Context(), input("ABC-123"))
	require.NoError(t, err)

	_, err = repo.Create(t.Context(), input("ABC-123"))

	assert.ErrorIs(t, err, domain.ErrSKUAlreadyExists)
}

func TestCreateSameSKUConcurrently(t *testing.T) {
	repo := newRepo(t)
	const attempts = 8

	var wg sync.WaitGroup
	errs := make([]error, attempts)
	for i := range attempts {
		wg.Go(func() {
			_, errs[i] = repo.Create(t.Context(), input("RACE-1"))
		})
	}
	wg.Wait()

	succeeded := 0
	for _, err := range errs {
		if err == nil {
			succeeded++
			continue
		}
		assert.ErrorIs(t, err, domain.ErrSKUAlreadyExists)
	}
	assert.Equal(t, 1, succeeded, "exactly one creation wins")
}

func TestGet(t *testing.T) {
	repo := newRepo(t)
	created, err := repo.Create(t.Context(), input("ABC-123"))
	require.NoError(t, err)

	got, err := repo.Get(t.Context(), created.ID)

	require.NoError(t, err)
	assert.Equal(t, created, got)
}

func TestGetNotFound(t *testing.T) {
	repo := newRepo(t)
	created, err := repo.Create(t.Context(), input("ABC-123"))
	require.NoError(t, err)
	_, err = pool.Exec(t.Context(), "UPDATE products SET deleted_at = now() WHERE id = $1", created.ID)
	require.NoError(t, err)

	for name, id := range map[string]string{
		"unknown id": "00000000-0000-7000-8000-000000000000",
		"deleted":    created.ID,
	} {
		t.Run(name, func(t *testing.T) {
			_, err := repo.Get(t.Context(), id)

			assert.ErrorIs(t, err, domain.ErrNotFound)
		})
	}
}
