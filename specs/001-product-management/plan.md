# Implementation Plan: Product Management

**Branch**: `001-product-management` | **Date**: 2026-11-25 | **Spec**: [spec.md](spec.md)

**Input**: Feature specification from `specs/001-product-management/spec.md`, plus the technical direction the user provided with `/speckit-plan` (contract, database, sqlc, service, handlers, frontend, seed, tests, docs).

## Summary

An administrator creates, views, browses (filters, search, sort, pagination in the URL), edits and deletes products from the Spanish-language web app. The work is one vertical slice:

- **Contract first**: [contracts/products-api.yaml](contracts/products-api.yaml) is merged into `openapi.yaml` and the types are regenerated.
- **Database**: Postgres 18 with one goose migration, embedded and applied at startup. It adds an ICU collation for Spanish order, `unaccent` behind an immutable wrapper for search, partial indexes over non-deleted rows, and `uuidv7()` IDs.
- **API**: `handler → service → repository` with sqlc. The domain owns normalization and the rules. Versions prevent overwriting someone else's change, deletion is logical, and the middleware the README deferred to this feature lands here (rate limiting, body limit, security headers) together with `/docs`.
- **Web**: `features/products` with TanStack Query, the URL as the state of the catalog, a shared react-hook-form + zod form, Spanish texts centralized, and a responsive list.

Technical decisions are in [research.md](research.md), the model in [data-model.md](data-model.md), the web contract in [contracts/web-routes.md](contracts/web-routes.md), and the end-to-end validation in [quickstart.md](quickstart.md).

**Additions to the provided plan**, required by the spec, the constitution or the README's "deferred to the first feature" list:

- Rate limiting at 300/min per IP, with `CLIENT_IP_HEADER` (FR-043, VI; R11).
- A 64 KiB body limit with the new code `PAYLOAD_TOO_LARGE` (R12).
- Security headers (R13).
- `/docs` with Swagger UI and `make sync-openapi` (R14).
- Spanish order and accent-insensitive search (FR-019, FR-020; R1, R2).
- Translating the status page, setting `lang="es-MX"` and using Spanish texts throughout (FR-045 to FR-047).
- A responsive list (FR-048).
- `AssertResponseMatchesContract` for negative tests (R16).
- A CI check for `sqlc diff`, which root CLAUDE.md already claims runs but `ci.yml` doesn't have.

## Technical Context

**Language/Version**: Go 1.27.4 (API). TypeScript ~6.0 with React 19.3 (web). SQL on PostgreSQL 18.

**Primary Dependencies**:
- API: Gin 1.12, pgx v5.13 (`pgxpool`, `stdlib`), sqlc 1.31.1 (tool), go-playground/validator v10 (already indirect, now direct). New: goose v3.30.1 as a library, `golang.org/x/time/rate` and `testcontainers-go` (tests); kin-openapi (tests).
- Web: Mantine 9.8, TanStack Query 5, React Router 8, react-hook-form 7 + zod 4, openapi-fetch 0.17. **No new dependencies** on the web.

**Storage**: PostgreSQL 18: `products` table, ICU collation `catalog_es`, `unaccent` extension, `catalog_fold` function, built-in `uuidv7()`.

**Testing**:
- API: `testing` + testify, `httptest` with contract validation (kin-openapi), `testcontainers-go` (Postgres 18), native fuzzing.
- Web: Vitest 5 + Testing Library + user-event + MSW 3 + vitest-axe (WCAG 2.2 AA).

**Target Platform**: API in a distroless Linux container on Railway, with Postgres on Railway. Web as a static SPA on Vercel and nginx. Evergreen browsers, from 360 px wide.

**Project Type**: web application (REST API + SPA) in a monorepo (`api/`, `web/`).

**Performance Goals**: SC-002: 95% of list/filter/search/sort results in < 1 s with 10,000 products. Derived server-side target: list at p95 < 200 ms with 10,000 rows (sequential scan for search, indexes for filters and sorts).

**Constraints**:
- Money in integer cents, at most 99,999,999,999.
- `page_size` ≤ 100, and the web always uses 20.
- `q` ≤ 100 characters.
- 300 requests/min per IP on `/api/v1`.
- Body ≤ 64 KiB.
- UI 100% in es-MX, addressing the user as tú.
- WCAG 2.2 AA.
- No horizontal page scrolling at 360 px or wider.

**Scale/Scope**: About 10,000 products, one role (admin, no authentication), 5 endpoints + `/docs`, and 5 web routes + status.

All Technical Context unknowns are resolved in research.md. The user accepted the recommendations for C1 to C8 (Complexity Tracking) on 2026-11-25.

## Constitution Check

*GATE: Must pass before Phase 0 research. Re-checked after Phase 1 design.*

| # | Principle | How this plan satisfies it | Pre | Post |
|---|---|---|---|---|
| I | Spec-driven | Branch `001-product-management`, spec with 10 clarifications, vertical slice (API + web), nothing outside the spec. The additions above trace to FRs or to the constitution | ✅ | ✅ |
| II | TDD | Each layer starts with failing tests derived from the scenarios: domain (table-driven + fuzz), service (fakes), handlers (contract), repository (testcontainers), middleware, web pages (MSW + axe). In `tasks.md`, tests go first in every phase | ✅ | ✅ |
| III | Contract first | Phase 1 of the work merges `contracts/products-api.yaml` into `openapi.yaml`, then `make gen-types` + `make sync-openapi`. Every handler response is validated against the contract (`AssertMatchesContract` / `AssertResponseMatchesContract`). `schema.d.ts` and `repository/db` are never edited by hand | ✅ | ✅ |
| IV | Layers and DI | `handler` defines its service interface, and `service` defines its repository interface. Manual wiring in `cmd/api/main.go`. Web: components → hooks → `features/products/api.ts` → `apiClient`; `products` doesn't import from `status`. C5 avoids Gin globals | ✅ | ✅ |
| V | Quality | golangci-lint and ESLint/Prettier as configured; no `nolint`/`eslint-disable` without justification. Small functions; rules defined once in `domain` (C3) | ✅ | ✅ |
| VI | Security | Validation on the server (body, params, query); sqlc only, with a sort whitelist and escaped `LIKE`; explicit CORS (unchanged); no new secrets; generic 500 with request ID; per-IP rate limiting (R11); body limit; headers; govulncheck and `pnpm audit` in CI | ✅ | ✅ |
| VII | Accessibility and UX | Mantine components per web/CLAUDE.md; axe on every page (both layouts); loading, empty, no-results, error and out-of-range states; feedback on every action; buttons disabled while they run | ✅ | ✅ |
| VIII | Documentation in sync | Everything in the list below goes in the same PR | ✅ | ✅ |
| IX | Simplicity | No query builder or new web dependencies. New API dependencies are justified below and were already prescribed by api/CLAUDE.md. No trigram index, no metrics. The complexity added is in Complexity Tracking | ✅ | ✅ |
| X | Continuous delivery | Everything goes through the PR with full CI. The migration is applied at startup and Railway doesn't promote a failed deployment (C4) | ✅ | ✅ |

**Result**: PASS. There are no unjustified violations. C4 and C5 change a documented decision and a convention; the README and api/CLAUDE.md are updated in the same PR.

### New dependencies (IX)

| Dependency | Side | Why | Rejected alternative |
|---|---|---|---|
| `github.com/pressly/goose/v3` (library, v3.30.1 like the tool) | API | Apply embedded migrations at startup (api/CLAUDE.md) | Running the CLI in the container: distroless has no shell |
| `golang.org/x/time/rate` | API | Per-IP token bucket (api/CLAUDE.md) | Writing a limiter by hand |
| `github.com/testcontainers/testcontainers-go` (+ `modules/postgres`) | API, tests | A real Postgres in repository tests (api/CLAUDE.md) | Mocking pgx: wouldn't test SQL, collation or constraints |
| `github.com/go-playground/validator/v10` (indirect → direct) | API | DTO tags and custom validators | — |
| Swagger UI from jsDelivr (pinned + SRI) | API (HTML) | `/docs` without vendoring assets (R14) | `swaggo` (annotations duplicate the contract) |

## Project Structure

### Documentation (this feature)

```text
specs/001-product-management/
├── spec.md
├── plan.md              # this file
├── research.md          # Phase 0
├── data-model.md        # Phase 1
├── quickstart.md        # Phase 1
├── contracts/
│   ├── products-api.yaml   # OpenAPI changes, merged into openapi.yaml
│   └── web-routes.md       # routes and URL parameters
├── checklists/requirements.md
└── tasks.md             # /speckit-tasks (not created here)
```

### Source Code (repository root)

```text
openapi.yaml                          # + /api/v1/products*, /docs*, schemas and responses
Makefile                              # + sqlc, seed, sync-openapi
.github/workflows/ci.yml              # + sqlc diff, openapi sync check (same job names)

api/
├── cmd/api/main.go                   # wiring: migrate → repo → service → handlers; `seed` subcommand
├── migrations/
│   ├── 00001_create_products.sql     # table, CHECKs, partial indexes, catalog_es, unaccent, catalog_fold
│   └── embed.go                      # embed.FS
├── queries/products.sql              # Create, Get, List, Count, Update, SoftDelete, Exists
├── internal/
│   ├── config/config.go              # + CLIENT_IP_HEADER (optional)
│   ├── domain/                       # product.go (types, normalization, rules), errors.go, validation.go (+ tests, fuzz)
│   ├── service/products.go           # business rules; defines ProductRepository (+ tests with fakes)
│   ├── repository/
│   │   ├── products.go               # sqlc adapter, 23505 and 0-row translation, LIKE escaping (+ integration tests)
│   │   └── db/                       # sqlc output (generated)
│   ├── migrate/migrate.go            # goose Provider + session locker + retry (+ integration test)
│   ├── seed/                         # seed.go + products.go (20 products) (+ tests)
│   ├── apidocs/                      # openapi.yaml (synced copy), index.html, handler (+ tests)
│   ├── httpapi/
│   │   ├── router.go                 # /api/v1 group with rate limit and body limit; headers; /docs
│   │   ├── errors.go                 # + single domain → HTTP mapping, details[]
│   │   ├── handler/                  # products.go, dto.go, decode.go, params.go (+ tests)
│   │   └── middleware/               # ratelimit.go, bodylimit.go, security.go (+ tests)
│   └── testutil/                     # + postgres.go (testcontainers), AssertResponseMatchesContract
└── go.mod                            # + goose, x/time, testcontainers-go, validator (direct)

web/
├── index.html                        # lang="es-MX", Spanish title
└── src/
    ├── api/errors.ts                 # ApiError
    ├── app/
    │   ├── App.tsx                   # routes, redirect / → /products, /status
    │   └── AppLayout.tsx             # header with navigation (+ test)
    ├── components/ui/PageState.tsx   # generic empty/error state (title, text, action)
    ├── features/
    │   ├── status/                   # moves to /status; texts translated (tests updated to Spanish)
    │   └── products/
    │       ├── api.ts  hooks.ts  query-keys.ts
    │       ├── schema.ts             # zod for the form (+ tests)
    │       ├── search-params.ts      # URL ⇄ CatalogSearch (+ tests)
    │       ├── format.ts             # formatPrice, formatDateTime (+ tests)
    │       ├── messages.ts           # FR-046 texts and per-field messages (FR-047)
    │       ├── testing.ts            # fixtures and MSW handlers for the tests
    │       ├── components/           # ProductTable, ProductCardList, ProductFilters, ProductForm,
    │       │                         # ProductDetails, ConflictAlert, ResultsAnnouncer
    │       └── pages/                # ProductListPage, ProductCreatePage, ProductDetailPage,
    │                                 # ProductEditPage (+ integration tests with MSW and axe each)
    └── test/handlers.ts              # default handlers (health) unchanged
```

**Structure Decision**: The existing monorepo is kept. `api/` follows the layout in api/CLAUDE.md and adds `migrate`, `seed` and `apidocs` as internal packages with one responsibility each. `web/` adds the `products` feature and moves `status` to `/status`. No other projects or shared packages are created.

### Documentation updated in the same PR (VIII)

| File | Change |
|---|---|
| `openapi.yaml` (+ `api/internal/apidocs/openapi.yaml`) | Merge `contracts/products-api.yaml`; `info.version` 0.2.0 |
| `web/src/api/schema.d.ts`, `api/internal/repository/db/` | Regenerated |
| Root `CLAUDE.md` | Domain: `currency` MXN\|USD and price and inventory bounds. API: `currency` in the list parameters and the range rule; `DELETE ?version=`; complete errors (PATCH 400; DELETE 400/409; 413; 429). Codes: remove `INSUFFICIENT_STOCK`, add `PAYLOAD_TOO_LARGE`. Commands: `make sqlc`, `make seed`, `make sync-openapi` |
| `api/CLAUDE.md` | Remove `ErrInsufficientStock`. Middleware and `/docs` are no longer pending. Validation and decoding per C3/C5. Startup migrations per C4. `CLIENT_IP_HEADER` in config. `testutil` with Postgres and `AssertResponseMatchesContract`. Packages `migrate`, `seed`, `apidocs` |
| `web/CLAUDE.md` | UI copy in es-MX addressing the user as tú (`messages.ts`). Price "entered in currency units" (no longer "pesos"). Price and date formatting. Routes (`products`, `status` at `/status`). Optimistic updates per C7. Responsive list per C6 |
| `README.md` | Endpoints, `/docs`, `make seed`, `CLIENT_IP_HEADER` (env table + Railway step: `X-Real-IP`). New decisions (collation, unaccent, uuidv7, rate limit, migrations, docs). Remove the "Deferred to the first feature" line |
| `.env.example` | `CLIENT_IP_HEADER=` (empty locally) |

## Complexity Tracking

> Per the user's instruction, conflicts between the provided plan and the spec, CLAUDE.md or the constitution were flagged here rather than resolved unilaterally. The user accepted every recommendation on 2026-11-25; research.md marks each one as "(Cn)".

| # | Conflict | Source of the conflict | Resolution (accepted) |
|---|---|---|---|
| C1 | `Intl.NumberFormat("es-MX", { style: "currency", currency })` prints `$1,299.90` (MXN) and `USD 1,299.90` (USD) | Spec FR-012 and clarification: `$1,299.90 MXN` / `$1,299.90 USD` (that format was option B, which you rejected) | Use `currencyDisplay: "narrowSymbol"` and append the code (R22) |
| C2 | `NumberInput` with "2 decimales" (`decimalScale={2}`) and the default `clampBehavior="blur"` truncates or clamps silently | Spec edge case: more than two decimals or above the maximum → "rejected on the price field" (an error, not a silent correction) | No `decimalScale`, `clampBehavior="none"`; zod reports the error (R21) |
| C3 | The handler validates with DTO tags before the service normalizes | FR-002/FR-003: rules apply after trim/uppercase; plain tags would reject `" abc-123 "` and 120-character names with spaces | The domain owns normalization and rules; the custom validators (`sku`, `trimmed_len`) call it, and the service validates again (R7) |
| C4 | Applying migrations at startup forces a database connection before serving | README "Readiness without blocking startup" (the API starts without a database); api/CLAUDE.md does require migrations at startup | Migrate before serving, with retries for 30 s and exit on failure (Railway doesn't promote the deployment); update the README decision (R10) |
| C5 | `ShouldBindJSON` + rejecting unknown fields + custom validators require mutating Gin's global `binding` package | Constitution IV and api/CLAUDE.md: "no globals" | Own helper `decodeJSON` with an injected validator; update api/CLAUDE.md, which says "via `ShouldBindJSON`" (R8) |
| C6 | "Listado con `Table`" alone | FR-048/SC-010: usable at 360 px without horizontal scrolling | `Table` from 48em, stacked cards below, one rendered at a time (R23) |
| C7 | Optimistic update when editing (`useUpdateProduct`), also required by web/CLAUDE.md | FR-029/FR-030: the user waits on the form until the response (conflict → values stay; success → navigate). The optimistic update isn't visible and adds rollback, and it can misplace products in filtered or sorted lists | No optimistic update when editing (keep it for deleting); update web/CLAUDE.md (R20) |
| C8 | Spec defect introduced in the clarification: dropping a typed comma doesn't stop `199,90` from becoming `19990` (the comma is lost and the `90` is appended) | Spec edge case "can never become 19,990.00 by mistake" | Show the hint "Usa punto (.) para los decimales" when a comma is dropped; the spec's edge case, FR-005 and FR-046 were corrected (R21) |

**Complexity added and justified** (not a constitution violation):

| Addition | Why it's needed | Simpler alternative rejected because |
|---|---|---|
| ICU collation + `catalog_fold` function in the migration | FR-019/FR-020 (Spanish search and order) | Sorting or filtering in Go breaks pagination, and libc doesn't have es_MX on Alpine |
| Two list queries (page + count) in one transaction | Consistent `total` and the out-of-range state (FR-026) | `COUNT(*) OVER ()` gives no total when the page is empty |
| `AssertResponseMatchesContract` alongside the existing helper | Validating the response contract for intentionally invalid requests | Leaving negative tests without contract validation would break III |
| `make sync-openapi` + an embedded copy of the contract | `/docs` in a distroless image whose Docker context is `api/` | `//go:embed` can't reach `../openapi.yaml` |
| Two list layouts (table and cards) | FR-048 | See C6 |
