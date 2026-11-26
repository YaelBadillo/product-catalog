package repository_test

import (
	"slices"
	"testing"
	"time"

	"github.com/YaelBadillo/product-catalog/api/internal/domain"
	"github.com/YaelBadillo/product-catalog/api/internal/repository"
	"github.com/YaelBadillo/product-catalog/api/internal/service"
	"github.com/YaelBadillo/product-catalog/api/internal/testutil"
)

const benchProducts = 10_000

// seedBench inserts benchProducts varied rows in one statement.
func seedBench(b *testing.B) {
	b.Helper()
	testutil.ResetDB(b, pool)
	_, err := pool.Exec(b.Context(), `
		INSERT INTO products (sku, name, price_cents, currency, stock, category, is_active)
		SELECT 'BEN-' || lpad(i::text, 6, '0'),
		       (ARRAY['Café', 'Té', 'Árbol', 'Ñame', 'Silla', 'Camisa', 'Nuez'])[1 + i % 7] || ' modelo ' || i,
		       100 + (i * 7919) % 1000000,
		       CASE WHEN i % 3 = 0 THEN 'USD' ELSE 'MXN' END,
		       i % 500,
		       (ARRAY['electronics', 'home', 'food', 'clothing', 'other'])[1 + i % 5],
		       i % 4 <> 0
		FROM generate_series(1, $1) AS i`, benchProducts)
	if err != nil {
		b.Fatal(err)
	}
	if _, err := pool.Exec(b.Context(), "ANALYZE products"); err != nil {
		b.Fatal(err)
	}
}

// BenchmarkList measures list, filter, search and sort over 10,000 products
// and reports the p95 (SC-002: server target under 200 ms).
func BenchmarkList(b *testing.B) {
	seedBench(b)
	repo := repository.NewProducts(pool)
	cases := map[string]func(*service.ListParams){
		"default": func(*service.ListParams) {},
		"filter": func(p *service.ListParams) {
			p.Category = ptr(domain.CategoryFood)
			p.IsActive = ptr(true)
			p.Currency = ptr(domain.CurrencyMXN)
			p.MinPriceCents = ptr[int64](10_000)
			p.MaxPriceCents = ptr[int64](500_000)
		},
		"search":     func(p *service.ListParams) { p.Query = "cafe modelo 12" },
		"sort name":  func(p *service.ListParams) { p.Sort = service.SortName; p.Order = service.OrderAsc },
		"sort price": func(p *service.ListParams) { p.Sort = service.SortPrice; p.Page = 250 },
	}
	for name, mutate := range cases {
		b.Run(name, func(b *testing.B) {
			p := params(mutate)
			durations := make([]time.Duration, 0, b.N)
			for b.Loop() {
				start := time.Now()
				if _, err := repo.List(b.Context(), p); err != nil {
					b.Fatal(err)
				}
				durations = append(durations, time.Since(start))
			}
			slices.Sort(durations)
			p95 := durations[len(durations)*95/100]
			b.ReportMetric(float64(p95.Microseconds())/1000, "p95-ms")
		})
	}
}
