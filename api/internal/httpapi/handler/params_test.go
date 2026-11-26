package handler_test

import (
	"net/url"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/YaelBadillo/product-catalog/api/internal/domain"
	"github.com/YaelBadillo/product-catalog/api/internal/httpapi/handler"
	"github.com/YaelBadillo/product-catalog/api/internal/service"
)

func ptr[T any](v T) *T { return &v }

func TestParseListParamsDefaults(t *testing.T) {
	got, err := handler.ParseListParams(url.Values{"unknown": {"x"}})

	require.NoError(t, err)
	assert.Equal(t, service.ListParams{
		Page: 1, PageSize: 20, Sort: service.SortCreatedAt, Order: service.OrderDesc,
	}, got)
}

func TestParseListParamsValues(t *testing.T) {
	got, err := handler.ParseListParams(url.Values{
		"page": {"3"}, "page_size": {"100"}, "category": {"food"}, "is_active": {"false"},
		"currency": {"USD"}, "min_price": {"0"}, "max_price": {"99999999999"},
		"q": {"  café  "}, "sort": {"price"}, "order": {"asc"},
	})

	require.NoError(t, err)
	assert.Equal(t, service.ListParams{
		Page: 3, PageSize: 100, Category: ptr(domain.CategoryFood), IsActive: ptr(false),
		Currency: ptr(domain.CurrencyUSD), MinPriceCents: ptr[int64](0), MaxPriceCents: ptr[int64](99_999_999_999),
		Query: "café", Sort: service.SortPrice, Order: service.OrderAsc,
	}, got)
}

func TestParseListParamsAcceptsEqualBoundsAndA100CharacterQuery(t *testing.T) {
	got, err := handler.ParseListParams(url.Values{
		"currency": {"MXN"}, "min_price": {"500"}, "max_price": {"500"}, "q": {strings.Repeat("ñ", 100)},
	})

	require.NoError(t, err)
	assert.Equal(t, int64(500), *got.MinPriceCents)
	assert.Equal(t, strings.Repeat("ñ", 100), got.Query)
}

func TestParseListParamsErrors(t *testing.T) {
	tests := []struct {
		name      string
		query     url.Values
		wantField string
	}{
		{"page zero", url.Values{"page": {"0"}}, "page"},
		{"page not a number", url.Values{"page": {"abc"}}, "page"},
		{"page size zero", url.Values{"page_size": {"0"}}, "page_size"},
		{"page size over 100", url.Values{"page_size": {"101"}}, "page_size"},
		{"unknown category", url.Values{"category": {"toys"}}, "category"},
		{"is_active not a boolean", url.Values{"is_active": {"yes"}}, "is_active"},
		{"unknown currency", url.Values{"currency": {"EUR"}}, "currency"},
		{"negative min price", url.Values{"currency": {"MXN"}, "min_price": {"-1"}}, "min_price"},
		{"min price not a whole number", url.Values{"currency": {"MXN"}, "min_price": {"1.5"}}, "min_price"},
		{"max price over the maximum", url.Values{"currency": {"MXN"}, "max_price": {"100000000000"}}, "max_price"},
		{"min price without currency", url.Values{"min_price": {"100"}}, "min_price"},
		{"max price without currency", url.Values{"max_price": {"100"}}, "max_price"},
		{"min over max", url.Values{"currency": {"MXN"}, "min_price": {"500"}, "max_price": {"100"}}, "min_price"},
		{"query over 100 characters", url.Values{"q": {" " + strings.Repeat("a", 100)}}, "q"},
		{"unknown sort", url.Values{"sort": {"color"}}, "sort"},
		{"unknown order", url.Values{"order": {"up"}}, "order"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := handler.ParseListParams(tt.query)

			assert.Equal(t, tt.wantField, requestError(t, err).Field)
		})
	}
}
