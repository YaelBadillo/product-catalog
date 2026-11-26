package seed_test

import (
	"context"
	"testing"
	"unicode"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/YaelBadillo/product-catalog/api/internal/domain"
	"github.com/YaelBadillo/product-catalog/api/internal/seed"
	"github.com/YaelBadillo/product-catalog/api/internal/service"
)

// memoryRepo keeps products by SKU, refusing duplicates like Postgres.
type memoryRepo struct {
	bySKU map[string]domain.ProductInput
}

func newMemoryRepo() *memoryRepo { return &memoryRepo{bySKU: map[string]domain.ProductInput{}} }

func (r *memoryRepo) Create(_ context.Context, in domain.ProductInput) (domain.Product, error) {
	if _, ok := r.bySKU[in.SKU]; ok {
		return domain.Product{}, domain.ErrSKUAlreadyExists
	}
	r.bySKU[in.SKU] = in
	return domain.Product{SKU: in.SKU}, nil
}

func (r *memoryRepo) Get(context.Context, string) (domain.Product, error) {
	return domain.Product{}, domain.ErrNotFound
}

func (r *memoryRepo) List(context.Context, service.ListParams) (service.ProductPage, error) {
	return service.ProductPage{}, nil
}

func (r *memoryRepo) Update(context.Context, string, domain.ProductPatch) (domain.Product, error) {
	return domain.Product{}, domain.ErrNotFound
}

func (r *memoryRepo) Delete(context.Context, string, int32) error { return domain.ErrNotFound }

func TestRunCreatesTwentyValidProducts(t *testing.T) {
	repo := newMemoryRepo()

	created, err := seed.Run(t.Context(), service.NewProducts(repo))

	require.NoError(t, err, "every sample product passes the domain rules")
	assert.Equal(t, 20, created)
	require.Len(t, repo.bySKU, 20)

	categories := map[domain.Category]bool{}
	currencies := map[domain.Currency]bool{}
	inactive := 0
	var accented, withEnye bool
	for _, p := range repo.bySKU {
		categories[p.Category] = true
		currencies[p.Currency] = true
		if !*p.IsActive {
			inactive++
		}
		for _, r := range p.Name {
			if r == 'ñ' || r == 'Ñ' {
				withEnye = true
			} else if r > unicode.MaxASCII && unicode.IsLetter(r) {
				accented = true
			}
		}
	}
	assert.Len(t, categories, 5, "all five categories")
	assert.Len(t, currencies, 2, "both currencies")
	assert.GreaterOrEqual(t, inactive, 4)
	assert.True(t, accented, "names with accents")
	assert.True(t, withEnye, "names with ñ")
}

func TestRunTwiceSkipsExistingSKUs(t *testing.T) {
	repo := newMemoryRepo()
	svc := service.NewProducts(repo)
	_, err := seed.Run(t.Context(), svc)
	require.NoError(t, err)

	created, err := seed.Run(t.Context(), svc)

	require.NoError(t, err)
	assert.Zero(t, created)
	assert.Len(t, repo.bySKU, 20)
}
