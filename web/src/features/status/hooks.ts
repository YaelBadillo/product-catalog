import { useQuery } from "@tanstack/react-query";
import { fetchSystemStatus } from "./api";
import { statusKeys } from "./query-keys";

/**
 * Availability of the API and the database. Unavailability is data, so the
 * last result stays on screen while `refetch` checks again.
 */
export function useSystemStatus() {
  return useQuery({
    queryKey: statusKeys.system(),
    queryFn: fetchSystemStatus,
  });
}
