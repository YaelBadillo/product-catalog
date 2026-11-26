import { Text } from "@mantine/core";
import { modals } from "@mantine/modals";
import { messages } from "../messages";
import type { Product } from "../types";

/**
 * Asks to confirm a deletion, naming the product (FR-032). The confirm button
 * shows progress and is disabled until `onConfirm` settles (FR-039); the
 * modal traps focus and returns it on close.
 */
export function openDeleteConfirm(
  product: Pick<Product, "name" | "sku">,
  onConfirm: () => Promise<void>,
): void {
  const modalId = modals.openConfirmModal({
    title: messages.deleteConfirm.title,
    children: (
      <Text size="sm" style={{ overflowWrap: "anywhere" }}>
        {messages.deleteConfirm.body(product.name, product.sku)}
      </Text>
    ),
    labels: {
      confirm: messages.actions.delete,
      cancel: messages.actions.cancel,
    },
    confirmProps: { color: "red.9" },
    closeButtonProps: { "aria-label": messages.actions.close },
    closeOnConfirm: false,
    onConfirm: () => {
      modals.updateModal({
        modalId,
        confirmProps: { color: "red.9", loading: true },
        cancelProps: { disabled: true },
      });
      void onConfirm().finally(() => {
        modals.close(modalId);
      });
    },
  });
}
