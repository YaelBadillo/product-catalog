import { http, HttpResponse } from "msw";
import type { components } from "../api/schema";
import { env } from "../lib/env";

type Schemas = components["schemas"];

/** Absolute URL of an API path, as the client requests it. */
export function apiUrl(path: string): string {
  return new URL(path, env.VITE_API_URL).toString();
}

/** Default handlers: every service healthy. Override per test with `server.use`. */
export const handlers = [
  http.get(apiUrl("/healthz"), () =>
    HttpResponse.json<Schemas["HealthResponse"]>({ status: "ok" }),
  ),
  http.get(apiUrl("/readyz"), () =>
    HttpResponse.json<Schemas["ReadinessResponse"]>({
      status: "ok",
      checks: { database: "ok" },
    }),
  ),
];
