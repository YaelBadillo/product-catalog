import { z } from "zod";

const envSchema = z.object({
  /** Root of the API host; endpoint paths are appended to it. */
  VITE_API_URL: z.url({ protocol: /^https?$/ }),
});

export type Env = z.infer<typeof envSchema>;

/** Validates environment variables; throws if any is missing or invalid. */
export function parseEnv(source: Record<string, unknown>): Env {
  return envSchema.parse(source);
}

/** Validated at startup: the app fails fast on a bad configuration. */
export const env = parseEnv(import.meta.env);
