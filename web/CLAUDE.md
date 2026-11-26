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

`src/api/schema.d.ts` is generated: never edit it by hand.

## Architecture

- `src/app` — providers and router.
- `src/api` — HTTP infrastructure only: `openapi-fetch` client, generated `schema.d.ts`, error types.
- `src/features/products/` — `api.ts`, `hooks.ts`, `query-keys.ts`, `schema.ts`, `components/`, `pages/`.
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
- HTTP: `openapi-fetch` with `openapi-typescript` types. No axios.
- Env vars validated with zod at startup (`VITE_API_URL`). No secrets in `VITE_*`.
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
- Render with `renderWithProviders`, using a fresh `QueryClient` per test with `retry: false`.
- Interactions always via `user-event`, never `fireEvent`.
- Run an `axe` check on every page.

## Dependency notes

- `vitest-axe` is pinned to `1.0.0-pre.5`: its `latest` tag (`0.1.0`) has types incompatible with current Vitest. Don't move it to `latest`.
- Known unmet peer warnings, verified working: `eslint-plugin-jsx-a11y` (declares ESLint ≤ 9) and `openapi-typescript` (declares TypeScript 5).
