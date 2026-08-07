// Shared React Query hooks over api.ts's VPN surface — the typed data layer a
// VPN screen (a later task) consumes, so no screen has to hand-roll its own
// load()/poll()/invalidate bookkeeping. Mirrors hooks/gateway.ts.
import { useMutation, useQuery, useQueryClient, type UseQueryResult } from "@tanstack/react-query";
import * as api from "../api";

// useVpns lists every BYO overlay. The wire returns a plain array (never null
// here), so no coalescing is needed — unlike getGateways' nested response.
export function useVpns(): UseQueryResult<api.VpnView[]> {
  return useQuery({
    queryKey: ["vpns"],
    queryFn: () => api.getVpns(),
  });
}

// useVpnServers lists every provisioned WireGuard server.
export function useVpnServers(): UseQueryResult<api.VpnServerView[]> {
  return useQuery({
    queryKey: ["vpnServers"],
    queryFn: () => api.getVpnServers(),
  });
}

// useVpnStatus is one overlay's live status heartbeat — a 5s poll, disabled
// whenever there is no id or the caller says it isn't needed. refetchInterval
// keeps polling even while the query is otherwise fresh.
export function useVpnStatus(
  id: string | undefined,
  enabled: boolean,
): UseQueryResult<api.VpnStatus> {
  return useQuery({
    queryKey: ["vpnStatus", id],
    queryFn: () => api.getVpnStatus(id as string),
    enabled: enabled && !!id,
    refetchInterval: 5000,
  });
}

// useVpnServerStatus is one server's live status heartbeat, same cadence and
// gating as useVpnStatus.
export function useVpnServerStatus(
  id: string | undefined,
  enabled: boolean,
): UseQueryResult<api.VpnStatus> {
  return useQuery({
    queryKey: ["vpnServerStatus", id],
    queryFn: () => api.getVpnServerStatus(id as string),
    enabled: enabled && !!id,
    refetchInterval: 5000,
  });
}

// useSaveVpn upserts a BYO overlay. Its id, binding, config, validity and
// autostart all live on the list, so ["vpns"] is invalidated; a change of
// binding can also flip a status, so the acted-on overlay's status is too.
export function useSaveVpn() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (body: api.SaveVpnRequest) => api.saveVpn(body),
    onSuccess: (_data, body) => {
      void qc.invalidateQueries({ queryKey: ["vpns"] });
      void qc.invalidateQueries({ queryKey: ["vpnStatus", body.id] });
    },
  });
}

// useDeleteVpn removes a BYO overlay.
export function useDeleteVpn() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (id: string) => api.deleteVpn(id),
    onSuccess: (_data, id) => {
      void qc.invalidateQueries({ queryKey: ["vpns"] });
      void qc.invalidateQueries({ queryKey: ["vpnStatus", id] });
    },
  });
}

// useVpnAction brings a BYO overlay up or down. Both change its status and its
// list-level up/valid/peers reading, so both are invalidated. "up" returns a
// VpnStatus, "down" returns nothing (204) — the mutation resolves either way
// and callers read status from a fresh useVpnStatus, not the return value.
export function useVpnAction() {
  const qc = useQueryClient();
  return useMutation<api.VpnStatus | void, Error, { id: string; action: "up" | "down" }>({
    mutationFn: ({ id, action }) => (action === "up" ? api.vpnUp(id) : api.vpnDown(id)),
    onSuccess: (_data, { id }) => {
      void qc.invalidateQueries({ queryKey: ["vpns"] });
      void qc.invalidateQueries({ queryKey: ["vpnStatus", id] });
    },
  });
}

// useProvisionVpnServer stands up (or re-applies) a WireGuard server. The
// result carries the server plus a firewall hint the caller surfaces; the
// server list changes, so ["vpnServers"] is invalidated.
export function useProvisionVpnServer() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (body: api.ProvisionVpnServerRequest) => api.provisionVpnServer(body),
    onSuccess: () => {
      void qc.invalidateQueries({ queryKey: ["vpnServers"] });
    },
  });
}

// useEnrollVpnDevice adds a peer to a server. The returned VpnEnrollResult
// carries the peer's full client config — emitted only here — so it is left on
// the resolved data for the caller to surface (mutation.data?.config); the
// server's peer list grew, so ["vpnServers"] is invalidated.
export function useEnrollVpnDevice() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: ({ id, body }: { id: string; body: api.EnrollVpnDeviceRequest }) =>
      api.enrollVpnDevice(id, body),
    onSuccess: () => {
      void qc.invalidateQueries({ queryKey: ["vpnServers"] });
    },
  });
}

// useRevokeVpnDevice removes a peer (by public key) from a server.
export function useRevokeVpnDevice() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: ({ id, publicKey }: { id: string; publicKey: string }) =>
      api.revokeVpnDevice(id, publicKey),
    onSuccess: () => {
      void qc.invalidateQueries({ queryKey: ["vpnServers"] });
    },
  });
}

// useDeleteVpnServer tears a provisioned server down.
export function useDeleteVpnServer() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (id: string) => api.deleteVpnServer(id),
    onSuccess: (_data, id) => {
      void qc.invalidateQueries({ queryKey: ["vpnServers"] });
      void qc.invalidateQueries({ queryKey: ["vpnServerStatus", id] });
    },
  });
}
