import { act, renderHook, waitFor } from "@testing-library/react";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { describe, it, expect, vi, beforeEach } from "vitest";
import type { ReactNode } from "react";
import * as api from "../api";
import type { DiskUsage, EndpointInfo, Snapshot } from "../api";
import { useClearService, useDiskUsage, useEndpoints, useMonitorStream, useServiceActions } from "./dashboard";

vi.mock("../api", async () => {
  const actual = await vi.importActual<typeof import("../api")>("../api");
  return {
    ...actual,
    streamMonitor: vi.fn(),
    getDiskUsage: vi.fn(),
    getEndpoints: vi.fn(),
    serviceAction: vi.fn(),
    clearService: vi.fn(),
  };
});

function snap(overrides: Partial<Snapshot> = {}): Snapshot {
  return {
    at: "2026-08-01T12:00:00.000Z",
    execSyncing: false,
    execHead: 100,
    refHead: 100,
    beaconSlot: 100,
    beaconDistance: 0,
    execPeers: 5,
    beaconPeers: 5,
    diskUsedPct: 10,
    execActive: true,
    beaconActive: true,
    ...overrides,
  };
}

function wrapper({ children }: { children: ReactNode }) {
  const queryClient = new QueryClient({
    defaultOptions: { queries: { retry: false }, mutations: { retry: false } },
  });
  return <QueryClientProvider client={queryClient}>{children}</QueryClientProvider>;
}

beforeEach(() => {
  vi.mocked(api.streamMonitor).mockReset();
  vi.mocked(api.getDiskUsage).mockReset();
  vi.mocked(api.getEndpoints).mockReset();
  vi.mocked(api.serviceAction).mockReset();
  vi.mocked(api.clearService).mockReset();
});

describe("useMonitorStream", () => {
  it("does not open a stream when disabled", () => {
    renderHook(() => useMonitorStream("t1", false));
    expect(api.streamMonitor).not.toHaveBeenCalled();
  });

  it("opens the stream when enabled and surfaces snapshots as they arrive", () => {
    const stop = vi.fn();
    let onSnapshot: ((s: Snapshot) => void) | null = null;
    vi.mocked(api.streamMonitor).mockImplementation((_id, cb) => {
      onSnapshot = cb;
      return stop;
    });

    const { result } = renderHook(() => useMonitorStream("t1", true));
    expect(api.streamMonitor).toHaveBeenCalledWith("t1", expect.any(Function));
    expect(result.current.snapshot).toBeNull();

    const s1 = snap({ execHead: 100 });
    act(() => onSnapshot!(s1));
    expect(result.current.snapshot).toEqual(s1);
    // First tick has no prior snapshot to diff against, so the rate stays null.
    expect(result.current.execBlocksPerSec).toBeNull();
  });

  it("smooths a blocks/sec rate across ticks", () => {
    let onSnapshot: ((s: Snapshot) => void) | null = null;
    vi.mocked(api.streamMonitor).mockImplementation((_id, cb) => {
      onSnapshot = cb;
      return vi.fn();
    });

    const { result } = renderHook(() => useMonitorStream("t1", true));

    act(() => onSnapshot!(snap({ at: "2026-08-01T12:00:00.000Z", execHead: 100 })));
    act(() => onSnapshot!(snap({ at: "2026-08-01T12:00:10.000Z", execHead: 150 })));

    expect(result.current.execBlocksPerSec).toBe(5);
  });

  it("tears down the stream on unmount", () => {
    const stop = vi.fn();
    vi.mocked(api.streamMonitor).mockReturnValue(stop);

    const { unmount } = renderHook(() => useMonitorStream("t1", true));
    unmount();

    expect(stop).toHaveBeenCalledTimes(1);
  });

  it("closes the old stream and resets state when disabled after having data", () => {
    const stop = vi.fn();
    let onSnapshot: ((s: Snapshot) => void) | null = null;
    vi.mocked(api.streamMonitor).mockImplementation((_id, cb) => {
      onSnapshot = cb;
      return stop;
    });

    const { result, rerender } = renderHook(({ enabled }) => useMonitorStream("t1", enabled), {
      initialProps: { enabled: true },
    });
    act(() => onSnapshot!(snap()));
    expect(result.current.snapshot).not.toBeNull();

    rerender({ enabled: false });

    expect(stop).toHaveBeenCalledTimes(1);
    expect(result.current.snapshot).toBeNull();
    expect(result.current.execBlocksPerSec).toBeNull();
  });
});

describe("useDiskUsage", () => {
  it("fetches disk usage when enabled", async () => {
    const du: DiskUsage = {
      ExecBytes: 1,
      BeaconBytes: 2,
      DiskFreeBytes: 3,
      ExpectedExecBytes: 4,
      ExpectedBeaconBytes: 5,
      SyncLabel: "fast",
      GenesisSyncLabel: "slow",
    };
    vi.mocked(api.getDiskUsage).mockResolvedValue(du);

    const { result } = renderHook(() => useDiskUsage("t1", true), { wrapper });

    await waitFor(() => expect(result.current.isSuccess).toBe(true));
    expect(result.current.data).toEqual(du);
    expect(api.getDiskUsage).toHaveBeenCalledWith("t1");
  });

  it("does not fetch when disabled", () => {
    renderHook(() => useDiskUsage("t1", false), { wrapper });
    expect(api.getDiskUsage).not.toHaveBeenCalled();
  });

  it("surfaces a rejection as query error without retrying", async () => {
    vi.mocked(api.getDiskUsage).mockRejectedValue(new Error("du down"));

    const { result } = renderHook(() => useDiskUsage("t1", true), { wrapper });

    await waitFor(() => expect(result.current.isError).toBe(true));
    expect(api.getDiskUsage).toHaveBeenCalledTimes(1);
  });
});

describe("useEndpoints", () => {
  it("fetches endpoints when enabled", async () => {
    const ep: EndpointInfo = {
      ExecHTTP: "http://127.0.0.1:8545",
      BeaconHTTP: "http://127.0.0.1:5052",
      ExecReachable: true,
      BeaconReachable: true,
      ChainIDMatches: true,
      Access: "local",
      TunnelHint: "",
    };
    vi.mocked(api.getEndpoints).mockResolvedValue(ep);

    const { result } = renderHook(() => useEndpoints("t1", true), { wrapper });

    await waitFor(() => expect(result.current.isSuccess).toBe(true));
    expect(result.current.data).toEqual(ep);
  });

  it("does not fetch when disabled", () => {
    renderHook(() => useEndpoints("t1", false), { wrapper });
    expect(api.getEndpoints).not.toHaveBeenCalled();
  });
});

describe("useServiceActions", () => {
  it("runs an action, tracking it as pending until it resolves", async () => {
    let resolveAction: (() => void) | null = null;
    vi.mocked(api.serviceAction).mockReturnValue(
      new Promise((resolve) => {
        resolveAction = () => resolve({ Active: true });
      }),
    );

    const { result } = renderHook(() => useServiceActions("t1"));

    let runPromise!: Promise<void>;
    await waitFor(() => {
      runPromise = result.current.run("exec", "start");
    });
    await waitFor(() => expect(result.current.pending.exec).toBe("start"));
    expect(result.current.pending.beacon).toBeNull();

    resolveAction!();
    await runPromise;
    await waitFor(() => expect(result.current.pending.exec).toBeNull());
    expect(result.current.error).toBeNull();
    expect(api.serviceAction).toHaveBeenCalledWith("t1", "exec", "start");
  });

  it("ignores a second call for a service already in flight", async () => {
    let resolveAction: (() => void) | null = null;
    vi.mocked(api.serviceAction).mockReturnValue(
      new Promise((resolve) => {
        resolveAction = () => resolve({ Active: true });
      }),
    );

    const { result } = renderHook(() => useServiceActions("t1"));

    let first!: Promise<void>;
    await waitFor(() => {
      first = result.current.run("exec", "start");
    });
    await waitFor(() => expect(result.current.pending.exec).toBe("start"));

    const second = result.current.run("exec", "restart");

    resolveAction!();
    await Promise.all([first, second]);
    expect(api.serviceAction).toHaveBeenCalledTimes(1);
    expect(api.serviceAction).toHaveBeenCalledWith("t1", "exec", "start");
  });

  it("surfaces a failed action as a formatted error and clears pending", async () => {
    vi.mocked(api.serviceAction).mockRejectedValue(new Error("connection refused"));

    const { result } = renderHook(() => useServiceActions("t1"));

    await waitFor(() => result.current.run("beacon", "stop"));

    await waitFor(() =>
      expect(result.current.error).toBe("Beacon stop failed: connection refused"),
    );
    expect(result.current.pending.beacon).toBeNull();
  });
});

describe("useClearService", () => {
  it("calls clearService and invalidates the disk-usage query on success", async () => {
    vi.mocked(api.clearService).mockResolvedValue({ status: "ok" });
    vi.mocked(api.getDiskUsage).mockResolvedValue({
      ExecBytes: 0,
      BeaconBytes: 0,
      DiskFreeBytes: 0,
      ExpectedExecBytes: 0,
      ExpectedBeaconBytes: 0,
      SyncLabel: "",
      GenesisSyncLabel: "",
    });

    const queryClient = new QueryClient({
      defaultOptions: { queries: { retry: false }, mutations: { retry: false } },
    });
    function localWrapper({ children }: { children: ReactNode }) {
      return <QueryClientProvider client={queryClient}>{children}</QueryClientProvider>;
    }

    const duResult = renderHook(() => useDiskUsage("t1", true), { wrapper: localWrapper });
    await waitFor(() => expect(duResult.result.current.isSuccess).toBe(true));
    expect(api.getDiskUsage).toHaveBeenCalledTimes(1);

    const { result } = renderHook(() => useClearService("t1"), { wrapper: localWrapper });
    result.current.mutate("exec");

    await waitFor(() => expect(result.current.isSuccess).toBe(true));
    expect(api.clearService).toHaveBeenCalledWith("t1", "exec");
    await waitFor(() => expect(api.getDiskUsage).toHaveBeenCalledTimes(2));
  });

  it("surfaces a rejection as mutation error", async () => {
    vi.mocked(api.clearService).mockRejectedValue(new Error("clear failed"));

    const { result } = renderHook(() => useClearService("t1"), { wrapper });
    result.current.mutate("exec");

    await waitFor(() => expect(result.current.isError).toBe(true));
    expect(result.current.error).toBeInstanceOf(Error);
  });
});
