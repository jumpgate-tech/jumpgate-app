// Shared React Query hooks over api.ts's gateway surface — the data layer the
// panel (next task) and later screens consume, so no screen has to hand-roll
// its own load()/poll()/invalidate bookkeeping the way panel.ts (the vanilla
// reference this mirrors) does.
import { useCallback, useRef } from "react";
import { useMutation, useQuery, useQueryClient, type UseQueryResult } from "@tanstack/react-query";
import * as api from "../api";

// useGateways lists every gateway in the fleet. api.getGateways() itself
// returns the wider GatewaysResponse (gateways + targets + sources + presets
// + orphans) — this hook narrows to the `gateways` array callers actually
// asked for, coalescing the wire's possible null (see GatewaysResponse) to [].
export function useGateways(): UseQueryResult<api.GatewayView[]> {
  // Shares the ["gateways"] cache (the full GatewaysResponse) with the eRPC
  // screen's useGatewaysFull, narrowing to the array via `select` so both
  // observers read one consistent cached shape and one invalidation refreshes
  // both — rather than a second queryFn that could store a different shape
  // under the same key.
  return useQuery({
    queryKey: ["gateways"],
    queryFn: () => api.getGateways(),
    select: (res) => res.gateways ?? [],
  });
}

// useGatewayHealth is the panel's live-dot heartbeat (netHealth in panel.ts):
// a 5s poll of one gateway's analytics scrape, disabled whenever there is no
// gateway id or the caller says it isn't needed (e.g. a screen that never
// shows a dot). refetchInterval keeps polling even while the query is
// otherwise fresh, matching panel.ts's unconditional setInterval.
export function useGatewayHealth(
  gid: string | undefined,
  enabled: boolean,
): UseQueryResult<api.GatewayAnalytics> {
  return useQuery({
    queryKey: ["gwHealth", gid],
    queryFn: () => api.getGatewayAnalytics(gid as string),
    enabled: enabled && !!gid,
    refetchInterval: 5000,
  });
}

// useGatewayCapabilities mirrors panel.ts's netCaps: probed lazily (opening
// real sockets against real endpoints), cached indefinitely (staleTime:
// Infinity — nothing here should ride a poll cadence), and re-probed only on
// an explicit ask. getGatewayCapabilities(gid, refresh) forces the SERVER to
// re-probe (rather than serve its own cached read) only when refresh=true —
// see api.ts's own comment on that function — so a plain `refetch()` (refresh
// omitted, defaulting to false) would just re-fetch this hook's cache without
// forcing a fresh probe server-side. The "recheck" action needs the latter, so
// the returned refetch takes an optional `refresh` flag: it's stashed in a
// ref the queryFn reads (React Query's refetch() takes no argument of its
// own), then a real refetch is triggered.
export function useGatewayCapabilities(
  gid: string | undefined,
  enabled: boolean,
): Omit<UseQueryResult<api.GatewayCapabilities>, "refetch"> & {
  refetch: (refresh?: boolean) => Promise<unknown>;
} {
  const forceRefresh = useRef(false);
  const query = useQuery({
    queryKey: ["gwCaps", gid],
    queryFn: () => {
      const refresh = forceRefresh.current;
      forceRefresh.current = false;
      return api.getGatewayCapabilities(gid as string, refresh);
    },
    enabled: enabled && !!gid,
    staleTime: Infinity,
  });
  const refetch = useCallback(
    (refresh = false) => {
      forceRefresh.current = refresh;
      return query.refetch();
    },
    [query.refetch],
  );
  return { ...query, refetch };
}

// useGatewayAction runs start/stop/restart (mirrors panel.ts's runAction).
// Both the gateway list (its status.State) and the health poll (analytics
// availability tracks whether the container is running) can change as a
// result, so both are invalidated.
export function useGatewayAction() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: ({ gid, action }: { gid: string; action: api.ContainerActionKind }) =>
      api.gatewayAction(gid, action),
    onSuccess: (_data, { gid }) => {
      void qc.invalidateQueries({ queryKey: ["gateways"] });
      void qc.invalidateQueries({ queryKey: ["gwHealth", gid] });
    },
  });
}

// usePutGatewayConfig stores the desired config (mirrors panel.ts's config
// writes, e.g. withNetwork/withUpstream call sites). It never touches a
// running container on its own — see putGatewayConfig's own comment —  so
// only the gateway list (config, warnings) is invalidated; applying the
// change is provisionGateway's job, outside this hook.
export function usePutGatewayConfig() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: ({ gid, config }: { gid: string; config: api.GatewayConfig }) =>
      api.putGatewayConfig(gid, config),
    onSuccess: () => {
      void qc.invalidateQueries({ queryKey: ["gateways"] });
    },
  });
}

// useWipeGateway mirrors panel.ts's runWipe. wipeGateway resolves NORMALLY
// even on a partial failure (the wipe happened; a cascade in front of it did
// not) — see WipeResult/wipeGateway's own comments — so this hook must not
// swallow `.error`: it's left on the resolved data for the caller to read
// (e.g. `mutation.data?.error`), exactly as runWipe reads `result.error` into
// actionErr rather than treating a resolved promise as unconditional success.
export function useWipeGateway() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (gid: string) => api.wipeGateway(gid),
    onSuccess: (_data, gid) => {
      void qc.invalidateQueries({ queryKey: ["gateways"] });
      void qc.invalidateQueries({ queryKey: ["gwHealth", gid] });
    },
  });
}
