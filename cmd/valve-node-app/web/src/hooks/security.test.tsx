import { renderHook, waitFor } from "@testing-library/react";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { describe, it, expect, vi, beforeEach } from "vitest";
import type { ReactNode } from "react";
import * as api from "../api";
import type { CheckItem } from "../api";
import { useFirewallChecklist } from "./security";

vi.mock("../api", async () => {
  const actual = await vi.importActual<typeof import("../api")>("../api");
  return {
    ...actual,
    getFirewallChecklist: vi.fn(),
  };
});

function wrapper({ children }: { children: ReactNode }) {
  const queryClient = new QueryClient({
    defaultOptions: { queries: { retry: false }, mutations: { retry: false } },
  });
  return <QueryClientProvider client={queryClient}>{children}</QueryClientProvider>;
}

beforeEach(() => {
  vi.mocked(api.getFirewallChecklist).mockReset();
});

describe("useFirewallChecklist", () => {
  it("fetches the checklist for the target when enabled", async () => {
    const items: CheckItem[] = [
      { ID: "c1", Title: "Port 30303 open", Why: "peers", Status: "pass", Detail: "ok", Fix: "" },
    ];
    vi.mocked(api.getFirewallChecklist).mockResolvedValue(items);

    const { result } = renderHook(() => useFirewallChecklist("t1", true), { wrapper });

    await waitFor(() => expect(result.current.isSuccess).toBe(true));
    expect(api.getFirewallChecklist).toHaveBeenCalledWith("t1");
    expect(result.current.data).toEqual(items);
  });

  it("is disabled (does not call the api) when enabled=false", async () => {
    const { result } = renderHook(() => useFirewallChecklist("t1", false), { wrapper });

    await new Promise((r) => setTimeout(r, 0));

    expect(api.getFirewallChecklist).not.toHaveBeenCalled();
    expect(result.current.fetchStatus).toBe("idle");
  });

  it("is disabled when the target id is empty", async () => {
    const { result } = renderHook(() => useFirewallChecklist("", true), { wrapper });

    await new Promise((r) => setTimeout(r, 0));

    expect(api.getFirewallChecklist).not.toHaveBeenCalled();
    expect(result.current.fetchStatus).toBe("idle");
  });
});
