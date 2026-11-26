import type { components } from "./schema";

export type ErrorCode = components["schemas"]["ErrorCode"];
export type ErrorDetail = components["schemas"]["ErrorDetail"];

const errorCodes: readonly ErrorCode[] = [
  "VALIDATION_ERROR",
  "NOT_FOUND",
  "SKU_ALREADY_EXISTS",
  "VERSION_CONFLICT",
  "PAYLOAD_TOO_LARGE",
  "RATE_LIMITED",
  "INTERNAL_ERROR",
];

/**
 * A failed API call. `status` is 0 when the request never got a response
 * (network failure). `code` is `INTERNAL_ERROR` whenever the answer is not the
 * API's error envelope. An Error subclass, so it can be thrown and rejected.
 */
export class ApiError extends Error {
  readonly status: number;
  readonly code: ErrorCode;
  readonly details: ErrorDetail[];
  readonly retryAfter: number | undefined;

  constructor(init: {
    status: number;
    code: ErrorCode;
    message: string;
    details: ErrorDetail[];
    retryAfter?: number;
  }) {
    super(init.message);
    this.name = "ApiError";
    this.status = init.status;
    this.code = init.code;
    this.details = init.details;
    this.retryAfter = init.retryAfter;
  }
}

function isRecord(value: unknown): value is Record<string, unknown> {
  return typeof value === "object" && value !== null;
}

function isErrorCode(value: unknown): value is ErrorCode {
  return errorCodes.includes(value as ErrorCode);
}

function readDetails(value: unknown): ErrorDetail[] {
  if (!Array.isArray(value)) {
    return [];
  }
  return value.filter(
    (d): d is ErrorDetail =>
      isRecord(d) &&
      typeof d.field === "string" &&
      typeof d.message === "string",
  );
}

function readRetryAfter(response: Response | undefined): number | undefined {
  const header = response?.headers.get("Retry-After");
  const seconds = header ? Number.parseInt(header, 10) : Number.NaN;
  return Number.isFinite(seconds) ? seconds : undefined;
}

/**
 * Builds an ApiError from a failed response and its parsed body. Without a
 * response (network failure) the status is 0.
 */
export function toApiError(
  response: Response | undefined,
  body: unknown,
): ApiError {
  const status = response?.status ?? 0;
  const retryAfter = readRetryAfter(response);
  const envelope = isRecord(body) ? body.error : undefined;
  if (isRecord(envelope) && isErrorCode(envelope.code)) {
    return new ApiError({
      status,
      code: envelope.code,
      message:
        typeof envelope.message === "string" ? envelope.message : "API error",
      details: readDetails(envelope.details),
      retryAfter,
    });
  }
  return new ApiError({
    status,
    code: "INTERNAL_ERROR",
    message: `Unexpected API response (status ${String(status)})`,
    details: [],
    retryAfter,
  });
}
