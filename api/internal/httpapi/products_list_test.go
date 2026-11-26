package httpapi_test

import (
	"encoding/json"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/YaelBadillo/product-catalog/api/internal/domain"
	"github.com/YaelBadillo/product-catalog/api/internal/service"
	"github.com/YaelBadillo/product-catalog/api/internal/testutil"
)

type listBody struct {
	Data []map[string]any `json:"data"`
	Meta struct {
		Page       int `json:"page"`
		PageSize   int `json:"page_size"`
		Total      int `json:"total"`
		TotalPages int `json:"total_pages"`
	} `json:"meta"`
}

func TestListProducts(t *testing.T) {
	var got service.ListParams
	svc := &fakeService{list: func(p service.ListParams) (service.ProductPage, error) {
		got = p
		return service.ProductPage{Items: []domain.Product{sampleProduct()}, Total: 41}, nil
	}}
	req := request(t, http.MethodGet, "/api/v1/products?page=2&category=food&q=cafe&sort=name&order=asc", "")

	rec := do(t, productsRouter(t, svc, nil), req)

	require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
	var body listBody
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &body))
	assert.Len(t, body.Data, 1)
	assert.Equal(t, 2, body.Meta.Page)
	assert.Equal(t, 20, body.Meta.PageSize)
	assert.Equal(t, 41, body.Meta.Total)
	assert.Equal(t, 3, body.Meta.TotalPages)
	assert.Equal(t, 2, got.Page)
	assert.Equal(t, domain.CategoryFood, *got.Category)
	assert.Equal(t, "cafe", got.Query)
	assert.Equal(t, service.SortName, got.Sort)
	testutil.AssertMatchesContract(t, req, rec)
}

func TestListProductsEmpty(t *testing.T) {
	svc := &fakeService{list: func(service.ListParams) (service.ProductPage, error) {
		return service.ProductPage{}, nil
	}}
	req := request(t, http.MethodGet, "/api/v1/products", "")

	rec := do(t, productsRouter(t, svc, nil), req)

	require.Equal(t, http.StatusOK, rec.Code)
	assert.JSONEq(t, `{"data":[],"meta":{"page":1,"page_size":20,"total":0,"total_pages":0}}`, rec.Body.String())
	testutil.AssertMatchesContract(t, req, rec)
}

func TestListProductsPageBeyondTheLast(t *testing.T) {
	svc := &fakeService{list: func(service.ListParams) (service.ProductPage, error) {
		return service.ProductPage{Items: []domain.Product{}, Total: 12}, nil
	}}
	req := request(t, http.MethodGet, "/api/v1/products?page=999", "")

	rec := do(t, productsRouter(t, svc, nil), req)

	require.Equal(t, http.StatusOK, rec.Code)
	var body listBody
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &body))
	assert.Empty(t, body.Data)
	assert.Equal(t, 12, body.Meta.Total)
	assert.Equal(t, 1, body.Meta.TotalPages)
	testutil.AssertMatchesContract(t, req, rec)
}

func TestListProductsInvalidParams(t *testing.T) {
	tests := []struct {
		target    string
		wantField string
	}{
		{"/api/v1/products?min_price=100", "min_price"},
		{"/api/v1/products?currency=MXN&min_price=500&max_price=100", "min_price"},
		{"/api/v1/products?sort=color", "sort"},
	}
	for _, tt := range tests {
		t.Run(tt.target, func(t *testing.T) {
			svc := &fakeService{}
			req := request(t, http.MethodGet, tt.target, "")

			rec := do(t, productsRouter(t, svc, nil), req)

			assert.Equal(t, http.StatusBadRequest, rec.Code)
			assert.Equal(t, []string{tt.wantField}, detailFields(decodeEnvelope(t, rec)))
			testutil.AssertResponseMatchesContract(t, req, rec)
		})
	}
}
