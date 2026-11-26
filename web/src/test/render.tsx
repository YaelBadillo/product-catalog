import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { render } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import type { ReactElement } from "react";
import { createMemoryRouter, MemoryRouter } from "react-router";
import { RouterProvider } from "react-router/dom";
import { UiProvider } from "../app/providers";
import { routes } from "../app/routes";

/**
 * Renders `ui` with the app's providers: Mantine with the app theme (test env:
 * no transitions or portals), a fresh QueryClient per call (no retries) and an
 * in-memory router starting at `route`.
 */
export function renderWithProviders(ui: ReactElement, { route = "/" } = {}) {
  const queryClient = createTestQueryClient();
  return {
    user: userEvent.setup(),
    ...render(
      <UiProvider env="test">
        <QueryClientProvider client={queryClient}>
          <MemoryRouter initialEntries={[route]}>{ui}</MemoryRouter>
        </QueryClientProvider>
      </UiProvider>,
    ),
  };
}

function createTestQueryClient() {
  return new QueryClient({ defaultOptions: { queries: { retry: false } } });
}

/**
 * Renders the app's routes at `path` with the same providers as
 * renderWithProviders. Returns the memory `router`, so tests can check the
 * location, navigate and go back.
 */
export function renderRoute(path: string) {
  const queryClient = createTestQueryClient();
  const router = createMemoryRouter(routes, { initialEntries: [path] });
  return {
    user: userEvent.setup(),
    router,
    queryClient,
    ...render(
      <UiProvider env="test">
        <QueryClientProvider client={queryClient}>
          <RouterProvider router={router} />
        </QueryClientProvider>
      </UiProvider>,
    ),
  };
}
