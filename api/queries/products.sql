-- name: CreateProduct :one
INSERT INTO products (sku, name, description, price_cents, currency, stock, category, is_active)
VALUES (@sku, @name, @description, @price_cents, @currency, @stock, @category, @is_active)
RETURNING *;

-- name: GetProduct :one
SELECT * FROM products
WHERE id = @id AND deleted_at IS NULL;

-- name: ListProducts :many
-- Optional filters are NULL when unset. The sort comes from a whitelist: each
-- (column, direction) pair is one CASE, so nothing from the input reaches the
-- SQL text. Sorting by price groups by currency first (MXN, then USD), and id
-- breaks every tie so pages never repeat or skip products.
SELECT * FROM products
WHERE deleted_at IS NULL
  AND (sqlc.narg(category)::text IS NULL OR category = sqlc.narg(category)::text)
  AND (sqlc.narg(is_active)::boolean IS NULL OR is_active = sqlc.narg(is_active)::boolean)
  AND (sqlc.narg(currency)::text IS NULL OR currency = sqlc.narg(currency)::text)
  AND (sqlc.narg(min_price)::bigint IS NULL OR price_cents >= sqlc.narg(min_price)::bigint)
  AND (sqlc.narg(max_price)::bigint IS NULL OR price_cents <= sqlc.narg(max_price)::bigint)
  AND (sqlc.narg(pattern)::text IS NULL
    OR catalog_fold(name) LIKE '%' || catalog_fold(sqlc.narg(pattern)::text) || '%' ESCAPE '\'
    OR catalog_fold(sku) LIKE '%' || catalog_fold(sqlc.narg(pattern)::text) || '%' ESCAPE '\')
ORDER BY
  CASE WHEN @sort::text = 'name' AND @sort_order::text = 'asc' THEN name END ASC,
  CASE WHEN @sort::text = 'name' AND @sort_order::text = 'desc' THEN name END DESC,
  CASE WHEN @sort::text = 'price' THEN currency END ASC,
  CASE WHEN @sort::text = 'price' AND @sort_order::text = 'asc' THEN price_cents END ASC,
  CASE WHEN @sort::text = 'price' AND @sort_order::text = 'desc' THEN price_cents END DESC,
  CASE WHEN @sort::text = 'stock' AND @sort_order::text = 'asc' THEN stock END ASC,
  CASE WHEN @sort::text = 'stock' AND @sort_order::text = 'desc' THEN stock END DESC,
  CASE WHEN @sort::text = 'created_at' AND @sort_order::text = 'asc' THEN created_at END ASC,
  CASE WHEN @sort::text = 'created_at' AND @sort_order::text = 'desc' THEN created_at END DESC,
  id ASC
LIMIT @page_limit OFFSET @page_offset;

-- name: CountProducts :one
-- The same filters as ListProducts.
SELECT count(*) FROM products
WHERE deleted_at IS NULL
  AND (sqlc.narg(category)::text IS NULL OR category = sqlc.narg(category)::text)
  AND (sqlc.narg(is_active)::boolean IS NULL OR is_active = sqlc.narg(is_active)::boolean)
  AND (sqlc.narg(currency)::text IS NULL OR currency = sqlc.narg(currency)::text)
  AND (sqlc.narg(min_price)::bigint IS NULL OR price_cents >= sqlc.narg(min_price)::bigint)
  AND (sqlc.narg(max_price)::bigint IS NULL OR price_cents <= sqlc.narg(max_price)::bigint)
  AND (sqlc.narg(pattern)::text IS NULL
    OR catalog_fold(name) LIKE '%' || catalog_fold(sqlc.narg(pattern)::text) || '%' ESCAPE '\'
    OR catalog_fold(sku) LIKE '%' || catalog_fold(sqlc.narg(pattern)::text) || '%' ESCAPE '\');

-- name: UpdateProduct :one
-- Changes only the given fields, and only if the version still matches.
-- description_set distinguishes "remove the description" (NULL) from "keep it".
UPDATE products SET
    name        = COALESCE(sqlc.narg(name)::text, name),
    description = CASE WHEN @description_set::boolean THEN sqlc.narg(description)::text ELSE description END,
    price_cents = COALESCE(sqlc.narg(price_cents)::bigint, price_cents),
    currency    = COALESCE(sqlc.narg(currency)::text, currency),
    stock       = COALESCE(sqlc.narg(stock)::integer, stock),
    category    = COALESCE(sqlc.narg(category)::text, category),
    is_active   = COALESCE(sqlc.narg(is_active)::boolean, is_active),
    version     = version + 1,
    updated_at  = now()
WHERE id = @id AND version = @version AND deleted_at IS NULL
RETURNING *;

-- name: ProductExists :one
SELECT EXISTS (SELECT 1 FROM products WHERE id = @id AND deleted_at IS NULL);

-- name: SoftDeleteProduct :execrows
-- Logical deletion, only if the version still matches. The row stays, but no
-- query returns it again and its SKU is free.
UPDATE products SET deleted_at = now(), version = version + 1
WHERE id = @id AND version = @version AND deleted_at IS NULL;
