import type { components, operations } from "../../api/schema";

type Schemas = components["schemas"];

export type Product = Schemas["Product"];
export type Category = Schemas["Category"];
export type Currency = Schemas["Currency"];
export type ProductList = Schemas["ProductList"];
export type CreateProductRequest = Schemas["CreateProductRequest"];
export type UpdateProductRequest = Schemas["UpdateProductRequest"];

/** Query parameters of GET /api/v1/products. */
export type ListQuery = NonNullable<
  operations["listProducts"]["parameters"]["query"]
>;

export const categories: readonly Category[] = [
  "electronics",
  "home",
  "food",
  "clothing",
  "other",
];
export const currencies: readonly Currency[] = ["MXN", "USD"];
