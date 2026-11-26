import { afterAll, beforeAll, describe, expect, it, vi } from "vitest";
import { formatDateTime, formatPrice } from "./format";

describe("formatPrice", () => {
  it.each([
    [129990, "MXN", "$1,299.90 MXN"],
    [129990, "USD", "$1,299.90 USD"],
    [19990, "MXN", "$199.90 MXN"],
    [1, "MXN", "$0.01 MXN"],
    [99999999999, "USD", "$999,999,999.99 USD"],
  ] as const)("formats %i %s as %s", (cents, currency, expected) => {
    expect(formatPrice(cents, currency)).toBe(expected);
  });
});

describe("formatDateTime", () => {
  beforeAll(() => {
    vi.stubEnv("TZ", "America/Mexico_City");
  });
  afterAll(() => {
    vi.unstubAllEnvs();
  });

  it("formats in Mexican Spanish and the local time zone", () => {
    expect(formatDateTime("2026-11-25T20:30:00Z")).toBe("25 nov 2026, 14:30");
  });

  it("uses a 24-hour clock", () => {
    expect(formatDateTime("2026-01-05T06:05:00Z")).toBe("5 ene 2026, 00:05");
  });
});
