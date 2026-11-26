package httpapi_test

import (
	"bytes"
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/YaelBadillo/product-catalog/api/internal/domain"
	"github.com/YaelBadillo/product-catalog/api/internal/httpapi/handler"
	"github.com/YaelBadillo/product-catalog/api/internal/service"
)

var errNotStubbed = errors.New("fake service: call not stubbed")

// fakeService answers with its funcs and records what it received.
type fakeService struct {
	create func(domain.ProductInput) (domain.Product, error)
	get    func(id string) (domain.Product, error)
	list   func(service.ListParams) (service.ProductPage, error)
	update func(id string, p domain.ProductPatch) (domain.Product, error)

	createdWith []domain.ProductInput
}

func (f *fakeService) Create(_ context.Context, in domain.ProductInput) (domain.Product, error) {
	f.createdWith = append(f.createdWith, in)
	if f.create == nil {
		return domain.Product{}, errNotStubbed
	}
	return f.create(in)
}

func (f *fakeService) Get(_ context.Context, id string) (domain.Product, error) {
	if f.get == nil {
		return domain.Product{}, errNotStubbed
	}
	return f.get(id)
}

func (f *fakeService) List(_ context.Context, p service.ListParams) (service.ProductPage, error) {
	if f.list == nil {
		return service.ProductPage{}, errNotStubbed
	}
	return f.list(p)
}

func (f *fakeService) Update(_ context.Context, id string, p domain.ProductPatch) (domain.Product, error) {
	if f.update == nil {
		return domain.Product{}, errNotStubbed
	}
	return f.update(id, p)
}

const productID = "0193a5b0-7c1e-7000-8000-000000000001"

func sampleProduct() domain.Product {
	at := time.Date(2026, 11, 25, 20, 30, 0, 0, time.UTC)
	return domain.Product{
		ID:         productID,
		SKU:        "ABC-123",
		Name:       "Café molido",
		PriceCents: 19990,
		Currency:   domain.CurrencyMXN,
		Stock:      5,
		Category:   domain.CategoryFood,
		IsActive:   true,
		Version:    1,
		CreatedAt:  at,
		UpdatedAt:  at,
	}
}

// productsRouter is the real router with the products routes on svc.
func productsRouter(t *testing.T, svc *fakeService, logs *bytes.Buffer) *gin.Engine {
	t.Helper()
	if logs == nil {
		logs = &bytes.Buffer{}
	}
	h := handler.NewProducts(svc, handler.NewValidator())
	return newRouterWith(t, logs, "", h.Register)
}

func request(t *testing.T, method, target, body string) *http.Request {
	t.Helper()
	req := httptest.NewRequestWithContext(t.Context(), method, target, strings.NewReader(body))
	// Lets the contract assertion validate a body the handler already read.
	req.GetBody = func() (io.ReadCloser, error) { return io.NopCloser(strings.NewReader(body)), nil }
	if body != "" {
		req.Header.Set("Content-Type", "application/json")
	}
	return req
}
