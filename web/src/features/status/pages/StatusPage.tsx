import { Alert, Button, Container, Stack, Text, Title } from "@mantine/core";
import { CircleAlert } from "lucide-react";
import { useDocumentTitle } from "../../../lib/use-document-title";
import { StatusIndicator } from "../components/StatusIndicator";
import { useSystemStatus } from "../hooks";

export function StatusPage() {
  const { data: status, isFetching, refetch } = useSystemStatus();
  useDocumentTitle("Estado del servicio");

  return (
    <Container component="main" size="xs" py="xl">
      <Stack gap="md">
        <Title order={1} size="h2">
          Estado del servicio
        </Title>
        {status ? (
          <>
            {status.api === "unavailable" && (
              <Alert
                variant="outline"
                color="red.9"
                icon={<CircleAlert aria-hidden="true" />}
                title="La API no responde."
              >
                <Button
                  onClick={() => {
                    void refetch();
                  }}
                  disabled={isFetching}
                >
                  Reintentar
                </Button>
              </Alert>
            )}
            <StatusIndicator name="API" state={status.api} />
            <StatusIndicator name="Base de datos" state={status.database} />
          </>
        ) : (
          <Text role="status" c="dimmed">
            Comprobando el estado…
          </Text>
        )}
      </Stack>
    </Container>
  );
}
