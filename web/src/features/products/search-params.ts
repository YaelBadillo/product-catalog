import { z } from "zod";
import { toCents } from "./schema";
import {
  categories,
  currencies,
  type Category,
  type Currency,
  type ListQuery,
} from "./types";

export const sortFields = ["created_at", "name", "price", "stock"] as const;
export const sortOrders = ["desc", "asc"] as const;
export type SortField = (typeof sortFields)[number];
export type SortOrder = (typeof sortOrders)[number];

/** The catalog view, as the URL holds it. Prices in currency units. */
export interface CatalogSearch {
  page: number;
  category?: Category;
  isActive?: boolean;
  currency?: Currency;
  minPrice?: number;
  maxPrice?: number;
  q?: string;
  sort: SortField;
  order: SortOrder;
}

export const defaultCatalogSearch: CatalogSearch = {
  page: 1,
  sort: "created_at",
  order: "desc",
};

export const pageSize = 20;
const maxQueryLength = 100;
const maxPrice = 999_999_999.99;

const page = z.coerce.number().int().min(1).catch(1);
const category = z.enum(categories).optional().catch(undefined);
const isActive = z
  .enum(["true", "false"])
  .transform((v) => v === "true")
  .optional()
  .catch(undefined);
const currency = z.enum(currencies).optional().catch(undefined);
const price = z
  .string()
  .regex(/^\d+(\.\d{1,2})?$/)
  .transform(Number)
  .refine((v) => v <= maxPrice)
  .optional()
  .catch(undefined);
const sort = z.enum(sortFields).catch("created_at");
const order = z.enum(sortOrders).catch("desc");

function readSearchText(raw: string | null): string | undefined {
  const text = raw?.trim().slice(0, maxQueryLength).trim();
  return text ? text : undefined;
}

function optional(params: URLSearchParams, key: string): string | undefined {
  return params.get(key) ?? undefined;
}

/** Reads the URL, falling back to defaults for anything invalid (FR-025). */
export function parseCatalogSearch(params: URLSearchParams): CatalogSearch {
  const search: CatalogSearch = {
    page: page.parse(params.get("page") ?? undefined),
    category: category.parse(optional(params, "category")),
    isActive: isActive.parse(optional(params, "is_active")),
    currency: currency.parse(optional(params, "currency")),
    minPrice: price.parse(optional(params, "min_price")),
    maxPrice: price.parse(optional(params, "max_price")),
    q: readSearchText(params.get("q")),
    sort: sort.parse(optional(params, "sort")),
    order: order.parse(optional(params, "order")),
  };
  return withValidPriceRange(search);
}

/** Drops a price range that has no currency or is inverted. */
export function withValidPriceRange(search: CatalogSearch): CatalogSearch {
  const inverted =
    search.minPrice !== undefined &&
    search.maxPrice !== undefined &&
    search.minPrice > search.maxPrice;
  if (!search.currency || inverted) {
    return { ...search, minPrice: undefined, maxPrice: undefined };
  }
  return search;
}

function compact<T extends object>(value: T): T {
  return Object.fromEntries(
    Object.entries(value).filter(([, v]) => v !== undefined),
  ) as T;
}

/** The URL of a view, omitting defaults. */
export function serializeCatalogSearch(search: CatalogSearch): URLSearchParams {
  const s = withValidPriceRange(search);
  const params = new URLSearchParams();
  const set = (key: string, value: string | number | boolean | undefined) => {
    if (value !== undefined) {
      params.set(key, String(value));
    }
  };
  set("page", s.page === 1 ? undefined : s.page);
  set("category", s.category);
  set("is_active", s.isActive);
  set("currency", s.currency);
  set("min_price", s.minPrice);
  set("max_price", s.maxPrice);
  set("q", s.q);
  set("sort", s.sort === defaultCatalogSearch.sort ? undefined : s.sort);
  set("order", s.order === defaultCatalogSearch.order ? undefined : s.order);
  return params;
}

/** The API query of a view: prices in cents, always 20 per page. */
export function toListQuery(search: CatalogSearch): ListQuery {
  const s = withValidPriceRange(search);
  return compact({
    page: s.page,
    page_size: pageSize,
    category: s.category,
    is_active: s.isActive,
    currency: s.currency,
    min_price: s.minPrice === undefined ? undefined : toCents(s.minPrice),
    max_price: s.maxPrice === undefined ? undefined : toCents(s.maxPrice),
    q: s.q,
    sort: s.sort,
    order: s.order,
  });
}

/** Whether any filter or search narrows the catalog. */
export function hasFilters(search: CatalogSearch): boolean {
  return (
    search.category !== undefined ||
    search.isActive !== undefined ||
    search.currency !== undefined ||
    search.q !== undefined
  );
}
