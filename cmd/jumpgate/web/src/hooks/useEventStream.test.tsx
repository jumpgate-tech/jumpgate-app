import { renderHook } from "@testing-library/react";
import { describe, it, expect, vi, beforeEach } from "vitest";
import type { SetupEvent } from "../api";
import * as api from "../api";
import { useEventStream } from "./useEventStream";

vi.mock("../api", () => ({
  streamSetup: vi.fn(),
}));

describe("useEventStream", () => {
  beforeEach(() => {
    vi.mocked(api.streamSetup).mockReset();
  });

  it("subscribes on mount when targetId is non-null", () => {
    const stop = vi.fn();
    vi.mocked(api.streamSetup).mockReturnValue(stop);
    const onEvent = vi.fn();

    renderHook(() => useEventStream("target-1", onEvent));

    expect(api.streamSetup).toHaveBeenCalledTimes(1);
    expect(api.streamSetup).toHaveBeenCalledWith("target-1", expect.any(Function));
    expect(stop).not.toHaveBeenCalled();
  });

  it("does not subscribe when targetId is null", () => {
    const onEvent = vi.fn();

    renderHook(() => useEventStream(null, onEvent));

    expect(api.streamSetup).not.toHaveBeenCalled();
  });

  it("tears down (calls stop) on unmount", () => {
    const stop = vi.fn();
    vi.mocked(api.streamSetup).mockReturnValue(stop);
    const onEvent = vi.fn();

    const { unmount } = renderHook(() => useEventStream("target-1", onEvent));
    expect(stop).not.toHaveBeenCalled();
    unmount();
    expect(stop).toHaveBeenCalledTimes(1);
  });

  it("re-subscribes (stops the old stream, opens a new one) when targetId changes", () => {
    const stopA = vi.fn();
    const stopB = vi.fn();
    vi.mocked(api.streamSetup).mockReturnValueOnce(stopA).mockReturnValueOnce(stopB);
    const onEvent = vi.fn();

    const { rerender } = renderHook<void, { targetId: string }>(
      ({ targetId }) => useEventStream(targetId, onEvent),
      { initialProps: { targetId: "target-1" } },
    );
    expect(api.streamSetup).toHaveBeenCalledTimes(1);
    expect(api.streamSetup).toHaveBeenLastCalledWith("target-1", expect.any(Function));

    rerender({ targetId: "target-2" });

    expect(stopA).toHaveBeenCalledTimes(1);
    expect(api.streamSetup).toHaveBeenCalledTimes(2);
    expect(api.streamSetup).toHaveBeenLastCalledWith("target-2", expect.any(Function));
    expect(stopB).not.toHaveBeenCalled();
  });

  it("does NOT re-subscribe when only the onEvent identity changes", () => {
    const stop = vi.fn();
    vi.mocked(api.streamSetup).mockReturnValue(stop);

    const { rerender } = renderHook<void, { onEvent: (ev: SetupEvent) => void }>(
      ({ onEvent }) => useEventStream("target-1", onEvent),
      { initialProps: { onEvent: vi.fn() } },
    );
    expect(api.streamSetup).toHaveBeenCalledTimes(1);

    rerender({ onEvent: vi.fn() });

    expect(api.streamSetup).toHaveBeenCalledTimes(1);
    expect(stop).not.toHaveBeenCalled();
  });

  it("calls the CURRENT onEvent even after its identity changed (kept in a ref)", () => {
    const captured: { fn: ((ev: SetupEvent) => void) | null } = { fn: null };
    vi.mocked(api.streamSetup).mockImplementation((_id: string, onEvent: (ev: SetupEvent) => void) => {
      captured.fn = onEvent;
      return vi.fn();
    });

    const onEventA = vi.fn();
    const onEventB = vi.fn();
    const { rerender } = renderHook<void, { onEvent: (ev: SetupEvent) => void }>(
      ({ onEvent }) => useEventStream("target-1", onEvent),
      { initialProps: { onEvent: onEventA } },
    );

    rerender({ onEvent: onEventB });

    const ev: SetupEvent = { stepId: "run", done: true };
    captured.fn?.(ev);

    expect(onEventA).not.toHaveBeenCalled();
    expect(onEventB).toHaveBeenCalledWith(ev);
  });
});
