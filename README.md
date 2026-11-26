# Product Catalog

Product catalog management platform: a REST API in Go and a React web client that consumes it, in one monorepo.

| Path           | Contents                                              |
| -------------- | ----------------------------------------------------- |
| `api/`         | Go + Gin API                                          |
| `web/`         | React + Vite + TypeScript client                      |
| `openapi.yaml` | API contract, the single source of truth              |
| `Makefile`     | Every development command (`make help`)               |
| `.github/`     | CI workflow                                           |

## Requirements

- Docker with Compose v2 (for `make up` and the API integration tests)
- Go 1.27.4+
- Node.js 22.22+ or 24+, and pnpm (`corepack enable` installs the pinned version)
- make

## Getting started

```sh
cp .env.example .env   # optional: compose has local defaults
make up                # Postgres, API and web, waiting until they are healthy
```

- Web: <http://localhost:3000> (status page: API and database availability)
- API: <http://localhost:8080> (`/healthz`, `/readyz`)
- Postgres: `localhost:5432`

`make down` stops everything.

### Development mode

```sh
cp .env.example .env
make db        # Postgres only
make dev-api   # API with go run on :8080
make dev-web   # Vite dev server on :5173
```

The Makefile loads `.env` and exports its variables to these commands.

## Commands

`make help` lists every command with its description: environment (`up`, `down`, `db`, `dev-api`, `dev-web`), tests (`test`, `test-unit`, `test-api`, `test-web`), quality (`lint`, `lint-api`, `lint-web`, `fmt`) and generation (`gen-types`, `migrate-new name=...`).

## Environment variables

`.env.example` documents every variable. Local defaults are for development only; production values live in Railway and Vercel.

| Variable | Used by | Local default | Description |
| --- | --- | --- | --- |
| `POSTGRES_USER`, `POSTGRES_PASSWORD`, `POSTGRES_DB` | `db` (compose) | `catalog` | Local database credentials. |
| `DATABASE_URL` | API | required | Postgres connection string. Compose builds one for the `db` host; `.env` points at `localhost` for `make dev-api`. |
| `PORT` | API | `8080` | HTTP port. |
| `CORS_ALLOWED_ORIGINS` | API | required | Comma-separated origins, `http(s)://host[:port]`. `*` is rejected. |
| `GIN_MODE` | API | `debug` | Use `release` in production. |
| `VITE_API_URL` | web (build time) | required | Root URL of the API host, e.g. `http://localhost:8080`. |

## Deployment

Every merge to `main` deploys the API to Railway and the web to Vercel.

### API and Postgres on Railway

1. Create a project and add a **PostgreSQL** service (major version 18, as in `docker-compose.yml`).
2. Add a service from this GitHub repository and configure it:
   - **Root Directory:** `/api`.
   - **Config file path:** `/api/railway.toml` (Railway does not resolve it relative to the root directory). It sets the Dockerfile builder and the `/healthz` health check.
   - **Watch Paths:** `/api/**`.
   - **Wait for CI:** on, so a deployment starts only after the GitHub checks pass.
3. Set the variables:
   - `DATABASE_URL` = `${{Postgres.DATABASE_URL}}` (reference to the Postgres service).
   - `CORS_ALLOWED_ORIGINS` = the web's production URL, e.g. `https://<project>.vercel.app`.
   - `PORT` = `8080`.
   - `GIN_MODE` = `release`.
4. Generate a public domain under **Networking**. It is the web's `VITE_API_URL`.

### Web on Vercel

1. Import the GitHub repository with **Root Directory** `web` and the **Vite** framework preset. `vercel.json` rewrites every route to `/index.html`.
2. Set `VITE_API_URL` to the Railway API domain (`https://...`).
3. If the build cannot resolve pnpm 12, also set `ENABLE_EXPERIMENTAL_COREPACK=1` so Vercel uses the version in `package.json`'s `packageManager`.

Vercel deploys each push to `main` without waiting for the workflow. The branch protection below is what guarantees that only code that passed CI reaches `main`.

### GitHub

Protect `main` (Settings → Branches):

- Require a pull request before merging.
- Require the status checks `OpenAPI`, `API (Go)` and `Web (React)` to pass.
- Require branches to be up to date before merging, so the tested code is the merged code.

## Production

- Web: _pending_
- API: _pending_

## Technical decisions

- **One contract.** `openapi.yaml` (OpenAPI 3.1) drives both sides: the web generates its types from it (CI fails if they are stale) and API handler tests validate every response against it with kin-openapi.
- **Health probes at the host root.** `/healthz` and `/readyz` sit outside `/api/v1` because they serve platforms, not API clients. `/readyz` answers 503 with its own `{ status, checks }` shape instead of the error envelope.
- **Readiness without blocking startup.** `/readyz` pings Postgres with a 2 s timeout, and `pgxpool` connects lazily, so the API starts and reports the database as unavailable instead of crashing.
- **`Pinger` instead of a service.** The health handler depends on a one-method interface it defines; a dependency check is infrastructure, not domain, so it skips the service layer.
- **Safe failures.** A custom recovery answers panics with a generic `INTERNAL_ERROR` and the request ID; the panic and stack trace go only to the logs.
- **Request IDs.** An incoming `X-Request-ID` is reused only if it matches `[A-Za-z0-9-]{1,64}`, to prevent log injection; otherwise one is generated with `crypto/rand`.
- **Structured logs.** One JSON line per request via `slog`, with the path but not the query string, which may carry sensitive data.
- **Strict CORS.** Origins are explicit and validated at startup; `*` or a malformed origin stops the API before it serves anything.
- **Small, non-root images.** The API runs on distroless `static-debian13:nonroot`; since it has no curl, the binary's `healthcheck` subcommand backs container health checks. The web runs on `nginx-unprivileged` with an SPA fallback.
- **Build-time API URL.** Vite inlines `VITE_API_URL`, so the web image is built per environment, and its Docker build fails without the argument; the app also validates it with zod at startup.
- **Pinned versions.** Postgres 18 (Railway's major), Go from `go.mod`, pnpm via `packageManager` (recorded in the lockfile), dev tools in one `go tool` modfile each so their dependencies never shift, and the CI golangci-lint pinned to the same version.
- **Status page.** One query turns "unavailable" into data, so the last result stays visible and the Retry button stays disabled while a new check runs.
- **Interceptable client.** The openapi-fetch client resolves `fetch` on each call, so MSW can intercept it in tests.
- **CI.** One workflow with three jobs that double as required checks; newer runs on the same branch cancel older ones; actions are pinned by major version and the Redocly CLI by exact version.
- **Deploy gating.** Railway waits for CI on `main`; Vercel relies on branch protection, since it does not wait for checks.
- **Spec Kit scope.** Specs cover user-facing features; infrastructure and tooling work such as this foundation needs none and is planned in Claude Code plan mode on `chore/...` branches. The rules live in the constitution (`.specify/memory/constitution.md`).
- **Mandatory tests in Spec Kit.** Stock Spec Kit treats test tasks as optional; a `tasks-template.md` override and an edit to the `speckit-tasks` skill make them mandatory and first in every phase, as TDD requires.
- **Deferred to the first feature.** Rate limiting, body size limit, security headers, Swagger UI at `/docs`, migrations wiring, `make sqlc` and `make seed`.
