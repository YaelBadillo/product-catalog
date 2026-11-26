import type { Currency } from "./types";

/** `$1,299.90 MXN`: sign, thousands separator, two decimals and the code (R22). */
export function formatPrice(cents: number, currency: Currency): string {
  const amount = new Intl.NumberFormat("es-MX", {
    style: "currency",
    currency,
    currencyDisplay: "narrowSymbol",
  }).format(cents / 100);
  return `${amount} ${currency}`;
}

/** `25 nov 2026, 14:30` in the user's time zone. */
export function formatDateTime(iso: string): string {
  return new Intl.DateTimeFormat("es-MX", {
    dateStyle: "medium",
    timeStyle: "short",
    hourCycle: "h23",
  }).format(new Date(iso));
}
