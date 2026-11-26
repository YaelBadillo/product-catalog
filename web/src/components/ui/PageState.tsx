import { Stack, Text, Title } from "@mantine/core";
import type { ReactNode } from "react";

interface PageStateProps {
  title: string;
  text?: string;
  action?: ReactNode;
  /** 1 when the state is the page's main content (e.g. not found). */
  headingLevel?: 1 | 2;
}

/** Empty, error and not-found states: a heading, optional text and action. */
export function PageState({
  title,
  text,
  action,
  headingLevel = 2,
}: PageStateProps) {
  return (
    <Stack align="center" ta="center" gap="sm" py="xl">
      <Title order={headingLevel} size="h3">
        {title}
      </Title>
      {text && <Text c="dimmed">{text}</Text>}
      {action}
    </Stack>
  );
}
