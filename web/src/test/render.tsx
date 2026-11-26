import { QueryClientProvider } from "@tanstack/react-query";
import { render } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import type { ReactElement } from "react";
import {
  createMemoryRouter,
  type InitialEntry,
  MemoryRouter,
} from "react-router";
import { RouterProvider } from "react-router/dom";
import { UiProvider } from "../app/providers";
import { createQueryClient } from "../app/query-client";
import { routes } from "../app/routes";

/**
 * Renders `ui` with the app's providers: Mantine with the app theme (test env:
 * no transitions or portals), a fresh QueryClient per call (the app's retry
 * policy, without delays) and an in-memory router starting at `route`.
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
  return createQueryClient({ retryDelay: 0 });
}

/**
 * Renders the app's routes at `path` with the same providers as
 * renderWithProviders. `path` may be a list of history entries; the last one
 * is current. Returns the memory `router`, so tests can check the location,
 * navigate and go back.
 */
export function renderRoute(path: string | InitialEntry[]) {
  const queryClient = createTestQueryClient();
  const router = createMemoryRouter(routes, {
    initialEntries: typeof path === "string" ? [path] : path,
  });
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
