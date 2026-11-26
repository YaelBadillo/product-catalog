import { NumberInput, type NumberInputProps } from "@mantine/core";
import { useState, type KeyboardEvent } from "react";
import { messages } from "../messages";

type PriceInputProps = Omit<
  NumberInputProps,
  "thousandSeparator" | "decimalSeparator" | "allowNegative" | "clampBehavior"
>;

/**
 * Price in currency units with thousands separators ("1,299.90"). A typed
 * comma is dropped and a hint says to use a dot (C8). No decimalScale or
 * clamping: the schema reports too many decimals or the maximum (C2).
 */
export function PriceInput({
  onKeyDown,
  description,
  ...props
}: PriceInputProps) {
  const [commaTyped, setCommaTyped] = useState(false);

  function handleKeyDown(event: KeyboardEvent<HTMLInputElement>) {
    if (event.key === ",") {
      event.preventDefault();
      setCommaTyped(true);
    }
    onKeyDown?.(event);
  }

  return (
    <NumberInput
      {...props}
      inputMode="decimal"
      thousandSeparator=","
      decimalSeparator="."
      allowNegative={false}
      clampBehavior="none"
      onKeyDown={handleKeyDown}
      description={commaTyped ? messages.hints.comma : description}
    />
  );
}
