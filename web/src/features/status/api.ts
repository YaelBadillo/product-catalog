import { apiClient } from "../../api/client";
import type { components } from "../../api/schema";

type ReadinessResponse = components["schemas"]["ReadinessResponse"];

export type ServiceState = "operational" | "unavailable" | "unknown";

export interface SystemStatus {
  api: ServiceState;
  database: ServiceState;
}

/** GET /healthz: resolves if the API process is alive. */
async function fetchHealth(): Promise<void> {
  const { data, response } = await apiClient.GET("/healthz");
  if (!data) {
    throw new Error(
      `GET /healthz failed with status ${String(response.status)}`,
    );
  }
}

/**
 * GET /readyz. A 503 is a valid answer, not a failure: the API is up and
 * reports which dependency is unavailable.
 */
async function fetchReadiness(): Promise<ReadinessResponse> {
  const { data, error, response } = await apiClient.GET("/readyz");
  if (data) {
    return data;
  }
  if (response.status === 503) {
    return error;
  }
  throw new Error(`GET /readyz failed with status ${String(response.status)}`);
}

/**
 * Checks the API and, through it, the database. An unreachable service is a
 * result, not an error, so it never rejects.
 */
export async function fetchSystemStatus(): Promise<SystemStatus> {
  try {
    await fetchHealth();
  } catch {
    return { api: "unavailable", database: "unknown" };
  }
  try {
    const readiness = await fetchReadiness();
    return {
      api: "operational",
      database:
        readiness.checks.database === "ok" ? "operational" : "unavailable",
    };
  } catch {
    return { api: "operational", database: "unknown" };
  }
}
