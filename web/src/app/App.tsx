import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { ErrorBoundary } from "react-error-boundary";
import { createBrowserRouter } from "react-router";
import { RouterProvider } from "react-router/dom";
import { StatusPage } from "../features/status/pages/StatusPage";

const queryClient = new QueryClient();

const router = createBrowserRouter([{ path: "/", element: <StatusPage /> }]);

function RenderErrorFallback() {
  return (
    <p role="alert" className="p-6">
      Something went wrong. Reload the page to try again.
    </p>
  );
}

export function App() {
  return (
    <ErrorBoundary FallbackComponent={RenderErrorFallback}>
      <QueryClientProvider client={queryClient}>
        <RouterProvider router={router} />
      </QueryClientProvider>
    </ErrorBoundary>
  );
}
