# Product Catalog

[Versión en español](README.md) (main version)

Product catalog management: a REST API in Go and a React web app, in one repository.

## Requirements

- Docker with Compose v2
- make

Only development mode also needs Go 1.27.4+, Node.js 22.22+ and pnpm (`corepack enable`).

## Run the project

```sh
make up     # Postgres, API and web; waits until they are ready
make seed   # optional: 20 sample products
```

- Web: <http://localhost:3000>
- API: <http://localhost:8080> (docs at <http://localhost:8080/docs>)

`make down` stops it. It works with no setup; to change values, copy `.env.example` to `.env`.

## Development mode

With hot reload (needs Go and Node):

```sh
cp .env.example .env
make db        # Postgres only
make dev-api   # API on :8080
make dev-web   # web on http://localhost:5173
```

## Tests and quality

With Docker only (same Go and Node versions as CI):

```sh
make test-docker   # API and web
make lint-docker   # lint, formatting and typecheck
```

With Go and Node installed, faster day to day: `make test` and `make lint`.

The API integration tests start their own Postgres with testcontainers: Docker must be running.

`make help` lists every command.

## Layout

| Path | Contents |
| --- | --- |
| `api/` | Go + Gin API ([api/CLAUDE.md](api/CLAUDE.md)) |
| `web/` | React + Vite + TypeScript web app ([web/CLAUDE.md](web/CLAUDE.md)) |
| `openapi.yaml` | API contract, the single source of truth |
| `specs/` | Per-feature specifications (Spec Kit) |

## Environment variables

`.env.example` documents all of them. Production values live in Railway and Vercel.

| Variable | Used by | Description |
| --- | --- | --- |
| `POSTGRES_USER`, `POSTGRES_PASSWORD`, `POSTGRES_DB` | `db` (compose) | Local credentials (`catalog` by default). |
| `DATABASE_URL` | API | Postgres connection string. |
| `PORT` | API | HTTP port (`8080`). |
| `CORS_ALLOWED_ORIGINS` | API | Comma-separated origins, `http(s)://host[:port]`. `*` is rejected. |
| `GIN_MODE` | API | `release` in production. |
| `CLIENT_IP_HEADER` | API | Header with the real client IP, for rate limiting. Empty locally; `X-Real-IP` on Railway. |
| `VITE_API_URL` | web (build time) | Root URL of the API, e.g. `http://localhost:8080`. |

## Deployment

Every merge to `main` deploys the API to Railway and the web to Vercel.

**API and Postgres on Railway**

1. Create a project with a **PostgreSQL** 18 service.
2. Add a service from this repository: Root Directory `/api`, Config file path `/api/railway.toml`, Watch Paths `/api/**` and **Wait for CI** on.
3. Variables: `DATABASE_URL=${{Postgres.DATABASE_URL}}`, `CORS_ALLOWED_ORIGINS=<web URL>`, `PORT=8080`, `GIN_MODE=release`, `CLIENT_IP_HEADER=X-Real-IP`.
4. Generate a public domain under **Networking**: it is the web's `VITE_API_URL`.

**Web on Vercel**

1. Import the repository with Root Directory `web` and the **Vite** preset.
2. Set `VITE_API_URL` to the API domain. If the build cannot find pnpm 12, add `ENABLE_EXPERIMENTAL_COREPACK=1`.

**GitHub**: protect `main` by requiring a PR, the `OpenAPI`, `API (Go)` and `Web (React)` checks, and up-to-date branches before merging. Vercel does not wait for CI, so this protection is what guarantees only tested code ships.
