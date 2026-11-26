import { apiClient } from "../../api/client";
import { toApiError } from "../../api/errors";
import type {
  CreateProductRequest,
  ListQuery,
  Product,
  ProductList,
  UpdateProductRequest,
} from "./types";

interface ClientResult<T> {
  data?: T;
  error?: unknown;
  response: Response;
}

/**
 * Resolves with the data of a successful call; rejects with an ApiError on an
 * error response or a network failure.
 */
async function unwrap<T>(call: () => Promise<ClientResult<T>>): Promise<T> {
  let result: ClientResult<T>;
  try {
    result = await call();
  } catch (cause) {
    throw toApiError(undefined, cause);
  }
  if (!result.response.ok) {
    throw toApiError(result.response, result.error);
  }
  return result.data as T;
}

export function getProduct(id: string): Promise<Product> {
  return unwrap(() =>
    apiClient.GET("/api/v1/products/{id}", { params: { path: { id } } }),
  );
}

export function createProduct(body: CreateProductRequest): Promise<Product> {
  return unwrap(() => apiClient.POST("/api/v1/products", { body }));
}

export function listProducts(query: ListQuery): Promise<ProductList> {
  return unwrap(() => apiClient.GET("/api/v1/products", { params: { query } }));
}

export function updateProduct(
  id: string,
  body: UpdateProductRequest,
): Promise<Product> {
  return unwrap(() =>
    apiClient.PATCH("/api/v1/products/{id}", {
      params: { path: { id } },
      body,
    }),
  );
}

export async function deleteProduct(
  id: string,
  version: number,
): Promise<void> {
  await unwrap(() =>
    apiClient.DELETE("/api/v1/products/{id}", {
      params: { path: { id }, query: { version } },
    }),
  );
}
