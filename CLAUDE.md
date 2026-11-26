# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Overview

Product catalog management platform. Monorepo: REST API in Go (`api/`) + React web client (`web/`) that consumes every API endpoint. Side-specific conventions live in `api/CLAUDE.md` and `web/CLAUDE.md`.

```
api/                # Go + Gin
web/                # React + Vite + TypeScript
specs/NNN-feature/  # Spec Kit specs, one folder per feature
.specify/           # Spec Kit constitution and templates
openapi.yaml        # API contract: single source of truth
docker-compose.yml
Makefile
.github/workflows/
```

`openapi.yaml` drives both sides: the API serves it at `/docs` and its handler tests validate every response against it; the web generates its TS types from it. Change the contract first, then regenerate (`make gen-types`) and implement.

## Commands

`make help` lists every command. Environment: `make up`, `make down`, `make db`, `make dev-api`, `make dev-web`. Tests: `make test` (all), `make test-unit` (backend with `-short` + frontend), `make test-api`, `make test-web`. Quality: `make lint`, `make lint-api`, `make lint-web`, `make fmt`. Generation: `make gen-types`, `make sqlc`, `make sync-openapi`, `make migrate-new name=...`. Data: `make seed` (20 sample products, idempotent). Every new command goes into the Makefile with its `##` description.

## Domain: `Product`

| Field | Type | Rules |
|---|---|---|
| `id` | UUID | Server-generated |
| `sku` | string | Required, trimmed and uppercased, then `^[A-Z0-9-]{4,32}$`; unique among non-deleted; immutable |
| `name` | string | Required, trimmed, then 2–120 chars |
| `description` | string? | Trimmed, max 2000; empty means none (`null`) |
| `price_cents` | int64 | Required, 1 to 99,999,999,999. Money is always integer cents, never float |
| `currency` | string | `MXN` or `USD`, default `MXN`; changing it never converts the amount |
| `stock` | int32 | Required, 0 to 2,147,483,647 |
| `category` | enum | Required: `electronics`, `home`, `food`, `clothing`, `other` |
| `is_active` | bool | Default `true` |
| `version` | int | Optimistic concurrency; +1 on every update and deletion |
| `created_at` / `updated_at` | timestamptz | Server-managed |
| `deleted_at` | timestamptz? | Soft delete: no query returns the row and its SKU is free |

## API (`/api/v1`)

| Method | Path | Success | Errors |
|---|---|---|---|
| GET | `/products` | 200 | 400 |
| GET | `/products/:id` | 200 | 400, 404 |
| POST | `/products` | 201 + `Location` | 400, 409, 413, 422 |
| PATCH | `/products/:id` | 200 | 400, 404, 409, 413, 422 |
| DELETE | `/products/:id?version=N` | 204 | 400, 404, 409 |

Every `/api/v1` route can also answer 429 (rate limit: a sustained 300/min per IP with bursts of 300) and 500.

- List params: `page`, `page_size` (max 100), `category`, `is_active`, `currency`, `min_price` and `max_price` (cents, inclusive; both require `currency`, and min ≤ max), `q` (≤ 100 chars; partial match on name and sku ignoring case and accents, `%`/`_`/`\` literal), `sort` (`name`, `price`, `stock`, `created_at`) + `order`. Name sorts in Spanish order; price sorts group by currency (MXN, then USD); `id` breaks ties. Response: `{ data: [...], meta: { page, page_size, total, total_pages } }`; a page beyond the last returns an empty `data` with the real `total`.
- `PATCH` (body `version`) and `DELETE` (query `version`) check the version; a mismatch returns `409 VERSION_CONFLICT`. `PATCH` changes only the fields present, rejects `sku` (unknown field, 400) and a body with only `version` (422); an empty `description` removes it.
- Single error shape: `{ "error": { "code", "message", "details": [{ "field", "message" }] } }`. Codes: `VALIDATION_ERROR` (400 malformed request or invalid parameter; 422 broken business rule, with `details`), `NOT_FOUND`, `SKU_ALREADY_EXISTS`, `VERSION_CONFLICT`, `PAYLOAD_TOO_LARGE` (body over 64 KiB), `RATE_LIMITED` (with `Retry-After`), `INTERNAL_ERROR`.
- 500s never expose internals: generic message + request ID; details go only to logs.
- Health probes live at the host root, outside `/api/v1`: `GET /healthz` (process alive) and `GET /readyz` (database ping; 503 when unavailable). They use their own `{ status, checks }` shape, not the error envelope. `GET /docs` serves Swagger UI over the embedded contract (`/docs/openapi.yaml`). Neither the probes nor `/docs` are rate-limited. `VITE_API_URL` is the API host root.

## Workflow

The constitution (`.specify/memory/constitution.md`) prevails over every CLAUDE.md file; these add technical detail and must not contradict it.

### Spec Kit (spec-driven development)
- Every feature gets its own branch and `specs/NNN-feature/`, following: constitution → specify → clarify → plan → tasks → analyze → implement.
- Features are vertical slices (API + web), worked from the repo root.
- Implement nothing that is not in the spec. When a requirement changes: update the spec first, then the tests, then the code.
- A feature's `contracts/` describes contract changes; every plan includes a task to apply them to `openapi.yaml` and regenerate types.
- Spec Kit applies to user-facing features. Infrastructure and tooling work (`chore`, `ci`) needs no spec: plan it in Claude Code plan mode on a `chore/...` branch. It still follows TDD.
- Customize Spec Kit templates in `.specify/templates/overrides/` (the `tasks-template.md` override makes tests mandatory). `specify init` restores the core templates and `.claude/skills/speckit-*`, so after an upgrade re-apply the "tests are mandatory" edits to `speckit-tasks`.

### TDD (mandatory)
- Write tests **before** code, derived from the spec's acceptance criteria: red → green → refactor. Confirm each new test fails for the right reason before implementing.
- **Code adapts to tests, never tests to code.** Never modify, weaken or delete a test to make it pass. If a test looks wrong, stop and check it against the spec; if the spec is wrong, fix the spec first.
- In `tasks.md`, test tasks come before implementation tasks in every phase.
- A feature is done only when all tests, formatting, lint and typecheck pass.

### Docs stay in sync
Every change includes, in the same commit/PR, updates to whatever it affects: `openapi.yaml`, specs, README (`README.md` in Spanish, the main one, and `README.en.md`, kept in sync), CLAUDE.md files, constitution, `.env.example`, generated code (`make sqlc`, `make gen-types`). A change without its docs is incomplete.

### Code and commits
- SOLID, separation of concerns, small functions, no duplication. Tooling and style rules per side are in the side's CLAUDE.md.
- Conventional Commits, brief: `type(scope): summary` (imperative, lowercase, ≤ 72 chars); scope is the area (`api`, `web`; none for root files), one area per commit. Optional 1–3 line body with what and why. Example:
  ```
  chore(api): set up Go module and dev tooling

  Pin sqlc, goose, govulncheck and golangci-lint via go tool, configure
  golangci-lint and sqlc, and add a hello endpoint to verify the setup.
  ```
- One PR per feature. `main` is protected (merge via PR with green checks only) and always deployable.

## Security (cross-cutting)

- Validate all input on the server (body, params, query), even when the web also validates.
- Secrets only in env vars: never in the repo, logs, or `VITE_*` variables. Keep `.env.example` current.
- Minimal, justified dependencies; `govulncheck` and `pnpm audit` run in CI.
- Side-specific OWASP rules are in `api/CLAUDE.md` and `web/CLAUDE.md`.

## CI/CD

- GitHub Actions on every PR and push to `main`:
  - OpenAPI lint (Redocly).
  - API: golangci-lint (format + lint), `sqlc diff`, embedded OpenAPI copy in sync, `govulncheck`, `go test -race` (including integration), build, docker build.
  - Web: generated-types check, `pnpm format:check`, `pnpm lint`, typecheck, `pnpm audit`, tests, build.
- Deploy: API + Postgres on Railway (Wait for CI, health check on `/healthz`); web on Vercel (SPA rewrite to `index.html`).
- The job names in `.github/workflows/ci.yml` (`OpenAPI`, `API (Go)`, `Web (React)`) are the required checks of `main`: renaming one breaks the branch protection. Vercel does not wait for CI; only merges of PRs with green checks reach `main`.
