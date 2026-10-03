import { render, screen, cleanup, fireEvent, waitFor } from "@testing-library/react";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { MemoryRouter, Routes, Route } from "react-router-dom";
import { describe, it, expect, vi, beforeEach, afterEach } from "vitest";
import type { ReactNode } from "react";
import * as api from "../../api";
import type { Target, Catalog, CheckItem } from "../../api";
import { Security } from "./Security";

vi.mock("../../api", async () => {
  const actual = await vi.importActual<typeof import("../../api")>("../../api");
  return {
    ...actual,
    listTargets: vi.fn(),
    getCatalog: vi.fn(),
    getFirewallChecklist: vi.fn(),
  };
});

const CATALOG: Catalog = {
  networks: [
    {
      ChainID: 1,
      Name: "Ethereum",
      CheckpointURL: "",
      ExecClients: [],
      BeaconClients: [],
      LearnURL: "https://learn.valve.city/eth",
      SnapshotSizeTB: 0,
      SyncLabel: "",
      GenesisSyncLabel: "",
    },
  ],
  clients: [],
};

const WIRED_TARGET: Target = {
  id: "t1",
  mode: "local",
  wire: {
    ChainID: 1,
    ExecID: "reth",
    BeaconID: "lighthouse",
    DataDir: "/data",
    JWTPath: "/data/jwt.hex",
    Archive: false,
  },
};

const CHECK_ITEMS: CheckItem[] = [
  {
    ID: "c1",
    Title: "Port 30303 reachable",
    Why: "Peers can't connect without it.",
    Status: "fail",
    Detail: "closed",
    Fix: "sudo ufw allow 30303",
  },
];

function renderAt(id = "t1") {
  const queryClient = new QueryClient({
    defaultOptions: { queries: { retry: false }, mutations: { retry: false } },
  });
  function wrapper({ children }: { children: ReactNode }) {
    return (
      <QueryClientProvider client={queryClient}>
        <MemoryRouter initialEntries={[`/security/${id}`]}>{children}</MemoryRouter>
      </QueryClientProvider>
    );
  }
  return render(
    <Routes>
      <Route path="/security/:id" element={<Security />} />
    </Routes>,
    { wrapper },
  );
}

beforeEach(() => {
  vi.mocked(api.listTargets).mockReset();
  vi.mocked(api.getCatalog).mockReset();
  vi.mocked(api.getFirewallChecklist).mockReset();
});
afterEach(cleanup);

describe("Security", () => {
  it("shows Loading… while the target/catalog fetch is in flight", () => {
    vi.mocked(api.listTargets).mockReturnValue(new Promise(() => {}));
    vi.mocked(api.getCatalog).mockReturnValue(new Promise(() => {}));

    renderAt();

    expect(screen.getByText("Loading…")).toBeInTheDocument();
  });

  it("shows a failed-to-load error when the target/catalog fetch rejects", async () => {
    vi.mocked(api.listTargets).mockRejectedValue(new Error("network down"));
    vi.mocked(api.getCatalog).mockResolvedValue(CATALOG);

    renderAt();

    expect(await screen.findByText(/Failed to load target/)).toBeInTheDocument();
  });

  it("shows a not-found message when no target matches the id", async () => {
    vi.mocked(api.listTargets).mockResolvedValue([]);
    vi.mocked(api.getCatalog).mockResolvedValue(CATALOG);

    renderAt("nope");

    expect(await screen.findByText(/not found/)).toBeInTheDocument();
  });

  it("shows a setup-wizard prompt when the target has no wire config", async () => {
    vi.mocked(api.listTargets).mockResolvedValue([{ id: "t1", mode: "local" }]);
    vi.mocked(api.getCatalog).mockResolvedValue(CATALOG);

    renderAt();

    expect(await screen.findByText(/hasn't completed setup yet/)).toBeInTheDocument();
  });

  it("shows 'No checks returned.' when the checklist is empty", async () => {
    vi.mocked(api.listTargets).mockResolvedValue([WIRED_TARGET]);
    vi.mocked(api.getCatalog).mockResolvedValue(CATALOG);
    vi.mocked(api.getFirewallChecklist).mockResolvedValue([]);

    renderAt();

    expect(await screen.findByText("No checks returned.")).toBeInTheDocument();
  });

  it("renders the checklist, the network's footer link, toggles an item, and copies its fix", async () => {
    vi.mocked(api.listTargets).mockResolvedValue([WIRED_TARGET]);
    vi.mocked(api.getCatalog).mockResolvedValue(CATALOG);
    vi.mocked(api.getFirewallChecklist).mockResolvedValue(CHECK_ITEMS);
    const writeText = vi.fn().mockResolvedValue(undefined);
    Object.defineProperty(navigator, "clipboard", { value: { writeText }, configurable: true });

    renderAt();

    expect(await screen.findByText("Port 30303 reachable")).toBeInTheDocument();
    expect(screen.getByText("fail")).toHaveClass("badge-bad");
    // The network's LearnURL, distinct from the base learn link, shows in the footer.
    expect(screen.getByText("Ethereum")).toHaveAttribute("href", "https://learn.valve.city/eth");

    const li = screen.getByText("Port 30303 reachable").closest(".check-item")!;
    expect(li).not.toHaveClass("expanded");
    fireEvent.click(screen.getByText("Port 30303 reachable"));
    expect(li).toHaveClass("expanded");

    fireEvent.click(screen.getByText("Copy"));
    await waitFor(() => expect(writeText).toHaveBeenCalledWith("sudo ufw allow 30303"));
    expect(await screen.findByText("Copied!")).toBeInTheDocument();
  });

  it("re-runs the checklist on button click", async () => {
    vi.mocked(api.listTargets).mockResolvedValue([WIRED_TARGET]);
    vi.mocked(api.getCatalog).mockResolvedValue(CATALOG);
    vi.mocked(api.getFirewallChecklist).mockResolvedValue(CHECK_ITEMS);

    renderAt();

    await screen.findByText("Port 30303 reachable");
    expect(api.getFirewallChecklist).toHaveBeenCalledTimes(1);

    fireEvent.click(screen.getByText("Re-run checks"));

    await waitFor(() => expect(api.getFirewallChecklist).toHaveBeenCalledTimes(2));
  });

  it("keeps the previous checklist visible with an error banner when a re-run fails", async () => {
    vi.mocked(api.listTargets).mockResolvedValue([WIRED_TARGET]);
    vi.mocked(api.getCatalog).mockResolvedValue(CATALOG);
    vi.mocked(api.getFirewallChecklist).mockResolvedValueOnce(CHECK_ITEMS);

    renderAt();
    await screen.findByText("Port 30303 reachable");

    vi.mocked(api.getFirewallChecklist).mockRejectedValueOnce(new Error("probe timed out"));
    fireEvent.click(screen.getByText("Re-run checks"));

    expect(await screen.findByText("probe timed out")).toBeInTheDocument();
    // Stale results stay on screen rather than being replaced by a blank state.
    expect(screen.getByText("Port 30303 reachable")).toBeInTheDocument();
  });
});
