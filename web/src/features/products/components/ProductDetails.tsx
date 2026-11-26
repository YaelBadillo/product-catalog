import { Box, Text } from "@mantine/core";
import type { ReactNode } from "react";
import { formatDateTime, formatPrice } from "../format";
import { messages } from "../messages";
import type { Product } from "../types";
import { StatusBadge } from "./StatusBadge";
import classes from "./ProductDetails.module.css";

interface ItemProps {
  label: string;
  children: ReactNode;
  /** Spans the whole row (long text). */
  wide?: boolean;
}

function Item({ label, children, wide = false }: ItemProps) {
  return (
    <div className={wide ? classes.wide : undefined}>
      <Text component="dt" size="sm" c="dimmed" fw={500}>
        {label}
      </Text>
      <Text component="dd" m={0} className={classes.value}>
        {children}
      </Text>
    </div>
  );
}

/** Every attribute of a product except internal ids (FR-015). */
export function ProductDetails({ product }: { product: Product }) {
  return (
    <Box component="dl" m={0} className={classes.list}>
      <Item label={messages.labels.sku}>{product.sku}</Item>
      <Item label={messages.labels.price}>
        {formatPrice(product.price_cents, product.currency)}
      </Item>
      <Item label={messages.labels.stock}>
        {product.stock.toLocaleString("es-MX")}
      </Item>
      <Item label={messages.labels.category}>
        {messages.categories[product.category]}
      </Item>
      <Item label={messages.labels.status}>
        <StatusBadge active={product.is_active} />
      </Item>
      <Item label={messages.labels.description} wide>
        {product.description ?? (
          <Text component="span" c="dimmed" fs="italic">
            {messages.noDescription}
          </Text>
        )}
      </Item>
      <Item label={messages.labels.createdAt}>
        {formatDateTime(product.created_at)}
      </Item>
      <Item label={messages.labels.updatedAt}>
        {formatDateTime(product.updated_at)}
      </Item>
    </Box>
  );
}
