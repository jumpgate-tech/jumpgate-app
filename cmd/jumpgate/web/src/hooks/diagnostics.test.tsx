import { renderHook, waitFor, act } from "@testing-library/react";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { describe, it, expect, vi, beforeEach } from "vitest";
import type { ReactNode } from "react";
import * as api from "../api";
import type { DiagReport } from "../api";
import { useLatestDiagnostics, useRunDiagnostics, diagnosticsQueryKey } from "./diagnostics";

vi.mock("../api", async () => {
  const actual = await vi.importActual<typeof import("../api")>("../api");
  return {
    ...actual,
    getLatestDiagnostics: vi.fn(),
    runNetworkDiagnostics: vi.fn(),
  };
});

function makeWrapper(queryClient: QueryClient) {
  return function wrapper({ children }: { children: ReactNode }) {
    return <QueryClientProvider client={queryClient}>{children}</QueryClientProvider>;
  };
}

const REPORT: DiagReport = {
  at: "2026-08-01T00:00:00Z",
  trigger: "manual",
  items: [{ ID: "c1", Title: "DNS resolves", Why: "needed", Status: "pass", Detail: "ok", Fix: "" }],
};

beforeEach(() => {
  vi.mocked(api.getLatestDiagnostics).mockReset();
  vi.mocked(api.runNetworkDiagnostics).mockReset();
});

describe("useLatestDiagnostics", () => {
  it("fetches the latest report for the target when enabled", async () => {
    vi.mocked(api.getLatestDiagnostics).mockResolvedValue(REPORT);
    const queryClient = new QueryClient({ defaultOptions: { queries: { retry: false } } });

    const { result } = renderHook(() => useLatestDiagnostics("t1", true), {
      wrapper: makeWrapper(queryClient),
    });

    await waitFor(() => expect(result.current.isSuccess).toBe(true));
    expect(api.getLatestDiagnostics).toHaveBeenCalledWith("t1");
    expect(result.current.data).toEqual(REPORT);
  });

  it("resolves to null when no diagnostics have run yet", async () => {
    vi.mocked(api.getLatestDiagnostics).mockResolvedValue(null);
    const queryClient = new QueryClient({ defaultOptions: { queries: { retry: false } } });

    const { result } = renderHook(() => useLatestDiagnostics("t1", true), {
      wrapper: makeWrapper(queryClient),
    });

    await waitFor(() => expect(result.current.isSuccess).toBe(true));
    expect(result.current.data).toBeNull();
  });

  it("is disabled (does not call the api) when enabled=false", async () => {
    const queryClient = new QueryClient({ defaultOptions: { queries: { retry: false } } });

    const { result } = renderHook(() => useLatestDiagnostics("t1", false), {
      wrapper: makeWrapper(queryClient),
    });

    await new Promise((r) => setTimeout(r, 0));

    expect(api.getLatestDiagnostics).not.toHaveBeenCalled();
    expect(result.current.fetchStatus).toBe("idle");
  });

  it("is disabled when the target id is empty", async () => {
    const queryClient = new QueryClient({ defaultOptions: { queries: { retry: false } } });

    const { result } = renderHook(() => useLatestDiagnostics("", true), {
      wrapper: makeWrapper(queryClient),
    });

    await new Promise((r) => setTimeout(r, 0));

    expect(api.getLatestDiagnostics).not.toHaveBeenCalled();
    expect(result.current.fetchStatus).toBe("idle");
  });
});

describe("useRunDiagnostics", () => {
  it("runs diagnostics and writes the result into the latest-report query cache", async () => {
    vi.mocked(api.getLatestDiagnostics).mockResolvedValue(null);
    vi.mocked(api.runNetworkDiagnostics).mockResolvedValue(REPORT);
    const queryClient = new QueryClient({
      defaultOptions: { queries: { retry: false }, mutations: { retry: false } },
    });
    const wrapper = makeWrapper(queryClient);

    const latest = renderHook(() => useLatestDiagnostics("t1", true), { wrapper });
    await waitFor(() => expect(latest.result.current.isSuccess).toBe(true));
    expect(latest.result.current.data).toBeNull();

    const run = renderHook(() => useRunDiagnostics("t1"), { wrapper });
    await act(async () => {
      await run.result.current.mutateAsync();
    });

    expect(api.runNetworkDiagnostics).toHaveBeenCalledWith("t1");
    expect(queryClient.getQueryData(diagnosticsQueryKey("t1"))).toEqual(REPORT);
  });

  it("surfaces a run failure without touching the cached report", async () => {
    vi.mocked(api.runNetworkDiagnostics).mockRejectedValue(new Error("probe timed out"));
    const queryClient = new QueryClient({
      defaultOptions: { queries: { retry: false }, mutations: { retry: false } },
    });
    queryClient.setQueryData(diagnosticsQueryKey("t1"), REPORT);
    const wrapper = makeWrapper(queryClient);

    const run = renderHook(() => useRunDiagnostics("t1"), { wrapper });
    await act(async () => {
      await expect(run.result.current.mutateAsync()).rejects.toThrow("probe timed out");
    });

    expect(run.result.current.isError).toBe(true);
    expect(queryClient.getQueryData(diagnosticsQueryKey("t1"))).toEqual(REPORT);
  });
});
