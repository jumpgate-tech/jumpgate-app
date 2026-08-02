import { renderHook, waitFor } from "@testing-library/react";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { describe, it, expect, vi, beforeEach } from "vitest";
import type { ReactNode } from "react";
import * as api from "../api";
import type { GatewayView, GatewayAnalytics, GatewayCapabilities, WipeResult } from "../api";
import {
  useGateways,
  useGatewayHealth,
  useGatewayCapabilities,
  useGatewayAction,
  usePutGatewayConfig,
  useWipeGateway,
} from "./gateway";

vi.mock("../api", async () => {
  const actual = await vi.importActual<typeof import("../api")>("../api");
  return {
    ...actual,
    getGateways: vi.fn(),
    getGatewayAnalytics: vi.fn(),
    getGatewayCapabilities: vi.fn(),
    gatewayAction: vi.fn(),
    putGatewayConfig: vi.fn(),
    wipeGateway: vi.fn(),
  };
});

const STATUS = {
  ID: "gw-1",
  ContainerName: "erpc-gw-1",
  State: "running" as const,
  Image: "erpc:latest",
  ImageID: "sha256:abc",
  ExitCode: 0,
  Platform: "linux/amd64",
  EnginePlatform: "linux/amd64",
  Emulated: false,
  Detail: "",
};

function makeGateway(id: string): GatewayView {
  return {
    id,
    label: id,
    containerName: `erpc-${id}`,
    placement: { targetId: "local", backend: "docker" },
    status: STATUS,
    docker: { present: true, reachable: true, flavor: "docker" },
    baseUrl: "http://localhost:4000",
    tls: { enabled: false, status: STATUS },
    networks: [],
    actions: ["start", "stop", "restart", "wipe"],
    wipeDiscards: "the container and its config",
    config: { ProjectID: id, BindAddr: "0.0.0.0", Port: 4000, Networks: [] },
  };
}

function wrapper({ children }: { children: ReactNode }) {
  const queryClient = new QueryClient({
    defaultOptions: { queries: { retry: false }, mutations: { retry: false } },
  });
  return <QueryClientProvider client={queryClient}>{children}</QueryClientProvider>;
}

beforeEach(() => {
  vi.mocked(api.getGateways).mockReset();
  vi.mocked(api.getGatewayAnalytics).mockReset();
  vi.mocked(api.getGatewayCapabilities).mockReset();
  vi.mocked(api.gatewayAction).mockReset();
  vi.mocked(api.putGatewayConfig).mockReset();
  vi.mocked(api.wipeGateway).mockReset();
});

describe("useGateways", () => {
  it("returns the gateways array from getGateways()'s response", async () => {
    const gws = [makeGateway("gw-1"), makeGateway("gw-2")];
    vi.mocked(api.getGateways).mockResolvedValue({
      gateways: gws,
      targets: [],
      sources: [],
      presets: [],
    });

    const { result } = renderHook(() => useGateways(), { wrapper });

    await waitFor(() => expect(result.current.isSuccess).toBe(true));
    expect(result.current.data).toEqual(gws);
  });

  it("coalesces a null gateways field to []", async () => {
    vi.mocked(api.getGateways).mockResolvedValue({
      gateways: null,
      targets: [],
      sources: [],
      presets: [],
    });

    const { result } = renderHook(() => useGateways(), { wrapper });

    await waitFor(() => expect(result.current.isSuccess).toBe(true));
    expect(result.current.data).toEqual([]);
  });
});

describe("useGatewayHealth", () => {
  it("fetches analytics when enabled with a gid", async () => {
    const analytics: GatewayAnalytics = {
      enabled: true,
      at: "now",
      since: "then",
      networks: [],
      endpoints: [],
    };
    vi.mocked(api.getGatewayAnalytics).mockResolvedValue(analytics);

    const { result } = renderHook(() => useGatewayHealth("gw-1", true), { wrapper });

    await waitFor(() => expect(result.current.isSuccess).toBe(true));
    expect(api.getGatewayAnalytics).toHaveBeenCalledWith("gw-1");
    expect(result.current.data).toEqual(analytics);
  });

  it("is disabled (does not call the api) when enabled=false", async () => {
    const { result } = renderHook(() => useGatewayHealth("gw-1", false), { wrapper });

    // Give any stray async work a tick to run.
    await new Promise((r) => setTimeout(r, 0));

    expect(api.getGatewayAnalytics).not.toHaveBeenCalled();
    expect(result.current.fetchStatus).toBe("idle");
  });

  it("is disabled when there is no gid", async () => {
    const { result } = renderHook(() => useGatewayHealth(undefined, true), { wrapper });

    await new Promise((r) => setTimeout(r, 0));

    expect(api.getGatewayAnalytics).not.toHaveBeenCalled();
    expect(result.current.fetchStatus).toBe("idle");
  });
});

describe("useGatewayCapabilities", () => {
  it("fetches (without forcing a re-probe) when enabled with a gid", async () => {
    const caps: GatewayCapabilities = { at: "now", endpoints: [] };
    vi.mocked(api.getGatewayCapabilities).mockResolvedValue(caps);

    const { result } = renderHook(() => useGatewayCapabilities("gw-1", true), { wrapper });

    await waitFor(() => expect(result.current.isSuccess).toBe(true));
    expect(api.getGatewayCapabilities).toHaveBeenCalledWith("gw-1", false);
    expect(result.current.data).toEqual(caps);
  });

  it("is disabled (does not call the api) when enabled=false", async () => {
    const { result } = renderHook(() => useGatewayCapabilities("gw-1", false), { wrapper });

    await new Promise((r) => setTimeout(r, 0));

    expect(api.getGatewayCapabilities).not.toHaveBeenCalled();
    expect(result.current.fetchStatus).toBe("idle");
  });

  it("refetch(true) forces a re-probe (refresh=1 passed to the api call)", async () => {
    const caps: GatewayCapabilities = { at: "now", endpoints: [] };
    vi.mocked(api.getGatewayCapabilities).mockResolvedValue(caps);

    const { result } = renderHook(() => useGatewayCapabilities("gw-1", true), { wrapper });
    await waitFor(() => expect(result.current.isSuccess).toBe(true));
    expect(api.getGatewayCapabilities).toHaveBeenLastCalledWith("gw-1", false);

    await result.current.refetch(true);

    expect(api.getGatewayCapabilities).toHaveBeenLastCalledWith("gw-1", true);
  });
});

describe("useGatewayAction", () => {
  it("invalidates gateways + health for the acted-on gid on success", async () => {
    vi.mocked(api.getGateways).mockResolvedValue({
      gateways: [makeGateway("gw-1")],
      targets: [],
      sources: [],
      presets: [],
    });
    vi.mocked(api.gatewayAction).mockResolvedValue({ status: STATUS });

    const queryClient = new QueryClient({
      defaultOptions: { queries: { retry: false }, mutations: { retry: false } },
    });
    function localWrapper({ children }: { children: ReactNode }) {
      return <QueryClientProvider client={queryClient}>{children}</QueryClientProvider>;
    }

    const gatewaysResult = renderHook(() => useGateways(), { wrapper: localWrapper });
    await waitFor(() => expect(gatewaysResult.result.current.isSuccess).toBe(true));
    expect(api.getGateways).toHaveBeenCalledTimes(1);

    const { result } = renderHook(() => useGatewayAction(), { wrapper: localWrapper });
    result.current.mutate({ gid: "gw-1", action: "restart" });

    await waitFor(() => expect(result.current.isSuccess).toBe(true));
    expect(api.gatewayAction).toHaveBeenCalledWith("gw-1", "restart");

    // A successful mutation invalidates ["gateways"], which refetches it.
    await waitFor(() => expect(api.getGateways).toHaveBeenCalledTimes(2));
  });
});

describe("usePutGatewayConfig", () => {
  it("invalidates gateways on success", async () => {
    vi.mocked(api.getGateways).mockResolvedValue({
      gateways: [makeGateway("gw-1")],
      targets: [],
      sources: [],
      presets: [],
    });
    const updated = makeGateway("gw-1");
    vi.mocked(api.putGatewayConfig).mockResolvedValue(updated);

    const queryClient = new QueryClient({
      defaultOptions: { queries: { retry: false }, mutations: { retry: false } },
    });
    function localWrapper({ children }: { children: ReactNode }) {
      return <QueryClientProvider client={queryClient}>{children}</QueryClientProvider>;
    }

    const gatewaysResult = renderHook(() => useGateways(), { wrapper: localWrapper });
    await waitFor(() => expect(gatewaysResult.result.current.isSuccess).toBe(true));
    expect(api.getGateways).toHaveBeenCalledTimes(1);

    const { result } = renderHook(() => usePutGatewayConfig(), { wrapper: localWrapper });
    result.current.mutate({ gid: "gw-1", config: updated.config });

    await waitFor(() => expect(result.current.isSuccess).toBe(true));
    await waitFor(() => expect(api.getGateways).toHaveBeenCalledTimes(2));
  });
});

describe("useWipeGateway", () => {
  it("resolves with a partial-failure error intact (not swallowed) and still invalidates gateways", async () => {
    vi.mocked(api.getGateways).mockResolvedValue({
      gateways: [makeGateway("gw-1")],
      targets: [],
      sources: [],
      presets: [],
    });
    const partial: WipeResult = {
      report: {
        ID: "gw-1",
        ContainerName: "erpc-gw-1",
        ContainerRemoved: true,
        VolumesRemoved: [],
        VolumesAbsent: [],
        Recreated: true,
        Cascaded: ["front-1"],
        CascadeSkipped: ["front-2"],
      },
      status: STATUS,
      error: "could not restart front-2",
    };
    vi.mocked(api.wipeGateway).mockResolvedValue(partial);

    const queryClient = new QueryClient({
      defaultOptions: { queries: { retry: false }, mutations: { retry: false } },
    });
    function localWrapper({ children }: { children: ReactNode }) {
      return <QueryClientProvider client={queryClient}>{children}</QueryClientProvider>;
    }

    const gatewaysResult = renderHook(() => useGateways(), { wrapper: localWrapper });
    await waitFor(() => expect(gatewaysResult.result.current.isSuccess).toBe(true));

    const { result } = renderHook(() => useWipeGateway(), { wrapper: localWrapper });
    result.current.mutate("gw-1");

    await waitFor(() => expect(result.current.isSuccess).toBe(true));
    expect(result.current.data?.error).toBe("could not restart front-2");
    await waitFor(() => expect(api.getGateways).toHaveBeenCalledTimes(2));
  });
});
