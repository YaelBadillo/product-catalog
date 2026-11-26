import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { render } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import type { ReactElement } from "react";
import { MemoryRouter } from "react-router";
import { UiProvider } from "../app/providers";

/**
 * Renders `ui` with the app's providers: Mantine with the app theme (test env:
 * no transitions or portals), a fresh QueryClient per call (no retries) and an
 * in-memory router starting at `route`.
 */
export function renderWithProviders(ui: ReactElement, { route = "/" } = {}) {
  const queryClient = new QueryClient({
    defaultOptions: { queries: { retry: false } },
  });
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
