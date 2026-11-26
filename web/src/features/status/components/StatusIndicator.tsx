import { Group, Paper, Title } from "@mantine/core";
import { CircleCheck, CircleHelp, CircleX } from "lucide-react";
import { useId } from "react";
import type { ServiceState } from "../api";

// The icon is decorative: the label always states the service state.
const presentation = {
  operational: {
    label: "En funcionamiento",
    Icon: CircleCheck,
    color: "var(--mantine-color-green-8)",
  },
  unavailable: {
    label: "No disponible",
    Icon: CircleX,
    color: "var(--mantine-color-red-9)",
  },
  unknown: {
    label: "Desconocido",
    Icon: CircleHelp,
    color: "var(--mantine-color-gray-5)",
  },
} as const;

interface StatusIndicatorProps {
  name: string;
  state: ServiceState;
}

/** One service and its state, as a labelled region. */
export function StatusIndicator({ name, state }: StatusIndicatorProps) {
  const headingId = useId();
  const { label, Icon, color } = presentation[state];

  return (
    <Paper component="section" aria-labelledby={headingId} withBorder p="md">
      <Group justify="space-between">
        <Title id={headingId} order={2} size="h4">
          {name}
        </Title>
        <Group component="p" gap="xs" m={0} fz="sm" fw={500}>
          <Icon aria-hidden="true" size={20} color={color} />
          {label}
        </Group>
      </Group>
    </Paper>
  );
}
