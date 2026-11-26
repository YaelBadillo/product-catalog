# web/CLAUDE.md

Vite + React + TypeScript in strict mode (no `any`). The domain model, API contract and workflow (Spec Kit, TDD) are in the root `CLAUDE.md`.

## Commands (from `web/`)

```sh
pnpm dev                                   # dev server
pnpm test                                  # all tests (vitest run)
pnpm test src/features/products/pages/     # a folder or a single file path
pnpm test -t "ProductListPage > shows empty state"  # by name; Vitest 5 joins describe/test names with " > "
pnpm typecheck && pnpm lint && pnpm format:check
pnpm format                                # apply Prettier
pnpm gen:api                               # regenerate src/api/schema.d.ts from ../openapi.yaml
```

`src/api/schema.d.ts` is generated: never edit it by hand. From the repo root the Makefile wraps these (`make dev-web`, `make test-web`, `make lint-web`, `make gen-types`).

## Architecture

- `src/app` — providers and router (`App.tsx`).
- `src/api` — HTTP infrastructure only: `openapi-fetch` client, generated `schema.d.ts`, error types.
- `src/features/<feature>/` — `api.ts`, `hooks.ts`, `query-keys.ts`, `schema.ts`, `components/`, `pages/`. Existing: `status` (API and database availability, at `/`).
- `src/components/ui` — generic components with no business logic.
- `src/lib` — utilities. `src/test` — test setup and MSW handlers. Tests live next to the file they test.
- Import rules: `features` may import from `components/ui`, `lib` and `api`; those never import from `features`; a feature never imports from another feature.
- Dependency flow: components → hooks → `features/*/api.ts` → HTTP client. Components never call `fetch` or the client directly. No DI containers or classes.
- Split container components (use hooks) from presentational ones (data and callbacks via props). Keep logic in hooks and pure functions.
- Shared dependencies are provided via Context (`QueryClientProvider`) so tests can swap them; the network is swapped with MSW.

## Conventions

- Server state: TanStack Query. Keys centralized in `query-keys.ts`; invalidate after mutations; optimistic updates for edit and delete.
- Routing: `react-router`. Filters, search and pagination live in URL query params.
- Forms: `react-hook-form` + `zod` + `@hookform/resolvers`. Map API validation `details[]` onto fields with `setError`. Price is entered in pesos and sent as cents with `Math.round(price * 100)`.
- HTTP: `openapi-fetch` with `openapi-typescript` types. No axios. The client (`src/api/client.ts`) resolves `fetch` per call; keep it that way or MSW can't intercept it.
- Env vars validated with zod at startup (`src/lib/env.ts`). `VITE_API_URL` is the API host root and is inlined at build time (the Docker build requires it). No secrets in `VITE_*`.
- Routing: React Router 8 — `RouterProvider` comes from `react-router/dom`, everything else from `react-router`.
- UI: no component libraries. Tailwind + own base components in `src/components/ui` (Button, Input, Select, Dialog, Table, Pagination, Skeleton, Toast); icons from `lucide-react`; class composition with `clsx`. Use native semantic elements: `<dialog>` with `showModal()`, `<select>`, `<table>`, `<button>`, `<label>`.
- Accessibility (required): associated labels, `aria-invalid` + `aria-describedby` on errors, `aria-label` on icon-only buttons, visible focus, keyboard navigation, `aria-live` region for notifications.
- Every view has loading, empty and error states; buttons are disabled during requests; `react-error-boundary` for render errors.
- Never use `dangerouslySetInnerHTML`.
- Lint/format: ESLint with `typescript-eslint` (`strictTypeChecked`), `react-hooks`, `react-refresh` (component files export only components, or Vite's Fast Refresh breaks), `jsx-a11y`, `@tanstack/eslint-plugin-query`; Prettier + `prettier-plugin-tailwindcss`; `eslint-config-prettier`. No `eslint-disable` without a justifying comment.

## Testing

- Vitest + jsdom + Testing Library (`react`, `dom`, `jest-dom`, `user-event`) + MSW + `vitest-axe`.
- `src/test/setup.ts` wires jest-dom and vitest-axe matchers, RTL cleanup, and the MSW `server` (`src/test/server.ts`; add per-test handlers with `server.use`). Vitest globals are off: import `describe`/`it`/`expect` from `vitest`.
- Priority: per-page integration tests derived from acceptance criteria (list, create, detail, edit, delete). MSW intercepts the network with `onUnhandledFrame: "error"` (MSW 3's name for `onUnhandledRequest`). Never mock the client or the hooks.
- Unit tests for pure functions and zod schemas.
- Query by role or label; `getByTestId` only as a last resort.
- Render with `renderWithProviders` (`src/test/render.tsx`): fresh `QueryClient` per test with `retry: false`, a `MemoryRouter`, and a `user` from `user-event`.
- Default MSW handlers (every service healthy) live in `src/test/handlers.ts`; build URLs with `apiUrl(path)`. Tests run with `VITE_API_URL=http://api.test` (`test.env` in `vite.config.ts`).
- Interactions always via `user-event`, never `fireEvent`.
- Run an `axe` check on every page.

## Dependency notes

- `vitest-axe` is pinned to `1.0.0-pre.5`: its `latest` tag (`0.1.0`) has types incompatible with current Vitest. Don't move it to `latest`.
- Known unmet peer warnings, verified working: `eslint-plugin-jsx-a11y` (declares ESLint ≤ 9) and `openapi-typescript` (declares TypeScript 5).
- pnpm is pinned by `packageManager` in `package.json` (used by corepack and CI). pnpm 12 records it in the lockfile, so changing it requires `pnpm install`.
- React Router 8 requires Node 22.22+.
