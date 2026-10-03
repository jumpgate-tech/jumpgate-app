// The Setup wizard section's data layer — a dedicated hook file (not
// hooks/target.ts/dashboard.ts/services.ts) both to keep this screen's own
// disk-probe and provisioning-stream bookkeeping out of those files, and so
// this section can land standalone while those files are mid-migration by
// other work.
//
// The client catalog and any existing target's wire config come from
// hooks/target.ts's useCatalog/useTargets — SetupWizard.tsx reads those
// directly, the same pair every other target-scoped screen shares — mirroring
// wizard.ts's own load(), which did the identical
// Promise.all([getCatalog(), listTargets()]) by hand.
//
// wizard.ts's remaining data comes from:
//   - the disk-free probe (api.getDiskFree) at the chosen data location —
//     useDiskProbe. Triggered on demand (entering the mode step, or the data
//     location field losing focus), never polled — mirrors probeDisk's own
//     manual diskProbing/diskError/freeBytes/probedPath bookkeeping. probe()
//     also RETURNS its outcome (not just setting state) so the caller can
//     chain wizardModel.evaluateFit's archive/full downgrade decision off
//     the very same result, exactly like probeDisk calling
//     evaluateFit(net, path) immediately after a successful fetch.
//   - starting a provisioning run and following its event stream —
//     useSetupRun. POST api.startSetup, then imperative api.streamSetup
//     (NOT useEventStream: this is a POST-triggered run, not a stream that
//     simply follows the mount) until unmount or a fresh start() call closes
//     it. wizard.ts's own startSetup NEVER closes the stream just because a
//     step finished — only a new run (its own `streamStop?.()` at the top of
//     startSetup) or the screen's disposal does, so this hook reproduces
//     that exactly rather than "helpfully" auto-closing on the handshake
//     step's done event. A 409 (a run is already in flight for this target)
//     is treated as success — attach to the live stream instead of erroring,
//     matching startSetup's own catch.
import { useCallback, useEffect, useRef, useState } from "react";
import * as api from "../api";

export interface DiskProbeOutcome {
  freeBytes?: number;
  error?: string;
}

export interface UseDiskProbeResult {
  freeBytes: number | null;
  probedPath: string | null;
  probing: boolean;
  error: string | null;
  probe: (path: string) => Promise<DiskProbeOutcome>;
}

export function useDiskProbe(targetId: string): UseDiskProbeResult {
  const [freeBytes, setFreeBytes] = useState<number | null>(null);
  const [probedPath, setProbedPath] = useState<string | null>(null);
  const [probing, setProbing] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const mountedRef = useRef(true);

  useEffect(() => {
    mountedRef.current = true;
    return () => {
      mountedRef.current = false;
    };
  }, []);

  const probe = useCallback(
    async (path: string): Promise<DiskProbeOutcome> => {
      setProbing(true);
      setError(null);
      try {
        const { freeBytes: bytes } = await api.getDiskFree(targetId, path);
        if (!mountedRef.current) return { freeBytes: bytes };
        setFreeBytes(bytes);
        setProbedPath(path);
        setProbing(false);
        return { freeBytes: bytes };
      } catch (err) {
        const message = err instanceof Error ? err.message : String(err);
        if (!mountedRef.current) return { error: message };
        setFreeBytes(null);
        setProbedPath(path);
        setError(message);
        setProbing(false);
        return { error: message };
      }
    },
    [targetId],
  );

  return { freeBytes, probedPath, probing, error, probe };
}

export interface UseSetupRunResult {
  events: api.SetupEvent[];
  starting: boolean;
  startError: string | null;
  // start() resolves to whether the run actually proceeded (POST accepted,
  // or a 409 meant one was already in flight) — the caller (SetupWizard)
  // uses this to decide whether to advance to the run step, mirroring
  // startSetup's own early `return` on a hard POST failure that leaves the
  // wizard on the review step with startError shown inline.
  start: (wire: api.StartSetupRequest) => Promise<boolean>;
}

// useSetupRun mirrors wizard.ts's own startSetup — see this file's header
// comment for the exact teardown/409 semantics it reproduces.
export function useSetupRun(targetId: string): UseSetupRunResult {
  const [events, setEvents] = useState<api.SetupEvent[]>([]);
  const [starting, setStarting] = useState(false);
  const [startError, setStartError] = useState<string | null>(null);
  // startingRef is the synchronous mirror of `starting`, guarding a
  // chained-async re-entrant call the way busy/busyRef pairs do elsewhere
  // (services.ts's own provision()) — `starting` state can't be read back
  // synchronously inside start() itself.
  const startingRef = useRef(false);
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

  const start = useCallback(
    async (wire: api.StartSetupRequest): Promise<boolean> => {
      if (startingRef.current) return false;
      startingRef.current = true;
      setStarting(true);
      setStartError(null);
      setEvents([]);
      // Reset run state up front — this is what makes "Retry setup" actually
      // retry: without stopping any stale stream and clearing events here, a
      // second click could leave the previous run's failed events on screen
      // even once a fresh POST + stream are issued below.
      streamRef.current?.();
      streamRef.current = null;

      try {
        await api.startSetup(targetId, wire);
      } catch (err) {
        // A 409 means a run is already in flight for this target — that's
        // fine, attach to its live stream below instead of starting a new
        // one.
        if (!(err instanceof api.ApiError && err.status === 409)) {
          startingRef.current = false;
          setStarting(false);
          setStartError(err instanceof Error ? err.message : String(err));
          return false;
        }
      }

      startingRef.current = false;
      if (!mountedRef.current) return true;
      setStarting(false);
      streamRef.current = api.streamSetup(targetId, (ev) => {
        if (!mountedRef.current) return;
        setEvents((prev) => [...prev, ev]);
      });
      return true;
    },
    [targetId],
  );

  return { events, starting, startError, start };
}
