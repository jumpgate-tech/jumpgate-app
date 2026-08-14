// React Query hooks over api.ts's update-check surface — the App shell's
// update banner data layer. The server caches the GitHub result itself (see
// internal/server/update.go), so a modest client poll here is cheap.
import { useMutation, useQuery, useQueryClient, type UseQueryResult } from "@tanstack/react-query";
import * as api from "../api";

// useUpdate reads the current update status. It polls hourly — a new release is
// rare, and the server answers from its own cache anyway, so a tight interval
// would buy nothing. A failed request is not retried on a loop: the status
// itself already carries checkError, so surfacing that beats React Query hiding
// a transient failure behind background retries.
export function useUpdate(): UseQueryResult<api.Update> {
  return useQuery({
    queryKey: ["update"],
    queryFn: () => api.getUpdate(),
    refetchInterval: 60 * 60 * 1000,
    retry: false,
  });
}

// useSkipUpdate records the skipped version and seeds the cache with the fresh
// status the POST returns, so the banner hides at once without a second GET.
export function useSkipUpdate() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (version: string) => api.skipUpdate(version),
    onSuccess: (updated) => {
      qc.setQueryData(["update"], updated);
    },
  });
}
