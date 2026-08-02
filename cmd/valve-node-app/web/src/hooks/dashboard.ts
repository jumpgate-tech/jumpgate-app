// The Dashboard section's data layer — a dedicated hook file (not
// hooks/gateway.ts/target.ts/logs.ts) both to keep this screen's own
// SSE-plus-rate-smoothing and per-service action bookkeeping out of those
// files, and so this section can land standalone while those files are
// mid-migration by other work.
//
// dashboard.ts's data comes from four independent sources, kept independent
// here too:
//   - the monitor SSE stream (services' active/inactive + sync/peer/disk
//     figures) — useMonitorStream. There is no polled "container list" to
//     wrap in a refetchInterval query: the server pushes a snapshot every 5s
//     itself (see internal/monitor), and every service's actual
//     active/inactive state comes ONLY from that stream, never re-derived
//     from a query or from an action's response.
//   - disk usage (/du) — a one-shot fetch with a manual "Retry" button in
//     the legacy UI, not a poll. useDiskUsage wraps it as a React Query
//     query (retry: false, so a failure surfaces immediately instead of
//     React Query's own background retries masking it) so DashboardSection
//     can drive its Retry button with `refetch()`.
//   - endpoints reachability — same shape as disk usage. useEndpoints.
//   - service actions (start/stop/restart/clear) — useServiceActions for
//     the first three (a per-service single-flight guard + a shared last
//     error message, exactly like dashboard.ts's own `pending`/`actionErr`)
//     and useClearService for the destructive one, as a mutation whose
//     success invalidates the disk-usage query — mirroring runClear's own
//     `void loadDiskUsage()` after a successful clear.
import { useEffect, useRef, useState } from "react";
import { useMutation, useQuery, useQueryClient, type UseQueryResult } from "@tanstack/react-query";
import * as api from "../api";
import { nextBlocksPerSec, SERVICE_LABEL } from "../screens/Machine/dashboardModel";

export interface UseMonitorStreamResult {
  snapshot: api.Snapshot | null;
  execBlocksPerSec: number | null;
}

// useMonitorStream mirrors dashboard.ts's own streamMonitor+updateRate: it
// opens the target's monitor SSE stream whenever enabled, and keeps a
// smoothed blocks/sec rate across ticks (see dashboardModel.nextBlocksPerSec
// for the exact smoothing). Disabling (or changing targetId) closes the
// stream and resets both to null — a fresh section open starts the rate
// estimate over, exactly like a fresh renderDashboard mount would.
export function useMonitorStream(targetId: string, enabled: boolean): UseMonitorStreamResult {
  const [snapshot, setSnapshot] = useState<api.Snapshot | null>(null);
  const [execBlocksPerSec, setExecBlocksPerSec] = useState<number | null>(null);
  const prevRef = useRef<api.Snapshot | null>(null);

  useEffect(() => {
    prevRef.current = null;
    if (!enabled || !targetId) {
      setSnapshot(null);
      setExecBlocksPerSec(null);
      return;
    }

    setSnapshot(null);
    setExecBlocksPerSec(null);

    const stop = api.streamMonitor(targetId, (snap) => {
      // Capture the previous snapshot into a local BEFORE mutating the ref:
      // the setExecBlocksPerSec updater below is a closure React calls
      // lazily (during its own flush), so if it read prevRef.current
      // directly it would see this tick's snapshot instead of the prior
      // one once that assignment below has already run.
      const prevSnap = prevRef.current;
      prevRef.current = snap;
      setExecBlocksPerSec((rate) => nextBlocksPerSec(prevSnap, snap, rate));
      setSnapshot(snap);
    });

    return () => stop();
  }, [targetId, enabled]);

  return { snapshot, execBlocksPerSec };
}

export function useDiskUsage(targetId: string, enabled: boolean): UseQueryResult<api.DiskUsage> {
  return useQuery({
    queryKey: ["diskUsage", targetId],
    queryFn: () => api.getDiskUsage(targetId),
    enabled: enabled && !!targetId,
    retry: false,
  });
}

export function useEndpoints(targetId: string, enabled: boolean): UseQueryResult<api.EndpointInfo> {
  return useQuery({
    queryKey: ["endpoints", targetId],
    queryFn: () => api.getEndpoints(targetId),
    enabled: enabled && !!targetId,
    retry: false,
  });
}

export interface UseServiceActionsResult {
  pending: Record<api.ServiceID, api.ServiceActionKind | null>;
  error: string | null;
  run: (svc: api.ServiceID, kind: api.ServiceActionKind) => Promise<void>;
}

const NO_PENDING: Record<api.ServiceID, api.ServiceActionKind | null> = { exec: null, beacon: null };

// useServiceActions mirrors dashboard.ts's own runServiceAction: a
// per-service single-flight guard (a second call for a service already
// in flight is a no-op, exactly like `if (pending[svc] !== null) return;`)
// and a shared last-error message, formatted the same way
// ("<Label> <kind> failed: <message>"). pendingRef mirrors dashboard.ts's
// plain `pending` object — it's read synchronously inside `run` for the
// guard, since a functional setState update alone can't be read back before
// the async call proceeds.
export function useServiceActions(targetId: string): UseServiceActionsResult {
  const [pending, setPending] = useState(NO_PENDING);
  const [error, setError] = useState<string | null>(null);
  const pendingRef = useRef(NO_PENDING);
  const mountedRef = useRef(true);
  useEffect(() => {
    mountedRef.current = true;
    return () => {
      mountedRef.current = false;
    };
  }, []);

  async function run(svc: api.ServiceID, kind: api.ServiceActionKind): Promise<void> {
    if (pendingRef.current[svc] !== null) return;
    pendingRef.current = { ...pendingRef.current, [svc]: kind };
    setPending(pendingRef.current);
    setError(null);
    let errMsg: string | null = null;
    try {
      await api.serviceAction(targetId, svc, kind);
    } catch (err) {
      errMsg = `${SERVICE_LABEL[svc]} ${kind} failed: ${err instanceof Error ? err.message : String(err)}`;
    }
    pendingRef.current = { ...pendingRef.current, [svc]: null };
    if (!mountedRef.current) return;
    setPending(pendingRef.current);
    setError(errMsg);
  }

  return { pending, error, run };
}

// useClearService mirrors dashboard.ts's own runClear: a mutation over
// api.clearService whose success invalidates the disk-usage query, the same
// way runClear calls `void loadDiskUsage()` after the clear resolves — the
// caller (the clear confirm modal) reads `isPending`/`error` off the
// returned mutation to drive its own "Clearing…" / error-message UI.
export function useClearService(targetId: string) {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (svc: api.ServiceID) => api.clearService(targetId, svc),
    onSuccess: () => {
      void qc.invalidateQueries({ queryKey: ["diskUsage", targetId] });
    },
  });
}
