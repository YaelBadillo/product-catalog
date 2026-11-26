import createClient from "openapi-fetch";
import { env } from "../lib/env";
import type { paths } from "./schema";

/** Typed HTTP client for the API described in openapi.yaml. */
export const apiClient = createClient<paths>({
  baseUrl: env.VITE_API_URL,
  // openapi-fetch captures `fetch` when the client is created. Resolving it per
  // call keeps requests interceptable (MSW patches the global after import).
  fetch: (request) => globalThis.fetch(request),
});
