package service_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/YaelBadillo/product-catalog/api/internal/domain"
	"github.com/YaelBadillo/product-catalog/api/internal/service"
)

func rawInput() domain.ProductInput {
	return domain.ProductInput{
		SKU:         "  abc-123 ",
		Name:        "  Café molido ",
		Description: ptr("   "),
		PriceCents:  19990,
		Stock:       5,
		Category:    domain.CategoryFood,
	}
}

func TestCreateNormalizesAndAppliesDefaults(t *testing.T) {
	repo := &fakeRepo{create: func(in domain.ProductInput) (domain.Product, error) {
		return domain.Product{ID: "id-1", SKU: in.SKU}, nil
	}}
	svc := service.NewProducts(repo)

	got, err := svc.Create(t.Context(), rawInput())

	require.NoError(t, err)
	assert.Equal(t, "id-1", got.ID)
	require.Len(t, repo.created, 1)
	stored := repo.created[0]
	assert.Equal(t, "ABC-123", stored.SKU)
	assert.Equal(t, "Café molido", stored.Name)
	assert.Nil(t, stored.Description)
	assert.Equal(t, domain.CurrencyMXN, stored.Currency)
	require.NotNil(t, stored.IsActive)
	assert.True(t, *stored.IsActive)
}

func TestCreateRejectsInvalidInputWithoutCallingTheRepository(t *testing.T) {
	repo := &fakeRepo{}
	svc := service.NewProducts(repo)
	in := rawInput()
	in.Name = " a "
	in.PriceCents = 0

	_, err := svc.Create(t.Context(), in)

	var verr *domain.ValidationError
	require.ErrorAs(t, err, &verr)
	assert.Len(t, verr.Fields, 2)
	assert.Empty(t, repo.created)
}

func TestCreatePropagatesDuplicateSKU(t *testing.T) {
	repo := &fakeRepo{create: func(domain.ProductInput) (domain.Product, error) {
		return domain.Product{}, domain.ErrSKUAlreadyExists
	}}

	_, err := service.NewProducts(repo).Create(t.Context(), rawInput())

	assert.ErrorIs(t, err, domain.ErrSKUAlreadyExists)
}

func TestGet(t *testing.T) {
	want := domain.Product{ID: "id-1", SKU: "ABC-123"}
	repo := &fakeRepo{get: func(id string) (domain.Product, error) {
		if id == want.ID {
			return want, nil
		}
		return domain.Product{}, domain.ErrNotFound
	}}
	svc := service.NewProducts(repo)

	got, err := svc.Get(t.Context(), "id-1")
	require.NoError(t, err)
	assert.Equal(t, want, got)

	_, err = svc.Get(t.Context(), "id-2")
	assert.ErrorIs(t, err, domain.ErrNotFound)
}
