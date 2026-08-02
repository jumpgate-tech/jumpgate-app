import { renderHook, waitFor } from "@testing-library/react";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { describe, it, expect, vi, beforeEach } from "vitest";
import type { ReactNode } from "react";
import * as api from "../api";
import type { Target, Catalog, Host } from "../api";
import { useTargets, useCatalog, useHost, useAddTarget, useDeleteTarget } from "./target";

vi.mock("../api", async () => {
  const actual = await vi.importActual<typeof import("../api")>("../api");
  return {
    ...actual,
    listTargets: vi.fn(),
    getCatalog: vi.fn(),
    getHost: vi.fn(),
    addTarget: vi.fn(),
    deleteTarget: vi.fn(),
  };
});

function wrapper({ children }: { children: ReactNode }) {
  const queryClient = new QueryClient({
    defaultOptions: { queries: { retry: false }, mutations: { retry: false } },
  });
  return <QueryClientProvider client={queryClient}>{children}</QueryClientProvider>;
}

beforeEach(() => {
  vi.mocked(api.listTargets).mockReset();
  vi.mocked(api.getCatalog).mockReset();
  vi.mocked(api.getHost).mockReset();
  vi.mocked(api.addTarget).mockReset();
  vi.mocked(api.deleteTarget).mockReset();
});

describe("useTargets", () => {
  it("returns the list from listTargets()", async () => {
    const targets: Target[] = [{ id: "t1", mode: "local" }];
    vi.mocked(api.listTargets).mockResolvedValue(targets);

    const { result } = renderHook(() => useTargets(), { wrapper });

    await waitFor(() => expect(result.current.isSuccess).toBe(true));
    expect(result.current.data).toEqual(targets);
  });

  it("surfaces a rejection as query error", async () => {
    vi.mocked(api.listTargets).mockRejectedValue(new Error("boom"));

    const { result } = renderHook(() => useTargets(), { wrapper });

    await waitFor(() => expect(result.current.isError).toBe(true));
    expect(result.current.error).toBeInstanceOf(Error);
  });
});

describe("useCatalog", () => {
  it("returns the catalog from getCatalog()", async () => {
    const catalog: Catalog = { networks: [], clients: [] };
    vi.mocked(api.getCatalog).mockResolvedValue(catalog);

    const { result } = renderHook(() => useCatalog(), { wrapper });

    await waitFor(() => expect(result.current.isSuccess).toBe(true));
    expect(result.current.data).toEqual(catalog);
  });
});

describe("useHost", () => {
  it("returns the host from getHost()", async () => {
    const host: Host = { os: "linux", arch: "amd64" };
    vi.mocked(api.getHost).mockResolvedValue(host);

    const { result } = renderHook(() => useHost(), { wrapper });

    await waitFor(() => expect(result.current.isSuccess).toBe(true));
    expect(result.current.data).toEqual(host);
  });
});

describe("useAddTarget", () => {
  it("calls addTarget and invalidates the targets list", async () => {
    const added: Target = { id: "t1", mode: "local" };
    vi.mocked(api.listTargets).mockResolvedValue([]);
    vi.mocked(api.addTarget).mockResolvedValue(added);

    const queryClient = new QueryClient({
      defaultOptions: { queries: { retry: false }, mutations: { retry: false } },
    });
    function localWrapper({ children }: { children: ReactNode }) {
      return <QueryClientProvider client={queryClient}>{children}</QueryClientProvider>;
    }

    const targetsResult = renderHook(() => useTargets(), { wrapper: localWrapper });
    await waitFor(() => expect(targetsResult.result.current.isSuccess).toBe(true));
    expect(api.listTargets).toHaveBeenCalledTimes(1);

    const { result } = renderHook(() => useAddTarget(), { wrapper: localWrapper });
    result.current.mutate({ id: "t1", mode: "local" });

    await waitFor(() => expect(result.current.isSuccess).toBe(true));
    expect(api.addTarget).toHaveBeenCalledWith({ id: "t1", mode: "local" });
    await waitFor(() => expect(api.listTargets).toHaveBeenCalledTimes(2));
  });

  it("surfaces a rejection as mutation error", async () => {
    vi.mocked(api.addTarget).mockRejectedValue(new Error("connection refused"));

    const { result } = renderHook(() => useAddTarget(), { wrapper });
    result.current.mutate({ id: "t1", mode: "local" });

    await waitFor(() => expect(result.current.isError).toBe(true));
    expect(result.current.error).toBeInstanceOf(Error);
  });
});

describe("useDeleteTarget", () => {
  it("calls deleteTarget and invalidates the targets list", async () => {
    vi.mocked(api.listTargets).mockResolvedValue([{ id: "t1", mode: "local" }]);
    vi.mocked(api.deleteTarget).mockResolvedValue(undefined);

    const queryClient = new QueryClient({
      defaultOptions: { queries: { retry: false }, mutations: { retry: false } },
    });
    function localWrapper({ children }: { children: ReactNode }) {
      return <QueryClientProvider client={queryClient}>{children}</QueryClientProvider>;
    }

    const targetsResult = renderHook(() => useTargets(), { wrapper: localWrapper });
    await waitFor(() => expect(targetsResult.result.current.isSuccess).toBe(true));

    const { result } = renderHook(() => useDeleteTarget(), { wrapper: localWrapper });
    result.current.mutate("t1");

    await waitFor(() => expect(result.current.isSuccess).toBe(true));
    expect(api.deleteTarget).toHaveBeenCalledWith("t1");
    await waitFor(() => expect(api.listTargets).toHaveBeenCalledTimes(2));
  });
});
