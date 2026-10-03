import { act, renderHook, waitFor } from "@testing-library/react";
import { describe, it, expect, vi, beforeEach } from "vitest";
import * as api from "../api";
import type { SetupEvent, StartSetupRequest } from "../api";
import { useDiskProbe, useSetupRun } from "./wizard";

vi.mock("../api", async () => {
  const actual = await vi.importActual<typeof import("../api")>("../api");
  return {
    ...actual,
    getDiskFree: vi.fn(),
    startSetup: vi.fn(),
    streamSetup: vi.fn(),
  };
});

beforeEach(() => {
  vi.mocked(api.getDiskFree).mockReset();
  vi.mocked(api.startSetup).mockReset();
  vi.mocked(api.streamSetup).mockReset();
});

const WIRE: StartSetupRequest = { ChainID: 369, ExecID: "reth", BeaconID: "lighthouse-pulse", Archive: true };

describe("useDiskProbe", () => {
  it("starts idle", () => {
    const { result } = renderHook(() => useDiskProbe("t1"));
    expect(result.current.freeBytes).toBeNull();
    expect(result.current.probedPath).toBeNull();
    expect(result.current.probing).toBe(false);
    expect(result.current.error).toBeNull();
  });

  it("probes a path, tracking probing until it resolves, and returns the outcome", async () => {
    let resolveFn: ((v: { path: string; freeBytes: number }) => void) | null = null;
    vi.mocked(api.getDiskFree).mockReturnValue(
      new Promise((resolve) => {
        resolveFn = resolve;
      }),
    );

    const { result } = renderHook(() => useDiskProbe("t1"));

    let outcome!: Promise<unknown>;
    act(() => {
      outcome = result.current.probe("/data");
    });
    expect(result.current.probing).toBe(true);
    expect(api.getDiskFree).toHaveBeenCalledWith("t1", "/data");

    await act(async () => {
      resolveFn!({ path: "/data", freeBytes: 5_000_000_000 });
      await outcome;
    });

    expect(result.current.probing).toBe(false);
    expect(result.current.freeBytes).toBe(5_000_000_000);
    expect(result.current.probedPath).toBe("/data");
    expect(result.current.error).toBeNull();
    await expect(outcome).resolves.toEqual({ freeBytes: 5_000_000_000 });
  });

  it("surfaces a rejection as an error, clears freeBytes, and still records the probed path", async () => {
    vi.mocked(api.getDiskFree).mockRejectedValue(new Error("permission denied"));

    const { result } = renderHook(() => useDiskProbe("t1"));

    let outcome!: Promise<unknown>;
    await act(async () => {
      outcome = result.current.probe("/no-access");
    });

    expect(result.current.probing).toBe(false);
    expect(result.current.freeBytes).toBeNull();
    expect(result.current.probedPath).toBe("/no-access");
    expect(result.current.error).toBe("permission denied");
    await expect(outcome).resolves.toEqual({ error: "permission denied" });
  });

  it("a later probe overwrites an earlier one's result", async () => {
    vi.mocked(api.getDiskFree)
      .mockResolvedValueOnce({ path: "/a", freeBytes: 1 })
      .mockResolvedValueOnce({ path: "/b", freeBytes: 2 });

    const { result } = renderHook(() => useDiskProbe("t1"));
    await act(async () => {
      await result.current.probe("/a");
    });
    await act(async () => {
      await result.current.probe("/b");
    });

    expect(result.current.freeBytes).toBe(2);
    expect(result.current.probedPath).toBe("/b");
  });
});

describe("useSetupRun", () => {
  function wireStream(): { push: (ev: SetupEvent) => void; stop: ReturnType<typeof vi.fn> } {
    const stop = vi.fn();
    let onEvent: ((ev: SetupEvent) => void) | null = null;
    vi.mocked(api.streamSetup).mockImplementation((_id, cb) => {
      onEvent = cb;
      return stop;
    });
    return { push: (ev: SetupEvent) => act(() => onEvent!(ev)), stop };
  }

  it("starts idle with no events", () => {
    const { result } = renderHook(() => useSetupRun("t1"));
    expect(result.current.events).toEqual([]);
    expect(result.current.starting).toBe(false);
    expect(result.current.startError).toBeNull();
  });

  it("posts, then opens the stream and accumulates events as they arrive", async () => {
    vi.mocked(api.startSetup).mockResolvedValue({ status: "accepted" });
    const stream = wireStream();

    const { result } = renderHook(() => useSetupRun("t1"));
    await act(async () => {
      await result.current.start(WIRE);
    });

    expect(api.startSetup).toHaveBeenCalledWith("t1", WIRE);
    expect(api.streamSetup).toHaveBeenCalledWith("t1", expect.any(Function));
    expect(result.current.starting).toBe(false);

    stream.push({ stepId: "preflight", line: "checking docker" });
    expect(result.current.events).toEqual([{ stepId: "preflight", line: "checking docker" }]);

    stream.push({ stepId: "handshake", done: true });
    expect(result.current.events).toHaveLength(2);
  });

  it("never closes the stream on its own once the terminal step reports done", async () => {
    vi.mocked(api.startSetup).mockResolvedValue({ status: "accepted" });
    const stream = wireStream();

    const { result } = renderHook(() => useSetupRun("t1"));
    await act(async () => {
      await result.current.start(WIRE);
    });
    stream.push({ stepId: "handshake", done: true });

    expect(stream.stop).not.toHaveBeenCalled();
  });

  it("treats a 409 (already provisioning) as success and attaches to the live stream", async () => {
    vi.mocked(api.startSetup).mockRejectedValue(new api.ApiError(409, "already running"));
    wireStream();

    const { result } = renderHook(() => useSetupRun("t1"));
    await act(async () => {
      await result.current.start(WIRE);
    });

    expect(result.current.starting).toBe(false);
    expect(result.current.startError).toBeNull();
    expect(api.streamSetup).toHaveBeenCalledWith("t1", expect.any(Function));
  });

  it("surfaces a non-409 POST failure without opening a stream", async () => {
    vi.mocked(api.startSetup).mockRejectedValue(new Error("network down"));

    const { result } = renderHook(() => useSetupRun("t1"));
    await act(async () => {
      await result.current.start(WIRE);
    });

    expect(result.current.starting).toBe(false);
    expect(result.current.startError).toBe("network down");
    expect(api.streamSetup).not.toHaveBeenCalled();
  });

  it("a fresh start() closes the previous stream and clears prior events", async () => {
    vi.mocked(api.startSetup).mockResolvedValue({ status: "accepted" });
    const first = wireStream();

    const { result } = renderHook(() => useSetupRun("t1"));
    await act(async () => {
      await result.current.start(WIRE);
    });
    first.push({ stepId: "preflight", err: "boom" });
    expect(result.current.events).toHaveLength(1);

    const second = wireStream();
    await act(async () => {
      await result.current.start(WIRE);
    });

    expect(first.stop).toHaveBeenCalledTimes(1);
    expect(result.current.events).toEqual([]);
    second.push({ stepId: "preflight", line: "retrying" });
    expect(result.current.events).toEqual([{ stepId: "preflight", line: "retrying" }]);
  });

  it("ignores a re-entrant start() call while one is already starting", async () => {
    let resolvePost: (() => void) | null = null;
    vi.mocked(api.startSetup).mockReturnValue(
      new Promise((resolve) => {
        resolvePost = () => resolve({ status: "accepted" });
      }),
    );
    wireStream();

    const { result } = renderHook(() => useSetupRun("t1"));
    let firstCall!: Promise<boolean>;
    act(() => {
      firstCall = result.current.start(WIRE);
    });
    expect(result.current.starting).toBe(true);

    const secondCall = result.current.start(WIRE);

    await act(async () => {
      resolvePost!();
      await Promise.all([firstCall, secondCall]);
    });

    expect(api.startSetup).toHaveBeenCalledTimes(1);
  });

  it("tears down an open stream on unmount", async () => {
    vi.mocked(api.startSetup).mockResolvedValue({ status: "accepted" });
    const stream = wireStream();

    const { result, unmount } = renderHook(() => useSetupRun("t1"));
    await act(async () => {
      await result.current.start(WIRE);
    });

    unmount();

    expect(stream.stop).toHaveBeenCalledTimes(1);
  });

  it("waitFor also confirms starting flips back to false after a successful post", async () => {
    vi.mocked(api.startSetup).mockResolvedValue({ status: "accepted" });
    wireStream();

    const { result } = renderHook(() => useSetupRun("t1"));
    act(() => {
      void result.current.start(WIRE);
    });
    await waitFor(() => expect(result.current.starting).toBe(false));
  });
});
