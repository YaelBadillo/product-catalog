package domain_test

import (
	"regexp"
	"testing"

	"github.com/YaelBadillo/product-catalog/api/internal/domain"
)

var skuPattern = regexp.MustCompile(`^[A-Z0-9-]{4,32}$`)

func FuzzNormalizeSKU(f *testing.F) {
	for _, seed := range []string{"  abc-123 ", "ABC-123", "AÑO-1", "ab c1", "", "ß-ss-ß", " abcd ", "ǅabc"} {
		f.Add(seed)
	}
	f.Fuzz(func(t *testing.T, raw string) {
		once := domain.NormalizeSKU(raw)

		if twice := domain.NormalizeSKU(once); twice != once {
			t.Fatalf("not idempotent: %q → %q → %q", raw, once, twice)
		}
		if domain.ValidateSKU(once) == nil && !skuPattern.MatchString(once) {
			t.Fatalf("accepted %q, which does not match %s", once, skuPattern)
		}
	})
}
