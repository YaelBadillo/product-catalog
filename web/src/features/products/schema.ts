import { z } from "zod";
import { messages } from "./messages";
import type { Category, CreateProductRequest, Currency } from "./types";

const skuPattern = /^[A-Z0-9-]{4,32}$/;
const maxPrice = 999_999_999.99;
const maxStock = 2_147_483_647;

/**
 * Length in Unicode code points, as the API (Go runes) and Postgres
 * (char_length) count it.
 */
function codePointLength(value: string): number {
  return Array.from(value).length;
}

/** True when `value` has at most two decimals, ignoring float noise. */
function hasAtMostTwoDecimals(value: number): boolean {
  const cents = value * 100;
  return Math.abs(cents - Math.round(cents)) < 1e-6;
}

const sku = z.string().superRefine((value, ctx) => {
  const normalized = value.trim().toUpperCase();
  if (normalized === "") {
    ctx.addIssue({ code: "custom", message: messages.rules.skuRequired });
  } else if (!skuPattern.test(normalized)) {
    ctx.addIssue({ code: "custom", message: messages.rules.sku });
  }
});

const name = z.string().superRefine((value, ctx) => {
  const length = codePointLength(value.trim());
  if (length === 0) {
    ctx.addIssue({ code: "custom", message: messages.rules.nameRequired });
  } else if (length < 2 || length > 120) {
    ctx.addIssue({ code: "custom", message: messages.rules.name });
  }
});

const description = z
  .string()
  .refine(
    (value) => codePointLength(value.trim()) <= 2000,
    messages.rules.description,
  );

const price = z.union([z.number(), z.literal("")]).superRefine((value, ctx) => {
  const issue = priceIssue(value);
  if (issue) {
    ctx.addIssue({ code: "custom", message: issue });
  }
});

/** The rule a price breaks, or undefined. Shared by the form and the filters. */
export function priceIssue(value: number | ""): string | undefined {
  if (value === "") {
    return messages.rules.priceRequired;
  }
  if (value <= 0) {
    return messages.rules.pricePositive;
  }
  if (!hasAtMostTwoDecimals(value)) {
    return messages.rules.priceDecimals;
  }
  if (value > maxPrice) {
    return messages.rules.priceMax;
  }
  return undefined;
}

const stock = z.union([z.number(), z.literal("")]).superRefine((value, ctx) => {
  if (value === "") {
    ctx.addIssue({ code: "custom", message: messages.rules.stockRequired });
  } else if (!Number.isInteger(value) || value < 0 || value > maxStock) {
    ctx.addIssue({ code: "custom", message: messages.rules.stock });
  }
});

const category = z
  .union([
    z.enum(["electronics", "home", "food", "clothing", "other"]),
    z.literal(""),
  ])
  .refine((value) => value !== "", messages.rules.category);

/** The product form, create and edit. Rules match the API's (FR-002 to FR-009). */
export const productFormSchema = z.object({
  sku,
  name,
  description,
  price,
  currency: z.enum(["MXN", "USD"], { error: messages.rules.currency }),
  stock,
  category,
  isActive: z.boolean(),
});

export interface ProductFormValues {
  sku: string;
  name: string;
  description: string;
  /** In currency units (199.90), "" while empty. */
  price: number | "";
  currency: Currency;
  stock: number | "";
  category: Category | "";
  isActive: boolean;
}

export const defaultFormValues: ProductFormValues = {
  sku: "",
  name: "",
  description: "",
  price: "",
  currency: "MXN",
  stock: "",
  category: "",
  isActive: true,
};

/** Units to cents, after the schema has validated the amount. */
export function toCents(units: number): number {
  return Math.round(units * 100);
}

/** The POST body of valid form values. */
export function toCreateBody(values: ProductFormValues): CreateProductRequest {
  const body: CreateProductRequest = {
    sku: values.sku.trim().toUpperCase(),
    name: values.name.trim(),
    price_cents: toCents(Number(values.price)),
    currency: values.currency,
    stock: Number(values.stock),
    category: values.category as Category,
    is_active: values.isActive,
  };
  const trimmed = values.description.trim();
  if (trimmed !== "") {
    body.description = trimmed;
  }
  return body;
}
