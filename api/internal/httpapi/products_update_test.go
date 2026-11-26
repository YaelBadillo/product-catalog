package httpapi_test

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/YaelBadillo/product-catalog/api/internal/domain"
	"github.com/YaelBadillo/product-catalog/api/internal/testutil"
)

const productURL = "/api/v1/products/" + productID

func TestUpdateProduct(t *testing.T) {
	var gotID string
	var gotPatch domain.ProductPatch
	svc := &fakeService{update: func(id string, p domain.ProductPatch) (domain.Product, error) {
		gotID, gotPatch = id, p
		product := sampleProduct()
		product.Stock = 7
		product.Version = 2
		return product, nil
	}}
	req := request(t, http.MethodPatch, productURL, `{"version":1,"stock":7,"description":"  ","is_active":false}`)

	rec := do(t, productsRouter(t, svc, nil), req)

	require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
	assert.Contains(t, rec.Body.String(), `"version":2`)
	testutil.AssertMatchesContract(t, req, rec)
	assert.Equal(t, productID, gotID)
	assert.Equal(t, int32(1), gotPatch.Version)
	assert.Equal(t, int64(7), *gotPatch.Stock)
	assert.Equal(t, "  ", *gotPatch.Description, "the service normalizes")
	assert.False(t, *gotPatch.IsActive)
	assert.Nil(t, gotPatch.Name)
	assert.Nil(t, gotPatch.PriceCents)
}

func TestUpdateProductBadRequests(t *testing.T) {
	tests := []struct {
		name, target, body, wantField string
	}{
		{"sku cannot change", productURL, `{"version":2,"sku":"NEW-1"}`, "sku"},
		{"malformed json", productURL, `{"version":`, ""},
		{"invalid id", "/api/v1/products/abc", `{"version":1,"stock":1}`, "id"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := request(t, http.MethodPatch, tt.target, tt.body)

			rec := do(t, productsRouter(t, &fakeService{}, nil), req)

			assert.Equal(t, http.StatusBadRequest, rec.Code)
			assert.Equal(t, []string{tt.wantField}, detailFields(decodeEnvelope(t, rec)))
			testutil.AssertResponseMatchesContract(t, req, rec)
		})
	}
}

func TestUpdateProductRuleViolations(t *testing.T) {
	tests := []struct {
		name, body string
		wantFields []string
	}{
		{"missing version", `{"stock":1}`, []string{"version"}},
		{"invalid fields", `{"version":1,"name":" a ","price_cents":0,"category":"toys"}`, []string{"name", "price_cents", "category"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := request(t, http.MethodPatch, productURL, tt.body)

			rec := do(t, productsRouter(t, &fakeService{}, nil), req)

			assert.Equal(t, http.StatusUnprocessableEntity, rec.Code)
			assert.ElementsMatch(t, tt.wantFields, detailFields(decodeEnvelope(t, rec)))
			testutil.AssertResponseMatchesContract(t, req, rec)
		})
	}

	t.Run("version alone", func(t *testing.T) {
		svc := &fakeService{update: func(string, domain.ProductPatch) (domain.Product, error) {
			return domain.Product{}, &domain.ValidationError{Fields: []domain.FieldError{{Field: "", Message: "no changes"}}}
		}}
		req := request(t, http.MethodPatch, productURL, `{"version":2}`)

		rec := do(t, productsRouter(t, svc, nil), req)

		assert.Equal(t, http.StatusUnprocessableEntity, rec.Code)
		testutil.AssertResponseMatchesContract(t, req, rec)
	})
}

func TestUpdateProductServiceErrors(t *testing.T) {
	tests := []struct {
		err        error
		wantStatus int
		wantCode   string
	}{
		{domain.ErrNotFound, http.StatusNotFound, "NOT_FOUND"},
		{domain.ErrVersionConflict, http.StatusConflict, "VERSION_CONFLICT"},
	}
	for _, tt := range tests {
		t.Run(tt.wantCode, func(t *testing.T) {
			svc := &fakeService{update: func(string, domain.ProductPatch) (domain.Product, error) {
				return domain.Product{}, tt.err
			}}
			req := request(t, http.MethodPatch, productURL, `{"version":1,"stock":7}`)

			rec := do(t, productsRouter(t, svc, nil), req)

			assert.Equal(t, tt.wantStatus, rec.Code)
			assert.Equal(t, tt.wantCode, decodeEnvelope(t, rec).Error.Code)
			testutil.AssertMatchesContract(t, req, rec)
		})
	}
}
