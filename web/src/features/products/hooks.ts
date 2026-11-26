import { ApiError } from "../../api/errors";
import {
  keepPreviousData,
  useMutation,
  useQuery,
  useQueryClient,
} from "@tanstack/react-query";
import {
  createProduct,
  deleteProduct,
  getProduct,
  listProducts,
  updateProduct,
} from "./api";
import { productKeys } from "./query-keys";
import { toListQuery, type CatalogSearch } from "./search-params";
import type { ProductList, UpdateProductRequest } from "./types";

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

interface DeleteVariables {
  id: string;
  /** The version the user saw (FR-035). */
  version: number;
}

function withoutProduct(list: ProductList | undefined, id: string) {
  if (!list?.data.some((product) => product.id === id)) {
    return list;
  }
  return {
    ...list,
    data: list.data.filter((product) => product.id !== id),
    meta: { ...list.meta, total: Math.max(0, list.meta.total - 1) },
  };
}

/**
 * Deletes a product, removing it from every cached list at once (R20). On
 * failure the lists come back, except when it was already gone (404).
 */
export function useDeleteProduct() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: ({ id, version }: DeleteVariables) =>
      deleteProduct(id, version),
    onMutate: async ({ id }) => {
      await queryClient.cancelQueries({ queryKey: productKeys.lists() });
      const snapshot = queryClient.getQueriesData<ProductList>({
        queryKey: productKeys.lists(),
      });
      queryClient.setQueriesData<ProductList>(
        { queryKey: productKeys.lists() },
        (list) => withoutProduct(list, id),
      );
      return { snapshot };
    },
    onError: (error, _variables, context) => {
      if (error instanceof ApiError && error.code === "NOT_FOUND") {
        return;
      }
      for (const [key, list] of context?.snapshot ?? []) {
        queryClient.setQueryData(key, list);
      }
    },
    onSuccess: (_data, { id }) => {
      queryClient.removeQueries({ queryKey: productKeys.detail(id) });
    },
    onSettled: () =>
      queryClient.invalidateQueries({ queryKey: productKeys.lists() }),
  });
}
