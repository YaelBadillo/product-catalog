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

- `src/app` — `App.tsx` (providers + `createBrowserRouter`), `routes.tsx` (the route array, shared with the tests' `renderRoute`), `AppLayout.tsx` (header navigation), `NotFoundPage.tsx`, Mantine theme (`theme.ts`) and `UiProvider` (`providers.tsx`). Routes: `/` → `/products`, `/products`, `/products/new`, `/products/:id`, `/products/:id/edit`, `/status`, `*` (contract in `specs/001-product-management/contracts/web-routes.md`).
- `src/api` — HTTP infrastructure only: `openapi-fetch` client, generated `schema.d.ts`, error types.
- `src/features/<feature>/` — `api.ts`, `hooks.ts`, `query-keys.ts`, `schema.ts`, `components/`, `pages/`. Existing: `products` (catalog, create, detail, edit, delete; also `search-params.ts` + `use-catalog-search.ts` for the URL state, `format.ts`, `messages.ts`, `testing.ts` with fixtures and MSW handlers for tests) and `status` (API and database availability, at `/status`).
- `src/components/ui` — own generic compositions of Mantine components, with no business logic. Never re-implement what Mantine already provides.
- `src/lib` — utilities (`env.ts`, `use-document-title.ts`). `src/test` — test setup and MSW handlers. Tests live next to the file they test.
- Import rules: `features` may import from `components/ui`, `lib` and `api`; those never import from `features`; a feature never imports from another feature.
- Dependency flow: components → hooks → `features/*/api.ts` → HTTP client. Components never call `fetch` or the client directly. No DI containers or classes, except `Error` subclasses such as `ApiError`, which must be throwable.
- Split container components (use hooks) from presentational ones (data and callbacks via props). Keep logic in hooks and pure functions.
- Shared dependencies are provided via Context (`QueryClientProvider`) so tests can swap them; the network is swapped with MSW.

## Conventions

- Server state: TanStack Query. Keys centralized in `query-keys.ts`; invalidate after mutations. Optimistic updates only for delete (removed from every cached list, rolled back on failure). Edits wait for the answer: a version conflict must leave every cache and the form untouched. Every API function throws `ApiError` (`src/api/errors.ts`).
- Routing: `react-router`. Filters, search, sort and pagination live in URL query params, parsed with defaults for invalid values (`search-params.ts`). Filter, sort and page changes push a history entry; the search applies 300 ms after typing (or on Enter) and pushes once per typing session, then replaces.
- Forms: `react-hook-form` + `zod` + `@hookform/resolvers`, not `@mantine/form`. Mantine inputs take `{...register("field")}` and `error={errors.field?.message}`; `Select` and `NumberInput` go through `Controller`. Map API validation `details[]` onto fields with `setError`, using the Spanish rule message of each field (the API's English message is never shown). Price is entered in currency units with `PriceInput` (thousands separators, no `decimalScale` or clamping: the schema reports the error; a typed comma is dropped with a hint) and sent as cents with `Math.round(price * 100)`.
- HTTP: `openapi-fetch` with `openapi-typescript` types. No axios. The client (`src/api/client.ts`) resolves `fetch` per call; keep it that way or MSW can't intercept it.
- Env vars validated with zod at startup (`src/lib/env.ts`). `VITE_API_URL` is the API host root and is inlined at build time (the Docker build requires it). No secrets in `VITE_*`.
- Routing: React Router 8 — `RouterProvider` comes from `react-router/dom`, everything else from `react-router`.
- UI: Mantine v9. `@mantine/core` and `@mantine/hooks` for components, `@mantine/notifications` for feedback, `@mantine/modals` for confirmations. Custom styles go through Mantine style props or CSS modules (`*.module.css`, processed by `postcss.config.cjs` with `postcss-preset-mantine` and the Mantine breakpoints); no Tailwind or other CSS framework. Icons from `lucide-react`. `UiProvider` applies the theme with the light scheme forced and mounts `Notifications` and `ModalsProvider`; the colors are in [Colors](#colors).
- Accessibility (required): every input has a `label`; Mantine inputs set `aria-invalid` and `aria-describedby` from `error`. `aria-label` on icon-only buttons, visible focus, keyboard navigation. Notifications come from `@mantine/notifications` (`role="alert"`). `jsx-a11y` can't see inside Mantine components, so the per-page `axe` check is the main guard.
- Every view has loading, empty and error states; buttons are disabled during requests; `react-error-boundary` for render errors.
- Never use `dangerouslySetInnerHTML`.
- UI copy: Mexican Spanish, addressing the user as tú, centralized in `features/products/messages.ts` (components never hard-code text). `index.html` declares `lang="es-MX"`, and each page sets `document.title` with `useDocumentTitle`. Code and docs stay in English.
- Formatting: `formatPrice(cents, currency)` → `$1,299.90 MXN` (`es-MX`, `currencyDisplay: "narrowSymbol"` plus the code); `formatDateTime(iso)` → `25 nov 2026, 14:30` (`es-MX`, medium date, short time, 24 h, user's time zone).
- Responsive list: `useMediaQuery("(min-width: 48em)")` renders a `Table` on wide screens and stacked cards below; only one is rendered. Pages never scroll horizontally from 360 px.
- Lint/format: ESLint with `typescript-eslint` (`strictTypeChecked`), `react-hooks`, `react-refresh` (component files export only components, or Vite's Fast Refresh breaks), `jsx-a11y`, `@tanstack/eslint-plugin-query`; Prettier with default options; `eslint-config-prettier`. No `eslint-disable` without a justifying comment.

## Colors

The theme (`src/app/theme.ts`) defines `brand` (generated from `#4c5fd5`, primary shade 6), black `#14161a` and white. Its `cssVariablesResolver` sets `dimmed` to `gray.7` and default borders to `gray.3`.

| Role                          | Color                                            | In Mantine                                |
| ----------------------------- | ------------------------------------------------ | ----------------------------------------- |
| App background                | `#ffffff`                                        | Default                                   |
| Header                        | `#14161a` with white text                        | `bg="black" c="white"`                    |
| Active header link            | `brand.2`, underline `brand.3`                   | `c="brand.2"`                             |
| Primary button                | `brand.6`, hover `brand.7`                       | `<Button>` default                        |
| Secondary button              | `gray.3` border, black text                      | `variant="default"`                       |
| Danger button                 | `red.9`                                          | `color="red.9"`                           |
| Error alert                   | `red.9` border and title, transparent background | `<Alert variant="outline" color="red.9">` |
| Subtle surfaces, table header | `gray.0`                                         | `bg="gray.0"`                             |
| Borders                       | `gray.3` (inputs keep Mantine's `gray.4`)        | Default                                   |
| Secondary text                | `gray.7`                                         | `c="dimmed"`                              |
| Active / inactive state       | `green.8` / `gray.5` dot + text                  | Decorative icon + text                    |

- Never use `red` at its default shades for white text or button backgrounds (3.3:1); always `red.9`.
- Color is never the only indicator of a state: always pair it with text or a labelled icon.
- No colors outside this palette (brand, grays, `red.9`, `green.8`) without updating this file.
- `src/app/theme.test.ts` checks every text pair of this table for WCAG AA (4.5:1); add the pair there when adding a role.

## Testing

- Vitest + jsdom + Testing Library (`react`, `dom`, `jest-dom`, `user-event`) + MSW + `vitest-axe`.
- `src/test/setup.ts` wires jest-dom and vitest-axe matchers, RTL cleanup, the MSW `server` (`src/test/server.ts`; add per-test handlers with `server.use`), and the jsdom stubs from Mantine's Vitest guide (`getComputedStyle` without pseudo-elements, `scrollIntoView`, `matchMedia`, `document.fonts`, `ResizeObserver`). Vitest globals are off: import `describe`/`it`/`expect` from `vitest`.
- Priority: per-page integration tests derived from acceptance criteria (list, create, detail, edit, delete). MSW intercepts the network with `onUnhandledFrame: "error"` (MSW 3's name for `onUnhandledRequest`). Never mock the client or the hooks.
- Unit tests for pure functions and zod schemas.
- Query by role or label; `getByTestId` only as a last resort.
- Pages are tested through `renderRoute(path)` (`src/test/render.tsx`): the app's routes in a memory router with the same providers; `path` may be a list of history entries, and it returns the `router` to check the location, navigate and go back. Components render with `renderWithProviders`: `UiProvider` with `env="test"` (app theme, no transitions or portals, so notifications and modals render inside `container`), a fresh `QueryClient` per test with `retry: false`, a `MemoryRouter`, and a `user` from `user-event`. Only `App`, which brings its own providers, uses plain `render`.
- Mantine notifications also have `role="alert"`: scope alert queries with `within` when a notification and an inline alert can both be on screen. `setup.ts` clears them after each test (Mantine keeps them in a global store).
- Default MSW handlers (every service healthy) live in `src/test/handlers.ts`; build URLs with `apiUrl(path)`. Tests run with `VITE_API_URL=http://api.test` (`test.env` in `vite.config.ts`).
- Interactions always via `user-event`, never `fireEvent`.
- Run an `axe` check on every page (both list layouts; with dialogs open), importing `axe` from `src/test/axe.ts` (WCAG 2.2 AA tags), not from `vitest-axe`.
- Query fields by accessible name (`getByRole`): Mantine's required asterisk is part of the label text but not of the name.

## Dependency notes

- `vitest-axe` is pinned to `1.0.0-pre.5`: its `latest` tag (`0.1.0`) has types incompatible with current Vitest. Don't move it to `latest`.
- Known unmet peer warnings, verified working: `eslint-plugin-jsx-a11y` (declares ESLint ≤ 9) and `openapi-typescript` (declares TypeScript 5).
- pnpm is pinned by `packageManager` in `package.json` (used by corepack and CI). pnpm 12 records it in the lockfile, so changing it requires `pnpm install`.
- React Router 8 requires Node 22.22+.
- Mantine 9 requires React 19.2+. The `@mantine/*` packages require each other at the exact same version: upgrade them together.
