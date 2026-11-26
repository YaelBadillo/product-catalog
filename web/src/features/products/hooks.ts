import {
  keepPreviousData,
  useMutation,
  useQuery,
  useQueryClient,
} from "@tanstack/react-query";
import { createProduct, getProduct, listProducts, updateProduct } from "./api";
import { productKeys } from "./query-keys";
import { toListQuery, type CatalogSearch } from "./search-params";
import type { UpdateProductRequest } from "./types";

/** One product. `enabled: false` skips the request (e.g. an invalid id). */
export function useProduct(id: string, { enabled = true } = {}) {
  return useQuery({
    queryKey: productKeys.detail(id),
    queryFn: () => getProduct(id),
    enabled,
  });
}

/** Creates a product, caches its detail and refreshes every list. */
export function useCreateProduct() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: createProduct,
    onSuccess: (product) => {
      queryClient.setQueryData(productKeys.detail(product.id), product);
      void queryClient.invalidateQueries({ queryKey: productKeys.lists() });
    },
  });
}

/** One page of the catalog. The previous page stays while the next loads. */
export function useProducts(search: CatalogSearch) {
  const query = toListQuery(search);
  return useQuery({
    queryKey: productKeys.list(query),
    queryFn: () => listProducts(query),
    placeholderData: keepPreviousData,
  });
}

/**
 * Saves an edit. No optimistic update (C7): the form waits for the answer,
 * and a conflict must leave every cache untouched.
 */
export function useUpdateProduct(id: string) {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (body: UpdateProductRequest) => updateProduct(id, body),
    onSuccess: (product) => {
      queryClient.setQueryData(productKeys.detail(product.id), product);
      void queryClient.invalidateQueries({ queryKey: productKeys.lists() });
    },
  });
}
