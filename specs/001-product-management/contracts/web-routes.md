# Web Contract: Routes and Catalog URL

The UI contract of feature 001: the routes and the query parameters that make the catalog view shareable (FR-023 to FR-025). Users see every text in Mexican Spanish (FR-045, FR-046); keys and paths are in English.

## Routes

| Path | View | `document.title` | Notes |
|---|---|---|---|
| `/` | — | — | Redirects to `/products` (`replace`) |
| `/products` | Catalog | `Catálogo · Catálogo de productos` | State in the query (below) |
| `/products/new` | Create form | `Nuevo producto · Catálogo de productos` | On success → `/products/:id` |
| `/products/:id` | Detail | `{name} · Catálogo de productos` | `:id` not a UUID → "Producto no encontrado" with no request |
| `/products/:id/edit` | Edit form | `Editar {name} · Catálogo de productos` | On success → `/products/:id` |
| `/status` | Service status (existing, translated) | `Estado del servicio · Catálogo de productos` | Moves here from `/` |
| `*` | Not found | `Página no encontrada · Catálogo de productos` | Link "Volver al catálogo" |

The header (every view, FR-044) has links "Catálogo" (`/products`), "Nuevo producto" (`/products/new`) and "Estado" (`/status`), with `aria-current="page"` on the active one.

Going back from the detail to the catalog restores the previous view: the detail and edit links preserve the catalog's query in the history, and "Volver al catálogo" uses `navigate(-1)` when the previous entry is the catalog, or `/products` otherwise.

## Catalog query parameters

| Key | Values | Default (omitted from the URL) | Sanitization (FR-025) |
|---|---|---|---|
| `page` | integer ≥ 1 | `1` | Invalid → `1` |
| `category` | `electronics` \| `home` \| `food` \| `clothing` \| `other` | all | Invalid → no filter |
| `is_active` | `true` \| `false` | both | Invalid → no filter |
| `currency` | `MXN` \| `USD` | all | Invalid → no filter |
| `min_price`, `max_price` | currency units with ≤ 2 decimals (`199.90`) | none | Ignored without a valid `currency`, if invalid, or if min > max (both) |
| `q` | text, ≤ 100 characters | none | Trimmed; longer → cut to 100 |
| `sort` | `name` \| `price` \| `stock` \| `created_at` | `created_at` | Invalid → default |
| `order` | `asc` \| `desc` | `desc` | Invalid → default |

- **Translation to the API**: same keys. `min_price`/`max_price` → cents (`Math.round(x * 100)`). `page_size` is always 20 and isn't in the URL.
- **History**: changes to filters, sort, order and page add an entry. Changes to filters, search and sort set `page=1`. A search applies 300 ms after the last keystroke or on Enter; the first update of a typing session adds an entry and the rest replace it.
- **Page out of range**: if `page > total_pages ≥ 1`, show "Esta página no tiene productos" with "Ir a la primera página" (FR-026). After a deletion that empties the last page, navigate to `page = total_pages` (US4, scenario 8).

## Examples

```text
/products
/products?category=food&is_active=true
/products?currency=MXN&min_price=100&max_price=499.90&sort=price&order=asc
/products?q=caf%C3%A9&page=2
```
