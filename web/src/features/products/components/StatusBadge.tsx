import { Group, Text } from "@mantine/core";
import { CircleCheck, CircleMinus } from "lucide-react";
import { messages } from "../messages";

/** Active or inactive, with text and a decorative icon (never color alone). */
export function StatusBadge({ active }: { active: boolean }) {
  const Icon = active ? CircleCheck : CircleMinus;
  return (
    <Group component="span" gap={6} wrap="nowrap" display="inline-flex">
      <Icon
        aria-hidden="true"
        size={16}
        color={`var(--mantine-color-${active ? "green-8" : "gray-5"})`}
      />
      <Text component="span" size="sm">
        {active ? messages.statuses.active : messages.statuses.inactive}
      </Text>
    </Group>
  );
}
