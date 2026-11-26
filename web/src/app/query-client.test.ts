import { QueryObserver } from "@tanstack/react-query";
import { describe, expect, it } from "vitest";
import { ApiError } from "../api/errors";
import { createQueryClient } from "./query-client";

function apiError(status: number) {
  return new ApiError({
    status,
    code: status === 0 || status >= 500 ? "INTERNAL_ERROR" : "NOT_FOUND",
    message: "x",
    details: [],
  });
}

/**
 * How many times a query that always fails with `error` runs, observed the way
 * useQuery observes it (fetchQuery never retries by default).
 */
async function attempts(error: Error) {
  let calls = 0;
  const failing = () => {
    calls += 1;
    throw error;
  };
  const observer = new QueryObserver(createQueryClient({ retryDelay: 0 }), {
    queryKey: ["test"],
    queryFn: failing,
  });
  await new Promise<void>((resolve) => {
    const unsubscribe = observer.subscribe((result) => {
      if (result.status === "error") {
        unsubscribe();
        resolve();
      }
    });
  });
  return calls;
}

describe("createQueryClient retry policy", () => {
  it.each([400, 404, 409, 422, 429])("does not retry a %i", async (status) => {
    expect(await attempts(apiError(status))).toBe(1);
  });

  it.each([500, 503])("retries a %i up to 3 times", async (status) => {
    expect(await attempts(apiError(status))).toBe(4);
  });

  it("retries a network failure up to 3 times", async () => {
    expect(await attempts(apiError(0))).toBe(4);
  });

  it("retries an error that is not an ApiError up to 3 times", async () => {
    expect(await attempts(new TypeError("Failed to fetch"))).toBe(4);
  });
});
