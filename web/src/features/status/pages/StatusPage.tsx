import { Button } from "../../../components/ui/Button";
import { StatusIndicator } from "../components/StatusIndicator";
import { useSystemStatus } from "../hooks";

export function StatusPage() {
  const { data: status, isFetching, refetch } = useSystemStatus();

  return (
    <main className="mx-auto flex max-w-md flex-col gap-4 p-6">
      <h1 className="text-2xl font-semibold">System status</h1>
      {status ? (
        <>
          {status.api === "unavailable" && (
            <div
              role="alert"
              className="flex items-center justify-between gap-4 rounded-lg border border-red-200 bg-red-50 p-4 text-red-800"
            >
              <p>The API is not responding.</p>
              <Button
                onClick={() => {
                  void refetch();
                }}
                disabled={isFetching}
              >
                Retry
              </Button>
            </div>
          )}
          <StatusIndicator name="API" state={status.api} />
          <StatusIndicator name="Database" state={status.database} />
        </>
      ) : (
        <p role="status" className="text-gray-600">
          Checking status…
        </p>
      )}
    </main>
  );
}
