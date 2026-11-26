---

description: "Task list for feature 001-product-management"
---

# Tasks: Product Management

**Input**: Design documents from `specs/001-product-management/`: [plan.md](plan.md), [spec.md](spec.md), [research.md](research.md), [data-model.md](data-model.md), [contracts/products-api.yaml](contracts/products-api.yaml), [contracts/web-routes.md](contracts/web-routes.md), [quickstart.md](quickstart.md)

**Prerequisites**: plan.md (required), spec.md (required for user stories), research.md, data-model.md, contracts/

**Tests**: Test tasks are MANDATORY (constitution, Principle II). They derive from the acceptance criteria and precede the implementation tasks of their phase. Each must fail for the right reason before implementation starts. Never weaken a test to make it pass: if one looks wrong, check it against the spec.

**Organization**: Tasks are grouped by user story to enable independent implementation and testing of each story.

## Format: `[ID] [P?] [Story] Description`

- **[P]**: Can run in parallel (different files, no dependencies on incomplete tasks)
- **[Story]**: Which user story this task belongs to (US1–US4)
- `Rn` = decision in research.md, `FR-xxx`/`SC-xxx` = requirement in spec.md, `Cn` = accepted resolution in plan.md (Complexity Tracking)

## Path Conventions

Monorepo from plan.md: API in `api/` (Go, `handler → service → repository`), web in `web/src/` (feature `features/products`), contract `openapi.yaml` at the root. Go tests are black-box (`package xxx_test`). Integration tests use testcontainers and skip under `-short`. Every web page test uses MSW and includes an `axe` check.

---

## Phase 1: Setup (Shared Infrastructure)

**Purpose**: The contract first (Constitution III) and the tooling that every phase uses.

- [X] T001 Add the `sqlc` (`$(call go_tool,sqlc) generate`) and `sync-openapi` (copy `openapi.yaml` to `api/internal/apidocs/openapi.yaml`) targets to Makefile, under `##@ Generation`, each with its `##` description and listed in `.PHONY`
- [X] T002 Merge contracts/products-api.yaml into openapi.yaml:
  - paths `/api/v1/products`, `/api/v1/products/{id}`, `/docs`, `/docs/openapi.yaml`;
  - tags `products` and `docs`;
  - `components.schemas` and `components.responses`;
  - `info.version: 0.2.0`.

  Keep `/healthz`, `/readyz` and their schemas unchanged, then lint with `npx --yes @redocly/cli@2.80.0 lint openapi.yaml`.
- [X] T003 Run `make gen-types` (regenerates web/src/api/schema.d.ts) and `make sync-openapi` (creates api/internal/apidocs/openapi.yaml), then `make test-unit` to confirm the existing health tests still pass against the merged contract
- [X] T004 [P] Add a step "OpenAPI copy in sync" (`make sync-openapi && git diff --exit-code api/internal/apidocs/openapi.yaml`, run from the repo root) to the `API (Go)` job in .github/workflows/ci.yml, without renaming any job

---

## Phase 2: Foundational (Blocking Prerequisites)

**Purpose**: Schema, domain, error envelope, middleware and web shell that every story needs.

**⚠️ CRITICAL**: No user story work can begin until this phase is complete.

### API: test infrastructure and database

- [ ] T005 [P] Write tests for `AssertResponseMatchesContract` in api/internal/testutil/openapi_test.go, using the existing `fakeT`/`runFake`:
  - A request outside the contract (`GET /api/v1/products/abc`) with a documented `400` error envelope passes.
  - An undocumented status or an invalid body fails.
  - A `200` for `GET /api/v1/products/{id}` with `"description": null` passes. This verifies that kin-openapi accepts `type: [string, "null"]` (R16).
- [ ] T006 Implement `AssertResponseMatchesContract` in api/internal/testutil/openapi.go: find the route, skip request validation, validate the response with `IncludeResponseStatus`, and share the contract loading with `AssertMatchesContract`. If T005 shows that kin-openapi rejects the type array, change `Product.description` to `anyOf` in openapi.yaml and in contracts/products-api.yaml, then rerun T003 (R16).
- [ ] T007 [P] Create api/internal/testutil/postgres.go:
  - `StartPostgres(ctx)` starts `postgres:18-alpine` with testcontainers-go `modules/postgres` (`go get` both modules), for use in a package's `TestMain`, and returns the DSN and a terminate func.
  - `ResetDB(t, pool)` truncates `products`.
  - Under `-short`, the helpers make the callers skip.
- [ ] T008 Write migration tests in api/internal/migrate/migrate_test.go (integration, `TestMain` with `testutil.StartPostgres`):
  - `Up` on an empty database creates `products`.
  - A second `Up` is a no-op.
  - Two concurrent `Up` calls both succeed (session locker).
  - `Up` against an unreachable DSN retries and returns an error once a short injected timeout expires.
- [ ] T009 Write schema tests in api/internal/migrate/schema_test.go (raw SQL after `Up`), covering the constraints of data-model.md Product.sku, Product.name, Product.description, Product.price_cents, Product.currency, Product.stock, Product.category and Product.version, and the indexes:
  - Each `CHECK` rejects an invalid row.
  - Defaults apply: Product.currency `MXN`, Product.is_active `true`, Product.version `1`, Product.id non-null and time-ordered (`uuidv7`), Product.created_at and Product.updated_at set.
  - A duplicate active SKU fails with `23505` on `products_sku_active_key`, but succeeds when the other row has Product.deleted_at set.
  - `ORDER BY name` returns `árbol, Arroz, nuez, ñame, Zapato` (R1).
  - `catalog_fold('Café')` returns `cafe` and `catalog_fold('Ñandú')` returns `nandu` (R2).
- [ ] T010 Create the migration with `make migrate-new name=create_products` (api/migrations/<timestamp>_create_products.sql). In goose `Up`/`Down` sections:
  - `CREATE EXTENSION IF NOT EXISTS unaccent`, the ICU collation `catalog_es` and the `IMMUTABLE` function `catalog_fold` (R1, R2);
  - the `products` table per data-model.md Product.id, Product.sku, Product.name, Product.description, Product.price_cents, Product.currency, Product.stock, Product.category, Product.is_active, Product.version, Product.created_at, Product.updated_at and Product.deleted_at, with every `CHECK`;
  - the four partial indexes from data-model.md "Indexes".

  `Down` drops everything in reverse order.
- [ ] T011 [P] Create api/migrations/embed.go exporting `FS embed.FS` with `//go:embed *.sql`
- [ ] T012 Implement `Up(ctx, pool, opts)` in api/internal/migrate/migrate.go (R10, C4):
  - goose `NewProvider(DialectPostgres, stdlib.OpenDBFromPool(pool), migrations.FS)` with `lock.NewPostgresSessionLocker()`, and no goose globals;
  - retries with backoff for up to 30 s by default (configurable for tests).

  `go get github.com/pressly/goose/v3@v3.30.1`, the same version as api/tools/goose.mod.
- [ ] T013 Add `NewMigratedPool(ctx, dsn)` to api/internal/testutil/postgres.go: it opens a `pgxpool` and applies `migrate.Up`. Repository tests use it.
- [ ] T014 Write black-box startup tests in api/cmd/api/main_test.go (integration, skipped under `-short`): build the binary into `t.TempDir()` and start it with `DATABASE_URL` from `testutil.StartPostgres`, a free `PORT` and a valid `CORS_ALLOWED_ORIGINS`. Check that `/healthz` answers 200 and that the `products` table exists, which proves migrations ran at startup. The failure path (`migrate.Up` returns an error once its retry window expires) is covered by T008.
- [ ] T015 Call `migrate.Up` in `run()` in api/cmd/api/main.go after creating the pool and before `ListenAndServe`. If it fails, return the error so the process exits (C4). This makes the startup test pass.

### API: domain

- [ ] T016 [P] Write table-driven domain tests in api/internal/domain/product_test.go, covering spec Edge Cases and FR-002 to FR-009:
  - `ProductInput.Normalize`/`Validate`: SKU trim and uppercase, inner spaces and `Ñ` rejected, name lengths after trim, whitespace-only description → nil, price bounds, stock bounds, enums, defaults.
  - `ProductPatch.Validate` and `IsEmpty`.
  - `Category.Valid`, `Currency.Valid`.
  - `*ValidationError` lists every failing field with its JSON name.
- [ ] T017 [P] Write `FuzzNormalizeSKU` in api/internal/domain/sku_fuzz_test.go:
  - normalization is idempotent;
  - any value `ValidateSKU` accepts matches `^[A-Z0-9-]{4,32}$`;
  - it never panics.
- [ ] T018 Implement api/internal/domain/errors.go (`ErrNotFound`, `ErrSKUAlreadyExists`, `ErrVersionConflict`) and api/internal/domain/validation.go (`ValidationError{Fields []FieldError}`, `FieldError{Field, Message}`, English messages)
- [ ] T019 Implement api/internal/domain/product.go:
  - `Product`, `Category`, `Currency` (constants and `Valid`), `ProductInput`, `ProductPatch`;
  - `NormalizeSKU`, `NormalizeName`, `NormalizeDescription`, `ValidateSKU`;
  - rules per data-model.md Product.sku, Product.name, Product.description, Product.price_cents, Product.currency, Product.stock, Product.category, Product.is_active and Product.version (R7, C3).

### API: HTTP foundation

- [ ] T020 [P] Write tests for the error middleware in api/internal/httpapi/errors_test.go. Each case is a handler that calls `c.Error(err)`, and each response is checked for envelope shape (`details` always an array):

  | Error | Expected |
  |---|---|
  | `domain.ErrNotFound` | 404 `NOT_FOUND` |
  | `ErrSKUAlreadyExists` | 409 `SKU_ALREADY_EXISTS` |
  | `ErrVersionConflict` | 409 `VERSION_CONFLICT` |
  | `*domain.ValidationError` | 422 `VALIDATION_ERROR` with `details` |
  | `*handler.RequestError` | 400 `VALIDATION_ERROR` with `field` |
  | `*http.MaxBytesError` | 413 `PAYLOAD_TOO_LARGE` |
  | `*middleware.RateLimitedError` | 429 `RATE_LIMITED` + `Retry-After` |
  | Any other error | 500 `INTERNAL_ERROR` with the generic message and request ID; the cause and `request_id` are logged (R26) |
- [ ] T021 [P] Write tests for `decodeJSON` and the validator in api/internal/httpapi/handler/decode_test.go:
  - Unknown field → `RequestError{Field: "color"}`; syntax error; wrong type, with the field; trailing data → `RequestError`.
  - `*http.MaxBytesError` passes through unchanged.
  - Rule failures → `*domain.ValidationError` with JSON names.
  - The custom tags delegate to the domain: `sku` accepts `"  abc-123 "`, and `trimmed_len` accepts 120 characters plus surrounding spaces (R7, R8, C3, C5).
- [ ] T022 Implement api/internal/httpapi/handler/decode.go (`decodeJSON(c, v, validate)`, `RequestError`) and api/internal/httpapi/handler/validator.go:
  - `NewValidator()` with `RegisterTagNameFunc` (JSON names) and the custom tags `sku` and `trimmed_len`, which call `internal/domain`;
  - translation of `validator.ValidationErrors` to `*domain.ValidationError`.

  `go get github.com/go-playground/validator/v10` as a direct dependency.
- [ ] T023 [P] Write tests in api/internal/httpapi/middleware/security_test.go: every response carries the R13 headers
- [ ] T024 [P] Write tests in api/internal/httpapi/middleware/bodylimit_test.go: a body of 64 KiB + 1 makes the next handler's read fail with `*http.MaxBytesError`; 64 KiB passes (R12)
- [ ] T025 [P] Write tests in api/internal/httpapi/middleware/ratelimit_test.go (injected clock):
  - 300 requests from one IP pass and the 301st is aborted with `*RateLimitedError` carrying `RetryAfter`.
  - Another IP is unaffected.
  - Tokens refill at 5/s.
  - The janitor evicts IPs idle for more than 3 min and stops when its context is canceled (R11).
- [ ] T026 [P] Implement `SecurityHeaders()` in api/internal/httpapi/middleware/security.go (R13)
- [ ] T027 [P] Implement `BodyLimit(64 << 10)` with `http.MaxBytesReader` in api/internal/httpapi/middleware/bodylimit.go (R12)
- [ ] T028 [P] Implement `RateLimit(ctx, opts)` in api/internal/httpapi/middleware/ratelimit.go: a per-IP `rate.Limiter` (5/s, burst 300) keyed by `c.ClientIP()`, a janitor, and `RateLimitedError` reported with `c.Error` + `Abort`. `go get golang.org/x/time` (R11).
- [ ] T029 Implement the error middleware in api/internal/httpapi/errors.go, the single domain → HTTP mapping: it reads `c.Errors.Last()` after `c.Next()` and maps the types tested in T020, including `handler.RequestError` (T022) and `middleware.RateLimitedError` (T028), which it needs to compile. Add the code constants (`PAYLOAD_TOO_LARGE`, `RATE_LIMITED`, …) and keep the existing `recovery`.
- [ ] T030 [P] Write tests in api/internal/config/config_test.go: `CLIENT_IP_HEADER` is optional (default empty) and is read when set
- [ ] T031 Add `ClientIPHeader string` (`env:"CLIENT_IP_HEADER"`) to `Config` in api/internal/config/config.go
- [ ] T032 Write router tests in api/internal/httpapi/router_test.go (update `newRouter` to the new signature):
  - security headers on every response;
  - `/healthz` returns 200 for 400 consecutive requests (never rate-limited);
  - a test route added to `/api/v1` through a route registrar returns 429 at request 301;
  - `X-Forwarded-For` is ignored;
  - with `ClientIPHeader: "X-Real-IP"`, two values get separate buckets.
- [ ] T033 Update api/internal/httpapi/router.go and its wiring in api/cmd/api/main.go:
  - `NewRouter(ctx, logger, cfg RouterConfig, health, api ...func(*gin.RouterGroup))`: `RouterConfig` holds the allowed origins and the client IP header, and each `api` registrar adds routes to the `/api/v1` group (the product routes later, a test route in the router tests);
  - `SetTrustedProxies(nil)` and `TrustedPlatform = cfg.ClientIPHeader`;
  - the error middleware and `SecurityHeaders` globally;
  - an `/api/v1` group with `BodyLimit` and `RateLimit`.

### Web: foundation

- [ ] T034 [P] Write tests in web/src/api/errors.test.ts:
  - `toApiError(response, body)` builds an `ApiError` from the envelope (`status`, `code`, `details`) and reads `Retry-After`;
  - a non-JSON body or network failure gives `code: "INTERNAL_ERROR"`.
- [ ] T035 [P] Implement web/src/api/errors.ts (`ApiError`, `toApiError`) per data-model.md ApiError (R19)
- [ ] T036 [P] Write tests in web/src/features/products/format.test.ts, with the time zone pinned to `America/Mexico_City`:
  - `formatPrice(129990, "MXN")` → `$1,299.90 MXN`, `formatPrice(129990, "USD")` → `$1,299.90 USD`, `formatPrice(19990, "MXN")` → `$199.90 MXN`;
  - `formatDateTime` → `25 nov 2026, 14:30`.
- [ ] T037 [P] Implement `formatPrice` (`currencyDisplay: "narrowSymbol"` + the code) and `formatDateTime` (`es-MX`, `dateStyle: "medium"`, `timeStyle: "short"`, `hourCycle: "h23"`) in web/src/features/products/format.ts (C1, R22)
- [ ] T038 [P] Create web/src/features/products/messages.ts with:
  - every text in spec FR-046, including the comma hint (C8);
  - the per-field rule messages (FR-047) for data-model.md Product.sku, Product.name, Product.description, Product.price_cents, Product.stock, Product.category and Product.currency;
  - the Spanish category labels (data-model.md Product.category) and status labels ("Activo"/"Inactivo").
- [ ] T039 [P] Create web/src/features/products/query-keys.ts (`all`, `lists()`, `list(params)`, `details()`, `detail(id)`) (R20)
- [ ] T040 [P] Create web/src/features/products/testing.ts:
  - `buildProduct(overrides)`, typed with `components["schemas"]["Product"]`;
  - MSW handler factories for GET (list and detail), POST, PATCH and DELETE, including error envelopes, all built with `apiUrl`.
- [ ] T041 [P] Write tests in web/src/components/ui/PageState.test.tsx: it renders the title, text and optional action, and passes `axe`
- [ ] T042 [P] Implement web/src/components/ui/PageState.tsx: a generic Mantine composition for empty, error and not-found states, with no business logic
- [ ] T043 Set up the shared web test infrastructure:
  - Move the route objects to web/src/app/routes.tsx (the file exports only the route array, to keep `react-refresh` happy).
  - Add `renderRoute(path)` to web/src/test/render.tsx: the same providers as `renderWithProviders`, plus `createMemoryRouter(routes)` and `RouterProvider`. It returns the `router` so tests can navigate and go back.
  - Create web/src/test/axe.ts exporting `axe = configureAxe({ runOnly: { type: "tag", values: ["wcag2a", "wcag2aa", "wcag21aa", "wcag22aa"] } })` (R24), and switch web/src/app/App.test.tsx and web/src/features/status/pages/StatusPage.test.tsx to import it. Every later test imports `axe` from there.
- [ ] T044 Write routing tests in web/src/app/App.test.tsx (update the existing one):
  - `/` redirects to `/products`;
  - `/status` shows the status page in Spanish;
  - an unknown path shows "Página no encontrada" with "Volver al catálogo";
  - the header shows "Catálogo", "Nuevo producto" and "Estado", with `aria-current="page"` on the active link;
  - `axe` passes.
- [ ] T045 [P] Update web/src/features/status/pages/StatusPage.test.tsx to expect Spanish texts (FR-045; the requirement changed in the spec first)
- [ ] T046 Translate web/src/features/status/pages/StatusPage.tsx, web/src/features/status/components/StatusIndicator.tsx and the render fallback in web/src/app/App.tsx to Spanish
- [ ] T047 Implement web/src/app/AppLayout.tsx (black header with white text per web/CLAUDE.md Colors, `NavLink`s, `<Outlet>`) and web/src/app/NotFoundPage.tsx (with `PageState`). Register in web/src/app/routes.tsx:
  - the layout;
  - `/` → `<Navigate to="/products" replace>`;
  - `/status`;
  - `*`.
- [ ] T048 [P] Set `lang="es-MX"` and `<title>Catálogo de productos</title>` in web/index.html

**Checkpoint**: The foundation is ready. `make test-unit` and `make test-api` pass. Until US2 lands, `/products` shows "Página no encontrada".

---

## Phase 3: User Story 1 - Register a product and view its detail (Priority: P1) 🎯 MVP

**Goal**: Create a product from the form and see its complete detail, with SKU normalization, Spanish errors on each field, a duplicate-SKU message, and a not-found view.

**Independent Test**: Create a product with valid data, land on its detail, and check every field (normalized SKU, `$199.90 MXN`). Repeat the SKU and see the error on the field. Open a nonexistent ID and see "Producto no encontrado".

### Tests for User Story 1 ⚠️

> Write these tests FIRST and make sure they FAIL before implementation.

- [ ] T049 [P] [US1] Write repository tests in api/internal/repository/products_test.go, plus api/internal/repository/main_test.go (`TestMain`: `StartPostgres` + `NewMigratedPool`; `ResetDB` per test):
  - `Create` returns the server fields (`id`, `version: 1`, timestamps) and stores `description` as NULL;
  - a duplicate active SKU → `domain.ErrSKUAlreadyExists`;
  - two concurrent `Create` calls with the same SKU: exactly one succeeds and the other gets `ErrSKUAlreadyExists` (spec Edge Cases);
  - `Get` by id;
  - `Get` of an unknown id → `domain.ErrNotFound`.
- [ ] T050 [P] [US1] Write service tests in api/internal/service/products_test.go, with a hand-written fake repository:
  - `Create` normalizes (`"  abc-123 "` → `ABC-123`, name trimmed, blank description → nil) and applies the defaults (MXN, active);
  - a `*domain.ValidationError` doesn't call the repository;
  - `ErrSKUAlreadyExists` propagates;
  - `Get` returns the product or `ErrNotFound`.
- [ ] T051 [P] [US1] Write handler tests in api/internal/httpapi/products_test.go, through `httpapi.NewRouter` with a fake service. Every response is validated with `AssertMatchesContract`, or with `AssertResponseMatchesContract` for invalid requests. Cases are quickstart.md API scenarios 1 to 5:
  - POST → 201 with `Location: /api/v1/products/{id}`;
  - 409 `SKU_ALREADY_EXISTS`;
  - 422 with `details` for `name`, `price_cents` and `category`;
  - 400 for the unknown field `color`, malformed JSON and a wrong type;
  - GET → 200, 404, and 400 for `abc`;
  - **platform responses on real routes** (Constitution III): a POST body over 64 KiB → 413 `PAYLOAD_TOO_LARGE`; request 301 from one IP → 429 `RATE_LIMITED` with `Retry-After`; an unexpected error from the fake service → 500 `INTERNAL_ERROR` with the generic message and request ID.
- [ ] T052 [P] [US1] Write tests in web/src/features/products/schema.test.ts for `productFormSchema` and `toCreateBody`:
  - a lowercase or space-padded SKU is valid;
  - name length is counted after trimming;
  - price `0`, more than 2 decimals and above 999,999,999.99 each give their FR-047 message;
  - stock must be a whole number ≥ 0;
  - category is required;
  - `toCreateBody` trims, uppercases the SKU, sends `price_cents = Math.round(price * 100)` and omits an empty description.
- [ ] T053 [P] [US1] Write tests in web/src/features/products/pages/ProductCreatePage.test.tsx (`renderRoute("/products/new")`):
  - **US1 scenarios 1 to 6 and 9**: MXN, active and an empty category by default. A valid submission sends the POST and shows "Producto creado", then the detail opens. Client errors appear on each field and focus moves to the first one. A 422 maps to the Spanish message per field. A 409 shows "Ya existe un producto con este SKU". A 500 shows "Algo salió mal. Inténtalo de nuevo." and keeps the values. A 429 shows "Demasiadas solicitudes. Espera un momento e inténtalo de nuevo." and keeps the values (FR-043). The button is disabled while the request is pending.
  - The price shows thousands separators. A typed comma isn't entered and "Usa punto (.) para los decimales" appears (C8).
  - `axe` passes.
- [ ] T054 [P] [US1] Write tests in web/src/features/products/pages/ProductDetailPage.test.tsx:
  - **US1 scenarios 7 and 8, and FR-015**: every field shows, including `$199.90 MXN`, "Sin descripción", the dates as `25 nov 2026, 14:30`, the category in Spanish, and the status as text and icon. The product's internal `id` doesn't appear anywhere on the page.
  - A 120-character name and a 2000-character description appear in full (spec Edge Cases).
  - "Volver al catálogo" returns to the previous catalog view, filters included, when the user came from the catalog, and goes to `/products` otherwise (FR-023, contracts/web-routes.md).
  - A 404 shows "Producto no encontrado", "Es posible que se haya eliminado." and "Volver al catálogo".
  - `/products/abc` shows not-found without a request (MSW `onUnhandledFrame: "error"` proves it).
  - Loading state; an error shows "Reintentar", which refetches.
  - `document.title` is set.
  - `axe` passes.

### Implementation for User Story 1

- [ ] T055 [US1] Write the `CreateProduct` and `GetProduct` queries (`deleted_at IS NULL`) in api/queries/products.sql and run `make sqlc` to generate api/internal/repository/db/
- [ ] T056 [P] [US1] Add a "sqlc diff" step (`go tool -modfile=tools/sqlc.mod sqlc diff`) to the `API (Go)` job in .github/workflows/ci.yml
- [ ] T057 [US1] Implement api/internal/repository/products.go:
  - `NewProducts(pool)`;
  - `Create`, translating `23505` on `products_sku_active_key` to `domain.ErrSKUAlreadyExists` (R6);
  - `Get`, translating `pgx.ErrNoRows` to `domain.ErrNotFound`;
  - mapping between the db and domain types per data-model.md Product.id, Product.description (NULL ↔ nil), Product.price_cents, Product.stock and Product.version.
- [ ] T058 [US1] Implement api/internal/service/products.go: the `ProductRepository` interface (only `Create` and `Get` for now), `NewProducts(repo)`, and `Create` (domain normalization and validation, defaults per data-model.md Product.currency and Product.is_active) and `Get`
- [ ] T059 [US1] Implement api/internal/httpapi/handler/products.go and api/internal/httpapi/handler/dto.go, and wire repository → service → handler in api/cmd/api/main.go:
  - **Handler**: a `ProductService` interface defined by the handler; `Create` (decode with `decodeJSON`, then 201 with `Location`); `Get` (an invalid UUID gives a `RequestError` → 400).
  - **DTOs**: `CreateProductRequest`, with shape tags per data-model.md Product.sku, Product.name, Product.description, Product.price_cents, Product.currency, Product.stock, Product.category and Product.is_active; `ProductResponse`, where Product.description is `*string` → `null`.
  - **Routes**: `(*Products).Register(*gin.RouterGroup)` adds `POST /products` and `GET /products/:id`. main.go passes it to `NewRouter`, which mounts it under `/api/v1`.
- [ ] T060 [P] [US1] Implement `productFormSchema` (zod, messages from messages.ts) and `toCreateBody` in web/src/features/products/schema.ts, per data-model.md ProductFormValues.sku, ProductFormValues.name, ProductFormValues.description, ProductFormValues.price, ProductFormValues.currency, ProductFormValues.stock, ProductFormValues.category and ProductFormValues.isActive. The rules are those of Product.sku, Product.name, Product.description, Product.price_cents, Product.stock and Product.category. Use no `decimalScale`: the 2-decimal check is the schema's (C2).
- [ ] T061 [US1] Implement `createProduct` and `getProduct` in web/src/features/products/api.ts (throwing `ApiError`) and `useProduct(id)` and `useCreateProduct` (on success: `setQueryData(detail)` + invalidate `lists()`) in web/src/features/products/hooks.ts
- [ ] T062 [US1] Implement web/src/features/products/components/ProductForm.tsx, the form shared by create and edit (react-hook-form + `zodResolver`; `Select` and `NumberInput` through `Controller`):
  - **Price** (R21, C2, C8): `NumberInput` with `thousandSeparator=","`, `decimalSeparator="."`, `allowNegative={false}` and `clampBehavior="none"`. A typed comma is dropped and shows the hint.
  - **Edit mode**: a read-only SKU, and the submit button disabled while the form isn't dirty.
  - **Server errors**: mapped to fields with `setError(field, messages.field[field], { shouldFocus })`.
  - The button is disabled while the request is pending.
- [ ] T063 [US1] Implement web/src/features/products/pages/ProductCreatePage.tsx (notifications "Producto creado" / "Algo salió mal…", navigation to `/products/:id`, `document.title`) and register `/products/new` in web/src/app/routes.tsx
- [ ] T064 [US1] Implement web/src/features/products/components/ProductDetails.tsx (presentational) and web/src/features/products/pages/ProductDetailPage.tsx:
  - an `:id` that isn't a UUID → not-found with no request;
  - loading, not-found and error states with `PageState`;
  - a "Volver al catálogo" link: `navigate(-1)` when the previous history entry is the catalog, `/products` otherwise (contracts/web-routes.md);
  - `document.title`.

  Register `/products/:id` in web/src/app/routes.tsx.

**Checkpoint**: US1 works on its own, tested API to UI. This is the MVP.

---

## Phase 4: User Story 2 - Browse the catalog (Priority: P2)

**Goal**: A paginated catalog with filters, accent-insensitive search, sorting (Spanish order, prices grouped by currency) and every state, all reflected in the URL.

**Independent Test**: With a varied product set, apply each filter, the search and each sort, and check the results, the URL, reload and Back (quickstart.md, US2).

### Tests for User Story 2 ⚠️

- [ ] T065 [P] [US2] Write repository tests in api/internal/repository/products_list_test.go:
  - **Filters**: category, `is_active`, currency, and a price range (inclusive) combine.
  - **Search** (R2): partial match, ignoring case and accents (`cafe` → `Café molido`, `0123` → `CAF-0123`); `%`, `_` and `\` match literally.
  - **Sort**: by each field in both directions. Price groups MXN first, then USD. Name follows Spanish order (R1).
  - **Pagination**: ties never repeat or skip a product across pages; `total` respects the filters; a page beyond the last returns empty with the real `total`.
  - Rows with `deleted_at` set via SQL are excluded.
- [ ] T066 [P] [US2] Write service tests in api/internal/service/products_list_test.go: `List` passes `ListParams` through and returns `ProductPage`; repository errors propagate
- [ ] T067 [P] [US2] Write table-driven tests for `parseListParams` in api/internal/httpapi/handler/params_test.go:
  - defaults;
  - invalid `page`, `page_size`, enums, `is_active` and prices → `FieldError` naming the parameter;
  - `min_price`/`max_price` without `currency` → an error on that parameter;
  - min > max;
  - `q` longer than 100 characters as received is an error; otherwise it's trimmed;
  - unknown parameters are ignored.
- [ ] T068 [P] [US2] Write handler tests in api/internal/httpapi/products_list_test.go: `GET /api/v1/products` → 200 with `meta` (`page`, `page_size`, `total`, `total_pages`, which is 0 when `total` is 0) that matches the contract; the parameters reach the fake service; quickstart.md API scenarios 8, 9 and 12
- [ ] T069 [P] [US2] Write tests in web/src/features/products/search-params.test.ts, per contracts/web-routes.md:
  - parsing with defaults;
  - sanitizing invalid values (FR-025);
  - the price range is dropped without a currency or when min > max;
  - `q` is trimmed and cut to 100;
  - serializing omits defaults;
  - `toListQuery` converts units to cents and sets `page_size: 20`.
- [ ] T070 [P] [US2] Write tests in web/src/features/products/pages/ProductListPage.test.tsx (`renderRoute("/products?...")`), covering US2 scenarios 1 to 12:
  - **Rows**: columns with Spanish labels and formatted price; a 120-character name is truncated visually but stays complete in its accessible name.
  - **Filters**: each one updates the URL and the request.
  - **Search**: applies after 300 ms or on Enter; Back returns to the view before the search.
  - **Price range**: disabled without a currency, with "Elige una moneda para filtrar por precio". Clearing the currency clears the range. Min > max shows an error and sends no request.
  - **Sort and pages**: sort and order work; changing a filter returns to page 1.
  - **States**: loading; "Aún no hay productos" with a link to "Registrar producto"; "Ningún producto coincide con tu búsqueda o filtros" with "Limpiar filtros"; an error with "Reintentar"; a 429 with "Demasiadas solicitudes. Espera un momento e inténtalo de nuevo." and "Reintentar" (FR-043); "Esta página no tiene productos" with "Ir a la primera página".
  - **Navigation**: a row → detail, and Back restores the query.
  - **Accessibility**: an `aria-live` region announces "N productos".
  - **Layouts**: table and cards (`matchMedia` stub true and false), with `axe` on each.

### Implementation for User Story 2

- [ ] T071 [US2] Write the `ListProducts` query (optional filters with `sqlc.narg`, `catalog_fold … LIKE … ESCAPE '\'`, the `CASE` sort with `currency` first for price and `id` as tiebreaker, `LIMIT`/`OFFSET`) and the `CountProducts` query (same `WHERE`) in api/queries/products.sql, then run `make sqlc` (R2, R3)
- [ ] T072 [US2] Implement `List` in api/internal/repository/products.go: escape `\`, `%` and `_` in data-model.md ListParams.Query; run both queries in a read-only `REPEATABLE READ` transaction; map data-model.md ListParams.Sort and ListParams.Order to the query's parameters
- [ ] T073 [US2] Implement `ListParams`, `SortField`, `SortOrder`, `ProductPage` and `List` in api/internal/service/products.go (add `List` to `ProductRepository`)
- [ ] T074 [US2] Implement `parseListParams(url.Values)` in api/internal/httpapi/handler/params.go, per data-model.md ListParams.Page, ListParams.PageSize, ListParams.Category, ListParams.IsActive, ListParams.Currency, ListParams.MinPriceCents, ListParams.MaxPriceCents, ListParams.Query, ListParams.Sort and ListParams.Order (R9)
- [ ] T075 [US2] Implement the `List` handler and the `ProductList`/`PageMeta` DTOs in api/internal/httpapi/handler/products.go and api/internal/httpapi/handler/dto.go (`total_pages = ceil(total / page_size)`), and add `GET /products` to `(*Products).Register`
- [ ] T076 [P] [US2] Implement `parseCatalogSearch`, `serializeCatalogSearch` and `toListQuery` in web/src/features/products/search-params.ts, per data-model.md CatalogSearch.page, CatalogSearch.category, CatalogSearch.isActive, CatalogSearch.currency, CatalogSearch.minPrice, CatalogSearch.maxPrice, CatalogSearch.q, CatalogSearch.sort and CatalogSearch.order, and contracts/web-routes.md (R18)
- [ ] T077 [US2] Implement `listProducts` in web/src/features/products/api.ts and `useProducts(search)` (`placeholderData: keepPreviousData`) in web/src/features/products/hooks.ts
- [ ] T078 [US2] Implement `useCatalogSearch()` in web/src/features/products/use-catalog-search.ts:
  - URL state with `useSearchParams`;
  - filter, sort and page changes push a history entry, and filter and sort changes reset to page 1;
  - search uses `useDebouncedCallback` (300 ms) and pushes on the first update of a typing session, then replaces;
  - Enter applies immediately (R18).
- [ ] T079 [P] [US2] Implement web/src/features/products/components/ProductFilters.tsx and web/src/features/products/components/ResultsAnnouncer.tsx (`aria-live="polite"`):
  - category, status and currency selects;
  - the search box;
  - the price range, disabled without a currency (with its hint) and with the min > max error.
- [ ] T080 [P] [US2] Implement web/src/features/products/components/ProductTable.tsx (Mantine `Table`) and web/src/features/products/components/ProductCardList.tsx (stacked cards). Both show the same fields and Spanish labels, show status as text and icon, and link to the detail while preserving the catalog query (R23, C6).
- [ ] T081 [US2] Implement web/src/features/products/pages/ProductListPage.tsx and register `/products` in web/src/app/routes.tsx:
  - `useMediaQuery("(min-width: 48em)")` picks table or cards;
  - `Pagination`;
  - every FR-026 state with `PageState`;
  - `document.title`.

**Checkpoint**: US1 and US2 both work on their own.

---

## Phase 5: User Story 3 - Edit a product (Priority: P3)

**Goal**: Edit every field except the SKU, with no-change, version-conflict and not-found handling.

**Independent Test**: Edit each field and check the detail. Save the same product from two tabs: the second save gets the conflict alert, keeps its values, and offers "Recargar".

**Depends on**: US1 (`ProductForm`, detail). The edit action in the list needs US2.

### Tests for User Story 3 ⚠️

- [ ] T082 [P] [US3] Write repository tests in api/internal/repository/products_update_test.go:
  - `Update` changes only the provided fields;
  - `description_set` with nil sets NULL;
  - `version` +1 and `updated_at` change;
  - a wrong version → `ErrVersionConflict` with the row unchanged;
  - missing or deleted → `ErrNotFound`;
  - **concurrency**: two goroutines update with the same version, and exactly one succeeds while the other gets `ErrVersionConflict`.
- [ ] T083 [P] [US3] Write service tests in api/internal/service/products_update_test.go:
  - `Update` normalizes and validates the patch (domain);
  - an empty patch → `*domain.ValidationError`;
  - a blank description clears it;
  - `ErrVersionConflict` and `ErrNotFound` propagate.
- [ ] T084 [P] [US3] Write handler tests in api/internal/httpapi/products_update_test.go (contract-validated). Cases are quickstart.md API scenarios 13 to 17:
  - `PATCH` → 200;
  - 400 for `sku` in the body, malformed JSON and an invalid UUID;
  - 422 for `version` alone and for invalid fields;
  - 404;
  - 409 `VERSION_CONFLICT`.
- [ ] T085 [P] [US3] Write tests for `toPatchBody(dirtyFields, values, version)` in web/src/features/products/schema.test.ts: only changed fields plus `version`; an emptied description → `""`; price in cents
- [ ] T086 [P] [US3] Write tests in web/src/features/products/pages/ProductEditPage.test.tsx, covering US3 scenarios 1 to 8:
  - **Form**: prefilled, with a read-only SKU; "Guardar" is disabled until something changes.
  - **Save**: the PATCH carries only the dirty fields plus `version`, then "Cambios guardados" and the detail updated. "Guardar" is disabled while the request is pending (FR-039).
  - **Values**: an empty description sends `""`; changing the currency keeps the amount.
  - **Errors**: client errors and a 422 appear on each field.
  - **Conflict**: a 409 shows an `Alert` with "Otra persona modificó este producto. Recarga para ver la versión actual." and keeps the values. "Recargar" refetches and resets the form.
  - **Not found**: a 404 on save shows "Producto no encontrado".
  - Loading, not-found and error on load; `axe`.
  - Add the "Editar" action tests to web/src/features/products/pages/ProductDetailPage.test.tsx and to web/src/features/products/pages/ProductListPage.test.tsx.

### Implementation for User Story 3

- [ ] T087 [US3] Write the `UpdateProduct` query (`COALESCE(sqlc.narg)`, `description_set`, `WHERE id AND version AND deleted_at IS NULL`, `version + 1`, `updated_at = now()`, `RETURNING`) and the `ProductExists` query in api/queries/products.sql, then run `make sqlc` (R5)
- [ ] T088 [US3] Implement `Update` in api/internal/repository/products.go: when 0 rows change, run `ProductExists` → `ErrNotFound` or `ErrVersionConflict` (R6)
- [ ] T089 [US3] Implement `Update` in api/internal/service/products.go (`domain.ProductPatch`; `IsEmpty` → `ValidationError`) and add it to `ProductRepository`
- [ ] T090 [US3] Implement the `Update` handler and the `UpdateProductRequest` DTO in api/internal/httpapi/handler/products.go and api/internal/httpapi/handler/dto.go, and add `PATCH /products/:id` to `(*Products).Register`. The DTO has pointers per data-model.md Product.name, Product.description, Product.price_cents, Product.currency, Product.stock, Product.category and Product.is_active, a required Product.version, and no `sku`.
- [ ] T091 [P] [US3] Implement `toPatchBody` in web/src/features/products/schema.ts, per data-model.md ProductPatchBody
- [ ] T092 [US3] Implement `updateProduct` in web/src/features/products/api.ts and `useUpdateProduct` in web/src/features/products/hooks.ts. There's no optimistic update (C7): on success, `setQueryData(detail)` and invalidate `lists()`.
- [ ] T093 [P] [US3] Implement web/src/features/products/components/ConflictAlert.tsx: `<Alert variant="outline" color="red.9">` with the FR-046 text and a "Recargar" button that receives `onReload`
- [ ] T094 [US3] Implement web/src/features/products/pages/ProductEditPage.tsx (`ProductForm` in edit mode, `ConflictAlert`, not-found, `document.title`) and register `/products/:id/edit` in web/src/app/routes.tsx. Add the "Editar" action to ProductDetailPage.tsx, ProductTable.tsx and ProductCardList.tsx.

**Checkpoint**: US1, US2 and US3 all work on their own.

---

## Phase 6: User Story 4 - Delete a product (Priority: P4)

**Goal**: Delete from the list (immediately, with rollback) and from the detail, with a confirmation, version checking, and the SKU freed.

**Independent Test**: Delete from the list and from the detail. Check the confirmation, the immediate removal, that the product no longer appears anywhere, and that its SKU can be reused (quickstart.md, US4).

**Depends on**: US2 (list) and US1 (detail).

### Tests for User Story 4 ⚠️

- [ ] T095 [P] [US4] Write repository tests in api/internal/repository/products_delete_test.go:
  - `SoftDelete` sets `deleted_at` and `version` +1;
  - afterwards, `Get` → `ErrNotFound`, `List` excludes it, and `Create` with the same SKU succeeds;
  - a wrong version → `ErrVersionConflict`;
  - already deleted → `ErrNotFound`.
- [ ] T096 [P] [US4] Write service tests in api/internal/service/products_delete_test.go: `Delete` passes `id` and `version` through, and errors propagate
- [ ] T097 [P] [US4] Write handler tests in api/internal/httpapi/products_delete_test.go (contract-validated). Cases are quickstart.md API scenarios 18 to 20:
  - `DELETE ?version=N` → 204;
  - a missing or non-numeric `version` → 400;
  - an invalid UUID → 400;
  - 404;
  - 409.
- [ ] T098 [P] [US4] Write tests in web/src/features/products/pages/ProductListPage.delete.test.tsx, covering US4 scenarios 1 to 3 and 5 to 8:
  - **Confirmation**: "¿Eliminar producto?" with the name and SKU. Cancel sends no request and returns focus to the row's delete button (FR-040).
  - **Confirm**: the row disappears immediately, then "Producto eliminado". The confirm button is disabled while the request is pending (FR-039).
  - **Errors**: a 500 brings the row back with "No se pudo eliminar el producto. Inténtalo de nuevo.". A 409 keeps the row and shows `ConflictAlert`. A 404 removes the row with "Este producto ya no existe".
  - **Last page**: deleting the only product on it navigates to the previous page.
  - **Request**: `DELETE` sends the `version` from the list.
  - `axe` with the modal open.
- [ ] T099 [P] [US4] Write tests in web/src/features/products/pages/ProductDetailPage.delete.test.tsx, covering US4 scenario 4:
  - success → `/products` with "Producto eliminado";
  - the delete action is disabled while the request is pending (FR-039);
  - failure → stays with the notice;
  - 409 → `ConflictAlert`;
  - 404 → "Este producto ya no existe" and navigation to the catalog.

### Implementation for User Story 4

- [ ] T100 [US4] Write the `SoftDeleteProduct` query (`deleted_at = now()`, `version + 1`, `WHERE id AND version AND deleted_at IS NULL`) in api/queries/products.sql and run `make sqlc`
- [ ] T101 [US4] Implement `Delete` in api/internal/repository/products.go (0 rows → `ProductExists` → `ErrNotFound` or `ErrVersionConflict`), per data-model.md Product.deleted_at and Product.version
- [ ] T102 [US4] Implement `Delete` in api/internal/service/products.go and add it to `ProductRepository`
- [ ] T103 [US4] Implement the `Delete` handler in api/internal/httpapi/handler/products.go (parse the `version` query parameter into a `RequestError` field `version`, per data-model.md Product.version) and add `DELETE /products/:id` to `(*Products).Register`
- [ ] T104 [US4] Implement `deleteProduct` in web/src/features/products/api.ts and `useDeleteProduct` in web/src/features/products/hooks.ts:
  - optimistic removal from every cached `list` that contains the product, with a snapshot and rollback on error;
  - invalidate `lists()` on settle;
  - remove `detail(id)` on success (R20).
- [ ] T105 [P] [US4] Implement `openDeleteConfirm(product, onConfirm)` in web/src/features/products/components/delete-confirm.tsx: `modals.openConfirmModal` with the FR-046 texts, confirm button `color="red.9"` and cancel "Cancelar"
- [ ] T106 [US4] Add the delete action to ProductTable.tsx and ProductCardList.tsx, and handle in web/src/features/products/pages/ProductListPage.tsx:
  - notifications;
  - `ConflictAlert` on 409;
  - the 404 notice;
  - navigation to `total_pages` when the last page empties.
- [ ] T107 [US4] Add the delete action to web/src/features/products/pages/ProductDetailPage.tsx: success → `/products` with the notification; 409 → `ConflictAlert`; 404 → the notice and the catalog

**Checkpoint**: All four stories work on their own.

---

## Phase 7: Polish & Cross-Cutting Concerns

**Purpose**: Sample data, browsable docs, performance, documentation in sync (VIII), and final validation.

- [ ] T108 [P] Write tests in api/internal/seed/seed_test.go (real service, fake repository):
  - the 20 products pass the domain rules;
  - they cover all 5 categories, both currencies and at least 4 inactive products, and include names with accents and ñ;
  - a second `Run` skips `ErrSKUAlreadyExists` without returning an error (R15).
- [ ] T109 Extend api/cmd/api/main_test.go: after `testutil.ResetDB` on the package's container, `api seed` exits 0 and leaves 20 products, and a second run also exits 0 and still leaves 20 (R15)
- [ ] T110 Implement api/internal/seed/products.go and api/internal/seed/seed.go (`Run(ctx, creator)`), the `seed` subcommand in api/cmd/api/main.go (migrate → repository → service → `seed.Run`), and the `seed` target in Makefile (`cd $(API_DIR) && go run ./cmd/api seed`, with a `##` description)
- [ ] T111 [P] Write tests in api/internal/apidocs/apidocs_test.go:
  - `GET /docs` → 200 `text/html`, with its own CSP allowing `cdn.jsdelivr.net` and `swagger-ui-dist` pinned with `integrity`;
  - `GET /docs/openapi.yaml` → the embedded contract, byte-identical to the root `openapi.yaml`;
  - both are validated against the contract and aren't rate-limited.
- [ ] T112 Implement api/internal/apidocs/apidocs.go (embedding `openapi.yaml` and `index.html`) and api/internal/apidocs/index.html, and register `/docs` and `/docs/openapi.yaml` in api/internal/httpapi/router.go outside `/api/v1` (R14)
- [ ] T113 [P] Write `BenchmarkList` in api/internal/repository/products_bench_test.go: 10,000 products, then list, filter, search and sort. Record the p95 in the PR description (SC-002: server target < 200 ms).
- [ ] T114 [P] Update the root CLAUDE.md per plan.md "Documentation updated in the same PR": domain (`currency` MXN|USD, bounds); API table (errors and `DELETE ?version=`); list parameters (`currency`, range rule); error codes (remove `INSUFFICIENT_STOCK`, add `PAYLOAD_TOO_LARGE`); commands (`make sqlc`, `make seed`, `make sync-openapi`)
- [ ] T115 [P] Update api/CLAUDE.md:
  - remove `ErrInsufficientStock`;
  - the middleware and `/docs` are no longer pending;
  - `decodeJSON` with an injected validator instead of `ShouldBindJSON` (C5), and the domain owning normalization (C3);
  - startup migrations with retry (C4);
  - `CLIENT_IP_HEADER`;
  - packages `migrate`, `seed`, `apidocs`, and the new `testutil` helpers;
  - routes: `NewRouter` mounts `api ...func(*gin.RouterGroup)` registrars on `/api/v1`, and each handler exposes `Register`;
  - testing: black-box tests in api/cmd/api/main_test.go build and run the binary (startup migrations, `seed`).
- [ ] T116 [P] Update web/CLAUDE.md:
  - UI copy in es-MX addressing the user as tú (`messages.ts`);
  - price "entered in currency units";
  - the `formatPrice`/`formatDateTime` rules;
  - routes (`products`, `status` at `/status`);
  - optimistic updates for delete only (C7);
  - the responsive list (C6);
  - in Testing: `renderRoute`, and `axe` imported from `src/test/axe.ts` (WCAG 2.2 AA tags) instead of from `vitest-axe`.
- [ ] T117 [P] Update README.md:
  - endpoints and `/docs`;
  - `make seed`;
  - `CLIENT_IP_HEADER` in the env table, and a Railway step (`X-Real-IP`);
  - new technical decisions (collation, unaccent, uuidv7, rate limit, migrations at startup, docs);
  - remove the "Deferred to the first feature" line.
- [ ] T118 [P] Add `CLIENT_IP_HEADER=` to the "API" section of .env.example, after `GIN_MODE`: empty locally, with a comment saying to set it to `X-Real-IP` on Railway. web/.env.example doesn't change, because there are no new `VITE_*` variables.
- [ ] T119 Run `make fmt`, `make lint` and `make test`, and the generated-code checks in quickstart.md "Automated checks" (`make sqlc`, `make gen-types` and `make sync-openapi` with no diff)
- [ ] T120 Run the quickstart.md validation: the 25 API scenarios, the web walkthrough per story at 1280 px and 360 px, and keyboard plus screen reader. Fix anything found by writing the failing test first.

---

## Dependencies & Execution Order

### Phase Dependencies

- **Setup (Phase 1)**: no dependencies. T002 → T003. T001 before T003.
- **Foundational (Phase 2)**: depends on Setup and blocks every story.
  - Database: T007 → T008/T009 → T010, T011 → T012 → T013; T014 (startup test) → T015.
  - Domain: T016, T017 → T018 → T019.
  - HTTP: T020 → T029 (T029 also needs T022 and T028, whose error types it maps); T021 → T022 (needs T019); T023–T025 → T026–T028; T030 → T031; T032 → T033 (needs T029, T026–T028, T031).
  - Web: T034 → T035; T036 → T037; T041 → T042; T043 → T044/T045 → T046, T047.
- **US1 (Phase 3)**: depends on Foundational.
- **US2 (Phase 4)**: depends on Foundational and can run in parallel with US1. Its row links go to the US1 detail but don't need it in order to be tested.
- **US3 (Phase 5)**: depends on US1 (`ProductForm`, detail). The "Editar" action in the list depends on US2.
- **US4 (Phase 6)**: depends on US2 (list) and US1 (detail).
- **Polish (Phase 7)**: T108–T110 depend on US1 (`service.Create`). T111–T112 depend on Foundational. T113 depends on US2. T114–T120 run last.

### Within Each User Story

- Tests first, and they must fail for the right reason before any implementation.
- API: query + `make sqlc` → repository → service → handler + route → wiring.
- Web: schema / search-params → api.ts + hooks → components → page + route.
- Repeated files within a story (`products.go`, `dto.go`, `router.go`, `api.ts`, `hooks.ts`, `routes.tsx`) are edited in sequence, never in parallel.

### Parallel Opportunities

- Setup: T004 in parallel with T001–T003.
- Foundational: all the API tests marked [P] (T005, T007, T016, T017, T020, T021, T023–T025, T030) and all the web tasks marked [P] (T034–T042, T045, T048) can run together. API and web advance in parallel.
- Within each story, every test marked [P] can run at once, and the API and web halves progress in parallel.
- US1 and US2 can be developed in parallel once Foundational is done.

---

## Parallel Example: User Story 1

```bash
# Every US1 test at once (all must fail first):
Task: "T049 Repository tests in api/internal/repository/products_test.go"
Task: "T050 Service tests in api/internal/service/products_test.go"
Task: "T051 Handler tests in api/internal/httpapi/products_test.go"
Task: "T052 Schema tests in web/src/features/products/schema.test.ts"
Task: "T053 ProductCreatePage tests"
Task: "T054 ProductDetailPage tests"

# Then two parallel lines:
API: T055 → T057 → T058 → T059   (T056 alongside)
Web: T060 → T061 → T062 → T063 → T064
```

## Parallel Example: User Story 2

```bash
Task: "T065 Repository list tests"   Task: "T067 parseListParams tests"
Task: "T069 search-params tests"     Task: "T070 ProductListPage tests"
# Implementation: T079 and T080 (filters and the two layouts) in parallel after T078
```

---

## Implementation Strategy

### MVP First (User Story 1 Only)

1. Phase 1 (contract and tooling) → Phase 2 (schema, domain, envelope, middleware, web shell).
2. Phase 3 (US1): create and view, tested API to UI.
3. **Stop and validate**: quickstart.md, US1 section and API scenarios 1 to 5.

### Incremental Delivery

1. Setup + Foundational → the foundation is ready.
2. US1 → validate (MVP).
3. US2 → validate the catalog.
4. US3 → validate editing and conflicts.
5. US4 → validate deleting.
6. Polish → seed, `/docs`, docs, `make lint` + `make test` + quickstart.
7. One PR for the whole feature. Main must stay deployable at every merge.

### Commits

Conventional Commits with one area per commit (`api`, `web`, or no scope for root files such as openapi.yaml, Makefile, CI, CLAUDE.md and README). Commit at each logical group, for example "contract and generated types", "migration and migrate", "US1 API" and "US1 web".

---

## Notes

- [P] tasks touch different files and have no pending dependencies.
- [Story] traces each task to its user story in spec.md.
- Every text visible in the UI comes from messages.ts (SC-009). API messages stay in English.
- Generated code (web/src/api/schema.d.ts, api/internal/repository/db/) is never edited by hand.
- A task that adds a Go dependency runs `go get` and `go mod tidy`. Dev tools stay in api/tools/.
- Avoid vague tasks, simultaneous edits to the same file, and dependencies between stories beyond those documented.
