-- name: CreateProduct :one
INSERT INTO products (sku, name, description, price_cents, currency, stock, category, is_active)
VALUES (@sku, @name, @description, @price_cents, @currency, @stock, @category, @is_active)
RETURNING *;

-- name: GetProduct :one
SELECT * FROM products
WHERE id = @id AND deleted_at IS NULL;
