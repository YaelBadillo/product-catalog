package service_test

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/YaelBadillo/product-catalog/api/internal/domain"
	"github.com/YaelBadillo/product-catalog/api/internal/service"
)

func TestList(t *testing.T) {
	want := service.ProductPage{Items: []domain.Product{{ID: "id-1"}}, Total: 41}
	var got service.ListParams
	repo := &fakeRepo{list: func(p service.ListParams) (service.ProductPage, error) {
		got = p
		return want, nil
	}}
	in := service.ListParams{
		Page: 3, PageSize: 20, Category: ptr(domain.CategoryFood), Query: "cafe",
		Sort: service.SortPrice, Order: service.OrderAsc,
	}

	page, err := service.NewProducts(repo).List(t.Context(), in)

	require.NoError(t, err)
	assert.Equal(t, want, page)
	assert.Equal(t, in, got)
}

func TestListPropagatesErrors(t *testing.T) {
	boom := errors.New("boom")
	repo := &fakeRepo{list: func(service.ListParams) (service.ProductPage, error) {
		return service.ProductPage{}, boom
	}}

	_, err := service.NewProducts(repo).List(t.Context(), service.ListParams{})

	assert.ErrorIs(t, err, boom)
}
