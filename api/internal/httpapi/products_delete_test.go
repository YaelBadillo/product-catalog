package httpapi_test

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/YaelBadillo/product-catalog/api/internal/domain"
	"github.com/YaelBadillo/product-catalog/api/internal/testutil"
)

func TestDeleteProduct(t *testing.T) {
	var gotID string
	var gotVersion int32
	svc := &fakeService{remove: func(id string, version int32) error {
		gotID, gotVersion = id, version
		return nil
	}}
	req := request(t, http.MethodDelete, productURL+"?version=3", "")

	rec := do(t, productsRouter(t, svc, nil), req)

	require.Equal(t, http.StatusNoContent, rec.Code, rec.Body.String())
	assert.Empty(t, rec.Body.String())
	assert.Equal(t, productID, gotID)
	assert.Equal(t, int32(3), gotVersion)
	testutil.AssertMatchesContract(t, req, rec)
}

func TestDeleteProductBadRequests(t *testing.T) {
	tests := []struct{ name, target, wantField string }{
		{"missing version", productURL, "version"},
		{"non-numeric version", productURL + "?version=abc", "version"},
		{"version zero", productURL + "?version=0", "version"},
		{"invalid id", "/api/v1/products/abc?version=1", "id"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := request(t, http.MethodDelete, tt.target, "")

			rec := do(t, productsRouter(t, &fakeService{}, nil), req)

			assert.Equal(t, http.StatusBadRequest, rec.Code)
			assert.Equal(t, []string{tt.wantField}, detailFields(decodeEnvelope(t, rec)))
			testutil.AssertResponseMatchesContract(t, req, rec)
		})
	}
}

func TestDeleteProductServiceErrors(t *testing.T) {
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
			svc := &fakeService{remove: func(string, int32) error { return tt.err }}
			req := request(t, http.MethodDelete, productURL+"?version=1", "")

			rec := do(t, productsRouter(t, svc, nil), req)

			assert.Equal(t, tt.wantStatus, rec.Code)
			assert.Equal(t, tt.wantCode, decodeEnvelope(t, rec).Error.Code)
			testutil.AssertMatchesContract(t, req, rec)
		})
	}
}
