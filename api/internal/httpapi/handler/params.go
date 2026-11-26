package handler

import (
	"net/url"
	"strconv"
	"strings"
	"unicode/utf8"

	"github.com/YaelBadillo/product-catalog/api/internal/domain"
	"github.com/YaelBadillo/product-catalog/api/internal/service"
)

const (
	defaultPage     = 1
	defaultPageSize = 20
	maxPageSize     = 100
	maxQueryLen     = 100
)

// ParseListParams reads the catalog's query parameters. It returns a
// *RequestError naming the first invalid parameter. Unknown parameters are
// ignored and empty values count as absent.
func ParseListParams(query url.Values) (service.ListParams, error) {
	p := service.ListParams{
		Page: defaultPage, PageSize: defaultPageSize,
		Sort: service.SortCreatedAt, Order: service.OrderDesc,
	}
	r := paramReader{query: query}

	r.int("page", 1, 1<<31-1, &p.Page)
	r.int("page_size", 1, maxPageSize, &p.PageSize)
	p.Category = enumParam(&r, "category", func(v domain.Category) bool { return v.Valid() })
	p.IsActive = r.bool("is_active")
	p.Currency = enumParam(&r, "currency", func(v domain.Currency) bool { return v.Valid() })
	p.MinPriceCents = r.price("min_price")
	p.MaxPriceCents = r.price("max_price")
	p.Query = r.search("q")
	if sort := enumParam(&r, "sort", func(v service.SortField) bool { return v.Valid() }); sort != nil {
		p.Sort = *sort
	}
	if order := enumParam(&r, "order", func(v service.SortOrder) bool { return v.Valid() }); order != nil {
		p.Order = *order
	}
	if r.err != nil {
		return service.ListParams{}, r.err
	}
	if err := checkPriceRange(p); err != nil {
		return service.ListParams{}, err
	}
	return p, nil
}

// checkPriceRange: prices in different currencies are never compared, and
// the range must not be inverted.
func checkPriceRange(p service.ListParams) error {
	if p.Currency == nil {
		if p.MinPriceCents != nil {
			return &RequestError{Field: "min_price", Message: "requires currency"}
		}
		if p.MaxPriceCents != nil {
			return &RequestError{Field: "max_price", Message: "requires currency"}
		}
	}
	if p.MinPriceCents != nil && p.MaxPriceCents != nil && *p.MinPriceCents > *p.MaxPriceCents {
		return &RequestError{Field: "min_price", Message: "must not exceed max_price"}
	}
	return nil
}

// paramReader parses parameters and keeps the first error.
type paramReader struct {
	query url.Values
	err   error
}

func (r *paramReader) get(name string) (string, bool) {
	v := r.query.Get(name)
	return v, v != "" && r.err == nil
}

func (r *paramReader) fail(name, message string) {
	if r.err == nil {
		r.err = &RequestError{Field: name, Message: message}
	}
}

func (r *paramReader) int(name string, lo, hi int, dst *int) {
	raw, ok := r.get(name)
	if !ok {
		return
	}
	v, err := strconv.Atoi(raw)
	if err != nil || v < lo || v > hi {
		r.fail(name, "must be a whole number from "+strconv.Itoa(lo)+" to "+strconv.Itoa(hi))
		return
	}
	*dst = v
}

func (r *paramReader) bool(name string) *bool {
	raw, ok := r.get(name)
	if !ok {
		return nil
	}
	switch raw {
	case "true":
		v := true
		return &v
	case "false":
		v := false
		return &v
	default:
		r.fail(name, "must be true or false")
		return nil
	}
}

func (r *paramReader) price(name string) *int64 {
	raw, ok := r.get(name)
	if !ok {
		return nil
	}
	v, err := strconv.ParseInt(raw, 10, 64)
	if err != nil || v < 0 || v > domain.MaxPriceCents {
		r.fail(name, "must be a whole number of cents from 0 to 99999999999")
		return nil
	}
	return &v
}

func (r *paramReader) search(name string) string {
	raw, ok := r.get(name)
	if !ok {
		return ""
	}
	if utf8.RuneCountInString(raw) > maxQueryLen {
		r.fail(name, "must be at most 100 characters")
		return ""
	}
	return strings.TrimSpace(raw)
}

func enumParam[T ~string](r *paramReader, name string, valid func(T) bool) *T {
	raw, ok := r.get(name)
	if !ok {
		return nil
	}
	v := T(raw)
	if !valid(v) {
		r.fail(name, "is not an allowed value")
		return nil
	}
	return &v
}
