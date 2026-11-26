import { Button } from "@mantine/core";
import { Pencil } from "lucide-react";
import { Link } from "react-router";
import { messages } from "../messages";
import type { CatalogState } from "../navigation";
import type { Product } from "../types";

interface RowActionsProps {
  product: Product;
  linkState: CatalogState;
}

/**
 * Actions of a product in the catalog. Each names its product for screen
 * readers; the accessible name starts with the visible label (WCAG 2.5.3).
 */
export function RowActions({ product, linkState }: RowActionsProps) {
  return (
    <Button
      component={Link}
      to={`/products/${product.id}/edit`}
      state={{ catalogSearch: linkState.catalogSearch }}
      aria-label={`${messages.actions.edit} ${product.name}`}
      size="xs"
      variant="default"
      leftSection={<Pencil aria-hidden="true" size={14} />}
    >
      {messages.actions.edit}
    </Button>
  );
}
