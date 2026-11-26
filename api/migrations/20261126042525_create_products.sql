-- +goose Up
-- Accent-insensitive search (R2): unaccent behind an IMMUTABLE wrapper, so it
-- can be used in expressions and sqlc only needs its signature.
CREATE EXTENSION IF NOT EXISTS unaccent;

-- Spanish alphabetical order for names (R1): case and accents ignored at the
-- primary level, ñ after n. ICU, because musl (Alpine) has no libc locales.
CREATE COLLATION catalog_es (provider = icu, locale = 'es-MX');

-- +goose StatementBegin
CREATE FUNCTION catalog_fold(input text) RETURNS text
LANGUAGE sql IMMUTABLE PARALLEL SAFE STRICT
AS $$
    SELECT lower(public.unaccent('public.unaccent'::regdictionary, input))
$$;
-- +goose StatementEnd

CREATE TABLE products (
    id          uuid        PRIMARY KEY DEFAULT uuidv7(),
    sku         text        NOT NULL CHECK (sku ~ '^[A-Z0-9-]{4,32}$'),
    name        text        COLLATE catalog_es NOT NULL CHECK (char_length(name) BETWEEN 2 AND 120),
    description text        CHECK (description IS NULL OR char_length(description) BETWEEN 1 AND 2000),
    price_cents bigint      NOT NULL CHECK (price_cents BETWEEN 1 AND 99999999999),
    currency    text        NOT NULL DEFAULT 'MXN' CHECK (currency IN ('MXN', 'USD')),
    stock       integer     NOT NULL CHECK (stock >= 0),
    category    text        NOT NULL CHECK (category IN ('electronics', 'home', 'food', 'clothing', 'other')),
    is_active   boolean     NOT NULL DEFAULT true,
    version     integer     NOT NULL DEFAULT 1 CHECK (version >= 1),
    created_at  timestamptz NOT NULL DEFAULT now(),
    updated_at  timestamptz NOT NULL DEFAULT now(),
    deleted_at  timestamptz
);

-- Every index covers only non-deleted rows: a deleted product frees its SKU.
CREATE UNIQUE INDEX products_sku_active_key ON products (sku) WHERE deleted_at IS NULL;
CREATE INDEX products_category_idx ON products (category) WHERE deleted_at IS NULL;
CREATE INDEX products_created_at_idx ON products (created_at DESC, id) WHERE deleted_at IS NULL;
CREATE INDEX products_currency_price_idx ON products (currency, price_cents, id) WHERE deleted_at IS NULL;

-- +goose Down
DROP TABLE IF EXISTS products;
DROP FUNCTION IF EXISTS catalog_fold(text);
DROP COLLATION IF EXISTS catalog_es;
DROP EXTENSION IF EXISTS unaccent;
