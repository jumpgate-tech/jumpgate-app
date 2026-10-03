// React Query hooks over api.ts's diagnostics endpoints — the Diagnostics
// screen's data layer (mirrors diag.ts's report/loaded/running bookkeeping).
// getLatestDiagnostics reads whatever ran last, including runs the server
// triggered on its own from a journal error signature or a dead connection
// (inactive service, zero peers); runNetworkDiagnostics is the "Run
// diagnostics" button's manual re-probe. A successful run writes straight
// into the latest-report query's cache so the two endpoints share one source
// of truth, exactly like diag.ts's single `report` variable did — and the
// query keeps its last-loaded value across a failed run, so a re-run that
// errors shows the error banner above the still-valid previous report rather
// than a blank screen.
import {
  useMutation,
  useQuery,
  useQueryClient,
  type UseMutationResult,
  type UseQueryResult,
} from "@tanstack/react-query";
import * as api from "../api";

export function diagnosticsQueryKey(targetId: string): readonly [string, string, string] {
  return ["diagnostics", targetId, "latest"] as const;
}

export function useLatestDiagnostics(
  targetId: string,
  enabled: boolean,
): UseQueryResult<api.DiagReport | null> {
  return useQuery({
    queryKey: diagnosticsQueryKey(targetId),
    queryFn: () => api.getLatestDiagnostics(targetId),
    enabled: enabled && !!targetId,
  });
}

export function useRunDiagnostics(targetId: string): UseMutationResult<api.DiagReport, unknown, void> {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: () => api.runNetworkDiagnostics(targetId),
    onSuccess: (report) => {
      queryClient.setQueryData(diagnosticsQueryKey(targetId), report);
    },
  });
}
