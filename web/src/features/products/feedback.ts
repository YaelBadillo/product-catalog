import { notifications } from "@mantine/notifications";
import type { UseFormSetError } from "react-hook-form";
import { ApiError } from "../../api/errors";
import { messages, serverFieldMessages } from "./messages";
import type { ProductFormValues } from "./schema";

/** Confirmation of a successful action. */
export function notifySuccess(message: string) {
  notifications.show({ message, color: "green.8", closeButtonProps });
}

const closeButtonProps = { "aria-label": messages.actions.close };

/** Spanish text for a failed request: rate limited or unexpected. */
export function errorText(error: unknown): string {
  return error instanceof ApiError && error.code === "RATE_LIMITED"
    ? messages.feedback.rateLimited
    : messages.feedback.unexpected;
}

/** Notifies a failure that has no field to show it on. */
export function notifyError(error: unknown, message = errorText(error)) {
  notifications.show({ message, color: "red.9", closeButtonProps });
}

function isServerField(
  field: string,
): field is keyof typeof serverFieldMessages {
  return field in serverFieldMessages;
}

/**
 * Shows the field errors of a failed save next to their fields, in Spanish,
 * focusing the first. Returns false when the error has no field to show.
 */
export function applyServerErrors(
  error: unknown,
  setError: UseFormSetError<ProductFormValues>,
): boolean {
  if (!(error instanceof ApiError)) {
    return false;
  }
  if (error.code === "SKU_ALREADY_EXISTS") {
    setError(
      "sku",
      { message: messages.feedback.skuTaken },
      { shouldFocus: true },
    );
    return true;
  }
  if (error.code !== "VALIDATION_ERROR") {
    return false;
  }
  const fields = error.details
    .map((detail) => detail.field)
    .filter(isServerField)
    .map((field) => serverFieldMessages[field]);
  fields.forEach(({ field, message }, index) => {
    setError(field, { message }, { shouldFocus: index === 0 });
  });
  return fields.length > 0;
}
