import { Alert, Button } from "@mantine/core";
import { CircleAlert } from "lucide-react";
import { messages } from "../messages";

interface ConflictAlertProps {
  onReload: () => void;
  reloading?: boolean;
}

/** Someone else changed the product: offers to reload the current version. */
export function ConflictAlert({
  onReload,
  reloading = false,
}: ConflictAlertProps) {
  return (
    <Alert
      variant="outline"
      color="red.9"
      role="alert"
      icon={<CircleAlert aria-hidden="true" />}
      title={messages.feedback.conflict}
    >
      <Button variant="default" onClick={onReload} loading={reloading}>
        {messages.actions.reload}
      </Button>
    </Alert>
  );
}
