import { describe, expect, it } from "vitest";
import { parseEnv } from "./env";

describe("parseEnv", () => {
  it.each([["http://localhost:8080"], ["https://api.example.com"]])(
    "accepts %s as VITE_API_URL",
    (url) => {
      expect(parseEnv({ VITE_API_URL: url })).toEqual({ VITE_API_URL: url });
    },
  );

  it.each([
    ["a missing value", {}],
    ["an empty value", { VITE_API_URL: "" }],
    ["a value that is not a URL", { VITE_API_URL: "localhost:8080" }],
    ["a non-HTTP URL", { VITE_API_URL: "ftp://api.example.com" }],
  ])("rejects %s", (_, source) => {
    expect(() => parseEnv(source)).toThrow();
  });
});
