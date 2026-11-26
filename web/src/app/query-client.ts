import { QueryClient } from "@tanstack/react-query";
import { ApiError } from "../api/errors";

const maxRetries = 3;

/**
 * Retries server errors (5xx) and network failures up to 3 times. A 4xx is
 * the API's final answer (not found, conflict, invalid, rate limited), so
 * retrying only delays it.
 */
export function shouldRetry(failureCount: number, error: unknown): boolean {
  const clientError =
    error instanceof ApiError && error.status >= 400 && error.status < 500;
  return !clientError && failureCount < maxRetries;
}

interface QueryClientOptions {
  /** Delay between retries; tests pass 0. Defaults to TanStack's backoff. */
  retryDelay?: number;
}

/** The app's QueryClient. Tests build theirs here too, with the same policy. */
export function createQueryClient({ retryDelay }: QueryClientOptions = {}) {
  return new QueryClient({
    defaultOptions: {
      queries: {
        retry: shouldRetry,
        ...(retryDelay === undefined ? {} : { retryDelay }),
      },
    },
  });
}
