// React Query hook over api.ts's firewall-checklist endpoint — the Security
// screen's data layer (mirrors security.ts's load()/items bookkeeping).
// `data` sticks to the last successfully-loaded checklist across a failed
// re-run (React Query's default caching behavior), which is exactly what
// security.ts's own `items` array did by never being reassigned on a load()
// failure — a re-run that errors shows the error banner ABOVE the still-valid
// previous results, not a blank screen.
import { useQuery, type UseQueryResult } from "@tanstack/react-query";
import * as api from "../api";

export function useFirewallChecklist(
  targetId: string,
  enabled: boolean,
): UseQueryResult<api.CheckItem[]> {
  return useQuery({
    queryKey: ["firewallChecklist", targetId],
    queryFn: () => api.getFirewallChecklist(targetId),
    enabled: enabled && !!targetId,
  });
}
