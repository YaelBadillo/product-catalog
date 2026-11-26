import { describe, expect, it } from "vitest";
import { ApiError, toApiError } from "./errors";

function jsonResponse(status: number, headers: Record<string, string> = {}) {
  return new Response(null, { status, headers });
}

describe("toApiError", () => {
  it("builds an ApiError from the error envelope", () => {
    const error = toApiError(jsonResponse(422), {
      error: {
        code: "VALIDATION_ERROR",
        message: "the request breaks a business rule",
        details: [{ field: "name", message: "must be 2 to 120 characters" }],
      },
    });

    expect(error).toBeInstanceOf(ApiError);
    expect(error).toBeInstanceOf(Error);
    expect(error.status).toBe(422);
    expect(error.code).toBe("VALIDATION_ERROR");
    expect(error.details).toEqual([
      { field: "name", message: "must be 2 to 120 characters" },
    ]);
    expect(error.retryAfter).toBeUndefined();
  });

  it("reads Retry-After in seconds", () => {
    const error = toApiError(jsonResponse(429, { "Retry-After": "3" }), {
      error: { code: "RATE_LIMITED", message: "too many", details: [] },
    });

    expect(error.code).toBe("RATE_LIMITED");
    expect(error.retryAfter).toBe(3);
  });

  it.each([
    ["a non-JSON body", "<html>Bad gateway</html>"],
    ["an empty body", undefined],
    ["an unknown code", { error: { code: "NOPE", message: "x", details: [] } }],
  ])("gives INTERNAL_ERROR for %s", (_, body) => {
    const error = toApiError(jsonResponse(502), body);

    expect(error.status).toBe(502);
    expect(error.code).toBe("INTERNAL_ERROR");
    expect(error.details).toEqual([]);
  });

  it("gives INTERNAL_ERROR with status 0 for a network failure", () => {
    const error = toApiError(undefined, new TypeError("Failed to fetch"));

    expect(error.status).toBe(0);
    expect(error.code).toBe("INTERNAL_ERROR");
  });
});
