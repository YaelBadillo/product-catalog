package handler_test

import (
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/YaelBadillo/product-catalog/api/internal/domain"
	"github.com/YaelBadillo/product-catalog/api/internal/httpapi/handler"
)

type sample struct {
	SKU         string  `json:"sku" validate:"required,sku"`
	Name        string  `json:"name" validate:"required,trimmed_len=2-120"`
	Description *string `json:"description" validate:"omitempty,trimmed_len=0-2000"`
	PriceCents  *int64  `json:"price_cents" validate:"required,min=1"`
}

func decode(t *testing.T, body io.Reader) (sample, error) {
	t.Helper()
	rec := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rec)
	c.Request = httptest.NewRequestWithContext(t.Context(), http.MethodPost, "/", body)
	var got sample
	err := handler.DecodeJSON(c, &got, handler.NewValidator())
	return got, err
}

func requestError(t *testing.T, err error) *handler.RequestError {
	t.Helper()
	var reqErr *handler.RequestError
	require.ErrorAs(t, err, &reqErr)
	assert.NotEmpty(t, reqErr.Message)
	return reqErr
}

func TestDecodeJSONValid(t *testing.T) {
	name := "  " + strings.Repeat("ñ", 120) + "  "
	got, err := decode(t, strings.NewReader(`{"sku":"  abc-123 ","name":"`+name+`","price_cents":1}`))

	require.NoError(t, err, "sku is checked after normalizing and name after trimming")
	assert.Equal(t, "  abc-123 ", got.SKU, "decoding does not normalize: the service does")
}

func TestDecodeJSONRequestErrors(t *testing.T) {
	tests := []struct {
		name      string
		body      string
		wantField string
	}{
		{name: "unknown field", body: `{"sku":"ABC-1","name":"ab","price_cents":1,"color":"red"}`, wantField: "color"},
		{name: "malformed json", body: `{"sku":`, wantField: ""},
		{name: "syntax error", body: `{"sku" "x"}`, wantField: ""},
		{name: "wrong type", body: `{"sku":"ABC-1","name":"ab","price_cents":"12"}`, wantField: "price_cents"},
		{name: "fractional integer", body: `{"sku":"ABC-1","name":"ab","price_cents":1.5}`, wantField: "price_cents"},
		{name: "trailing data", body: `{"sku":"ABC-1","name":"ab","price_cents":1} {}`, wantField: ""},
		{name: "empty body", body: ``, wantField: ""},
		{name: "not an object", body: `[]`, wantField: ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := decode(t, strings.NewReader(tt.body))

			assert.Equal(t, tt.wantField, requestError(t, err).Field)
		})
	}
}

func TestDecodeJSONPassesMaxBytesErrorThrough(t *testing.T) {
	rec := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rec)
	body := `{"sku":"ABC-1","name":"` + strings.Repeat("a", 100) + `","price_cents":1}`
	c.Request = httptest.NewRequestWithContext(t.Context(), http.MethodPost, "/", strings.NewReader(body))
	c.Request.Body = http.MaxBytesReader(rec, c.Request.Body, 10)

	err := handler.DecodeJSON(c, &sample{}, handler.NewValidator())

	var maxBytes *http.MaxBytesError
	require.ErrorAs(t, err, &maxBytes)
	var reqErr *handler.RequestError
	assert.False(t, errors.As(err, &reqErr))
}

func TestDecodeJSONRuleFailures(t *testing.T) {
	_, err := decode(t, strings.NewReader(`{"sku":"ab c","name":" a ","description":"`+strings.Repeat("d", 2001)+`","price_cents":0}`))

	var verr *domain.ValidationError
	require.ErrorAs(t, err, &verr)
	fields := map[string]string{}
	for _, f := range verr.Fields {
		fields[f.Field] = f.Message
	}
	assert.Len(t, fields, 4)
	for _, field := range []string{"sku", "name", "description", "price_cents"} {
		assert.NotEmpty(t, fields[field], "JSON name %q", field)
	}
}

func TestDecodeJSONRequiredFields(t *testing.T) {
	_, err := decode(t, strings.NewReader(`{}`))

	var verr *domain.ValidationError
	require.ErrorAs(t, err, &verr)
	var fields []string
	for _, f := range verr.Fields {
		fields = append(fields, f.Field)
	}
	assert.ElementsMatch(t, []string{"sku", "name", "price_cents"}, fields)
}
