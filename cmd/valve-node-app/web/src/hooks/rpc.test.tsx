import { renderHook, act, waitFor } from "@testing-library/react";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { describe, it, expect, vi, beforeEach } from "vitest";
import type { ReactNode } from "react";
import type { SetupEvent } from "../api";
import * as api from "../api";
import { useGatewayOps, useTrustCert } from "./rpc";

vi.mock("../api", async () => {
  const actual = await vi.importActual<typeof import("../api")>("../api");
  return {
    ...actual,
    gatewayAction: vi.fn(),
    provisionGateway: vi.fn(),
    streamSetup: vi.fn(),
    trustGatewayCert: vi.fn(),
  };
});

function wrapper({ children }: { children: ReactNode }) {
  const qc = new QueryClient({ defaultOptions: { queries: { retry: false }, mutations: { retry: false } } });
  return <QueryClientProvider client={qc}>{children}</QueryClientProvider>;
}

beforeEach(() => {
  vi.mocked(api.gatewayAction).mockReset();
  vi.mocked(api.provisionGateway).mockReset();
  vi.mocked(api.streamSetup).mockReset();
  vi.mocked(api.trustGatewayCert).mockReset();
});

describe("useGatewayOps.runAction", () => {
  it("fires the action and clears busy", async () => {
    vi.mocked(api.gatewayAction).mockResolvedValue({ status: {} as api.ContainerStatus });
    const { result } = renderHook(() => useGatewayOps("gw-1"), { wrapper });
    await act(async () => {
      await result.current.runAction("restart");
    });
    expect(api.gatewayAction).toHaveBeenCalledWith("gw-1", "restart");
    expect(result.current.busy).toBeNull();
  });

  it("records the error on failure", async () => {
    vi.mocked(api.gatewayAction).mockRejectedValue(new Error("engine down"));
    const { result } = renderHook(() => useGatewayOps("gw-1"), { wrapper });
    await act(async () => {
      await result.current.runAction("start");
    });
    expect(result.current.actionErr).toContain("start failed: engine down");
  });
});

describe("useGatewayOps.provision", () => {
  function wireStream(): { push: (ev: SetupEvent) => void; stop: ReturnType<typeof vi.fn> } {
    const stop = vi.fn();
    let onEvent: ((ev: SetupEvent) => void) | null = null;
    vi.mocked(api.streamSetup).mockImplementation((_id, cb) => {
      onEvent = cb;
      return stop;
    });
    return { push: (ev) => act(() => onEvent!(ev)), stop };
  }

  it("follows the placement machine's stream and accumulates lines", async () => {
    vi.mocked(api.provisionGateway).mockResolvedValue({ status: "accepted", targetId: "local" });
    const { push } = wireStream();
    const { result } = renderHook(() => useGatewayOps("gw-1"), { wrapper });
    await act(async () => {
      await result.current.provision();
    });
    expect(api.provisionGateway).toHaveBeenCalledWith("gw-1");
    expect(api.streamSetup).toHaveBeenCalledWith("local", expect.any(Function));
    expect(result.current.busy).toBe("create");
    push({ stepId: "preflight", line: "checking" });
    expect(result.current.activity).toContain("preflight: checking");
    push({ stepId: "run", done: true });
    await waitFor(() => expect(result.current.busy).toBeNull());
  });

  it("resets without a stream when the POST is rejected (e.g. 409)", async () => {
    vi.mocked(api.provisionGateway).mockRejectedValue(new Error("already provisioning"));
    const { result } = renderHook(() => useGatewayOps("gw-1"), { wrapper });
    await act(async () => {
      await result.current.provision();
    });
    expect(result.current.busy).toBeNull();
    expect(result.current.actionErr).toContain("already provisioning");
    expect(api.streamSetup).not.toHaveBeenCalled();
  });

  it("surfaces a stream error and stops", async () => {
    vi.mocked(api.provisionGateway).mockResolvedValue({ status: "accepted", targetId: "local" });
    const { push, stop } = wireStream();
    const { result } = renderHook(() => useGatewayOps("gw-1"), { wrapper });
    await act(async () => {
      await result.current.provision();
    });
    push({ stepId: "config", err: "boom" });
    await waitFor(() => expect(result.current.busy).toBeNull());
    expect(result.current.actionErr).toContain("Provisioning failed");
    expect(stop).toHaveBeenCalled();
  });
});

describe("useTrustCert", () => {
  it("resolves a thrown error into a { ok:false } result carrying the message", async () => {
    vi.mocked(api.trustGatewayCert).mockRejectedValue(new Error("needs sudo"));
    const { result } = renderHook(() => useTrustCert(), { wrapper });
    let out: api.TrustCertResult | undefined;
    await act(async () => {
      out = await result.current.mutateAsync("gw-1");
    });
    expect(out).toEqual({ ok: false, message: "needs sudo" });
  });
});
