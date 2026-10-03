import { render, screen, cleanup, fireEvent, waitFor } from "@testing-library/react";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { MemoryRouter, Routes, Route } from "react-router-dom";
import { describe, it, expect, vi, beforeEach, afterEach } from "vitest";
import type { ReactNode } from "react";
import * as api from "../../api";
import type { Target, Catalog, DiagReport } from "../../api";
import { Diagnostics } from "./Diagnostics";

vi.mock("../../api", async () => {
  const actual = await vi.importActual<typeof import("../../api")>("../../api");
  return {
    ...actual,
    listTargets: vi.fn(),
    getCatalog: vi.fn(),
    getLatestDiagnostics: vi.fn(),
    runNetworkDiagnostics: vi.fn(),
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

const PASSING_REPORT: DiagReport = {
  at: "2026-08-01T12:00:00Z",
  trigger: "manual",
  items: [{ ID: "c1", Title: "DNS resolves", Why: "needed to reach peers", Status: "pass", Detail: "ok", Fix: "" }],
};

const FAILING_REPORT: DiagReport = {
  at: "2026-08-01T12:05:00Z",
  trigger: "journal: dial tcp: connection refused",
  items: [
    { ID: "c1", Title: "DNS resolves", Why: "needed to reach peers", Status: "pass", Detail: "ok", Fix: "" },
    {
      ID: "c2",
      Title: "Port 30303 reachable",
      Why: "Peers can't connect without it.",
      Status: "fail",
      Detail: "closed",
      Fix: "sudo ufw allow 30303",
    },
  ],
  failedId: "c2",
};

function renderAt(id = "t1") {
  const queryClient = new QueryClient({
    defaultOptions: { queries: { retry: false }, mutations: { retry: false } },
  });
  function wrapper({ children }: { children: ReactNode }) {
    return (
      <QueryClientProvider client={queryClient}>
        <MemoryRouter initialEntries={[`/diag/${id}`]}>{children}</MemoryRouter>
      </QueryClientProvider>
    );
  }
  return render(
    <Routes>
      <Route path="/diag/:id" element={<Diagnostics />} />
    </Routes>,
    { wrapper },
  );
}

beforeEach(() => {
  vi.mocked(api.listTargets).mockReset();
  vi.mocked(api.getCatalog).mockReset();
  vi.mocked(api.getLatestDiagnostics).mockReset();
  vi.mocked(api.runNetworkDiagnostics).mockReset();
});
afterEach(cleanup);

describe("Diagnostics", () => {
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

  it("shows the no-diagnostics-yet prompt when the latest report is null", async () => {
    vi.mocked(api.listTargets).mockResolvedValue([WIRED_TARGET]);
    vi.mocked(api.getCatalog).mockResolvedValue(CATALOG);
    vi.mocked(api.getLatestDiagnostics).mockResolvedValue(null);

    renderAt();

    expect(await screen.findByText(/No diagnostics have run yet/)).toBeInTheDocument();
  });

  it("renders a passing report, the network's footer link, and the last-run/trigger line", async () => {
    vi.mocked(api.listTargets).mockResolvedValue([WIRED_TARGET]);
    vi.mocked(api.getCatalog).mockResolvedValue(CATALOG);
    vi.mocked(api.getLatestDiagnostics).mockResolvedValue(PASSING_REPORT);

    renderAt();

    expect(await screen.findByText("All checks passed.")).toBeInTheDocument();
    expect(screen.getByText("DNS resolves")).toBeInTheDocument();
    expect(screen.getByText(/trigger: manual/)).toBeInTheDocument();
    expect(screen.getByText("Ethereum")).toHaveAttribute("href", "https://learn.valve.city/eth");
  });

  it("auto-expands the failed check with 'failed here', and can toggle + copy a passing item", async () => {
    vi.mocked(api.listTargets).mockResolvedValue([WIRED_TARGET]);
    vi.mocked(api.getCatalog).mockResolvedValue(CATALOG);
    vi.mocked(api.getLatestDiagnostics).mockResolvedValue(FAILING_REPORT);
    const writeText = vi.fn().mockResolvedValue(undefined);
    Object.defineProperty(navigator, "clipboard", { value: { writeText }, configurable: true });

    renderAt();

    expect(await screen.findByText("failed here")).toBeInTheDocument();
    expect(screen.getByText(/Failed at: Port 30303 reachable/)).toBeInTheDocument();

    const failedLi = screen.getByText("Port 30303 reachable").closest(".check-item")!;
    expect(failedLi).toHaveClass("expanded");

    const passingLi = screen.getByText("DNS resolves").closest(".check-item")!;
    expect(passingLi).not.toHaveClass("expanded");
    fireEvent.click(screen.getByText("DNS resolves"));
    expect(passingLi).toHaveClass("expanded");

    fireEvent.click(screen.getByText("Copy"));
    await waitFor(() => expect(writeText).toHaveBeenCalledWith("sudo ufw allow 30303"));
    expect(await screen.findByText("Copied!")).toBeInTheDocument();
  });

  it("disables the button and shows Running… while a run is in flight", async () => {
    vi.mocked(api.listTargets).mockResolvedValue([WIRED_TARGET]);
    vi.mocked(api.getCatalog).mockResolvedValue(CATALOG);
    vi.mocked(api.getLatestDiagnostics).mockResolvedValue(null);
    vi.mocked(api.runNetworkDiagnostics).mockReturnValue(new Promise(() => {}));

    renderAt();

    await screen.findByText(/No diagnostics have run yet/);
    fireEvent.click(screen.getByText("Run diagnostics"));

    expect(await screen.findByText("Running…")).toBeDisabled();
  });

  it("runs diagnostics on button click and shows the fresh report", async () => {
    vi.mocked(api.listTargets).mockResolvedValue([WIRED_TARGET]);
    vi.mocked(api.getCatalog).mockResolvedValue(CATALOG);
    vi.mocked(api.getLatestDiagnostics).mockResolvedValue(null);
    vi.mocked(api.runNetworkDiagnostics).mockResolvedValue(FAILING_REPORT);

    renderAt();

    await screen.findByText(/No diagnostics have run yet/);
    fireEvent.click(screen.getByText("Run diagnostics"));

    expect(await screen.findByText(/Failed at: Port 30303 reachable/)).toBeInTheDocument();
    expect(api.runNetworkDiagnostics).toHaveBeenCalledWith("t1");
  });

  it("keeps the previous report visible with an error banner when a run fails", async () => {
    vi.mocked(api.listTargets).mockResolvedValue([WIRED_TARGET]);
    vi.mocked(api.getCatalog).mockResolvedValue(CATALOG);
    vi.mocked(api.getLatestDiagnostics).mockResolvedValue(PASSING_REPORT);
    vi.mocked(api.runNetworkDiagnostics).mockRejectedValue(new Error("probe timed out"));

    renderAt();
    await screen.findByText("All checks passed.");

    fireEvent.click(screen.getByText("Run diagnostics"));

    expect(await screen.findByText("probe timed out")).toBeInTheDocument();
    // Stale report stays on screen rather than being replaced by a blank state.
    expect(screen.getByText("All checks passed.")).toBeInTheDocument();
  });
});
