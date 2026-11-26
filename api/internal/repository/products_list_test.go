package repository_test

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/YaelBadillo/product-catalog/api/internal/domain"
	"github.com/YaelBadillo/product-catalog/api/internal/repository"
	"github.com/YaelBadillo/product-catalog/api/internal/service"
)

// seed creates one product per mutation of a valid input, in order.
func seed(t *testing.T, repo *repository.Products, mutations ...func(*domain.ProductInput)) []domain.Product {
	t.Helper()
	products := make([]domain.Product, 0, len(mutations))
	for i, mutate := range mutations {
		in := input(fmt.Sprintf("SKU-%04d", i))
		mutate(&in)
		p, err := repo.Create(t.Context(), in)
		require.NoError(t, err)
		products = append(products, p)
	}
	return products
}

func params(mutate func(*service.ListParams)) service.ListParams {
	p := service.ListParams{Page: 1, PageSize: 20, Sort: service.SortCreatedAt, Order: service.OrderDesc}
	if mutate != nil {
		mutate(&p)
	}
	return p
}

func names(page service.ProductPage) []string {
	out := make([]string, 0, len(page.Items))
	for _, p := range page.Items {
		out = append(out, p.Name)
	}
	return out
}

func TestListFiltersCombine(t *testing.T) {
	repo := newRepo(t)
	seed(t, repo,
		func(in *domain.ProductInput) { in.Name = "Match"; in.PriceCents = 100 },
		func(in *domain.ProductInput) { in.Name = "Match top"; in.PriceCents = 500 },
		func(in *domain.ProductInput) { in.Name = "Too cheap"; in.PriceCents = 99 },
		func(in *domain.ProductInput) { in.Name = "Too expensive"; in.PriceCents = 501 },
		func(in *domain.ProductInput) { in.Name = "Other category"; in.Category = domain.CategoryHome },
		func(in *domain.ProductInput) { in.Name = "Inactive"; in.IsActive = ptr(false) },
		func(in *domain.ProductInput) { in.Name = "Dollars"; in.Currency = domain.CurrencyUSD },
	)

	page, err := repo.List(t.Context(), params(func(p *service.ListParams) {
		p.Category = ptr(domain.CategoryFood)
		p.IsActive = ptr(true)
		p.Currency = ptr(domain.CurrencyMXN)
		p.MinPriceCents = ptr[int64](100)
		p.MaxPriceCents = ptr[int64](500)
		p.Sort = service.SortName
		p.Order = service.OrderAsc
	}))

	require.NoError(t, err)
	assert.Equal(t, []string{"Match", "Match top"}, names(page))
	assert.Equal(t, int64(2), page.Total)
}

func TestListFiltersInactive(t *testing.T) {
	repo := newRepo(t)
	seed(t, repo,
		func(in *domain.ProductInput) { in.Name = "Active" },
		func(in *domain.ProductInput) { in.Name = "Inactive"; in.IsActive = ptr(false) },
	)

	page, err := repo.List(t.Context(), params(func(p *service.ListParams) { p.IsActive = ptr(false) }))

	require.NoError(t, err)
	assert.Equal(t, []string{"Inactive"}, names(page))
}

func TestListSearch(t *testing.T) {
	repo := newRepo(t)
	seed(t, repo,
		func(in *domain.ProductInput) { in.Name = "Café molido"; in.SKU = "CAF-0123" },
		func(in *domain.ProductInput) { in.Name = "Té verde"; in.SKU = "TEA-0001" },
		func(in *domain.ProductInput) { in.Name = "Camisa 100% algodón" },
		func(in *domain.ProductInput) { in.Name = "Pan_integral" },
		func(in *domain.ProductInput) { in.Name = `Barra\acero` },
		func(in *domain.ProductInput) { in.Name = "Pan integral" },
	)

	tests := []struct {
		query string
		want  []string
	}{
		{query: "cafe", want: []string{"Café molido"}},
		{query: "CAFÉ MOL", want: []string{"Café molido"}},
		{query: "0123", want: []string{"Café molido"}},
		{query: "caf-0", want: []string{"Café molido"}},
		{query: "TÉ VER", want: []string{"Té verde"}},
		{query: "%", want: []string{"Camisa 100% algodón"}},
		{query: "_", want: []string{"Pan_integral"}},
		{query: `\`, want: []string{`Barra\acero`}},
		{query: "nada", want: []string{}},
	}
	for _, tt := range tests {
		t.Run(tt.query, func(t *testing.T) {
			page, err := repo.List(t.Context(), params(func(p *service.ListParams) {
				p.Query = tt.query
				p.Sort = service.SortName
				p.Order = service.OrderAsc
			}))

			require.NoError(t, err)
			assert.Equal(t, tt.want, names(page))
			assert.Equal(t, int64(len(tt.want)), page.Total)
		})
	}
}

func TestListSort(t *testing.T) {
	repo := newRepo(t)
	seed(t, repo,
		func(in *domain.ProductInput) { in.Name = "Zapato"; in.PriceCents = 300; in.Stock = 1 },
		func(in *domain.ProductInput) {
			in.Name = "árbol"
			in.PriceCents = 100
			in.Stock = 3
			in.Currency = domain.CurrencyUSD
		},
		func(in *domain.ProductInput) { in.Name = "Arroz"; in.PriceCents = 200; in.Stock = 5 },
		func(in *domain.ProductInput) {
			in.Name = "ñame"
			in.PriceCents = 50
			in.Stock = 2
			in.Currency = domain.CurrencyUSD
		},
		func(in *domain.ProductInput) { in.Name = "nuez"; in.PriceCents = 400; in.Stock = 4 },
	)

	tests := []struct {
		sort  service.SortField
		order service.SortOrder
		want  []string
	}{
		{service.SortName, service.OrderAsc, []string{"árbol", "Arroz", "nuez", "ñame", "Zapato"}},
		{service.SortName, service.OrderDesc, []string{"Zapato", "ñame", "nuez", "Arroz", "árbol"}},
		{service.SortPrice, service.OrderAsc, []string{"Arroz", "Zapato", "nuez", "ñame", "árbol"}},
		{service.SortPrice, service.OrderDesc, []string{"nuez", "Zapato", "Arroz", "árbol", "ñame"}},
		{service.SortStock, service.OrderAsc, []string{"Zapato", "ñame", "árbol", "nuez", "Arroz"}},
		{service.SortStock, service.OrderDesc, []string{"Arroz", "nuez", "árbol", "ñame", "Zapato"}},
		{service.SortCreatedAt, service.OrderAsc, []string{"Zapato", "árbol", "Arroz", "ñame", "nuez"}},
		{service.SortCreatedAt, service.OrderDesc, []string{"nuez", "ñame", "Arroz", "árbol", "Zapato"}},
	}
	for _, tt := range tests {
		t.Run(string(tt.sort)+" "+string(tt.order), func(t *testing.T) {
			page, err := repo.List(t.Context(), params(func(p *service.ListParams) {
				p.Sort = tt.sort
				p.Order = tt.order
			}))

			require.NoError(t, err)
			assert.Equal(t, tt.want, names(page))
		})
	}
}

func TestListPaginationNeverRepeatsOrSkips(t *testing.T) {
	repo := newRepo(t)
	mutations := make([]func(*domain.ProductInput), 25)
	for i := range mutations {
		mutations[i] = func(in *domain.ProductInput) { in.Name = fmt.Sprintf("Item %02d", i) }
	}
	created := seed(t, repo, mutations...)
	// Every product ties on stock, so only the id breaks ties.
	seen := map[string]bool{}

	for pageNumber := 1; pageNumber <= 3; pageNumber++ {
		page, err := repo.List(t.Context(), params(func(p *service.ListParams) {
			p.Page = pageNumber
			p.PageSize = 10
			p.Sort = service.SortStock
			p.Order = service.OrderAsc
		}))
		require.NoError(t, err)
		assert.Equal(t, int64(25), page.Total)
		for _, p := range page.Items {
			assert.False(t, seen[p.ID], "product %s repeated", p.Name)
			seen[p.ID] = true
		}
	}
	assert.Len(t, seen, len(created), "no product skipped")
}

func TestListPageBeyondTheLast(t *testing.T) {
	repo := newRepo(t)
	seed(t, repo,
		func(in *domain.ProductInput) { in.Name = "Uno" },
		func(in *domain.ProductInput) { in.Name = "Dos" },
	)

	page, err := repo.List(t.Context(), params(func(p *service.ListParams) { p.Page = 999 }))

	require.NoError(t, err)
	assert.Empty(t, page.Items)
	assert.NotNil(t, page.Items)
	assert.Equal(t, int64(2), page.Total)
}

func TestListExcludesDeleted(t *testing.T) {
	repo := newRepo(t)
	products := seed(t, repo,
		func(in *domain.ProductInput) { in.Name = "Visible" },
		func(in *domain.ProductInput) { in.Name = "Borrado" },
	)
	_, err := pool.Exec(t.Context(), "UPDATE products SET deleted_at = now() WHERE id = $1", products[1].ID)
	require.NoError(t, err)

	page, err := repo.List(t.Context(), params(nil))

	require.NoError(t, err)
	assert.Equal(t, []string{"Visible"}, names(page))
	assert.Equal(t, int64(1), page.Total)
}
