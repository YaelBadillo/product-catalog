import { Alert, Button, Container, Stack, Text, Title } from "@mantine/core";
import { CircleAlert } from "lucide-react";
import { StatusIndicator } from "../components/StatusIndicator";
import { useSystemStatus } from "../hooks";

export function StatusPage() {
  const { data: status, isFetching, refetch } = useSystemStatus();

  return (
    <Container component="main" size="xs" py="xl">
      <Stack gap="md">
        <Title order={1} size="h2">
          System status
        </Title>
        {status ? (
          <>
            {status.api === "unavailable" && (
              <Alert
                variant="outline"
                color="red.9"
                icon={<CircleAlert aria-hidden="true" />}
                title="The API is not responding."
              >
                <Button
                  onClick={() => {
                    void refetch();
                  }}
                  disabled={isFetching}
                >
                  Retry
                </Button>
              </Alert>
            )}
            <StatusIndicator name="API" state={status.api} />
            <StatusIndicator name="Database" state={status.database} />
          </>
        ) : (
          <Text role="status" c="dimmed">
            Checking status…
          </Text>
        )}
      </Stack>
    </Container>
  );
}
