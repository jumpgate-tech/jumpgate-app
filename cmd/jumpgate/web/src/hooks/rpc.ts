// The eRPC screen's data layer — the rpc-specific hooks over api.ts that the
// converted #/rpc screen consumes, kept out of hooks/gateway.ts (the shared
// gateway CRUD the panel also uses) so this screen's own create/traffic/
// verify/trust/provision bookkeeping lives beside the screen it serves.
//
// The gateway-list + lifecycle + config + wipe hooks it also needs already
// exist in hooks/gateway.ts and are reused verbatim; this file adds only what
// the fuller operator view needs on top.
import { useCallback, useEffect, useRef, useState } from "react";
import { useMutation, useQuery, useQueryClient, type UseQueryResult } from "@tanstack/react-query";
import * as api from "../api";
import { FINAL_STEP } from "../screens/Rpc/rpcModel";

function message(err: unknown): string {
  return err instanceof Error ? err.message : String(err);
}

// hintOf surfaces the server's operator-facing hint verbatim (mirrors rpc.ts's
// own hintOf) — it is written for exactly this moment.
function hintOf(err: unknown): string {
  return err instanceof api.ApiError && err.hint ? ` — ${err.hint}` : "";
}

// useGatewaysFull is the whole GatewaysResponse — gateways PLUS the targets,
// sources, presets and orphans the operator view needs. It shares the
// ["gateways"] key (and queryFn) with hooks/gateway.ts's useGateways, which
// narrows the same cached response to the array; every gateway mutation already
// invalidates that key, so both refresh together.
export function useGatewaysFull(): UseQueryResult<api.GatewaysResponse> {
  return useQuery({ queryKey: ["gateways"], queryFn: () => api.getGateways() });
}

// useGatewayTraffic reads one gateway's request counters. It never polls (the
// legacy screen read it once per load, then again after an action), and a
// failure is a query error the caller renders as an explained blank column —
// never a banner, because unreadable counters say nothing about serving.
export function useGatewayTraffic(gid: string | undefined): UseQueryResult<api.GatewayTraffic> {
  return useQuery({
    queryKey: ["gwTraffic", gid],
    queryFn: () => api.getGatewayTraffic(gid as string),
    enabled: !!gid,
  });
}

// useCreateGateway registers a new gateway on a machine that has none.
export function useCreateGateway() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (body: api.CreateGatewayRequest) => api.createGateway(body),
    onSuccess: () => void qc.invalidateQueries({ queryKey: ["gateways"] }),
  });
}

// useDeleteGateway is "Forget": it drops the saved configuration only and never
// touches the container (see deleteGateway's own note).
export function useDeleteGateway() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (gid: string) => api.deleteGateway(gid),
    onSuccess: () => void qc.invalidateQueries({ queryKey: ["gateways"] }),
  });
}

// useDismissOrphan forgets a leftover-container record only.
export function useDismissOrphan() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (name: string) => api.dismissOrphan(name),
    onSuccess: () => void qc.invalidateQueries({ queryKey: ["gateways"] }),
  });
}

// useKnownSet reads valve's vetted, measured set for a chain — an on-demand
// read (opened from a modal), so a mutation rather than a mounted query.
export function useKnownSet() {
  return useMutation({
    mutationFn: ({ gid, chainId }: { gid: string; chainId: number }) => api.knownSet(gid, chainId),
  });
}

// useDiscoverEndpoints probes chainid.network for a chain, on demand.
export function useDiscoverEndpoints() {
  return useMutation({ mutationFn: (chainId: number) => api.discoverEndpoints(chainId) });
}

// useVerifyTls runs the live HTTPS check. The result is on mutation.data, the
// reason a run failed to start on mutation.error, and the in-flight state on
// isPending — the three the legacy verifyResult/verifyErr/verifyBusy held.
export function useVerifyTls() {
  return useMutation({ mutationFn: (gid: string) => api.verifyGatewayTls(gid) });
}

// useTrustCert installs the gateway's internal-CA root into the trust store of
// the machine it runs on. It always RESOLVES to a TrustCertResult — a thrown
// error becomes { ok:false, message } — so the result line can render the exact
// command to run by hand, exactly as the legacy trustCert catch did.
export function useTrustCert() {
  return useMutation({
    mutationFn: async (gid: string): Promise<api.TrustCertResult> => {
      try {
        return await api.trustGatewayCert(gid);
      } catch (err) {
        return { ok: false, message: `${message(err)}${hintOf(err)}` };
      }
    },
  });
}

// useResetDevnet throws a devnet's chain away and rebuilds it from genesis,
// cascading a restart to the gateways in front of it. wipeGateway/resetDevnet
// resolve even on a partial failure (the reset happened; a cascade did not), so
// the resolved data carries `.error` for the caller to read.
export function useResetDevnet() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (targetId: string) => api.resetDevnet(targetId),
    onSuccess: () => void qc.invalidateQueries({ queryKey: ["gateways"] }),
  });
}

export interface UseGatewayOpsResult {
  // busy holds the action kind in flight ("start"/"stop"/"restart"/"create"),
  // or null when idle — the single source every lifecycle button disables from.
  busy: string | null;
  actionErr: string | null;
  // activity is the provisioning progress log, kept in state (not a ref/DOM) so
  // it survives a re-render triggered by any other card.
  activity: string[];
  setActionErr: (s: string | null) => void;
  note: (text: string) => void;
  clearActivity: () => void;
  runAction: (kind: api.ContainerActionKind) => Promise<void>;
  provision: () => Promise<void>;
}

// useGatewayOps mirrors rpc.ts's busy/actionErr/activity closures plus
// runAction/provision. Provisioning follows the PLACEMENT machine's setup event
// stream imperatively (api.streamSetup, like services.ts/wizard.ts — NOT
// useEventStream), single-flighted through a busyRef so the guard sees the
// latest value synchronously, and torn down on unmount and on finish.
export function useGatewayOps(gid: string): UseGatewayOpsResult {
  const qc = useQueryClient();
  const [busy, setBusyState] = useState<string | null>(null);
  const [actionErr, setActionErr] = useState<string | null>(null);
  const [activity, setActivity] = useState<string[]>([]);
  const busyRef = useRef<string | null>(null);
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

  const setBusy = useCallback((v: string | null) => {
    busyRef.current = v;
    setBusyState(v);
  }, []);
  const reload = useCallback(() => qc.invalidateQueries({ queryKey: ["gateways"] }), [qc]);

  const runAction = useCallback(
    async (kind: api.ContainerActionKind) => {
      if (busyRef.current) return;
      setBusy(kind);
      setActionErr(null);
      try {
        await api.gatewayAction(gid, kind);
      } catch (err) {
        setActionErr(`${kind} failed: ${message(err)}${hintOf(err)}`);
      }
      setBusy(null);
      await reload();
    },
    [gid, reload, setBusy],
  );

  const provision = useCallback(async () => {
    if (busyRef.current) return;
    setBusy("create");
    setActionErr(null);
    setActivity(["starting…"]);
    let started: { targetId: string };
    try {
      started = await api.provisionGateway(gid);
    } catch (err) {
      setActionErr(`${message(err)}${hintOf(err)}`);
      setActivity([]);
      setBusy(null);
      return;
    }
    streamRef.current?.();
    streamRef.current = api.streamSetup(started.targetId, (ev) => {
      if (!mountedRef.current) return;
      const line = ev.err ? `${ev.stepId}: ${ev.err}` : ev.line ? `${ev.stepId}: ${ev.line}` : `${ev.stepId}: done`;
      setActivity((prev) => [...prev.filter((l) => l !== "starting…"), line]);
      const finished = !!ev.err || (ev.stepId === FINAL_STEP && !!ev.done);
      if (!finished) return;
      streamRef.current?.();
      streamRef.current = null;
      setBusy(null);
      if (ev.err) setActionErr("Provisioning failed — see the log below.");
      void reload();
    });
  }, [gid, reload, setBusy]);

  const note = useCallback((text: string) => setActivity([text]), []);
  const clearActivity = useCallback(() => setActivity([]), []);

  return { busy, actionErr, activity, setActionErr, note, clearActivity, runAction, provision };
}
