import { Anchor, Group, Table } from "@mantine/core";
import type { ReactNode } from "react";
import { Link } from "react-router";
import { formatPrice } from "../format";
import { messages } from "../messages";
import type { CatalogState } from "../navigation";
import type { Product } from "../types";
import { StatusBadge } from "./StatusBadge";

interface ProductTableProps {
  products: Product[];
  /** History state for the detail links, so Back restores the catalog. */
  linkState: CatalogState;
  /** Row actions (edit, delete) for a product. */
  renderActions?: (product: Product) => ReactNode;
}

/** The catalog as a table, from 48em wide (R23). */
export function ProductTable({
  products,
  linkState,
  renderActions,
}: ProductTableProps) {
  return (
    <Table.ScrollContainer minWidth={720}>
      <Table striped highlightOnHover verticalSpacing="sm">
        <Table.Thead bg="gray.0">
          <Table.Tr>
            <Table.Th scope="col">{messages.labels.sku}</Table.Th>
            <Table.Th scope="col">{messages.labels.name}</Table.Th>
            <Table.Th scope="col">{messages.labels.category}</Table.Th>
            <Table.Th scope="col" ta="right">
              {messages.labels.price}
            </Table.Th>
            <Table.Th scope="col" ta="right">
              {messages.labels.stock}
            </Table.Th>
            <Table.Th scope="col">{messages.labels.status}</Table.Th>
            <Table.Th scope="col">{messages.labels.actions}</Table.Th>
          </Table.Tr>
        </Table.Thead>
        <Table.Tbody>
          {products.map((product) => (
            <Table.Tr key={product.id}>
              <Table.Td ff="monospace">{product.sku}</Table.Td>
              <Table.Td>
                <Anchor
                  component={Link}
                  to={`/products/${product.id}`}
                  state={linkState}
                  truncate="end"
                  display="block"
                  maw={260}
                >
                  {product.name}
                </Anchor>
              </Table.Td>
              <Table.Td>{messages.categories[product.category]}</Table.Td>
              <Table.Td ta="right" style={{ whiteSpace: "nowrap" }}>
                {formatPrice(product.price_cents, product.currency)}
              </Table.Td>
              <Table.Td ta="right">
                {product.stock.toLocaleString("es-MX")}
              </Table.Td>
              <Table.Td>
                <StatusBadge active={product.is_active} />
              </Table.Td>
              <Table.Td>
                <Group gap="xs" wrap="nowrap">
                  {renderActions?.(product)}
                </Group>
              </Table.Td>
            </Table.Tr>
          ))}
        </Table.Tbody>
      </Table>
    </Table.ScrollContainer>
  );
}
