import { Anchor, Group, Paper, Stack, Text, Title } from "@mantine/core";
import { useId, type ReactNode } from "react";
import { Link } from "react-router";
import { formatPrice } from "../format";
import { messages } from "../messages";
import type { CatalogState } from "../navigation";
import type { Product } from "../types";
import { StatusBadge } from "./StatusBadge";

interface ProductCardListProps {
  products: Product[];
  linkState: CatalogState;
  renderActions?: (product: Product) => ReactNode;
}

function Field({ label, children }: { label: string; children: ReactNode }) {
  return (
    <div>
      <Text component="dt" size="xs" c="dimmed" fw={500}>
        {label}
      </Text>
      <Text component="dd" m={0} size="sm">
        {children}
      </Text>
    </div>
  );
}

function ProductCard({
  product,
  linkState,
  renderActions,
}: { product: Product } & Omit<ProductCardListProps, "products">) {
  const headingId = useId();
  return (
    <Paper component="article" aria-labelledby={headingId} withBorder p="md">
      <Stack gap="sm">
        <Title
          id={headingId}
          order={2}
          size="h4"
          style={{ overflowWrap: "anywhere" }}
        >
          <Anchor
            component={Link}
            to={`/products/${product.id}`}
            state={linkState}
            inherit
          >
            {product.name}
          </Anchor>
        </Title>
        <Group component="dl" m={0} gap="lg" align="flex-start">
          <Field label={messages.labels.sku}>{product.sku}</Field>
          <Field label={messages.labels.price}>
            {formatPrice(product.price_cents, product.currency)}
          </Field>
          <Field label={messages.labels.stock}>
            {product.stock.toLocaleString("es-MX")}
          </Field>
          <Field label={messages.labels.category}>
            {messages.categories[product.category]}
          </Field>
          <Field label={messages.labels.status}>
            <StatusBadge active={product.is_active} />
          </Field>
        </Group>
        {renderActions && <Group gap="xs">{renderActions(product)}</Group>}
      </Stack>
    </Paper>
  );
}

/** The catalog as stacked cards, below 48em wide (R23). */
export function ProductCardList({ products, ...rest }: ProductCardListProps) {
  return (
    <Stack gap="sm">
      {products.map((product) => (
        <ProductCard key={product.id} product={product} {...rest} />
      ))}
    </Stack>
  );
}
