package domain

import (
	"errors"
	"math"
	"regexp"
	"strings"
	"time"
	"unicode/utf8"
)

// Category is the product category.
type Category string

// Categories.
const (
	CategoryElectronics Category = "electronics"
	CategoryHome        Category = "home"
	CategoryFood        Category = "food"
	CategoryClothing    Category = "clothing"
	CategoryOther       Category = "other"
)

// Valid reports whether c is a known category.
func (c Category) Valid() bool {
	switch c {
	case CategoryElectronics, CategoryHome, CategoryFood, CategoryClothing, CategoryOther:
		return true
	default:
		return false
	}
}

// Currency is the ISO 4217 code of a price.
type Currency string

// Currencies.
const (
	CurrencyMXN Currency = "MXN"
	CurrencyUSD Currency = "USD"
)

// Valid reports whether c is a supported currency.
func (c Currency) Valid() bool {
	return c == CurrencyMXN || c == CurrencyUSD
}

// Bounds of the product rules.
const (
	NameMinLen        = 2
	NameMaxLen        = 120
	DescriptionMaxLen = 2000
	MaxPriceCents     = 99_999_999_999
	MaxStock          = math.MaxInt32
)

// Rule messages, in English: the API is for developers.
const (
	msgSKU         = "must be 4 to 32 characters: uppercase letters A-Z, digits and hyphens"
	msgName        = "must be 2 to 120 characters"
	msgDescription = "must be at most 2000 characters"
	msgPrice       = "must be between 1 and 99999999999 cents"
	msgCurrency    = "must be MXN or USD"
	msgStock       = "must be a whole number between 0 and 2147483647"
	msgCategory    = "must be one of electronics, home, food, clothing, other"
	msgVersion     = "must be at least 1"
	msgEmptyPatch  = "at least one field besides version must be present"
)

// Product is a catalog product.
type Product struct {
	ID          string
	SKU         string
	Name        string
	Description *string
	PriceCents  int64
	Currency    Currency
	Stock       int32
	Category    Category
	IsActive    bool
	Version     int32
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

var skuPattern = regexp.MustCompile(`^[A-Z0-9-]{4,32}$`)

var errInvalidSKU = errors.New(msgSKU)

// NormalizeSKU trims surrounding whitespace and uppercases.
func NormalizeSKU(sku string) string {
	return strings.ToUpper(strings.TrimSpace(sku))
}

// ValidateSKU checks a normalized SKU.
func ValidateSKU(sku string) error {
	if !skuPattern.MatchString(sku) {
		return errInvalidSKU
	}
	return nil
}

// NormalizeName trims surrounding whitespace.
func NormalizeName(name string) string {
	return strings.TrimSpace(name)
}

// NormalizeDescription trims surrounding whitespace; empty means none (nil).
func NormalizeDescription(description *string) *string {
	if description == nil {
		return nil
	}
	trimmed := strings.TrimSpace(*description)
	if trimmed == "" {
		return nil
	}
	return &trimmed
}

// ProductInput is the data to create a product.
type ProductInput struct {
	SKU         string
	Name        string
	Description *string
	PriceCents  int64
	Currency    Currency
	Stock       int64
	Category    Category
	IsActive    *bool
}

// Normalize returns in with whitespace trimmed, the SKU uppercased and the
// defaults applied (currency MXN, active).
func (in ProductInput) Normalize() ProductInput {
	in.SKU = NormalizeSKU(in.SKU)
	in.Name = NormalizeName(in.Name)
	in.Description = NormalizeDescription(in.Description)
	if in.Currency == "" {
		in.Currency = CurrencyMXN
	}
	if in.IsActive == nil {
		active := true
		in.IsActive = &active
	}
	return in
}

// Validate checks every rule on a normalized input and reports every
// failing field.
func (in ProductInput) Validate() error {
	var errs fieldErrors
	if ValidateSKU(in.SKU) != nil {
		errs.add("sku", msgSKU)
	}
	validateName(&errs, in.Name)
	validateDescription(&errs, in.Description)
	validatePrice(&errs, in.PriceCents)
	validateCurrency(&errs, in.Currency)
	validateStock(&errs, in.Stock)
	validateCategory(&errs, in.Category)
	return errs.err()
}

// ProductPatch is a partial update. Nil fields stay as they are. An empty
// Description (after normalization) removes it.
type ProductPatch struct {
	Name        *string
	Description *string
	PriceCents  *int64
	Currency    *Currency
	Stock       *int64
	Category    *Category
	IsActive    *bool
	Version     int32
}

// Normalize trims the name and the description. An emptied description stays
// set (to ""), which means "remove it".
func (p ProductPatch) Normalize() ProductPatch {
	if p.Name != nil {
		name := NormalizeName(*p.Name)
		p.Name = &name
	}
	if p.Description != nil {
		description := strings.TrimSpace(*p.Description)
		p.Description = &description
	}
	return p
}

// IsEmpty reports whether the patch changes nothing.
func (p ProductPatch) IsEmpty() bool {
	return p.Name == nil && p.Description == nil && p.PriceCents == nil &&
		p.Currency == nil && p.Stock == nil && p.Category == nil && p.IsActive == nil
}

// Validate checks the version, that something changes, and the rules of
// every present field.
func (p ProductPatch) Validate() error {
	var errs fieldErrors
	if p.Version < 1 {
		errs.add("version", msgVersion)
	}
	if p.IsEmpty() {
		errs.add("", msgEmptyPatch)
	}
	if p.Name != nil {
		validateName(&errs, *p.Name)
	}
	if p.Description != nil && *p.Description != "" {
		validateDescription(&errs, p.Description)
	}
	if p.PriceCents != nil {
		validatePrice(&errs, *p.PriceCents)
	}
	if p.Currency != nil {
		validateCurrency(&errs, *p.Currency)
	}
	if p.Stock != nil {
		validateStock(&errs, *p.Stock)
	}
	if p.Category != nil {
		validateCategory(&errs, *p.Category)
	}
	return errs.err()
}

// RuleMessage returns the rule message of a field, by its JSON name, or ""
// for an unknown field. The handler's validators reuse these messages.
func RuleMessage(field string) string {
	return ruleMessages[field]
}

var ruleMessages = map[string]string{
	"sku":         msgSKU,
	"name":        msgName,
	"description": msgDescription,
	"price_cents": msgPrice,
	"currency":    msgCurrency,
	"stock":       msgStock,
	"category":    msgCategory,
	"version":     msgVersion,
}

func validateName(errs *fieldErrors, name string) {
	if n := utf8.RuneCountInString(name); n < NameMinLen || n > NameMaxLen {
		errs.add("name", msgName)
	}
}

func validateDescription(errs *fieldErrors, description *string) {
	if description != nil && utf8.RuneCountInString(*description) > DescriptionMaxLen {
		errs.add("description", msgDescription)
	}
}

func validatePrice(errs *fieldErrors, cents int64) {
	if cents < 1 || cents > MaxPriceCents {
		errs.add("price_cents", msgPrice)
	}
}

func validateCurrency(errs *fieldErrors, currency Currency) {
	if !currency.Valid() {
		errs.add("currency", msgCurrency)
	}
}

func validateStock(errs *fieldErrors, stock int64) {
	if stock < 0 || stock > MaxStock {
		errs.add("stock", msgStock)
	}
}

func validateCategory(errs *fieldErrors, category Category) {
	if !category.Valid() {
		errs.add("category", msgCategory)
	}
}
