import { renderHook, waitFor } from "@testing-library/react";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { describe, it, expect, vi, beforeEach } from "vitest";
import type { ReactNode } from "react";
import * as api from "../api";
import type {
  VpnView,
  VpnServerView,
  VpnStatus,
  VpnServerProvisionResult,
  VpnEnrollResult,
} from "../api";
import {
  useVpns,
  useVpnServers,
  useVpnStatus,
  useVpnServerStatus,
  useSaveVpn,
  useDeleteVpn,
  useVpnAction,
  useProvisionVpnServer,
  useEnrollVpnDevice,
  useRevokeVpnDevice,
  useDeleteVpnServer,
} from "./vpn";

vi.mock("../api", async () => {
  const actual = await vi.importActual<typeof import("../api")>("../api");
  return {
    ...actual,
    getVpns: vi.fn(),
    getVpnServers: vi.fn(),
    getVpnStatus: vi.fn(),
    getVpnServerStatus: vi.fn(),
    saveVpn: vi.fn(),
    deleteVpn: vi.fn(),
    vpnUp: vi.fn(),
    vpnDown: vi.fn(),
    provisionVpnServer: vi.fn(),
    enrollVpnDevice: vi.fn(),
    revokeVpnDevice: vi.fn(),
    deleteVpnServer: vi.fn(),
  };
});

function makeVpn(id: string): VpnView {
  return {
    id,
    provider: "wireguard",
    interface: "wg0",
    targetId: "local",
    autostart: false,
    configured: true,
    valid: true,
    endpoints: [],
    overlay: ["10.0.0.0/24"],
    peers: 0,
  };
}

function makeServer(id: string): VpnServerView {
  return {
    id,
    targetId: "local",
    interface: "wg0",
    address: "10.0.0.1/24",
    listenPort: 51820,
    publicKey: "srv-pub",
    endpoint: "vpn.example:51820",
    peers: [],
  };
}

const STATUS: VpnStatus = {
  id: "vpn-1",
  up: true,
  interface: "wg0",
  provider: "wireguard",
  addresses: ["10.0.0.2/32"],
  peers: 1,
  handshaked: true,
  lastHandshake: 1700000000,
};

function wrapper({ children }: { children: ReactNode }) {
  const queryClient = new QueryClient({
    defaultOptions: { queries: { retry: false }, mutations: { retry: false } },
  });
  return <QueryClientProvider client={queryClient}>{children}</QueryClientProvider>;
}

// Each invalidation test needs a shared client between the list observer and the
// mutation, so the mutation's invalidate can trigger a refetch we can assert on.
function makeSharedWrapper() {
  const queryClient = new QueryClient({
    defaultOptions: { queries: { retry: false }, mutations: { retry: false } },
  });
  function localWrapper({ children }: { children: ReactNode }) {
    return <QueryClientProvider client={queryClient}>{children}</QueryClientProvider>;
  }
  return localWrapper;
}

beforeEach(() => {
  vi.mocked(api.getVpns).mockReset();
  vi.mocked(api.getVpnServers).mockReset();
  vi.mocked(api.getVpnStatus).mockReset();
  vi.mocked(api.getVpnServerStatus).mockReset();
  vi.mocked(api.saveVpn).mockReset();
  vi.mocked(api.deleteVpn).mockReset();
  vi.mocked(api.vpnUp).mockReset();
  vi.mocked(api.vpnDown).mockReset();
  vi.mocked(api.provisionVpnServer).mockReset();
  vi.mocked(api.enrollVpnDevice).mockReset();
  vi.mocked(api.revokeVpnDevice).mockReset();
  vi.mocked(api.deleteVpnServer).mockReset();
});

describe("useVpns", () => {
  it("returns the overlays from getVpns()", async () => {
    const vpns = [makeVpn("vpn-1"), makeVpn("vpn-2")];
    vi.mocked(api.getVpns).mockResolvedValue(vpns);

    const { result } = renderHook(() => useVpns(), { wrapper });

    await waitFor(() => expect(result.current.isSuccess).toBe(true));
    expect(api.getVpns).toHaveBeenCalledTimes(1);
    expect(result.current.data).toEqual(vpns);
  });
});

describe("useVpnServers", () => {
  it("returns the servers from getVpnServers()", async () => {
    const servers = [makeServer("srv-1")];
    vi.mocked(api.getVpnServers).mockResolvedValue(servers);

    const { result } = renderHook(() => useVpnServers(), { wrapper });

    await waitFor(() => expect(result.current.isSuccess).toBe(true));
    expect(api.getVpnServers).toHaveBeenCalledTimes(1);
    expect(result.current.data).toEqual(servers);
  });
});

describe("useVpnStatus", () => {
  it("fetches status when enabled with an id", async () => {
    vi.mocked(api.getVpnStatus).mockResolvedValue(STATUS);

    const { result } = renderHook(() => useVpnStatus("vpn-1", true), { wrapper });

    await waitFor(() => expect(result.current.isSuccess).toBe(true));
    expect(api.getVpnStatus).toHaveBeenCalledWith("vpn-1");
    expect(result.current.data).toEqual(STATUS);
  });

  it("is disabled (does not call the api) when enabled=false", async () => {
    const { result } = renderHook(() => useVpnStatus("vpn-1", false), { wrapper });

    await new Promise((r) => setTimeout(r, 0));

    expect(api.getVpnStatus).not.toHaveBeenCalled();
    expect(result.current.fetchStatus).toBe("idle");
  });

  it("is disabled when there is no id", async () => {
    const { result } = renderHook(() => useVpnStatus(undefined, true), { wrapper });

    await new Promise((r) => setTimeout(r, 0));

    expect(api.getVpnStatus).not.toHaveBeenCalled();
    expect(result.current.fetchStatus).toBe("idle");
  });
});

describe("useVpnServerStatus", () => {
  it("fetches server status when enabled with an id", async () => {
    vi.mocked(api.getVpnServerStatus).mockResolvedValue(STATUS);

    const { result } = renderHook(() => useVpnServerStatus("srv-1", true), { wrapper });

    await waitFor(() => expect(result.current.isSuccess).toBe(true));
    expect(api.getVpnServerStatus).toHaveBeenCalledWith("srv-1");
    expect(result.current.data).toEqual(STATUS);
  });

  it("is disabled when there is no id", async () => {
    const { result } = renderHook(() => useVpnServerStatus(undefined, true), { wrapper });

    await new Promise((r) => setTimeout(r, 0));

    expect(api.getVpnServerStatus).not.toHaveBeenCalled();
    expect(result.current.fetchStatus).toBe("idle");
  });
});

describe("useSaveVpn", () => {
  it("upserts and invalidates the vpns list on success", async () => {
    const localWrapper = makeSharedWrapper();
    vi.mocked(api.getVpns).mockResolvedValue([makeVpn("vpn-1")]);
    vi.mocked(api.saveVpn).mockResolvedValue(makeVpn("vpn-1"));

    const list = renderHook(() => useVpns(), { wrapper: localWrapper });
    await waitFor(() => expect(list.result.current.isSuccess).toBe(true));
    expect(api.getVpns).toHaveBeenCalledTimes(1);

    const { result } = renderHook(() => useSaveVpn(), { wrapper: localWrapper });
    result.current.mutate({ id: "vpn-1", provider: "wireguard" });

    await waitFor(() => expect(result.current.isSuccess).toBe(true));
    expect(api.saveVpn).toHaveBeenCalledWith({ id: "vpn-1", provider: "wireguard" });
    await waitFor(() => expect(api.getVpns).toHaveBeenCalledTimes(2));
  });
});

describe("useDeleteVpn", () => {
  it("deletes and invalidates the vpns list on success", async () => {
    const localWrapper = makeSharedWrapper();
    vi.mocked(api.getVpns).mockResolvedValue([makeVpn("vpn-1")]);
    vi.mocked(api.deleteVpn).mockResolvedValue(undefined);

    const list = renderHook(() => useVpns(), { wrapper: localWrapper });
    await waitFor(() => expect(list.result.current.isSuccess).toBe(true));

    const { result } = renderHook(() => useDeleteVpn(), { wrapper: localWrapper });
    result.current.mutate("vpn-1");

    await waitFor(() => expect(result.current.isSuccess).toBe(true));
    expect(api.deleteVpn).toHaveBeenCalledWith("vpn-1");
    await waitFor(() => expect(api.getVpns).toHaveBeenCalledTimes(2));
  });
});

describe("useVpnAction", () => {
  it("calls vpnUp for the up action and invalidates the list", async () => {
    const localWrapper = makeSharedWrapper();
    vi.mocked(api.getVpns).mockResolvedValue([makeVpn("vpn-1")]);
    vi.mocked(api.vpnUp).mockResolvedValue(STATUS);

    const list = renderHook(() => useVpns(), { wrapper: localWrapper });
    await waitFor(() => expect(list.result.current.isSuccess).toBe(true));

    const { result } = renderHook(() => useVpnAction(), { wrapper: localWrapper });
    result.current.mutate({ id: "vpn-1", action: "up" });

    await waitFor(() => expect(result.current.isSuccess).toBe(true));
    expect(api.vpnUp).toHaveBeenCalledWith("vpn-1");
    expect(api.vpnDown).not.toHaveBeenCalled();
    await waitFor(() => expect(api.getVpns).toHaveBeenCalledTimes(2));
  });

  it("calls vpnDown for the down action", async () => {
    vi.mocked(api.vpnDown).mockResolvedValue(undefined);

    const { result } = renderHook(() => useVpnAction(), { wrapper });
    result.current.mutate({ id: "vpn-1", action: "down" });

    await waitFor(() => expect(result.current.isSuccess).toBe(true));
    expect(api.vpnDown).toHaveBeenCalledWith("vpn-1");
    expect(api.vpnUp).not.toHaveBeenCalled();
  });
});

describe("useProvisionVpnServer", () => {
  it("provisions and invalidates the servers list, surfacing the firewall hint", async () => {
    const localWrapper = makeSharedWrapper();
    vi.mocked(api.getVpnServers).mockResolvedValue([makeServer("srv-1")]);
    const provision: VpnServerProvisionResult = {
      server: makeServer("srv-1"),
      firewallHint: "open udp 51820",
      endpointConfigured: true,
    };
    vi.mocked(api.provisionVpnServer).mockResolvedValue(provision);

    const list = renderHook(() => useVpnServers(), { wrapper: localWrapper });
    await waitFor(() => expect(list.result.current.isSuccess).toBe(true));
    expect(api.getVpnServers).toHaveBeenCalledTimes(1);

    const { result } = renderHook(() => useProvisionVpnServer(), { wrapper: localWrapper });
    result.current.mutate({ id: "srv-1" });

    await waitFor(() => expect(result.current.isSuccess).toBe(true));
    expect(api.provisionVpnServer).toHaveBeenCalledWith({ id: "srv-1" });
    expect(result.current.data?.firewallHint).toBe("open udp 51820");
    await waitFor(() => expect(api.getVpnServers).toHaveBeenCalledTimes(2));
  });
});

describe("useEnrollVpnDevice", () => {
  it("enrolls a device, surfaces the returned config, and invalidates servers", async () => {
    const localWrapper = makeSharedWrapper();
    vi.mocked(api.getVpnServers).mockResolvedValue([makeServer("srv-1")]);
    const enroll: VpnEnrollResult = {
      name: "laptop",
      publicKey: "peer-pub",
      allowedIp: "10.0.0.2/32",
      config: "[Interface]\nPrivateKey = ...\n[Peer]\n...",
    };
    vi.mocked(api.enrollVpnDevice).mockResolvedValue(enroll);

    const list = renderHook(() => useVpnServers(), { wrapper: localWrapper });
    await waitFor(() => expect(list.result.current.isSuccess).toBe(true));

    const { result } = renderHook(() => useEnrollVpnDevice(), { wrapper: localWrapper });
    result.current.mutate({ id: "srv-1", body: { name: "laptop" } });

    await waitFor(() => expect(result.current.isSuccess).toBe(true));
    expect(api.enrollVpnDevice).toHaveBeenCalledWith("srv-1", { name: "laptop" });
    // The client config is emitted exactly once, here — the hook must surface it.
    expect(result.current.data?.config).toBe(enroll.config);
    await waitFor(() => expect(api.getVpnServers).toHaveBeenCalledTimes(2));
  });
});

describe("useRevokeVpnDevice", () => {
  it("revokes a peer by public key and invalidates servers", async () => {
    const localWrapper = makeSharedWrapper();
    vi.mocked(api.getVpnServers).mockResolvedValue([makeServer("srv-1")]);
    vi.mocked(api.revokeVpnDevice).mockResolvedValue(undefined);

    const list = renderHook(() => useVpnServers(), { wrapper: localWrapper });
    await waitFor(() => expect(list.result.current.isSuccess).toBe(true));

    const { result } = renderHook(() => useRevokeVpnDevice(), { wrapper: localWrapper });
    result.current.mutate({ id: "srv-1", publicKey: "peer-pub" });

    await waitFor(() => expect(result.current.isSuccess).toBe(true));
    expect(api.revokeVpnDevice).toHaveBeenCalledWith("srv-1", "peer-pub");
    await waitFor(() => expect(api.getVpnServers).toHaveBeenCalledTimes(2));
  });
});

describe("useDeleteVpnServer", () => {
  it("deletes a server and invalidates the servers list", async () => {
    const localWrapper = makeSharedWrapper();
    vi.mocked(api.getVpnServers).mockResolvedValue([makeServer("srv-1")]);
    vi.mocked(api.deleteVpnServer).mockResolvedValue(undefined);

    const list = renderHook(() => useVpnServers(), { wrapper: localWrapper });
    await waitFor(() => expect(list.result.current.isSuccess).toBe(true));

    const { result } = renderHook(() => useDeleteVpnServer(), { wrapper: localWrapper });
    result.current.mutate("srv-1");

    await waitFor(() => expect(result.current.isSuccess).toBe(true));
    expect(api.deleteVpnServer).toHaveBeenCalledWith("srv-1");
    await waitFor(() => expect(api.getVpnServers).toHaveBeenCalledTimes(2));
  });
});
