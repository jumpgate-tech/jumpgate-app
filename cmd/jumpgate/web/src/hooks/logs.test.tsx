import { renderHook, waitFor } from "@testing-library/react";
import { describe, it, expect, vi, beforeEach } from "vitest";
import type { Hit } from "../api";
import * as api from "../api";
import { maxRenderedLines } from "../screens/Machine/logsModel";
import { useLogStream, type UseLogStreamResult } from "./logs";

vi.mock("../api", () => ({
  getLogs: vi.fn(),
  streamLogs: vi.fn(),
}));

function hit(overrides: Partial<Hit> = {}): Hit {
  return {
    unit: "exec",
    line: "boom",
    at: "2026-08-01T12:00:00Z",
    signature: "sig",
    severity: "info",
    explain: "",
    ...overrides,
  };
}

describe("useLogStream", () => {
  beforeEach(() => {
    vi.mocked(api.getLogs).mockReset();
    vi.mocked(api.streamLogs).mockReset();
  });

  it("does not fetch or stream when disabled", () => {
    renderHook(() => useLogStream("t1", false));

    expect(api.getLogs).not.toHaveBeenCalled();
    expect(api.streamLogs).not.toHaveBeenCalled();
  });

  it("is loading, then fetches recent hits and opens the stream", async () => {
    vi.mocked(api.getLogs).mockResolvedValue([hit({ line: "a" }), hit({ line: "b" })]);
    const stop = vi.fn();
    vi.mocked(api.streamLogs).mockReturnValue(stop);

    const { result } = renderHook(() => useLogStream("t1", true));

    expect(result.current.loading).toBe(true);
    await waitFor(() => expect(result.current.loading).toBe(false));

    expect(api.getLogs).toHaveBeenCalledWith("t1", 200);
    expect(result.current.hits.map((h) => h.line)).toEqual(["a", "b"]);
    expect(result.current.error).toBeNull();
    expect(api.streamLogs).toHaveBeenCalledWith("t1", expect.any(Function));
  });

  it("surfaces a getLogs failure as error and never opens the stream", async () => {
    vi.mocked(api.getLogs).mockRejectedValue(new Error("network down"));

    const { result } = renderHook(() => useLogStream("t1", true));

    await waitFor(() => expect(result.current.loading).toBe(false));
    expect(result.current.error).toBe("network down");
    expect(result.current.hits).toEqual([]);
    expect(api.streamLogs).not.toHaveBeenCalled();
  });

  it("appends live hits from the stream onto the fetched seed", async () => {
    vi.mocked(api.getLogs).mockResolvedValue([hit({ line: "seed" })]);
    let onHit: ((h: Hit) => void) | null = null;
    vi.mocked(api.streamLogs).mockImplementation((_id, cb) => {
      onHit = cb;
      return vi.fn();
    });

    const { result } = renderHook(() => useLogStream("t1", true));
    await waitFor(() => expect(result.current.loading).toBe(false));

    onHit!(hit({ line: "live" }));

    await waitFor(() => expect(result.current.hits.map((h) => h.line)).toEqual(["seed", "live"]));
  });

  it("caps the hit list at maxRenderedLines as the stream keeps appending", async () => {
    vi.mocked(api.getLogs).mockResolvedValue([]);
    let onHit: ((h: Hit) => void) | null = null;
    vi.mocked(api.streamLogs).mockImplementation((_id, cb) => {
      onHit = cb;
      return vi.fn();
    });

    const { result } = renderHook(() => useLogStream("t1", true));
    await waitFor(() => expect(result.current.loading).toBe(false));

    for (let i = 0; i < maxRenderedLines + 20; i++) {
      onHit!(hit({ line: `l${i}` }));
    }

    await waitFor(() => expect(result.current.hits).toHaveLength(maxRenderedLines));
    expect(result.current.hits[0].line).toBe("l20");
    expect(result.current.hits[result.current.hits.length - 1].line).toBe(`l${maxRenderedLines + 19}`);
  });

  it("assigns a stable, unique _key to every row — even when signatures collide — that survives the front-splice", async () => {
    // Both seed rows share the default signature "sig": signature is a
    // classification pattern name, NOT a per-line id, so it cannot key a list.
    vi.mocked(api.getLogs).mockResolvedValue([hit({ line: "a" }), hit({ line: "b" })]);
    let onHit: ((h: Hit) => void) | null = null;
    vi.mocked(api.streamLogs).mockImplementation((_id, cb) => {
      onHit = cb;
      return vi.fn();
    });

    const { result } = renderHook(() => useLogStream("t1", true));
    await waitFor(() => expect(result.current.hits).toHaveLength(2));

    // Colliding signatures still get distinct keys.
    expect(new Set(result.current.hits.map((h) => h._key)).size).toBe(2);
    const bKey = result.current.hits[1]._key;

    // Append past the cap so the front ("a") is spliced off; "b" stays in view.
    for (let i = 0; i < maxRenderedLines - 1; i++) onHit!(hit({ line: `l${i}` }));
    await waitFor(() => expect(result.current.hits).toHaveLength(maxRenderedLines));

    // "b" survived the splice and kept the SAME _key — an array index would
    // have shifted from 1 to 0. Every rendered key is still unique.
    expect(result.current.hits.find((h) => h.line === "b")!._key).toBe(bKey);
    expect(new Set(result.current.hits.map((h) => h._key)).size).toBe(maxRenderedLines);
  });

  it("tears down (calls stop) on unmount", async () => {
    vi.mocked(api.getLogs).mockResolvedValue([]);
    const stop = vi.fn();
    vi.mocked(api.streamLogs).mockReturnValue(stop);

    const { result, unmount } = renderHook(() => useLogStream("t1", true));
    await waitFor(() => expect(result.current.loading).toBe(false));

    unmount();
    expect(stop).toHaveBeenCalledTimes(1);
  });

  it("stops the old stream and refetches when targetId changes", async () => {
    vi.mocked(api.getLogs).mockResolvedValue([hit({ line: "a" })]);
    const stopA = vi.fn();
    const stopB = vi.fn();
    vi.mocked(api.streamLogs).mockReturnValueOnce(stopA).mockReturnValueOnce(stopB);

    const { result, rerender } = renderHook<UseLogStreamResult, { targetId: string }>(
      ({ targetId }) => useLogStream(targetId, true),
      { initialProps: { targetId: "t1" } },
    );
    await waitFor(() => expect(result.current.loading).toBe(false));
    expect(api.getLogs).toHaveBeenCalledWith("t1", 200);

    rerender({ targetId: "t2" });

    expect(stopA).toHaveBeenCalledTimes(1);
    await waitFor(() => expect(api.getLogs).toHaveBeenCalledWith("t2", 200));
    expect(stopB).not.toHaveBeenCalled();
  });

  it("resets to empty when disabled after having loaded", async () => {
    vi.mocked(api.getLogs).mockResolvedValue([hit({ line: "a" })]);
    const stop = vi.fn();
    vi.mocked(api.streamLogs).mockReturnValue(stop);

    const { result, rerender } = renderHook<UseLogStreamResult, { enabled: boolean }>(
      ({ enabled }) => useLogStream("t1", enabled),
      { initialProps: { enabled: true } },
    );
    await waitFor(() => expect(result.current.hits).toHaveLength(1));

    rerender({ enabled: false });

    expect(stop).toHaveBeenCalledTimes(1);
    expect(result.current.hits).toEqual([]);
    expect(result.current.loading).toBe(false);
  });
});
