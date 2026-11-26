package handler

import (
	"time"

	"github.com/YaelBadillo/product-catalog/api/internal/domain"
	"github.com/YaelBadillo/product-catalog/api/internal/service"
)

// CreateProductRequest is the body of POST /products. Tags check shape; the
// custom tags delegate to the domain rules, measured after normalizing.
type CreateProductRequest struct {
	SKU         string  `json:"sku" validate:"required,sku"`
	Name        string  `json:"name" validate:"required,trimmed_len=2-120"`
	Description *string `json:"description" validate:"omitempty,trimmed_len=0-2000"`
	PriceCents  *int64  `json:"price_cents" validate:"required,min=1,max=99999999999"`
	Currency    *string `json:"currency" validate:"omitempty,oneof=MXN USD"`
	Stock       *int64  `json:"stock" validate:"required,min=0,max=2147483647"`
	Category    string  `json:"category" validate:"required,oneof=electronics home food clothing other"`
	IsActive    *bool   `json:"is_active"`
}

func (r CreateProductRequest) toInput() domain.ProductInput {
	in := domain.ProductInput{
		SKU:         r.SKU,
		Name:        r.Name,
		Description: r.Description,
		Category:    domain.Category(r.Category),
		IsActive:    r.IsActive,
	}
	if r.PriceCents != nil {
		in.PriceCents = *r.PriceCents
	}
	if r.Stock != nil {
		in.Stock = *r.Stock
	}
	if r.Currency != nil {
		in.Currency = domain.Currency(*r.Currency)
	}
	return in
}

// ProductResponse is a product in every response. Description is null when
// there is none.
type ProductResponse struct {
	ID          string    `json:"id"`
	SKU         string    `json:"sku"`
	Name        string    `json:"name"`
	Description *string   `json:"description"`
	PriceCents  int64     `json:"price_cents"`
	Currency    string    `json:"currency"`
	Stock       int32     `json:"stock"`
	Category    string    `json:"category"`
	IsActive    bool      `json:"is_active"`
	Version     int32     `json:"version"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

func toResponse(p domain.Product) ProductResponse {
	return ProductResponse{
		ID:          p.ID,
		SKU:         p.SKU,
		Name:        p.Name,
		Description: p.Description,
		PriceCents:  p.PriceCents,
		Currency:    string(p.Currency),
		Stock:       p.Stock,
		Category:    string(p.Category),
		IsActive:    p.IsActive,
		Version:     p.Version,
		CreatedAt:   p.CreatedAt.UTC(),
		UpdatedAt:   p.UpdatedAt.UTC(),
	}
}

// ProductList is the body of GET /products.
type ProductList struct {
	Data []ProductResponse `json:"data"`
	Meta PageMeta          `json:"meta"`
}

// PageMeta describes the page and the total that match the filters.
type PageMeta struct {
	Page       int   `json:"page"`
	PageSize   int   `json:"page_size"`
	Total      int64 `json:"total"`
	TotalPages int64 `json:"total_pages"`
}

func toListResponse(page service.ProductPage, p service.ListParams) ProductList {
	data := make([]ProductResponse, 0, len(page.Items))
	for _, item := range page.Items {
		data = append(data, toResponse(item))
	}
	pageSize := int64(p.PageSize)
	return ProductList{
		Data: data,
		Meta: PageMeta{
			Page:       p.Page,
			PageSize:   p.PageSize,
			Total:      page.Total,
			TotalPages: (page.Total + pageSize - 1) / pageSize,
		},
	}
}
