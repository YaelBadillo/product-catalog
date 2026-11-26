package domain_test

import (
	"errors"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/YaelBadillo/product-catalog/api/internal/domain"
)

func ptr[T any](v T) *T { return &v }

func validInput() domain.ProductInput {
	return domain.ProductInput{
		SKU:        "ABC-123",
		Name:       "Café molido",
		PriceCents: 19990,
		Currency:   domain.CurrencyMXN,
		Stock:      5,
		Category:   domain.CategoryFood,
		IsActive:   ptr(true),
	}
}

// failedFields returns the JSON names of the fields err reports.
func failedFields(t *testing.T, err error) []string {
	t.Helper()
	var verr *domain.ValidationError
	require.ErrorAs(t, err, &verr)
	fields := make([]string, 0, len(verr.Fields))
	for _, f := range verr.Fields {
		assert.NotEmpty(t, f.Message, "field %q needs a message", f.Field)
		fields = append(fields, f.Field)
	}
	return fields
}

func TestProductInputNormalize(t *testing.T) {
	in := domain.ProductInput{
		SKU:         "  abc-123 ",
		Name:        "  Café molido  ",
		Description: ptr("   "),
		PriceCents:  100,
		Stock:       1,
		Category:    domain.CategoryFood,
	}

	got := in.Normalize()

	assert.Equal(t, "ABC-123", got.SKU)
	assert.Equal(t, "Café molido", got.Name)
	assert.Nil(t, got.Description, "a whitespace-only description means none")
	assert.Equal(t, domain.CurrencyMXN, got.Currency, "currency defaults to MXN")
	require.NotNil(t, got.IsActive)
	assert.True(t, *got.IsActive, "status defaults to active")

	kept := domain.ProductInput{Description: ptr("  Tostado  "), Currency: domain.CurrencyUSD, IsActive: ptr(false)}.Normalize()
	assert.Equal(t, "Tostado", *kept.Description)
	assert.Equal(t, domain.CurrencyUSD, kept.Currency)
	assert.False(t, *kept.IsActive)
}

func TestProductInputValidate(t *testing.T) {
	tests := []struct {
		name       string
		mutate     func(*domain.ProductInput)
		wantFields []string
	}{
		{name: "valid", mutate: func(*domain.ProductInput) {}},
		{name: "sku at 4 characters", mutate: func(in *domain.ProductInput) { in.SKU = "AB-1" }},
		{name: "sku at 32 characters", mutate: func(in *domain.ProductInput) { in.SKU = strings.Repeat("A", 32) }},
		{name: "missing sku", mutate: func(in *domain.ProductInput) { in.SKU = "" }, wantFields: []string{"sku"}},
		{name: "short sku", mutate: func(in *domain.ProductInput) { in.SKU = "ABC" }, wantFields: []string{"sku"}},
		{name: "long sku", mutate: func(in *domain.ProductInput) { in.SKU = strings.Repeat("A", 33) }, wantFields: []string{"sku"}},
		{name: "sku with inner space", mutate: func(in *domain.ProductInput) { in.SKU = "AB C1" }, wantFields: []string{"sku"}},
		{name: "sku with Ñ", mutate: func(in *domain.ProductInput) { in.SKU = "AÑO-1" }, wantFields: []string{"sku"}},
		{name: "sku with a symbol", mutate: func(in *domain.ProductInput) { in.SKU = "ABC_123" }, wantFields: []string{"sku"}},
		{name: "name of spaces", mutate: func(in *domain.ProductInput) { in.Name = "   " }, wantFields: []string{"name"}},
		{name: "short name", mutate: func(in *domain.ProductInput) { in.Name = " a " }, wantFields: []string{"name"}},
		{name: "name at 2 characters", mutate: func(in *domain.ProductInput) { in.Name = "Té" }},
		{name: "name at 120 characters plus spaces", mutate: func(in *domain.ProductInput) { in.Name = "  " + strings.Repeat("ñ", 120) + "  " }},
		{name: "long name", mutate: func(in *domain.ProductInput) { in.Name = strings.Repeat("a", 121) }, wantFields: []string{"name"}},
		{name: "description at 2000", mutate: func(in *domain.ProductInput) { in.Description = ptr(strings.Repeat("é", 2000)) }},
		{name: "long description", mutate: func(in *domain.ProductInput) { in.Description = ptr(strings.Repeat("a", 2001)) }, wantFields: []string{"description"}},
		{name: "zero price", mutate: func(in *domain.ProductInput) { in.PriceCents = 0 }, wantFields: []string{"price_cents"}},
		{name: "negative price", mutate: func(in *domain.ProductInput) { in.PriceCents = -1 }, wantFields: []string{"price_cents"}},
		{name: "maximum price", mutate: func(in *domain.ProductInput) { in.PriceCents = 99_999_999_999 }},
		{name: "price above the maximum", mutate: func(in *domain.ProductInput) { in.PriceCents = 100_000_000_000 }, wantFields: []string{"price_cents"}},
		{name: "negative stock", mutate: func(in *domain.ProductInput) { in.Stock = -1 }, wantFields: []string{"stock"}},
		{name: "maximum stock", mutate: func(in *domain.ProductInput) { in.Stock = 2_147_483_647 }},
		{name: "stock above the maximum", mutate: func(in *domain.ProductInput) { in.Stock = 2_147_483_648 }, wantFields: []string{"stock"}},
		{name: "unknown currency", mutate: func(in *domain.ProductInput) { in.Currency = "EUR" }, wantFields: []string{"currency"}},
		{name: "missing category", mutate: func(in *domain.ProductInput) { in.Category = "" }, wantFields: []string{"category"}},
		{name: "unknown category", mutate: func(in *domain.ProductInput) { in.Category = "toys" }, wantFields: []string{"category"}},
		{
			name: "every failing field is listed",
			mutate: func(in *domain.ProductInput) {
				in.Name = "a"
				in.PriceCents = 0
				in.Category = "toys"
			},
			wantFields: []string{"name", "price_cents", "category"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			in := validInput()
			tt.mutate(&in)

			err := in.Normalize().Validate()

			if tt.wantFields == nil {
				assert.NoError(t, err)
				return
			}
			assert.ElementsMatch(t, tt.wantFields, failedFields(t, err))
		})
	}
}

func TestProductPatch(t *testing.T) {
	tests := []struct {
		name       string
		patch      domain.ProductPatch
		wantEmpty  bool
		wantFields []string
	}{
		{name: "one field", patch: domain.ProductPatch{Version: 1, Stock: ptr[int64](7)}},
		{name: "empty description removes it", patch: domain.ProductPatch{Version: 1, Description: ptr("  ")}},
		{name: "only version", patch: domain.ProductPatch{Version: 1}, wantEmpty: true, wantFields: []string{""}},
		{name: "missing version", patch: domain.ProductPatch{Stock: ptr[int64](1)}, wantFields: []string{"version"}},
		{name: "short name", patch: domain.ProductPatch{Version: 1, Name: ptr(" a ")}, wantFields: []string{"name"}},
		{name: "long description", patch: domain.ProductPatch{Version: 1, Description: ptr(strings.Repeat("a", 2001))}, wantFields: []string{"description"}},
		{name: "zero price", patch: domain.ProductPatch{Version: 1, PriceCents: ptr[int64](0)}, wantFields: []string{"price_cents"}},
		{name: "unknown currency", patch: domain.ProductPatch{Version: 1, Currency: ptr(domain.Currency("EUR"))}, wantFields: []string{"currency"}},
		{name: "negative stock", patch: domain.ProductPatch{Version: 1, Stock: ptr[int64](-1)}, wantFields: []string{"stock"}},
		{name: "unknown category", patch: domain.ProductPatch{Version: 1, Category: ptr(domain.Category("toys"))}, wantFields: []string{"category"}},
		{name: "status only", patch: domain.ProductPatch{Version: 1, IsActive: ptr(false)}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			patch := tt.patch.Normalize()

			assert.Equal(t, tt.wantEmpty, patch.IsEmpty())
			err := patch.Validate()
			if tt.wantFields == nil {
				assert.NoError(t, err)
				return
			}
			assert.ElementsMatch(t, tt.wantFields, failedFields(t, err))
		})
	}

	t.Run("normalizes name and description", func(t *testing.T) {
		got := domain.ProductPatch{Version: 1, Name: ptr("  Té  "), Description: ptr("   ")}.Normalize()

		assert.Equal(t, "Té", *got.Name)
		require.NotNil(t, got.Description, "an emptied description stays set, meaning remove")
		assert.Empty(t, *got.Description)
	})
}

func TestEnums(t *testing.T) {
	for _, c := range []domain.Category{"electronics", "home", "food", "clothing", "other"} {
		assert.True(t, c.Valid(), c)
	}
	for _, c := range []domain.Category{"", "toys", "Food"} {
		assert.False(t, c.Valid(), c)
	}
	assert.True(t, domain.CurrencyMXN.Valid())
	assert.True(t, domain.CurrencyUSD.Valid())
	for _, c := range []domain.Currency{"", "EUR", "mxn"} {
		assert.False(t, c.Valid(), c)
	}
}

func TestValidationErrorMessage(t *testing.T) {
	err := error(&domain.ValidationError{Fields: []domain.FieldError{{Field: "name", Message: "is required"}}})

	assert.Contains(t, err.Error(), "name")
	var verr *domain.ValidationError
	assert.True(t, errors.As(err, &verr))
}
