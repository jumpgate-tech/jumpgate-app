import { renderHook, waitFor } from "@testing-library/react";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { describe, it, expect, vi, beforeEach } from "vitest";
import type { ReactNode } from "react";
import * as api from "../api";
import type { Target, Catalog } from "../api";
import { useTargets, useCatalog } from "./target";

vi.mock("../api", async () => {
  const actual = await vi.importActual<typeof import("../api")>("../api");
  return {
    ...actual,
    listTargets: vi.fn(),
    getCatalog: vi.fn(),
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
