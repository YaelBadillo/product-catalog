import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { createProduct, getProduct } from "./api";
import { productKeys } from "./query-keys";

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
