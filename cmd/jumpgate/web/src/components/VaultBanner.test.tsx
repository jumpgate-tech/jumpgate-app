import { render, screen, cleanup, fireEvent, waitFor, act } from "@testing-library/react";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { describe, it, expect, vi, beforeEach, afterEach } from "vitest";
import type { ReactNode } from "react";
import { VaultBanner } from "./VaultBanner";

type View = { state: string; slots: string[]; plain?: string[] };
let view: View;
const fetchMock = vi.fn();

function json(status: number, body: unknown): Response {
  return new Response(JSON.stringify(body), { status, headers: { "Content-Type": "application/json" } });
}

function renderBanner() {
  const queryClient = new QueryClient({ defaultOptions: { queries: { retry: false }, mutations: { retry: false } } });
  function wrapper({ children }: { children: ReactNode }) {
    return <QueryClientProvider client={queryClient}>{children}</QueryClientProvider>;
  }
  return { queryClient, ...render(<VaultBanner />, { wrapper }) };
}

beforeEach(() => {
  fetchMock.mockReset();
  fetchMock.mockImplementation(async (path: string) => {
    if (path === "/api/vault/lock") {
      view = { ...view, state: "locked" };
    }
    return json(200, view);
  });
  vi.stubGlobal("fetch", fetchMock);
});
afterEach(() => {
  cleanup();
  vi.unstubAllGlobals();
});

describe("VaultBanner", () => {
  it("shows the locked banner and opens the dialog", async () => {
    view = { state: "locked", slots: ["passphrase"] };
    const { queryClient } = renderBanner();
    expect(await screen.findByText("Locked: box actions and plans wait until you unlock.")).toBeInTheDocument();
    expect(await screen.findByRole("dialog")).toBeInTheDocument();
    fireEvent.click(screen.getByText("Not now"));
    expect(screen.queryByRole("dialog")).toBeNull();
    await act(async () => {
      await queryClient.refetchQueries({ queryKey: ["vault"] });
    });
    expect(screen.queryByRole("dialog")).toBeNull();
    fireEvent.click(screen.getByRole("button", { name: "Unlock" }));
    expect(screen.getByRole("dialog")).toBeInTheDocument();
  });

  it("shows the plain-secrets notice", async () => {
    view = { state: "none", slots: [], plain: ["aiKey"] };
    renderBanner();
    expect(await screen.findByText("Secrets are stored unencrypted. Run jumpgate vault migrate.")).toBeInTheDocument();
    expect(screen.queryByRole("dialog")).toBeNull();
  });

  it("shows no locked banner when unlocked", async () => {
    view = { state: "unlocked", slots: ["passphrase"] };
    renderBanner();
    await screen.findByRole("button", { name: "Lock" });
    expect(screen.queryByText(/Locked:/)).toBeNull();
  });

  it("lock button calls lock", async () => {
    view = { state: "unlocked", slots: ["passphrase"] };
    renderBanner();
    fireEvent.click(await screen.findByRole("button", { name: "Lock" }));
    await waitFor(() =>
      expect(fetchMock).toHaveBeenCalledWith("/api/vault/lock", expect.objectContaining({ method: "POST" })),
    );
  });
});
