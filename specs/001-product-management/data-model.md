# Data Model: Product Management

Phase 1 of [plan.md](plan.md). Rules come from [spec.md](spec.md) (FR-001 to FR-009); decisions from [research.md](research.md).

## Product

One table, `products`. `deleted_at` implements logical deletion (FR-036). A deleted row never leaves the database, but no query returns it.

| Field | DB column | DB type | API (JSON) | Rule (after normalization) | UI label (es-MX) |
|---|---|---|---|---|---|
| ID | `id` | `uuid` PK, `DEFAULT uuidv7()` | `id` (uuid) | Server-generated (R4) | — (never shown, FR-015) |
| SKU | `sku` | `text NOT NULL` | `sku` | Trim + uppercase; `^[A-Z0-9-]{4,32}$`; unique among non-deleted; immutable | SKU |
| Name | `name` | `text COLLATE catalog_es NOT NULL` | `name` | Trim; 2–120 characters | Nombre |
| Description | `description` | `text NULL` | `description` (string \| null) | Trim; ≤ 2000; empty → `NULL` | Descripción |
| Price | `price_cents` | `bigint NOT NULL` | `price_cents` (int64) | 1 to 99,999,999,999 (= 999,999,999.99) | Precio |
| Currency | `currency` | `text NOT NULL DEFAULT 'MXN'` | `currency` | `MXN` \| `USD`; changing it doesn't convert | Moneda |
| Inventory | `stock` | `integer NOT NULL` | `stock` (int32) | 0 to 2,147,483,647 | Inventario |
| Category | `category` | `text NOT NULL` | `category` | `electronics` \| `home` \| `food` \| `clothing` \| `other`; required, no default | Categoría |
| Status | `is_active` | `boolean NOT NULL DEFAULT true` | `is_active` | — | Estado: Activo / Inactivo |
| Version | `version` | `integer NOT NULL DEFAULT 1` | `version` | +1 on every update and deletion | — |
| Created | `created_at` | `timestamptz NOT NULL DEFAULT now()` | `created_at` | Server-managed | Creado |
| Updated | `updated_at` | `timestamptz NOT NULL DEFAULT now()` | `updated_at` | `now()` on every update | Última modificación |
| Deleted | `deleted_at` | `timestamptz NULL` | — (not exposed) | `now()` on deletion | — |

Category names in the UI: `electronics` → Electrónica, `home` → Hogar, `food` → Alimentos, `clothing` → Ropa, `other` → Otra.

### Normalization (domain, before any validation)

| Field | Normalization | Examples |
|---|---|---|
| `sku` | `strings.TrimSpace` + `strings.ToUpper` | `"  abc-123 "` → `"ABC-123"`; `"ab c1"` → `"AB C1"` (invalid: inner space) |
| `name` | `strings.TrimSpace` | `"  Café  "` → `"Café"`; `"   "` → `""` (required) |
| `description` | `strings.TrimSpace`; `""` → `NULL` | `"  "` → `NULL` |

Lengths are counted in Unicode characters (`utf8.RuneCountInString` in Go, `char_length` in SQL).

### Database constraints (duplicate the domain rules)

- `CHECK (sku ~ '^[A-Z0-9-]{4,32}$')`
- `CHECK (char_length(name) BETWEEN 2 AND 120)`
- `CHECK (description IS NULL OR char_length(description) BETWEEN 1 AND 2000)`
- `CHECK (price_cents BETWEEN 1 AND 99999999999)`
- `CHECK (stock >= 0)` (the upper bound comes from `integer`)
- `CHECK (currency IN ('MXN', 'USD'))`
- `CHECK (category IN ('electronics', 'home', 'food', 'clothing', 'other'))`
- `CHECK (version >= 1)`

### Indexes (all partial, `WHERE deleted_at IS NULL`)

| Index | Columns | Purpose |
|---|---|---|
| `products_sku_active_key` | `UNIQUE (sku)` | SKU unique among non-deleted; a deleted product frees its SKU (FR-036). `23505` on this name → `ErrSKUAlreadyExists` |
| `products_category_idx` | `(category)` | Category filter |
| `products_created_at_idx` | `(created_at, id)` | Default sort (newest first) |
| `products_currency_price_idx` | `(currency, price_cents, id)` | Currency filter, price range and price sort |

No index for search: `catalog_fold(...) LIKE '%…%'` does a sequential scan, which is enough at 10,000 rows (R2, SC-002).

### Database objects created by the migration

- `CREATE EXTENSION IF NOT EXISTS unaccent`
- `CREATE COLLATION catalog_es (provider = icu, locale = 'es-MX')`: Spanish order for `name` (R1)
- `CREATE FUNCTION catalog_fold(text) RETURNS text IMMUTABLE PARALLEL SAFE`: `lower(unaccent('unaccent', $1))` (R2)

The `Down` migration drops them in reverse order.

## Lifecycle

```text
            create (version = 1)
 (none) ─────────────────────────▶ Active ◀──── edit is_active ────▶ Inactive
                                     │                                  │
                                     └──────── delete (version + 1) ────┘
                                                       │
                                                       ▼
                                          Deleted (terminal: deleted_at set,
                                          SKU free, no query returns it)
```

- **Active ⇄ Inactive**: a normal edit (`is_active`). An inactive product stays visible and editable (FR-009).
- **Edit**: applied only if `version` matches. It increments `version` and sets `updated_at` (FR-029).
- **Delete**: applied only if `version` matches. It sets `deleted_at` and increments `version` (FR-035, FR-036). There is no transition out of `Deleted` (recovery is out of scope).
- Editing or deleting a deleted product → `ErrNotFound` (FR-016, FR-037).

## Domain (`internal/domain`)

| Type | Content |
|---|---|
| `Product` | The fields above except `deleted_at`. `Description *string`, `PriceCents int64`, `Stock int32`, `Version int32` |
| `Category` | `string` with constants `CategoryElectronics`, `CategoryHome`, `CategoryFood`, `CategoryClothing`, `CategoryOther`, and `Valid()` |
| `Currency` | `string` with constants `CurrencyMXN`, `CurrencyUSD`, and `Valid()` |
| `NewProduct` / `ProductInput` | Creation input; `Normalize()` + `Validate()` return `*ValidationError` |
| `ProductPatch` | Pointers per field; `Description *string` (`""` = remove); `Version int32` required; `IsEmpty()` |
| `ValidationError` | `Fields []FieldError{Field, Message}`, with JSON names (`price_cents`, …) |
| Sentinel errors | `ErrNotFound`, `ErrSKUAlreadyExists`, `ErrVersionConflict` (`ErrInsufficientStock` is removed: out of scope) |

## List parameters (`service.ListParams`)

| Field | Type | Default | Rule |
|---|---|---|---|
| `Page` | int | 1 | ≥ 1 |
| `PageSize` | int | 20 | 1–100 (the web always uses 20) |
| `Category` | `*Category` | — | Valid enum |
| `IsActive` | `*bool` | — | — |
| `Currency` | `*Currency` | — | Valid enum |
| `MinPriceCents`, `MaxPriceCents` | `*int64` | — | Require `Currency`; 0 ≤ min ≤ max ≤ 99,999,999,999 |
| `Query` | string | `""` | Trimmed, ≤ 100 characters; `\`, `%` and `_` escaped in the repository |
| `Sort` | `SortField` | `created_at` | `name` \| `price` \| `stock` \| `created_at` |
| `Order` | `SortOrder` | `desc` | `asc` \| `desc` |

Result: `ProductPage{Items []Product, Total int64}`. `total_pages = ceil(total / page_size)` (0 if `total = 0`). A page beyond the last one returns `200` with `data: []` and the real `total` (FR-026).

## Web (`features/products`)

| Model | Shape | Notes |
|---|---|---|
| `Product` | `components["schemas"]["Product"]` (generated) | Never typed by hand |
| `ProductFormValues` | `{ sku, name, description, price: number \| "", currency, stock: number \| "", category: Category \| "", isActive }` | `price` in currency units. Creation: zod with Spanish messages (FR-047) |
| `ProductPatchBody` | Only the dirty fields + `version` | `price_cents = Math.round(price * 100)` after zod |
| `CatalogSearch` | `{ page, category?, isActive?, currency?, minPrice?, maxPrice?, q?, sort, order }` | Parsed from the URL with defaults (R18). `minPrice`/`maxPrice` in units |
| `ApiError` | `{ status, code, details[], retryAfter? }` | `src/api/errors.ts` (R19) |
