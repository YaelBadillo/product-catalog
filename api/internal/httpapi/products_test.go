package httpapi_test

import (
	"bytes"
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/YaelBadillo/product-catalog/api/internal/domain"
	"github.com/YaelBadillo/product-catalog/api/internal/testutil"
)

const validCreateBody = `{"sku":"  abc-123 ","name":" Café molido ","price_cents":19990,"stock":5,"category":"food"}`

func detailFields(body envelope) []string {
	fields := []string{}
	for _, d := range body.Error.Details {
		fields = append(fields, d.Field)
	}
	return fields
}

func TestCreateProduct(t *testing.T) {
	svc := &fakeService{create: func(domain.ProductInput) (domain.Product, error) { return sampleProduct(), nil }}
	router := productsRouter(t, svc, nil)
	req := request(t, http.MethodPost, "/api/v1/products", validCreateBody)

	rec := do(t, router, req)

	require.Equal(t, http.StatusCreated, rec.Code, rec.Body.String())
	assert.Equal(t, "/api/v1/products/"+productID, rec.Header().Get("Location"))
	assert.JSONEq(t, `{
		"id": "`+productID+`", "sku": "ABC-123", "name": "Café molido", "description": null,
		"price_cents": 19990, "currency": "MXN", "stock": 5, "category": "food",
		"is_active": true, "version": 1,
		"created_at": "2026-11-25T20:30:00Z", "updated_at": "2026-11-25T20:30:00Z"
	}`, rec.Body.String())
	testutil.AssertMatchesContract(t, req, rec)

	require.Len(t, svc.createdWith, 1)
	in := svc.createdWith[0]
	assert.Equal(t, "  abc-123 ", in.SKU, "the service normalizes, not the handler")
	assert.Equal(t, int64(19990), in.PriceCents)
	assert.Equal(t, int64(5), in.Stock)
	assert.Equal(t, domain.CategoryFood, in.Category)
	assert.Empty(t, in.Currency, "defaults are the service's job")
	assert.Nil(t, in.IsActive)
}

func TestCreateProductPassesOptionalFields(t *testing.T) {
	svc := &fakeService{create: func(domain.ProductInput) (domain.Product, error) { return sampleProduct(), nil }}
	router := productsRouter(t, svc, nil)
	req := request(t, http.MethodPost, "/api/v1/products",
		`{"sku":"ABC-123","name":"Café","description":"Tostado","price_cents":1,"currency":"USD","stock":0,"category":"food","is_active":false}`)

	rec := do(t, router, req)

	require.Equal(t, http.StatusCreated, rec.Code, rec.Body.String())
	in := svc.createdWith[0]
	assert.Equal(t, "Tostado", *in.Description)
	assert.Equal(t, domain.CurrencyUSD, in.Currency)
	assert.False(t, *in.IsActive)
	assert.Zero(t, in.Stock)
}

func TestCreateProductDuplicateSKU(t *testing.T) {
	svc := &fakeService{create: func(domain.ProductInput) (domain.Product, error) {
		return domain.Product{}, domain.ErrSKUAlreadyExists
	}}
	req := request(t, http.MethodPost, "/api/v1/products", validCreateBody)

	rec := do(t, productsRouter(t, svc, nil), req)

	assert.Equal(t, http.StatusConflict, rec.Code)
	assert.Equal(t, "SKU_ALREADY_EXISTS", decodeEnvelope(t, rec).Error.Code)
	testutil.AssertMatchesContract(t, req, rec)
}

func TestCreateProductRuleViolations(t *testing.T) {
	svc := &fakeService{}
	req := request(t, http.MethodPost, "/api/v1/products",
		`{"sku":"ABC-123","name":" a ","price_cents":0,"stock":1,"category":"toys"}`)

	rec := do(t, productsRouter(t, svc, nil), req)

	assert.Equal(t, http.StatusUnprocessableEntity, rec.Code)
	body := decodeEnvelope(t, rec)
	assert.Equal(t, "VALIDATION_ERROR", body.Error.Code)
	assert.ElementsMatch(t, []string{"name", "price_cents", "category"}, detailFields(body))
	assert.Empty(t, svc.createdWith, "the service is not called")
	testutil.AssertResponseMatchesContract(t, req, rec)
}

func TestCreateProductServiceValidationError(t *testing.T) {
	svc := &fakeService{create: func(domain.ProductInput) (domain.Product, error) {
		return domain.Product{}, &domain.ValidationError{Fields: []domain.FieldError{{Field: "sku", Message: "invalid"}}}
	}}
	req := request(t, http.MethodPost, "/api/v1/products", validCreateBody)

	rec := do(t, productsRouter(t, svc, nil), req)

	assert.Equal(t, http.StatusUnprocessableEntity, rec.Code)
	assert.Equal(t, []string{"sku"}, detailFields(decodeEnvelope(t, rec)))
	testutil.AssertMatchesContract(t, req, rec)
}

func TestCreateProductMalformedRequests(t *testing.T) {
	tests := []struct {
		name      string
		body      string
		wantField string
	}{
		{name: "unknown field", body: `{"sku":"ABC-123","name":"Café","price_cents":1,"stock":1,"category":"food","color":"red"}`, wantField: "color"},
		{name: "malformed json", body: `{"sku":`, wantField: ""},
		{name: "wrong type", body: `{"sku":"ABC-123","name":"Café","price_cents":"1","stock":1,"category":"food"}`, wantField: "price_cents"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc := &fakeService{}
			req := request(t, http.MethodPost, "/api/v1/products", tt.body)

			rec := do(t, productsRouter(t, svc, nil), req)

			assert.Equal(t, http.StatusBadRequest, rec.Code)
			body := decodeEnvelope(t, rec)
			assert.Equal(t, "VALIDATION_ERROR", body.Error.Code)
			assert.Equal(t, []string{tt.wantField}, detailFields(body))
			assert.Empty(t, svc.createdWith)
			testutil.AssertResponseMatchesContract(t, req, rec)
		})
	}
}

func TestGetProduct(t *testing.T) {
	description := "Tostado medio"
	svc := &fakeService{get: func(id string) (domain.Product, error) {
		if id != productID {
			return domain.Product{}, domain.ErrNotFound
		}
		p := sampleProduct()
		p.Description = &description
		return p, nil
	}}
	router := productsRouter(t, svc, nil)

	t.Run("found", func(t *testing.T) {
		req := request(t, http.MethodGet, "/api/v1/products/"+productID, "")

		rec := do(t, router, req)

		require.Equal(t, http.StatusOK, rec.Code)
		var got map[string]any
		require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &got))
		assert.Equal(t, description, got["description"])
		testutil.AssertMatchesContract(t, req, rec)
	})

	t.Run("not found", func(t *testing.T) {
		req := request(t, http.MethodGet, "/api/v1/products/00000000-0000-7000-8000-000000000000", "")

		rec := do(t, router, req)

		assert.Equal(t, http.StatusNotFound, rec.Code)
		assert.Equal(t, "NOT_FOUND", decodeEnvelope(t, rec).Error.Code)
		testutil.AssertMatchesContract(t, req, rec)
	})

	t.Run("invalid id", func(t *testing.T) {
		req := request(t, http.MethodGet, "/api/v1/products/abc", "")

		rec := do(t, router, req)

		assert.Equal(t, http.StatusBadRequest, rec.Code)
		assert.Equal(t, []string{"id"}, detailFields(decodeEnvelope(t, rec)))
		testutil.AssertResponseMatchesContract(t, req, rec)
	})
}

func TestProductRoutesPlatformResponses(t *testing.T) {
	t.Run("body over 64 KiB", func(t *testing.T) {
		svc := &fakeService{}
		body := `{"sku":"ABC-123","name":"` + strings.Repeat("a", 64<<10) + `","price_cents":1,"stock":1,"category":"food"}`
		req := request(t, http.MethodPost, "/api/v1/products", body)

		rec := do(t, productsRouter(t, svc, nil), req)

		assert.Equal(t, http.StatusRequestEntityTooLarge, rec.Code)
		assert.Equal(t, "PAYLOAD_TOO_LARGE", decodeEnvelope(t, rec).Error.Code)
		testutil.AssertResponseMatchesContract(t, req, rec)
	})

	t.Run("rate limited at request 301", func(t *testing.T) {
		svc := &fakeService{get: func(string) (domain.Product, error) { return sampleProduct(), nil }}
		router := productsRouter(t, svc, nil)
		for i := range 300 {
			require.Equal(t, http.StatusOK, do(t, router, request(t, http.MethodGet, "/api/v1/products/"+productID, "")).Code, "request %d", i+1)
		}
		req := request(t, http.MethodGet, "/api/v1/products/"+productID, "")

		rec := do(t, router, req)

		assert.Equal(t, http.StatusTooManyRequests, rec.Code)
		assert.NotEmpty(t, rec.Header().Get("Retry-After"))
		assert.Equal(t, "RATE_LIMITED", decodeEnvelope(t, rec).Error.Code)
		testutil.AssertMatchesContract(t, req, rec)
	})

	t.Run("unexpected error", func(t *testing.T) {
		var logs bytes.Buffer
		svc := &fakeService{get: func(string) (domain.Product, error) {
			return domain.Product{}, errors.New("connection reset by peer")
		}}
		req := request(t, http.MethodGet, "/api/v1/products/"+productID, "")

		rec := do(t, productsRouter(t, svc, &logs), req)

		assert.Equal(t, http.StatusInternalServerError, rec.Code)
		body := decodeEnvelope(t, rec)
		assert.Equal(t, "INTERNAL_ERROR", body.Error.Code)
		assert.Contains(t, body.Error.Message, rec.Header().Get("X-Request-ID"))
		assert.NotContains(t, rec.Body.String(), "connection reset")
		assert.Contains(t, logs.String(), "connection reset")
		testutil.AssertMatchesContract(t, req, rec)
	})
}
