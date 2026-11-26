package service_test

import (
	"context"
	"errors"

	"github.com/YaelBadillo/product-catalog/api/internal/domain"
	"github.com/YaelBadillo/product-catalog/api/internal/service"
)

var errUnexpected = errors.New("unexpected call")

// fakeRepo records calls and answers with its funcs.
type fakeRepo struct {
	created []domain.ProductInput
	create  func(domain.ProductInput) (domain.Product, error)
	get     func(id string) (domain.Product, error)
	list    func(service.ListParams) (service.ProductPage, error)
}

func (f *fakeRepo) List(_ context.Context, p service.ListParams) (service.ProductPage, error) {
	if f.list == nil {
		return service.ProductPage{}, errUnexpected
	}
	return f.list(p)
}

func (f *fakeRepo) Create(_ context.Context, in domain.ProductInput) (domain.Product, error) {
	f.created = append(f.created, in)
	if f.create == nil {
		return domain.Product{}, errUnexpected
	}
	return f.create(in)
}

func (f *fakeRepo) Get(_ context.Context, id string) (domain.Product, error) {
	if f.get == nil {
		return domain.Product{}, errUnexpected
	}
	return f.get(id)
}

func ptr[T any](v T) *T { return &v }
