package seed

import "github.com/YaelBadillo/product-catalog/api/internal/domain"

func ptr[T any](v T) *T { return &v }

// product builds a sample input. Prices in cents.
func product(sku, name string, cents int64, currency domain.Currency, stock int64, category domain.Category, active bool) domain.ProductInput {
	return domain.ProductInput{
		SKU: sku, Name: name, PriceCents: cents, Currency: currency,
		Stock: stock, Category: category, IsActive: ptr(active),
	}
}

func withDescription(in domain.ProductInput, description string) domain.ProductInput {
	in.Description = &description
	return in
}

// Products returns the 20 sample products: every category, both currencies,
// four inactive, and names with accents and ñ that also exercise the Spanish
// order (árbol next to arroz, ñame after nuez) and the accent-insensitive
// search by hand.
func Products() []domain.ProductInput {
	const mxn, usd = domain.CurrencyMXN, domain.CurrencyUSD
	const (
		electronics = domain.CategoryElectronics
		home        = domain.CategoryHome
		food        = domain.CategoryFood
		clothing    = domain.CategoryClothing
		other       = domain.CategoryOther
	)
	return []domain.ProductInput{
		withDescription(product("CAF-0123", "Café molido", 19990, mxn, 40, food, true), "Café de Veracruz, tostado medio, 500 g."),
		product("TEA-0001", "Té verde", 8950, mxn, 25, food, true),
		product("ARR-0500", "Arroz integral", 3490, mxn, 120, food, true),
		product("NUE-0250", "Nuez pecana", 15900, mxn, 18, food, true),
		product("NAM-0001", "Ñame fresco", 4200, mxn, 0, food, false),
		withDescription(product("ARB-2026", "árbol de Navidad artificial", 129990, mxn, 6, home, true), "1.80 m, con base metálica."),
		product("SIL-0001", "Silla plegable", 4500, usd, 32, home, true),
		product("LAM-0100", "Lámpara de escritorio", 64900, mxn, 12, home, true),
		product("EDR-0002", "Edredón matrimonial", 89900, mxn, 4, home, false),
		product("AUD-0300", "Audífonos inalámbricos", 5999, usd, 50, electronics, true),
		product("TEC-0101", "Teclado mecánico", 189900, mxn, 9, electronics, true),
		product("MON-2701", "Monitor 27 pulgadas", 24999, usd, 7, electronics, true),
		product("CAR-0065", "Cargador USB-C 65 W", 49900, mxn, 0, electronics, false),
		product("CAM-0100", "Camisa de algodón", 59900, mxn, 30, clothing, true),
		product("PAN-0032", "Pantalón de mezclilla", 3999, usd, 22, clothing, true),
		product("SUD-0001", "Sudadera con capucha", 79900, mxn, 15, clothing, true),
		product("ZAP-0042", "Zapato de piel", 129900, mxn, 3, clothing, false),
		product("CUA-0005", "Cuaderno de dibujo", 12900, mxn, 60, other, true),
		product("PIN-0001", "Piñata de estrella", 34900, mxn, 10, other, true),
		product("MOC-0010", "Mochila escolar", 2999, usd, 14, other, true),
	}
}
