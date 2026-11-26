# Quickstart: Validating Product Management

How to prove, end to end, that feature 001 works. Contract: [contracts/products-api.yaml](contracts/products-api.yaml), [contracts/web-routes.md](contracts/web-routes.md). Rules: [data-model.md](data-model.md).

## Prerequisites

The repository's usual ones (README): Docker with Compose v2, Go per `api/go.mod`, Node 22.22+ with pnpm, make. Work from the branch `001-product-management`.

## Setup

```sh
cp .env.example .env
make db          # Postgres 18
make dev-api     # migrates at startup, then listens on :8080
make seed        # 20 sample products; running it again creates no duplicates
make dev-web     # http://localhost:5173
```

The full stack with Docker: `make up` (web on http://localhost:3000), then `make seed`.

## Automated checks (all green before opening the PR)

```sh
make test        # API (unit + integration with testcontainers) and web
make lint        # golangci-lint, ESLint, Prettier, typecheck
make sqlc && git diff --exit-code api/internal/repository/db
make gen-types && git diff --exit-code web/src/api/schema.d.ts
make sync-openapi && git diff --exit-code api/internal/apidocs/openapi.yaml
```

## API: scenarios with curl

`API=http://localhost:8080/api/v1`. Every error follows `{ "error": { "code", "message", "details" } }`.

| # | Request | Expected |
|---|---|---|
| 1 | `POST $API/products` `{"sku":"  abc-123 ","name":" Café molido ","price_cents":19990,"stock":5,"category":"food"}` | `201`, `Location: /api/v1/products/{id}`, `sku: "ABC-123"`, `name: "Café molido"`, `currency: "MXN"`, `is_active: true`, `version: 1`, `description: null` |
| 2 | Same POST again | `409 SKU_ALREADY_EXISTS` |
| 3 | `POST` with `"name":" a "`, `"price_cents":0`, `"category":"toys"` | `422 VALIDATION_ERROR` with `details` for `name`, `price_cents`, `category` |
| 4 | `POST` with an unknown field `"color":"red"` | `400 VALIDATION_ERROR`, `details[0].field = "color"` |
| 5 | `GET $API/products/{id}` | `200` with the product. With a random UUID: `404 NOT_FOUND`. With `abc`: `400` |
| 6 | `GET $API/products?q=CAFE` | Finds "Café molido" (case and accents ignored) |
| 7 | `GET $API/products?q=%25` | Only products whose name or SKU contains a literal `%` |
| 8 | `GET $API/products?min_price=100` | `400`: `details[0].field = "min_price"` (needs `currency`) |
| 9 | `GET $API/products?currency=MXN&min_price=500&max_price=100` | `400` (min > max) |
| 10 | `GET $API/products?sort=price&order=asc` | All MXN products by price, then all USD ones |
| 11 | `GET $API/products?sort=name&order=asc` | Spanish order: `árbol` next to `arroz`, `ñame` after `nuez` (the seed includes these cases) |
| 12 | `GET $API/products?page=999` | `200`, `data: []`, real `meta.total` |
| 13 | `PATCH $API/products/{id}` `{"version":1,"stock":7}` | `200`, `stock: 7`, `version: 2` |
| 14 | Repeat 13 with `"version":1` | `409 VERSION_CONFLICT`; the product doesn't change |
| 15 | `PATCH` `{"version":2,"description":"  "}` on a product with a description | `200`, `description: null` |
| 16 | `PATCH` `{"version":2}` | `422` (no changes) |
| 17 | `PATCH` `{"version":2,"sku":"NEW-1"}` | `400` (unknown field: the SKU is immutable) |
| 18 | `DELETE $API/products/{id}?version=1` (outdated) | `409 VERSION_CONFLICT` |
| 19 | `DELETE $API/products/{id}?version={current}` | `204`. Then `GET` → `404`, it isn't in the list, and a `POST` with its SKU → `201` |
| 20 | `DELETE $API/products/{id}` without `version` | `400` |
| 21 | After a minute without requests, 301 quick ones: `for i in $(seq 301); do curl -s -o /dev/null -w '%{http_code}\n' $API/products; done \| sort \| uniq -c` | 300 × `200`, 1 × `429 RATE_LIMITED` with `Retry-After` |
| 22 | `POST` with a body > 64 KiB | `413 PAYLOAD_TOO_LARGE` |
| 23 | `curl -s -D - -o /dev/null $API/products` | `X-Content-Type-Options: nosniff`, `X-Frame-Options: DENY`, `Content-Security-Policy: default-src 'none'; frame-ancestors 'none'` |
| 24 | `curl -s localhost:8080/healthz` 400 times | Always `200` (the probes aren't rate-limited) |
| 25 | Open http://localhost:8080/docs | Swagger UI with the contract; `/docs/openapi.yaml` returns the YAML |

## Web: walkthrough per story

Open the web at 1280 px and repeat the key flows at 360 px (FR-048, SC-010). Every text must be in Spanish (SC-009).

**US1: Register and view**
1. "Nuevo producto": currency MXN, status active, category unselected.
2. Submit empty: errors on every required field, and focus on the first one.
3. SKU ` abc-999 `, name "Té verde", price `1,299.90`, inventory 3, category Alimentos → "Producto creado" and the detail with `ABC-999`, `$1,299.90 MXN`, "Sin descripción" and the dates as `25 nov 2026, 14:30`.
4. Repeat with the same SKU: "Ya existe un producto con este SKU" on the field.
5. Type a comma in the price: it isn't entered, and "Usa punto (.) para los decimales" appears.
6. Open `/products/00000000-0000-7000-8000-000000000000` and `/products/abc`: "Producto no encontrado" with "Volver al catálogo".

**US2: Browse**
1. `/` takes you to `/products`: 20 rows, total and pagination.
2. Filter Alimentos + Activo + MXN: the URL changes, and so do the results.
3. The price range is disabled without a currency, with the text "Elige una moneda para filtrar por precio". With MXN, set 100–500. Clear the currency: the range clears too.
4. Search "cafe": results after a pause. Back returns to the view before the search.
5. Sort by price with no currency: MXN first, then USD.
6. Copy the URL into another window: same view. Change a filter while on page 2: back to page 1.
7. Empty states: with no data, "Aún no hay productos" with "Registrar producto". With no results, "Ningún producto coincide con tu búsqueda o filtros" with "Limpiar filtros". Stopping the API shows the error with "Reintentar". `?page=999` shows "Esta página no tiene productos".

**US3: Edit**
1. "Editar": prefilled fields and a read-only SKU. "Guardar" is disabled until something changes.
2. Change the inventory and empty the description → "Cambios guardados", and the detail is updated.
3. Open the edit form in two tabs, save in one, then save in the other: "Otra persona modificó este producto. Recarga para ver la versión actual." with "Recargar", and your values stay in the form.

**US4: Delete**
1. Delete from the list: "¿Eliminar producto?" with the name and SKU. Cancel → nothing changes.
2. Confirm → the row disappears immediately, then "Producto eliminado".
3. With the API stopped, confirm: the row reappears with "No se pudo eliminar el producto. Inténtalo de nuevo."
4. Edit in another tab, then delete in this one: the version-conflict notice.
5. Delete the only product on the last page: you land on the previous page.
6. Create a product with the deleted product's SKU: it works.

**Accessibility**: complete US1 to US4 with the keyboard alone, and use a screen reader to check the field labels, the announced errors and the "N productos" count after filtering.
