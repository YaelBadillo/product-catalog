# Research: Product Management

Phase 0 of [plan.md](plan.md). Each entry records a decision, why, and what was rejected. Items marked **(Cn)** were flagged in the plan's Complexity Tracking; the user accepted every recommendation on 2026-11-25.

## Data and queries

### R1. Spanish alphabetical order for names (FR-020)

- **Decision**: The migration creates an ICU collation `catalog_es` (`provider = icu, locale = 'es-MX'`) and declares `products.name` with it, so `ORDER BY name` ignores case and accents at the primary level and puts `ñ` after `n`.
- **Rationale**: `postgres:18-alpine` uses musl, which has no libc locales, so ICU is the only option. Creating the collation in the migration avoids relying on the names `initdb` imports (`es-MX-x-icu`), which differ by platform. Declaring it on the column keeps queries free of `COLLATE`.
- **Alternatives**: Using `COLLATE "es-MX-x-icu"` in each query depends on imported names. Sorting in Go breaks pagination. A nondeterministic collation would also change equality, which this case doesn't need.
- **Verify first**: a repository test sorts `Zapato, árbol, Arroz, ñame, nuez` and expects `árbol, Arroz, nuez, ñame, Zapato`.

### R2. Case- and accent-insensitive partial search (FR-019)

- **Decision**: The migration enables `unaccent` and defines an `IMMUTABLE` wrapper `catalog_fold(text) = lower(unaccent('unaccent', $1))`. The query matches `catalog_fold(name) LIKE '%' || catalog_fold(@q) || '%' ESCAPE '\'`, and the same for `sku`. The repository escapes `\`, `%` and `_` in `q` before passing it as a parameter, so special characters match literally. `q` is trimmed, empty means no filter, and the maximum is 100 characters.
- **Rationale**: sqlc only needs the wrapper's signature, so it doesn't matter whether sqlc knows about extension functions. A sequential scan over 10,000 rows is well within SC-002, so no trigram index is needed (IX). `unaccent` is a trusted extension (PG13+) and ships in the official images and on Railway.
- **Alternatives**: `pg_trgm` with a GIN index on `catalog_fold(name)` would only pay off at much larger volumes. PG18 `LIKE` with a nondeterministic collation is newer, less familiar, and changes equality. Filtering on the client breaks pagination.

### R3. Dynamic sort, filters and total in sqlc

- **Decision**: sqlc only, with no query builder:
  - Optional filters use `sqlc.narg(...) IS NULL OR column = sqlc.narg(...)`.
  - `ORDER BY` uses one `CASE` per (column, direction) pair from the whitelist (`name`, `price`, `stock`, `created_at` × `asc`, `desc`).
  - Sorting by price always puts `currency ASC` first, then `price_cents` in the chosen direction. This groups by currency (MXN, then USD) when no currency is selected, and is harmless when one is.
  - Every sort ends with `id ASC` as a deterministic tiebreaker.
  - A separate `CountProducts` query repeats the same `WHERE`.
  - Both queries run in one read-only `REPEATABLE READ` transaction, so `total` and the page come from the same snapshot.
- **Rationale**: The whitelist is enforced twice, by the handler's enum check and by the `CASE` itself. Nothing from the input is concatenated into SQL (VI). Eight `CASE` branches are manageable, so `squirrel` isn't needed (IX).
- **Alternatives**: `squirrel` was kept as a fallback in the provided plan, but it's an extra dependency for a problem sqlc already solves. `COUNT(*) OVER ()` returns no total when the page is out of range, and FR-026 needs that total.

### R4. Product IDs

- **Decision**: `id uuid PRIMARY KEY DEFAULT uuidv7()`, the built-in in PG18.
- **Rationale**: IDs ordered by time give a tiebreaker that matches creation order and better index locality. No Go dependency is needed.
- **Alternatives**: `gen_random_uuid()` (v4) gives random order. Generating UUIDs in the app would need `google/uuid`.

### R5. Partial update (PATCH)

- **Decision**:
  - `UpdateProduct` sets each column to `COALESCE(sqlc.narg(col), col)`.
  - `description` uses `CASE WHEN @description_set THEN sqlc.narg(description) ELSE description END`, so it can be set to `NULL`.
  - The `WHERE` is `id = @id AND version = @version AND deleted_at IS NULL`, the `SET` includes `version = version + 1, updated_at = now()`, and the query ends with `RETURNING *`.
  - In the contract, `description` in the request is a `string`, never `null`. An empty or whitespace-only value removes it, and leaving the field out keeps it. This avoids the null-versus-absent ambiguity of Go pointers.
  - A PATCH with only `version` returns `422` (`minProperties: 2` in the contract), so an empty save never bumps the version.
  - The web sends only the fields that changed, plus `version`.
- **Alternatives**: JSON Merge Patch with a custom `Optional[T]` type needs more code for no gain. `PUT` with full replacement contradicts the `PATCH` already documented in CLAUDE.md.

### R6. Conflict vs. not found; duplicate SKU

- **Decision**:
  - When `UpdateProduct` or `SoftDeleteProduct` affects 0 rows, the repository runs `ProductExists(id)`, filtered by `deleted_at IS NULL`. It returns `ErrNotFound` if the product doesn't exist and `ErrVersionConflict` if it does.
  - A Postgres `23505` error on the constraint `products_sku_active_key` becomes `ErrSKUAlreadyExists`. Matching the constraint name keeps other unique violations from being mislabeled.
- **Rationale**: Matches the provided plan. The tiny window between the two statements can only turn a conflict into a not-found when someone deletes the product in between, and both outcomes are correct for the user.

### R7. Where validation lives **(C3)**

- **Decision**:
  - `internal/domain` owns normalization (`NormalizeSKU`, `NormalizeName`, `NormalizeDescription`) and every rule (FR-002 to FR-009), and returns `*domain.ValidationError{Fields []FieldError}`.
  - DTO tags check shape only (`required`, `oneof` for enums). The custom `sku` and `trimmed_len` validators call the domain's functions, so the rules are defined once and length is measured after trimming.
  - The service always normalizes and validates as well, because the seed and any future callers don't go through the handler.
  - Both kinds of error map to `422 VALIDATION_ERROR` with `details[]` that use JSON field names (`RegisterTagNameFunc`).
  - Messages are in English, because the API is for developers. The web shows its own Spanish texts (R19).
- **Why it was flagged**: In the provided plan, the handler validates with tags before the service normalizes. Plain tags on the raw input would reject valid names (120 characters plus surrounding spaces) and lowercase SKUs, which contradicts FR-002 and FR-003.

### R8. JSON decoding **(C5)**

- **Decision**: An own helper, `decodeJSON(c, &dto)`:
  - It uses `json.Decoder` with `DisallowUnknownFields`, rejects trailing data, and validates with a `*validator.Validate` injected into the handlers.
  - Syntax errors, unknown fields and wrong types return `400 VALIDATION_ERROR`, with `field` filled in when it's known. `*http.MaxBytesError` returns `413 PAYLOAD_TOO_LARGE`. Rule errors return `422`.
- **Why it was flagged**: `ShouldBindJSON` with unknown fields rejected and custom validators requires mutating global state in Gin's `binding` package (`EnableDecoderDisallowUnknownFields`, `binding.Validator.Engine()`). That goes against "no globals" (IV, api/CLAUDE.md). The alternative is to accept that one-time global configuration in `NewRouter` and keep the `ShouldBindJSON` convention as it is.

### R9. List query parameters

- **Decision**: `parseListParams(url.Values) (service.ListParams, []FieldError)` is an explicit function in the handler package with table-driven tests. It covers:
  - defaults (`page=1`, `page_size=20`, `sort=created_at`, `order=desc`) and ranges (`page ≥ 1`, `1 ≤ page_size ≤ 100`);
  - enums, `is_active` as `true|false`, and integer prices;
  - the cross-field rules: `min_price`/`max_price` require `currency`, and `min_price ≤ max_price`;
  - `q` at most 100 characters as received (the same bound as the contract's `maxLength`), then trimmed.

  Any error returns `400 VALIDATION_ERROR` with `details[].field` set to the parameter name. Unknown parameters are ignored.
- **Rationale**: The cross-field rules can't be expressed with tags, and Gin's query binding also uses its global validator.

## Platform

### R10. Migrations at startup **(C4)**

- **Decision**:
  - `internal/migrate.Up(ctx, pool)` uses the goose `Provider` with `migrations.FS` (`embed.FS`), a `*sql.DB` from `stdlib.OpenDBFromPool(pool)`, and `lock.NewPostgresSessionLocker()` so two instances never migrate at the same time. No goose globals are used.
  - `run()` migrates before serving, retrying with backoff for up to 30 s. If that fails, the process exits with an error, so Railway doesn't promote a deployment with an outdated schema and the previous one keeps serving.
  - `api migrate` doesn't exist; the seed migrates first (R15).
- **Why it was flagged**: The README documents "Readiness without blocking startup": the API starts even without a database. api/CLAUDE.md requires migrations at startup, which means connecting to the database before serving. The alternative is to migrate in the background and have `/readyz` report `migrations: pending` until they're applied, which needs more code and a change to the readiness contract.
- **Dependency**: `github.com/pressly/goose/v3` moves from tool to library in `go.mod`, at the same version as `tools/goose.mod` (v3.30.1).

### R11. Rate limiting (FR-043)

- **Decision**:
  - The middleware keeps one `rate.Limiter` per client IP, with `rate.Limit(5)` (300/min) and a burst of 300.
  - A janitor goroutine, stopped by the server's context, removes IPs that have been idle for more than 3 minutes.
  - A rejected request gets `429 RATE_LIMITED` with `Retry-After` in seconds.
  - It applies only to `/api/v1`, so health probes and `/docs` never return 429.
  - The client IP comes from `c.ClientIP()`. The router calls `SetTrustedProxies(nil)` so `X-Forwarded-For` is never trusted. If the new optional variable `CLIENT_IP_HEADER` is set, its value goes into `router.TrustedPlatform`. On Railway it's `X-Real-IP`: Railway staff say their edge overwrites that header (it isn't appended to), and they recommend it for IP-based rate limiting ([1], [2]). Without the variable (local, compose), the remote address of the connection is used.
- **Matches FR-043 as amended**: the token bucket is exactly "a sustained 300/min with bursts of up to 300". Request 301 of a burst gets 429. The first minute can reach about 600 if the burst arrives at the start, which the amended FR-043 allows. An exact per-minute window was rejected: it would mean keeping a timestamp per request.
- **Verify on deploy**: inspect `X-Real-IP` on the real deployment (community reports vary when there's a CDN in front) ([3]).
- **Dependency**: `golang.org/x/time` (rate), already prescribed by api/CLAUDE.md.

### R12. Body size limit

- **Decision**: `http.MaxBytesReader` at 64 KiB on `/api/v1` gives `413 PAYLOAD_TOO_LARGE`. That's a new error code in the contract and in CLAUDE.md. The largest valid product is about 3 KiB.
- **Alternatives**: Reusing `VALIDATION_ERROR` with 413 would mix two meanings in one code.

### R13. Security headers (API)

- **Decision**: One middleware sets `X-Content-Type-Options: nosniff`, `X-Frame-Options: DENY`, `Referrer-Policy: no-referrer`, `Strict-Transport-Security: max-age=31536000` (browsers ignore it over HTTP) and `Content-Security-Policy: default-src 'none'; frame-ancestors 'none'`. `/docs` overrides the CSP with its own policy (R14).
- **Scope**: API only, as the README's "deferred" list says. Hardening the web (CSP in nginx/Vercel) is out of scope for this feature.

### R14. Browsable documentation at `/docs`

- **Decision**:
  - `internal/apidocs` embeds `openapi.yaml` (a copy synced from the root with `make sync-openapi`) and an HTML page that loads `swagger-ui-dist` pinned to an exact version from jsDelivr, with SRI.
  - `GET /docs` serves the HTML with a CSP limited to `cdn.jsdelivr.net`, and `GET /docs/openapi.yaml` serves the contract.
  - CI checks that the copy matches the root file (`make sync-openapi && git diff --exit-code`).
- **Rationale**: `//go:embed` can't reach `../`, and the Docker build context is `api/`. A CDN avoids vendoring about 1.5 MB of assets and avoids adding dependencies.
- **Alternatives**: `swaggo/gin-swagger` generates the spec from annotations, which duplicates the contract. Vendoring `swagger-ui-dist` in the repository adds a large asset to maintain.

### R15. Sample data (`make seed`)

- **Decision**: An `api seed` subcommand in the same composition root migrates, builds the repository and service, and creates the 20 products from `internal/seed` through `service.Create`. On `ErrSKUAlreadyExists` it moves on, which makes it idempotent. `make seed` runs `cd api && go run ./cmd/api seed` with the `.env` `DATABASE_URL`.
- **Data**: All five categories, both currencies, 4 inactive products, and names with accents and ñ, which also cover R1/R2 by hand.
- **Note**: If a seeded product was deleted, its SKU is free and `seed` recreates it. That's expected, because the SKU no longer exists.

### R16. Contract validation in negative tests

- **Decision**: `testutil.AssertMatchesContract` validates the request as well, so tests that send invalid requests (bad UUID, missing `version`, query outside the enum) would fail before reaching the response. A sibling `AssertResponseMatchesContract` validates only the response, for those cases.
- **Verify first**: that kin-openapi accepts `type: [string, "null"]` (OpenAPI 3.1) for `description`. If it doesn't, use `anyOf: [{type: string}, {type: "null"}]`.

## Web

### R17. Routes and layout

- **Decision**:
  - `src/app/AppLayout.tsx` holds the header (black background, per web/CLAUDE.md) with `NavLink`s "Catálogo", "Nuevo producto" and "Estado", plus an `<Outlet>`.
  - The routes are `/` → redirect to `/products`, plus `/products`, `/products/new`, `/products/:id`, `/products/:id/edit` and `/status`.
  - An `:id` that isn't a UUID shows "Producto no encontrado" without calling the API (FR-016).
  - Each page sets `document.title` (for example, "Catálogo · Catálogo de productos").
- **Details**: [contracts/web-routes.md](contracts/web-routes.md).

### R18. URL as the catalog state (FR-023 to FR-025)

- **Decision**:
  - `search-params.ts` parses `URLSearchParams` with zod using `.catch()` per field, so invalid values fall back to defaults (FR-025). It drops any price range that has no currency or has min > max, and serializes while omitting defaults.
  - In the URL, prices are in currency units (`min_price=199.90`), so shared links are readable. They're converted to cents for the API.
  - Changing a filter, the sort or the page adds an entry to the history (and filter and sort changes go back to page 1).
  - Search uses `useDebouncedCallback` (300 ms, from `@mantine/hooks`). The first update of a typing session adds a history entry and the following ones replace it, so Back returns to the view before the search. Enter applies it immediately.
  - The price fields in the filter show the min > max error and don't touch the URL until it's fixed.

### R19. Errors and texts in Spanish (FR-045 to FR-047, SC-009)

- **Decision**:
  - `src/api/errors.ts` defines `ApiError` (`status`, `code`, `details`, `retryAfter`), which every function in `features/products/api.ts` throws.
  - `features/products/messages.ts` centralizes the FR-046 texts and the Spanish rule messages per field (FR-047). zod uses them.
  - For server errors with `details[]`, `setError(field, { message: messages.field[field] })` is called and the English message is ignored, so 100% of the text is in Spanish.
  - The status page and the render fallback are translated. `index.html` gets `lang="es-MX"` and the title "Catálogo de productos".

### R20. Server state (TanStack Query)

- **Decision**:
  - `query-keys.ts` defines `products.all`, `lists()`, `list(params)`, `details()` and `detail(id)`.
  - `useProduct(id)` always fetches with `GET /:id`.
  - **Create**: `setQueryData(detail)` plus invalidating `lists()`.
  - **Edit** **(C7)**: no optimistic update. On success, `setQueryData(detail)` with the response and invalidate `lists()`. A 409 leaves the form untouched.
  - **Delete**: optimistic removal in every cached `list` that contains the product, with a snapshot and rollback on error and invalidation on settle. If after the refetch `page > total_pages ≥ 1`, it navigates to `total_pages` (US4, scenario 8). In the list, the `version` comes from the list data (FR-035: "since they saw it").

### R21. Price input **(C2)**

- **Decision**:
  - Mantine `NumberInput` with `thousandSeparator=","`, `decimalSeparator="."`, `allowNegative={false}` and `clampBehavior="none"`, with no `decimalScale`. zod reports "más de 2 decimales" and "máximo 999,999,999.99" as field errors, as the spec's edge cases require.
  - Conversion happens only after zod: `Math.round(value * 100)`.
  - A test written first checks that a typed comma is not entered. Mantine's docs don't say what a typed comma does when it's the thousands separator ([4]). If the component lets it through, an `onKeyDown` drops it.
- **Why it was flagged**: The provided plan asks for "2 decimales" in the `NumberInput` (`decimalScale={2}`). That plus the default `clampBehavior="blur"` would truncate or clamp the value silently, while the spec requires showing an error.
- **Spec defect (C8)**: Dropping the comma doesn't stop `199,90` from becoming `19990` if the user keeps typing: the comma is lost, but the `90` is appended. The edge case "can never become 19,990.00" is therefore false. The field does show `19,990` as it's typed, so the mistake is visible but not prevented. Resolution: when a comma is dropped, the field shows the hint "Usa punto (.) para los decimales"; the spec's edge case and FR-005 were corrected.

### R22. Formatting (FR-012, FR-015) **(C1)**

- **Decision**:
  - `formatPrice(cents, currency)` = `Intl.NumberFormat("es-MX", { style: "currency", currency, currencyDisplay: "narrowSymbol" }).format(cents / 100) + " " + currency`, which gives `$1,299.90 MXN` and `$1,299.90 USD`.
  - `formatDateTime(iso)` = `Intl.DateTimeFormat("es-MX", { dateStyle: "medium", timeStyle: "short", hourCycle: "h23" })`, which gives `25 nov 2026, 14:30` in the user's time zone.
  - The exact output is checked in unit tests.
- **Why it was flagged**: The provided plan uses `{ style: "currency", currency }` as is. With `es-MX` that gives `$1,299.90` for MXN and `USD 1,299.90` for USD, which is the option the user rejected in the clarification.

### R23. Responsive list (FR-048) **(C6)**

- **Decision**: `useMediaQuery("(min-width: 48em)")` picks between `ProductTable` (Mantine `Table`) and `ProductCardList` (stacked cards with the same fields and actions). Only one is rendered, so neither tests nor screen readers see duplicates. Tests cover both layouts by changing the `matchMedia` stub.
- **Why it was flagged**: The provided plan asks for a `Table` alone. At 360 px, 6 columns plus actions force horizontal scrolling, which FR-048 forbids.

### R24. Accessibility (FR-040, SC-005)

- **Decision**:
  - The target is WCAG 2.2 AA. `src/test/axe.ts` exports `axe` from `configureAxe` with `runOnly` tags `wcag2a`, `wcag2aa`, `wcag21aa` and `wcag22aa`, and every test imports it instead of `vitest-axe`'s default.
  - An `aria-live="polite"` region announces "N productos" after each list update.
  - On a failed submit, react-hook-form already moves focus to the first invalid field (`shouldFocusError`). For server errors, `setError(..., { shouldFocus: true })` on the first one.
  - Confirmations use `@mantine/modals`, which traps focus and returns it on close.
  - Status is shown with text plus an icon, per web/CLAUDE.md.

### R25. Tests

- **Decision**:
  - **API**:
    - Domain: table-driven tests, plus a fuzz test for `NormalizeSKU`/`ValidateSKU` (idempotence; any normalized value that passes the regex).
    - Service: hand-written fakes.
    - Handlers: `httptest` with a fake service and contract checks.
    - Repository: `testcontainers-go` (`modules/postgres`, image `postgres:18-alpine`) through `testutil.StartPostgres` in `TestMain` (migrates once), with `testutil.ResetDB(t)` truncating per test, plus the concurrency case with two `UpdateProduct` calls on the same `version` (one gets `ErrVersionConflict`).
    - Middleware: rate limit with an injected clock or an IP map.
    - All integration tests skip under `-short`.
  - **Web**:
    - Per-page integration tests with MSW for every acceptance scenario.
    - Unit tests for `schema.ts`, `search-params.ts` and `format.ts`.
    - `axe` on every page, both layouts.
- **Dependency**: `github.com/testcontainers/testcontainers-go` (tests only), already prescribed by api/CLAUDE.md.

### R26. Observability

- **Decision**: The current per-request slog log stays as it is. The error mapper logs every unexpected error (500) with `request_id` and the wrapped cause, and the client only sees the generic message (VI). Rate-limit rejections are logged at `Info` with the IP, but never the query string. No metrics are added: neither the spec nor the platform requires them.

## Sources

- [1] [Which client-address headers does the Railway edge set, and are caller copies replaced?](https://station.railway.com/questions/which-client-address-headers-does-the-ra-fb9e36d7)
- [2] [Which header should I rely on for real client IP?](https://station.railway.com/questions/which-header-should-i-rely-on-for-real-c-d78a6f96)
- [3] [Edge/CDN layer leaking its own IP as client IP in X-Forwarded-For](https://station.railway.com/questions/edge-cdn-layer-leaking-its-own-ip-as-cli-1ce29b05)
- [4] [Mantine NumberInput](https://mantine.dev/core/number-input)
- [goose Provider](https://pressly.github.io/goose/documentation/provider), [PostgreSQL unaccent](https://www.postgresql.org/docs/18/unaccent.html)
