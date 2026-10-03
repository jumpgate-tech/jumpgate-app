import { renderHook, waitFor } from "@testing-library/react";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { describe, it, expect, vi, beforeEach } from "vitest";
import type { ReactNode } from "react";
import * as api from "../api";
import type { Settings } from "../api";
import { useSettings, usePutSettings } from "./settings";

vi.mock("../api", async () => {
  const actual = await vi.importActual<typeof import("../api")>("../api");
  return {
    ...actual,
    getSettings: vi.fn(),
    putSettings: vi.fn(),
  };
});

const SETTINGS: Settings = {
  aiProvider: "gemini",
  aiKeySet: true,
  refRpcBase: "https://ref.example",
  providerKeysSet: ["VALVE_API_KEY"],
};

function wrapper({ children }: { children: ReactNode }) {
  const queryClient = new QueryClient({
    defaultOptions: { queries: { retry: false }, mutations: { retry: false } },
  });
  return <QueryClientProvider client={queryClient}>{children}</QueryClientProvider>;
}

beforeEach(() => {
  vi.mocked(api.getSettings).mockReset();
  vi.mocked(api.putSettings).mockReset();
});

describe("useSettings", () => {
  it("returns settings from getSettings()", async () => {
    vi.mocked(api.getSettings).mockResolvedValue(SETTINGS);

    const { result } = renderHook(() => useSettings(), { wrapper });

    await waitFor(() => expect(result.current.isSuccess).toBe(true));
    expect(result.current.data).toEqual(SETTINGS);
  });
});

describe("usePutSettings", () => {
  it("seeds the settings cache with the response, without an extra GET", async () => {
    vi.mocked(api.getSettings).mockResolvedValue(SETTINGS);
    const updated: Settings = { ...SETTINGS, aiProvider: "groq" };
    vi.mocked(api.putSettings).mockResolvedValue(updated);

    const queryClient = new QueryClient({
      defaultOptions: { queries: { retry: false }, mutations: { retry: false } },
    });
    function localWrapper({ children }: { children: ReactNode }) {
      return <QueryClientProvider client={queryClient}>{children}</QueryClientProvider>;
    }

    const settingsResult = renderHook(() => useSettings(), { wrapper: localWrapper });
    await waitFor(() => expect(settingsResult.result.current.isSuccess).toBe(true));
    expect(api.getSettings).toHaveBeenCalledTimes(1);

    const { result } = renderHook(() => usePutSettings(), { wrapper: localWrapper });
    result.current.mutate({ aiProvider: "groq" });

    await waitFor(() => expect(result.current.isSuccess).toBe(true));
    expect(api.putSettings).toHaveBeenCalledWith({ aiProvider: "groq" });
    // No refetch: the cache was seeded directly from the mutation response.
    expect(api.getSettings).toHaveBeenCalledTimes(1);
    expect(queryClient.getQueryData(["settings"])).toEqual(updated);
  });
});
