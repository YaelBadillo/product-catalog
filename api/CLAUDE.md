# api/CLAUDE.md

Go + Gin REST API. The domain model, API contract and workflow (Spec Kit, TDD) are in the root `CLAUDE.md`.

## Commands (from `api/`)

```sh
go test -race -short ./...                              # unit tests only (fast TDD loop)
go test -race ./...                                     # everything, incl. integration (needs Docker)
go test -race -run '^TestName$/subtest' ./internal/service/
go test -fuzz='^FuzzName$' -fuzztime=30s ./internal/<pkg>/
go tool -modfile=tools/golangci-lint.mod golangci-lint fmt         # format (gofumpt + goimports)
go tool -modfile=tools/golangci-lint.mod golangci-lint run ./...   # lint
go tool -modfile=tools/sqlc.mod sqlc generate                      # after editing queries/ or migrations/
go tool -modfile=tools/goose.mod goose -dir migrations create <name> sql
go tool -modfile=tools/govulncheck.mod govulncheck ./...
```

Always run tests with `-race`.

Dev tools are pinned with `go tool`, each in its own modfile (`tools/<tool>.mod` + `.sum`), so no tool's dependencies get bumped by another's (golangci-lint breaks subtly if `golang.org/x/tools` moves). Never merge them or add them to `go.mod`. Upgrade one with `go get -tool -modfile=tools/<tool>.mod <pkg>@<version>`. `go.mod` gains a dependency only when code imports it (`go get`, then `go mod tidy`).

## Architecture

`handler → service → repository`, with dependency inversion at every boundary:
- The consuming package defines a small interface with only the methods it uses: `handler` defines the service interface, `service` defines the repository interface. Never depend on concrete implementations across layers.
- Constructors accept interfaces and return concrete structs.
- Manual constructor injection in `cmd/api/main.go` (composition root). No DI containers, no globals.
- No interface without a consumer that needs it (e.g. none for config or the logger).

Layout:
- `cmd/api/` — `main.go`, composition root.
- `internal/config` — config from env vars (`caarlos0/env`).
- `internal/domain` — `Product`, `Category`, sentinel errors (`ErrNotFound`, `ErrSKUAlreadyExists`, `ErrVersionConflict`, `ErrInsufficientStock`).
- `internal/service` — business rules.
- `internal/repository` — Postgres implementation. `repository/db` is sqlc output: never edit by hand.
- `internal/httpapi` — router and the **single** domain-error → HTTP mapping; `handler/` (handlers + DTOs) and `middleware/`.
- `internal/testutil` — testcontainers Postgres, migrations, fixtures.
- `migrations/` (`*.sql` + `embed.go`), `queries/` (SQL for sqlc).
- Never name a package after a stdlib package (hence `httpapi`, not `http`).

## Conventions

- Persistence: `pgx/v5` (`pgxpool`), queries via sqlc, goose migrations embedded with `embed.FS` and applied at startup.
- Validation: `go-playground/validator` tags via `ShouldBindJSON` (never `BindJSON`) for shape and format; business rules in the service; constraints duplicated in the DB (`CHECK`, `UNIQUE`). Reject unknown fields.
- DTOs are separate from the domain model. Every field of a `PATCH` DTO is a pointer.
- Errors: wrap with `%w`; check with `errors.Is` / `errors.As`; no `panic` in normal flow. Postgres `23505` becomes `ErrSKUAlreadyExists`. Domain errors are translated to HTTP only in `internal/httpapi`.
- `context.Context` is the first parameter of every function doing I/O and propagates down to the DB. Pass `c.Request.Context()` to the service, never `*gin.Context`.
- Logging: `log/slog`, structured, with request ID, no sensitive data.
- Middleware: recovery, request ID, logging, CORS (`gin-contrib/cors`, explicit origins from env, never `*`), per-IP rate limiting (`golang.org/x/time/rate`), body size limit, security headers.
- Server: timeouts configured; graceful shutdown via `signal.NotifyContext` + `srv.Shutdown`; Gin in release mode in production.
- Support endpoints: `/healthz` (process alive), `/readyz` (DB ping), `/docs` (Swagger UI serving the embedded `openapi.yaml`). `//go:embed` cannot reach files outside the module (no `..`), so the root `openapi.yaml` must be synced into `api/` to embed it.
- SQL safety: parameterized queries only (sqlc). Sort columns come from a whitelist; never concatenate input into SQL.
- Lint: golangci-lint v2 (`.golangci.yml`). Formatters `gofumpt` + `goimports`; linters `standard` + `gosec`, `errorlint`, `contextcheck`, `exhaustive`, `nilerr`, `revive`, `gocritic`, `misspell`. No `//nolint` without a justifying comment.

## Testing

- `testing` + `testify`: `require` for preconditions, `assert` for checks. Table-driven, black-box (`package xxx_test`).
- Service: unit tests with hand-written repository fakes.
- Handlers: `httptest` with a fake service. Assert status, headers (`Location`), error shape, and that every response conforms to `openapi.yaml` (`kin-openapi`).
- Repository: integration tests with `testcontainers-go` — real Postgres with migrations applied, one container per package via `TestMain`, data isolated per test. Cover soft delete, SKU uniqueness, and version conflicts under concurrent updates.
- Integration tests skip under `-short`; CI runs them all.
- Native fuzzing (`testing.F`) for SKU validation and normalization.
