import { screen, within } from "@testing-library/react";
import type { UserEvent } from "@testing-library/user-event";
import { delay, http, HttpResponse } from "msw";
import type { ErrorCode } from "../../api/errors";
import { apiUrl } from "../../test/handlers";
import type { Product, ProductList } from "./types";

/** A valid product; override any field. */
export function buildProduct(overrides: Partial<Product> = {}): Product {
  return {
    id: "0193a5b0-7c1e-7000-8000-000000000001",
    sku: "CAF-0123",
    name: "Café molido",
    description: null,
    price_cents: 19990,
    currency: "MXN",
    stock: 5,
    category: "food",
    is_active: true,
    version: 1,
    created_at: "2026-11-25T20:30:00Z",
    updated_at: "2026-11-25T20:30:00Z",
    ...overrides,
  };
}

/** A page of products with its meta. */
export function buildList(
  data: Product[],
  meta: Partial<ProductList["meta"]> = {},
): ProductList {
  const total = meta.total ?? data.length;
  const pageSize = meta.page_size ?? 20;
  return {
    data,
    meta: {
      page: 1,
      page_size: pageSize,
      total,
      total_pages: Math.ceil(total / pageSize),
      ...meta,
    },
  };
}

/** The error envelope, as the API answers it. */
export function errorResponse(
  status: number,
  code: ErrorCode,
  details: { field: string; message: string }[] = [],
  headers: Record<string, string> = {},
) {
  return HttpResponse.json(
    { error: { code, message: code.toLowerCase(), details } },
    { status, headers },
  );
}

const productsPath = "/api/v1/products";
const productPath = "/api/v1/products/:id";

type Resolver = Parameters<typeof http.get>[1];

export const productHandlers = {
  list: (resolver: Resolver) => http.get(apiUrl(productsPath), resolver),
  get: (resolver: Resolver) => http.get(apiUrl(productPath), resolver),
  create: (resolver: Resolver) => http.post(apiUrl(productsPath), resolver),
  update: (resolver: Resolver) => http.patch(apiUrl(productPath), resolver),
  remove: (resolver: Resolver) => http.delete(apiUrl(productPath), resolver),
  /** Answers GET /products with `list`. */
  listOf: (list: ProductList) =>
    http.get(apiUrl(productsPath), () => HttpResponse.json(list)),
  /** Answers GET /products/:id with `product`. */
  getOf: (product: Product) =>
    http.get(apiUrl(productPath), () => HttpResponse.json(product)),
  /** Never answers, to test loading and pending states. */
  pending: (method: "get" | "post" | "patch" | "delete", path = productPath) =>
    http[method](apiUrl(path), () => delay("infinite")),
};

/** Opens the Mantine Select labelled `label` and picks `option`. */
export async function chooseOption(
  user: UserEvent,
  label: string,
  option: string,
) {
  await user.click(field(label));
  const listbox = await screen.findByRole("listbox");
  await user.click(within(listbox).getByRole("option", { name: option }));
}

/** The text-like field (input, textarea, select) with accessible name `label`. */
export function field(label: string): HTMLInputElement {
  const found =
    screen.queryByRole("textbox", { name: label }) ??
    screen.getByRole("combobox", { name: label });
  return found as HTMLInputElement;
}
