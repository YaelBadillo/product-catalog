package service

import (
	"context"

	"github.com/YaelBadillo/product-catalog/api/internal/domain"
)

// SortField is a column the catalog can be sorted by (a whitelist).
type SortField string

// Sort fields.
const (
	SortName      SortField = "name"
	SortPrice     SortField = "price"
	SortStock     SortField = "stock"
	SortCreatedAt SortField = "created_at"
)

// Valid reports whether f is in the whitelist.
func (f SortField) Valid() bool {
	switch f {
	case SortName, SortPrice, SortStock, SortCreatedAt:
		return true
	default:
		return false
	}
}

// SortOrder is the sort direction.
type SortOrder string

// Sort orders.
const (
	OrderAsc  SortOrder = "asc"
	OrderDesc SortOrder = "desc"
)

// Valid reports whether o is asc or desc.
func (o SortOrder) Valid() bool { return o == OrderAsc || o == OrderDesc }

// ListParams selects one page of the catalog. Nil filters are unset; Query
// is already trimmed and matches literally.
type ListParams struct {
	Page          int
	PageSize      int
	Category      *domain.Category
	IsActive      *bool
	Currency      *domain.Currency
	MinPriceCents *int64
	MaxPriceCents *int64
	Query         string
	Sort          SortField
	Order         SortOrder
}

// ProductPage is one page of products and the total that match the filters.
type ProductPage struct {
	Items []domain.Product
	Total int64
}

// List returns one page of the non-deleted products that match p.
func (s *Products) List(ctx context.Context, p ListParams) (ProductPage, error) {
	return s.repo.List(ctx, p)
}
