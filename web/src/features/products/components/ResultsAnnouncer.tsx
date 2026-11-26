import { Text } from "@mantine/core";
import { messages } from "../messages";

/** "N productos", announced to screen readers after each update. */
export function ResultsAnnouncer({ total }: { total: number | undefined }) {
  return (
    <Text aria-live="polite" size="sm" c="dimmed">
      {total === undefined ? "" : messages.states.results(total)}
    </Text>
  );
}
