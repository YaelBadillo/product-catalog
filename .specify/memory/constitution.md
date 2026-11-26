# Product Catalog Constitution

## Core Principles

### I. Spec-Driven Development

- Every user-facing feature MUST go through Spec Kit (specify → clarify → plan → tasks → analyze →
  implement) as one vertical slice (API + web), on its own branch with its own `specs/NNN-feature/`.
- Infrastructure and tooling work MUST be planned in Claude Code plan mode on a `chore/...` branch;
  it needs no spec.
- Nothing outside the spec is implemented. When a requirement changes, the spec MUST be updated
  before the tests and the code.

### II. Test-Driven Development (NON-NEGOTIABLE)

- Tests MUST be written before the code they cover, derived from the acceptance criteria, and MUST
  fail first for the right reason (red → green → refactor). This applies to all code, `chore` work
  included.
- Code adapts to tests, never the reverse: modifying, weakening or deleting a test to make it pass
  is forbidden. A test that looks wrong is checked against the spec; if the spec is wrong, the spec
  is fixed first.
- In `tasks.md`, test tasks MUST precede implementation tasks in every phase.

### III. Contract First

- `openapi.yaml` is the single source of truth for the API. Contract changes MUST land in it before
  implementation, followed by type regeneration.
- Generated code (frontend TS types, sqlc output) MUST NOT be edited by hand.
- Backend tests MUST validate every response against the contract.

### IV. Layered Architecture and Dependency Inversion

- Backend: `handler → service → repository`. Each layer depends on small interfaces defined by its
  consumer; constructors accept interfaces and return concrete types.
- Dependencies are wired by hand in the composition root (`cmd/api/main.go`): no DI containers, no
  globals.
- Frontend is organized by feature: components → hooks → feature API → HTTP client. A feature MUST
  NOT import from another feature, and components MUST NOT call the HTTP client directly.

### V. Code Quality

- Code follows SOLID, Clean Code and the idioms of its language.
- Formatting and lint MUST pass as configured in `api/CLAUDE.md` and `web/CLAUDE.md`.
- A lint rule MUST NOT be disabled (`//nolint`, `eslint-disable` or config) without a written
  justification next to it.

### VI. Security (OWASP Top 10)

- All input (body, params, query) MUST be validated on the server, even when the web validates it.
- SQL MUST use parameterized queries only; never concatenate input into SQL.
- CORS MUST list explicit origins, never `*`.
- Secrets live only in environment variables: never in the repo, logs or `VITE_*` variables.
- Errors MUST NOT expose internals: a 500 returns a generic message and the request ID.
- The API MUST rate-limit requests, and CI MUST scan dependencies for known vulnerabilities.

### VII. Accessibility and User Experience

- The UI MUST follow the component strategy defined in `web/CLAUDE.md`.
- Every view MUST be operable by keyboard and screen reader, and its tests MUST include an axe
  check.
- Every view MUST handle loading, empty and error states, and every action MUST give visible
  feedback.

### VIII. Documentation in Sync

- Every change MUST update, in the same PR, whatever it affects: contract, specs, README, CLAUDE.md
  files, this constitution, `.env.example` and generated code.
- A change without its documentation is incomplete and MUST NOT be merged.

### IX. Simplicity

- Implement only what the spec asks for.
- Every new dependency MUST be justified in the plan.
- Any additional complexity MUST be justified in the plan's Complexity Tracking section.

### X. Continuous Delivery

- `main` MUST always be green and deployable.
- Every change MUST reach `main` through a PR with the full CI green; no direct pushes.
- Production deploys are automatic and MUST ship only changes that passed CI.

## Governance

- This constitution supersedes all other guidance. The CLAUDE.md files complement it with technical
  detail and MUST NOT contradict it.
- Every plan MUST pass the Constitution Check against each principle before implementation;
  exceptions MUST be justified in the plan's Complexity Tracking section.
- Amendments MUST bump the version, update **Last Amended**, state the reason in the commit and PR,
  and propagate to the CLAUDE.md files and affected templates in the same PR.
- Versioning: MAJOR removes or redefines a principle; MINOR adds a principle or materially expands
  guidance; PATCH clarifies wording.

**Version**: 1.0.0 | **Ratified**: 2026-11-25 | **Last Amended**: 2026-11-25
