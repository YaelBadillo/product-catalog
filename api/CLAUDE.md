# api/CLAUDE.md

Go + Gin REST API. The domain model, API contract and workflow (Spec Kit, TDD) are in the root `CLAUDE.md`.

## Commands (from `api/`)

```sh
go test -race -short ./...                              # unit tests only (fast TDD loop)
go test -race ./...                                     # everything, incl. integration (needs Docker)
go test -race -run '^TestName$/subtest' ./internal/service/
go test -fuzz='^FuzzName$' -fuzztime=30s ./internal/<pkg>/
go test -run='^$' -bench=BenchmarkList -benchtime=200x ./internal/repository/   # list p95 over 10,000 products
go tool -modfile=tools/golangci-lint.mod golangci-lint fmt         # format (gofumpt + goimports)
go tool -modfile=tools/golangci-lint.mod golangci-lint run ./...   # lint
go tool -modfile=tools/sqlc.mod sqlc generate                      # after editing queries/ or migrations/
go tool -modfile=tools/goose.mod goose -dir migrations create <name> sql
go tool -modfile=tools/govulncheck.mod govulncheck ./...
```

Always run tests with `-race`. From the repo root the Makefile wraps these (`make test-api`, `make lint-api`, `make sqlc`, `make sync-openapi`, `make seed`, `make migrate-new name=...`).

Dev tools are pinned with `go tool`, each in its own modfile (`tools/<tool>.mod` + `.sum`), so no tool's dependencies get bumped by another's (golangci-lint breaks subtly if `golang.org/x/tools` moves). Never merge them or add them to `go.mod`. Upgrade one with `go get -tool -modfile=tools/<tool>.mod <pkg>@<version>`. `go.mod` gains a dependency only when code imports it (`go get`, then `go mod tidy`).

## Architecture

`handler → service → repository`, with dependency inversion at every boundary:
- The consuming package defines a small interface with only the methods it uses: `handler` defines the service interface, `service` defines the repository interface. Never depend on concrete implementations across layers.
- Constructors accept interfaces and return concrete structs.
- Manual constructor injection in `cmd/api/main.go` (composition root). No DI containers, no globals.
- No interface without a consumer that needs it (e.g. none for config or the logger).

Layout:
- `cmd/api/` — `main.go`, composition root: migrate → repository → service → handlers. `api healthcheck` probes `/healthz` and exits 0/1 (the distroless image has no curl); `api seed` migrates and loads the sample products through the service.
- `internal/config` — config from env vars (`caarlos0/env`): `PORT` (8080), `DATABASE_URL`, `CORS_ALLOWED_ORIGINS` (explicit `http(s)://host[:port]` origins, validated at startup), `CLIENT_IP_HEADER` (optional: the header the platform's edge fills with the client IP, `X-Real-IP` on Railway; empty uses the connection's address).
- `internal/healthcheck` — the client behind `api healthcheck`.
- `internal/domain` — `Product`, `Category`, `Currency`, `ProductInput`, `ProductPatch`; normalization (`NormalizeSKU`, …) and every rule, defined once (`*ValidationError` with JSON field names); sentinel errors (`ErrNotFound`, `ErrSKUAlreadyExists`, `ErrVersionConflict`).
- `internal/service` — business rules: normalizes and validates every create and update (the seed goes through it too); `ListParams` (sort whitelist) and `ProductPage`.
- `internal/repository` — Postgres implementation. `repository/db` is sqlc output: never edit by hand. A versioned write that changes no row checks `ProductExists` to answer `ErrNotFound` or `ErrVersionConflict`; `List` runs page and count in one read-only `REPEATABLE READ` transaction and escapes `\`, `%`, `_` in the search.
- `internal/httpapi` — router and the **single** error → HTTP mapping (`ErrorHandler`: handlers report with `c.Error(err)`); `handler/` (handlers, DTOs, `DecodeJSON`, the validator, `ParseListParams`) and `middleware/`. `NewRouter(logger, RouterConfig, health, api ...func(*gin.RouterGroup))` mounts each registrar on `/api/v1`; each handler exposes `Register`.
- `internal/migrate` — `Up` applies the embedded migrations with a goose `Provider` and a Postgres session lock, waiting up to 30 s for the database.
- `internal/seed` — the 20 sample products and `Run`, which skips SKUs already in use.
- `internal/apidocs` — `/docs` (Swagger UI from jsDelivr, pinned with SRI, own CSP) and `/docs/openapi.yaml`, a copy of the root contract kept in sync by `make sync-openapi` (CI checks it).
- `internal/testutil` — `AssertMatchesContract` (validates a request and its response against `openapi.yaml`), `AssertResponseMatchesContract` (only the response, for intentionally invalid requests), and Postgres helpers: `StartPostgres` (testcontainers, for `TestMain`), `NewMigratedPool`, `ResetDB`, `RequireDSN` (skip under `-short`).
- `migrations/` (`*.sql` + `embed.go`), `queries/` (SQL for sqlc).
- Never name a package after a stdlib package (hence `httpapi`, not `http`).

## Conventions

- Persistence: `pgx/v5` (`pgxpool`), queries via sqlc, goose migrations embedded with `embed.FS` and applied at startup before serving: `migrate.Up` retries for up to 30 s and the process exits if it fails, so the platform never promotes a deployment with an outdated schema.
- Validation: handlers decode with `DecodeJSON(c, &dto, validate)` (unknown fields, trailing data and wrong types → 400 `*RequestError`; over the body limit → 413), never `ShouldBindJSON`, which would need Gin's global binding settings. The `*validator.Validate` is injected (`NewValidator`): tags check shape, and the custom tags `sku` and `trimmed_len` call the domain so rules are measured after normalizing. The service normalizes and validates again (422). Constraints are duplicated in the DB (`CHECK`, partial `UNIQUE`).
- DTOs are separate from the domain model. Every field of a `PATCH` DTO is a pointer.
- Errors: wrap with `%w`; check with `errors.Is` / `errors.As`; no `panic` in normal flow. Postgres `23505` becomes `ErrSKUAlreadyExists`. Domain errors are translated to HTTP only in `internal/httpapi`.
- `context.Context` is the first parameter of every function doing I/O and propagates down to the DB. Pass `c.Request.Context()` to the service, never `*gin.Context`.
- Logging: `log/slog`, structured, with request ID, no sensitive data.
- Middleware: request ID, logging, recovery, the error handler, security headers and CORS (`gin-contrib/cors`, explicit origins from env, never `*`) on every route; on `/api/v1` a 64 KiB body limit and per-IP rate limiting (`golang.org/x/time/rate`, 5/s with bursts of 300). The limiter is created in `main.go`, which runs its janitor. The client IP is `c.ClientIP()` with `SetTrustedProxies(nil)` (never `X-Forwarded-For`) and `TrustedPlatform = CLIENT_IP_HEADER`. Recovery is custom (`internal/httpapi/errors.go`) so panics answer with the error envelope.
- Server: timeouts configured; graceful shutdown via `signal.NotifyContext` + `srv.Shutdown`; Gin in release mode in production (`GIN_MODE=release`).
- Support endpoints: `/healthz` (process alive), `/readyz` (DB ping), `/docs` (Swagger UI serving the embedded `openapi.yaml`). The health handler depends on a `Pinger` it defines, not on a service: it checks infrastructure, not domain. `//go:embed` cannot reach files outside the module (no `..`), so the root `openapi.yaml` must be synced into `api/` to embed it.
- Go version: when bumping, update `go.mod` and `GO_VERSION` in `Dockerfile` together (CI reads `go.mod`).
- SQL safety: parameterized queries only (sqlc). Sort columns come from a whitelist; never concatenate input into SQL.
- Lint: golangci-lint v2 (`.golangci.yml`). Formatters `gofumpt` + `goimports`; linters `standard` + `gosec`, `errorlint`, `contextcheck`, `exhaustive`, `nilerr`, `revive`, `gocritic`, `misspell`. No `//nolint` without a justifying comment.

## Testing

- `testing` + `testify`: `require` for preconditions, `assert` for checks. Table-driven, black-box (`package xxx_test`).
- Service: unit tests with hand-written repository fakes.
- Handlers: `httptest` through `httpapi.NewRouter` with a fake service. Assert status, headers (`Location`), error shape, and that every response conforms to `openapi.yaml` with `testutil.AssertMatchesContract` (`kin-openapi`), or `AssertResponseMatchesContract` for intentionally invalid requests. Set `req.GetBody` so the assertion can validate a body the handler already read.
- Repository: integration tests with `testcontainers-go` — real Postgres with migrations applied, one container per package via `TestMain` (`testutil.StartPostgres` + `NewMigratedPool`), `testutil.ResetDB` per test. Cover soft delete, SKU uniqueness, and version conflicts under concurrent updates.
- Startup: black-box tests in `cmd/api/main_test.go` build and run the binary (migrations at startup, `api seed`).
- Integration tests skip under `-short`; CI runs them all.
- Native fuzzing (`testing.F`) for SKU validation and normalization.
