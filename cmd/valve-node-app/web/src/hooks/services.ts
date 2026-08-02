// The Services section's data layer — a dedicated hook file (not
// hooks/gateway.ts/target.ts/logs.ts/dashboard.ts) both to keep this
// screen's own action/provision bookkeeping out of those files, and so this
// section can land standalone while those files are mid-migration by other
// work.
//
// services.ts's data comes from:
//   - the containers list (api.getContainers) — useContainers, a plain
//     React Query with no polling: the legacy screen only ever re-reads it
//     after an action settles, never on a timer, so this hook does the same
//     (callers invalidate/refetch its ["containers", targetId] key).
//   - service actions (start/stop/restart) AND provisioning (create/
//     re-create) — useContainerOps. Both share the SAME busy/error state per
//     service in services.ts (busy[svc] holds whichever action id is
//     in-flight, be it "start" or "create"), so they share it here too,
//     rather than splitting into two hooks that could each think a service
//     is idle while the other is mid-action.
//   - saving the draft configuration (api.putContainerConfig) —
//     useSaveContainerConfig, a mutation that invalidates the containers
//     query on success, mirroring saveConfig's own `await load()` after a
//     successful save (and only after — a failed save leaves the list
//     alone, exactly like the legacy early return).
//   - wiping a service (api.wipeContainer) — useWipeContainer, a bare
//     mutation with NO auto-invalidate: services.ts only re-reads the list
//     when the wipe modal (confirm OR result) is dismissed, not the instant
//     the wipe call resolves, so that reload is the caller's job (see
//     ServicesSection's wipe-modal close handlers).
import { useCallback, useEffect, useRef, useState } from "react";
import { useMutation, useQuery, useQueryClient, type UseQueryResult } from "@tanstack/react-query";
import * as api from "../api";
import { provisionFinished, provisionLine } from "../screens/Machine/servicesModel";

export function useContainers(targetId: string): UseQueryResult<api.ContainersResponse> {
  return useQuery({
    queryKey: ["containers", targetId],
    queryFn: () => api.getContainers(targetId),
    enabled: !!targetId,
  });
}

function message(err: unknown): string {
  return err instanceof Error ? err.message : String(err);
}

// hintOf surfaces the server's operator-facing hint verbatim, mirroring
// services.ts's own hintOf.
function hintOf(err: unknown): string {
  return err instanceof api.ApiError && err.hint ? ` — ${err.hint}` : "";
}

export interface UseContainerOpsResult {
  // busy[svc] holds the action id currently in flight for that service
  // ("start"/"stop"/"restart"/"create"), or null when idle — the single
  // source both run() and provision() disable their buttons from.
  busy: Record<string, string | null>;
  error: Record<string, string | null>;
  // activity[svc] is the provisioning progress log — see services.ts's own
  // comment: kept here (not in a ref/DOM) so it survives a re-render
  // triggered by any other card.
  activity: Record<string, string[]>;
  run: (svc: api.ContainerServiceID, kind: api.ContainerActionKind) => Promise<void>;
  provision: (svc: api.ContainerServiceID) => Promise<void>;
}

const IDLE: Record<string, null> = { devnet: null };
const EMPTY_LINES: Record<string, string[]> = { devnet: [] };

// useContainerOps mirrors services.ts's own busy/actionErr/activity closures
// plus runAction/provision. One stream can be open at a time (streamRef is a
// single slot, not keyed per service) — matches the server's own one
// provisioning-run-per-target rule, and services.ts's identical
// `streamStop?.()` single variable.
export function useContainerOps(targetId: string): UseContainerOpsResult {
  const qc = useQueryClient();
  const [busy, setBusy] = useState<Record<string, string | null>>(IDLE);
  const [error, setError] = useState<Record<string, string | null>>(IDLE);
  const [activity, setActivity] = useState<Record<string, string[]>>(EMPTY_LINES);
  const busyRef = useRef(busy);
  const streamRef = useRef<(() => void) | null>(null);
  const mountedRef = useRef(true);

  useEffect(() => {
    mountedRef.current = true;
    return () => {
      mountedRef.current = false;
      streamRef.current?.();
      streamRef.current = null;
    };
  }, []);

  const reload = useCallback(
    () => qc.invalidateQueries({ queryKey: ["containers", targetId] }),
    [qc, targetId],
  );

  // run mirrors runAction: a per-service single-flight guard, then an
  // UNCONDITIONAL reload afterward — state always comes from a re-read,
  // never from the action's own response.
  const run = useCallback(
    async (svc: api.ContainerServiceID, kind: api.ContainerActionKind) => {
      if (busyRef.current[svc]) return;
      busyRef.current = { ...busyRef.current, [svc]: kind };
      setBusy(busyRef.current);
      setError((e) => ({ ...e, [svc]: null }));
      try {
        await api.containerAction(targetId, svc, kind);
      } catch (err) {
        setError((e) => ({ ...e, [svc]: `${kind} failed: ${message(err)}${hintOf(err)}` }));
      }
      busyRef.current = { ...busyRef.current, [svc]: null };
      setBusy(busyRef.current);
      await reload();
    },
    [targetId, reload],
  );

  // provision mirrors provision(): POST to kick off the plan, then follow
  // the target's shared setup event stream until FINAL_STEP reports done or
  // any step errors. A failure on the initial POST (most often a 409 —
  // something else is already provisioning this target) resets busy WITHOUT
  // reloading, exactly like services.ts's own early return.
  const provision = useCallback(
    async (svc: api.ContainerServiceID) => {
      if (busyRef.current[svc]) return;
      busyRef.current = { ...busyRef.current, [svc]: "create" };
      setBusy(busyRef.current);
      setError((e) => ({ ...e, [svc]: null }));
      setActivity((a) => ({ ...a, [svc]: ["starting…"] }));

      try {
        await api.provisionContainer(targetId, svc);
      } catch (err) {
        busyRef.current = { ...busyRef.current, [svc]: null };
        setBusy(busyRef.current);
        setError((e) => ({ ...e, [svc]: `${message(err)}${hintOf(err)}` }));
        setActivity((a) => ({ ...a, [svc]: [] }));
        return;
      }

      streamRef.current?.();
      streamRef.current = api.streamSetup(targetId, (ev) => {
        if (!mountedRef.current) return;
        const line = provisionLine(ev);
        setActivity((a) => ({ ...a, [svc]: [...(a[svc] ?? []).filter((l) => l !== "starting…"), line] }));
        if (!provisionFinished(ev)) return;

        streamRef.current?.();
        streamRef.current = null;
        busyRef.current = { ...busyRef.current, [svc]: null };
        setBusy(busyRef.current);
        if (ev.err) setError((e) => ({ ...e, [svc]: "Provisioning failed — see the log below." }));
        void reload();
      });
    },
    [targetId, reload],
  );

  return { busy, error, activity, run, provision };
}

// useSaveContainerConfig mirrors saveConfig's api.putContainerConfig call —
// it invalidates the containers list on success only, matching the legacy
// early return on a failed save (which leaves the list untouched).
export function useSaveContainerConfig(targetId: string) {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: ({ svc, config }: { svc: api.ContainerServiceID; config: api.DevnetConfig }) =>
      api.putContainerConfig(targetId, svc, config),
    onSuccess: () => {
      void qc.invalidateQueries({ queryKey: ["containers", targetId] });
    },
  });
}

// useWipeContainer wraps api.wipeContainer with NO auto-invalidate — see
// this file's header comment for why the reload is the caller's
// responsibility here, unlike every other mutation in this hook file.
export function useWipeContainer(targetId: string) {
  return useMutation({
    mutationFn: (svc: api.ContainerServiceID) => api.wipeContainer(targetId, svc),
  });
}
