# Product Catalog — Web

React client for the Product Catalog API. It consumes every endpoint of the contract in [`../openapi.yaml`](../openapi.yaml).

## Requirements

- Node.js 22.12+ or 24+
- pnpm

## Getting started

```sh
pnpm install
cp .env.example .env.local   # set VITE_API_URL to the API base URL
pnpm dev
```

## Scripts

| Script                              | Description                                       |
| ----------------------------------- | ------------------------------------------------- |
| `pnpm dev`                          | Start the dev server                              |
| `pnpm build`                        | Type-check and build for production               |
| `pnpm preview`                      | Serve the production build locally                |
| `pnpm test`                         | Run the test suite (Vitest)                       |
| `pnpm typecheck`                    | Type-check without emitting                       |
| `pnpm lint`                         | Lint with ESLint                                  |
| `pnpm format` / `pnpm format:check` | Format / check formatting with Prettier           |
| `pnpm gen:api`                      | Regenerate `src/api/schema.d.ts` from the OpenAPI |

## Stack

React + TypeScript (strict), Vite, TanStack Query, React Router, React Hook Form + Zod, openapi-fetch, Tailwind CSS. Tests: Vitest, Testing Library, MSW and vitest-axe.

Architecture and conventions are in [`CLAUDE.md`](CLAUDE.md).
