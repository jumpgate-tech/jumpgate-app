// React Query hook over api.ts's update-check surface. Two callers:
//   - the App-shell banner uses useUpdate() (no refresh), which the server
//     answers without a GitHub call when notices are off — so a "don't prompt
//     me" install makes no background request.
//   - the Settings page uses useUpdate(true), a manual pull that always asks
//     GitHub, and drives its "Check for updates" button with refetch().
import { useQuery, type UseQueryResult } from "@tanstack/react-query";
import * as api from "../api";

// useUpdate reads the update status. Pass refresh=true to force a live check.
// It does not poll — a new release is rare, and the check runs on mount (and on
// the Settings button's refetch). A failed request is not retried on a loop:
// the status carries checkError, so surfacing it beats hiding a transient
// failure behind background retries.
export function useUpdate(refresh = false): UseQueryResult<api.Update> {
  return useQuery({
    queryKey: ["update", refresh],
    queryFn: () => api.getUpdate(refresh),
    retry: false,
    refetchOnWindowFocus: false,
  });
}
