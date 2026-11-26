import { Alert } from "@mantine/core";
import { QueryClientProvider } from "@tanstack/react-query";
import { useState } from "react";
import { ErrorBoundary } from "react-error-boundary";
import { createBrowserRouter } from "react-router";
import { RouterProvider } from "react-router/dom";
import { UiProvider } from "./providers";
import { createQueryClient } from "./query-client";
import { routes } from "./routes";

const queryClient = createQueryClient();

function RenderErrorFallback() {
  return (
    <Alert variant="outline" color="red.9" m="xl">
      Algo salió mal. Recarga la página para intentarlo de nuevo.
    </Alert>
  );
}

export function App() {
  // Created on first render, so it reads the URL at mount time.
  const [router] = useState(() => createBrowserRouter(routes));
  return (
    <UiProvider>
      <ErrorBoundary FallbackComponent={RenderErrorFallback}>
        <QueryClientProvider client={queryClient}>
          <RouterProvider router={router} />
        </QueryClientProvider>
      </ErrorBoundary>
    </UiProvider>
  );
}
