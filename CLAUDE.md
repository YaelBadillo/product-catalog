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

| Command | Purpose |
|---|---|
| `make up` | Start the stack (docker compose) |
| `make test` | All tests, including integration (testcontainers, needs Docker) |
| `make test-unit` | Fast TDD loop: backend with `-short` + frontend |
| `make lint` / `make fmt` | Lint / format both sides |
| `make sqlc` | Regenerate sqlc code after editing `api/queries/` or migrations |
| `make migrate-new name=...` | Create a goose migration |
| `make gen-types` | Regenerate web types from `openapi.yaml` |
| `make seed` | Seed the database |

Running a single test: see `api/CLAUDE.md` and `web/CLAUDE.md`.

## Domain: `Product`

| Field | Type | Rules |
|---|---|---|
| `id` | UUID | Server-generated |
| `sku` | string | Required, unique among non-deleted, `^[A-Z0-9-]{4,32}$` |
| `name` | string | Required, 2–120 chars |
| `description` | string? | Max 2000 |
| `price_cents` | int64 | Required, > 0. Money is always integer cents, never float |
| `currency` | string | ISO 4217, default `MXN` |
| `stock` | int32 | Required, ≥ 0 |
| `category` | enum | `electronics`, `home`, `food`, `clothing`, `other` |
| `is_active` | bool | Default `true` |
| `version` | int | Optimistic concurrency |
| `created_at` / `updated_at` | timestamptz | Server-managed |
| `deleted_at` | timestamptz? | Soft delete |

## API (`/api/v1`)

| Method | Path | Success | Errors |
|---|---|---|---|
| GET | `/products` | 200 | 400 |
| GET | `/products/:id` | 200 | 400, 404 |
| POST | `/products` | 201 + `Location` | 400/422, 409 |
| PATCH | `/products/:id` | 200 | 404, 409, 422 |
| DELETE | `/products/:id` | 204 | 404 |

- List params: `page`, `page_size` (max 100), `category`, `is_active`, `min_price`, `max_price`, `q` (matches name and sku), `sort` + `order` (whitelisted columns only). Response: `{ data: [...], meta: { page, page_size, total, total_pages } }`.
- `PATCH` and `DELETE` check `version`; a mismatch returns `409 VERSION_CONFLICT`.
- Single error shape: `{ "error": { "code", "message", "details": [{ "field", "message" }] } }`. Codes: `VALIDATION_ERROR`, `NOT_FOUND`, `SKU_ALREADY_EXISTS`, `VERSION_CONFLICT`, `INSUFFICIENT_STOCK`, `RATE_LIMITED`, `INTERNAL_ERROR`.
- 500s never expose internals: generic message + request ID; details go only to logs.

## Workflow

### Spec Kit (spec-driven development)
- Every feature gets its own branch and `specs/NNN-feature/`, following: constitution → specify → clarify → plan → tasks → analyze → implement.
- Features are vertical slices (API + web), worked from the repo root.
- Implement nothing that is not in the spec. When a requirement changes: update the spec first, then the tests, then the code.
- A feature's `contracts/` describes contract changes; every plan includes a task to apply them to `openapi.yaml` and regenerate types.

### TDD (mandatory)
- Write tests **before** code, derived from the spec's acceptance criteria: red → green → refactor. Confirm each new test fails for the right reason before implementing.
- **Code adapts to tests, never tests to code.** Never modify, weaken or delete a test to make it pass. If a test looks wrong, stop and check it against the spec; if the spec is wrong, fix the spec first.
- In `tasks.md`, test tasks come before implementation tasks in every phase.
- A feature is done only when all tests, formatting, lint and typecheck pass.

### Docs stay in sync
Every change includes, in the same commit/PR, updates to whatever it affects: `openapi.yaml`, specs, README, CLAUDE.md files, constitution, `.env.example`, generated code (`make sqlc`, `make gen-types`). A change without its docs is incomplete.

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
  - API: `sqlc diff`, golangci-lint (format + lint), `govulncheck`, `go test -race` (including integration), build, docker build.
  - Web: generated-types check, `pnpm format:check`, `pnpm lint`, typecheck, `pnpm audit`, tests, build.
- Deploy: API + Postgres on Railway (Wait for CI, health check on `/healthz`); web on Vercel (SPA rewrite to `index.html`).
