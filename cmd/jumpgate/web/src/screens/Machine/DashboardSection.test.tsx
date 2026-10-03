import { render, screen, cleanup, fireEvent, waitFor, within, act } from "@testing-library/react";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { describe, it, expect, vi, beforeEach, afterEach } from "vitest";
import type { ReactNode } from "react";
import * as api from "../../api";
import type { Target, Catalog, Snapshot, DiskUsage, EndpointInfo } from "../../api";
import { DashboardSection } from "./DashboardSection";

vi.mock("../../api", async () => {
  const actual = await vi.importActual<typeof import("../../api")>("../../api");
  return {
    ...actual,
    listTargets: vi.fn(),
    getCatalog: vi.fn(),
    streamMonitor: vi.fn(),
    getDiskUsage: vi.fn(),
    getEndpoints: vi.fn(),
    serviceAction: vi.fn(),
    clearService: vi.fn(),
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

function snap(overrides: Partial<Snapshot> = {}): Snapshot {
  return {
    at: "2026-08-01T12:00:00.000Z",
    execSyncing: false,
    execHead: 100,
    refHead: 100,
    beaconSlot: 500,
    beaconDistance: 0,
    execPeers: 5,
    beaconPeers: 6,
    diskUsedPct: 10,
    execActive: true,
    beaconActive: true,
    ...overrides,
  };
}

function du(overrides: Partial<DiskUsage> = {}): DiskUsage {
  return {
    ExecBytes: 1_000_000_000,
    BeaconBytes: 500_000_000,
    DiskFreeBytes: 2_000_000_000,
    ExpectedExecBytes: 2_000_000_000,
    ExpectedBeaconBytes: 1_000_000_000,
    SyncLabel: "~2h",
    GenesisSyncLabel: "~2d",
    ...overrides,
  };
}

function endpoints(overrides: Partial<EndpointInfo> = {}): EndpointInfo {
  return {
    ExecHTTP: "http://127.0.0.1:8545",
    BeaconHTTP: "http://127.0.0.1:5052",
    ExecReachable: true,
    BeaconReachable: true,
    ChainIDMatches: true,
    Access: "local",
    TunnelHint: "",
    ...overrides,
  };
}

function renderSection(targetId = "t1") {
  const queryClient = new QueryClient({
    defaultOptions: { queries: { retry: false }, mutations: { retry: false } },
  });
  function wrapper({ children }: { children: ReactNode }) {
    return <QueryClientProvider client={queryClient}>{children}</QueryClientProvider>;
  }
  return render(<DashboardSection targetId={targetId} />, { wrapper });
}

// serviceRow returns the exec or beacon <div class="service-row">, in the
// order ServicesCard renders them (exec first, beacon second) — "Execution"
// and "Beacon" also appear as the Peers card's <dt> labels, so a plain
// getByText("Execution") is ambiguous and must be scoped to a row instead.
function serviceRow(which: "exec" | "beacon"): HTMLElement {
  const rows = document.querySelectorAll<HTMLElement>(".service-row");
  return which === "exec" ? rows[0] : rows[1];
}

// wireStream captures the onSnapshot callback so tests can push snapshots.
function wireStream(): { push: (s: Snapshot) => void; stop: ReturnType<typeof vi.fn> } {
  const stop = vi.fn();
  let onSnapshot: ((s: Snapshot) => void) | null = null;
  vi.mocked(api.streamMonitor).mockImplementation((_id, cb) => {
    onSnapshot = cb;
    return stop;
  });
  return {
    push: (s: Snapshot) => act(() => onSnapshot!(s)),
    stop,
  };
}

beforeEach(() => {
  vi.mocked(api.listTargets).mockReset();
  vi.mocked(api.getCatalog).mockReset();
  vi.mocked(api.streamMonitor).mockReset();
  vi.mocked(api.getDiskUsage).mockReset();
  vi.mocked(api.getEndpoints).mockReset();
  vi.mocked(api.serviceAction).mockReset();
  vi.mocked(api.clearService).mockReset();
});
afterEach(cleanup);

describe("DashboardSection", () => {
  it("shows Loading… while the target/catalog fetch is in flight", () => {
    vi.mocked(api.listTargets).mockReturnValue(new Promise(() => {}));
    vi.mocked(api.getCatalog).mockReturnValue(new Promise(() => {}));

    renderSection();

    expect(screen.getByText("Loading…")).toBeInTheDocument();
  });

  it("shows a failed-to-load error when the target/catalog fetch rejects", async () => {
    vi.mocked(api.listTargets).mockRejectedValue(new Error("network down"));
    vi.mocked(api.getCatalog).mockResolvedValue(CATALOG);

    renderSection();

    expect(await screen.findByText(/Failed to load target/)).toBeInTheDocument();
  });

  it("shows a not-found message when no target matches the id", async () => {
    vi.mocked(api.listTargets).mockResolvedValue([]);
    vi.mocked(api.getCatalog).mockResolvedValue(CATALOG);

    renderSection("nope");

    expect(await screen.findByText(/not found/)).toBeInTheDocument();
    expect(api.streamMonitor).not.toHaveBeenCalled();
  });

  it("shows a setup-wizard prompt when the target has no wire config", async () => {
    vi.mocked(api.listTargets).mockResolvedValue([{ id: "t1", mode: "local" }]);
    vi.mocked(api.getCatalog).mockResolvedValue(CATALOG);

    renderSection();

    expect(await screen.findByText(/hasn't completed setup yet/)).toBeInTheDocument();
    expect(api.streamMonitor).not.toHaveBeenCalled();
  });

  it("shows Connecting… once wired but before the first snapshot arrives", async () => {
    vi.mocked(api.listTargets).mockResolvedValue([WIRED_TARGET]);
    vi.mocked(api.getCatalog).mockResolvedValue(CATALOG);
    vi.mocked(api.getDiskUsage).mockReturnValue(new Promise(() => {}));
    vi.mocked(api.getEndpoints).mockReturnValue(new Promise(() => {}));
    wireStream();

    renderSection();

    expect(await screen.findByText("Connecting…")).toBeInTheDocument();
  });

  describe("once a snapshot has arrived", () => {
    async function renderWired(snapshotOverrides: Partial<Snapshot> = {}) {
      vi.mocked(api.listTargets).mockResolvedValue([WIRED_TARGET]);
      vi.mocked(api.getCatalog).mockResolvedValue(CATALOG);
      vi.mocked(api.getDiskUsage).mockResolvedValue(du());
      vi.mocked(api.getEndpoints).mockResolvedValue(endpoints());
      const stream = wireStream();

      renderSection();
      await screen.findByText("Connecting…");
      stream.push(snap(snapshotOverrides));
      await waitFor(() => expect(screen.queryByText("Connecting…")).not.toBeInTheDocument());
      return stream;
    }

    it("renders the top status, both services, and the network footer link", async () => {
      await renderWired();

      expect(screen.getByText("Running · synced")).toBeInTheDocument();
      expect(within(serviceRow("exec")).getByText("Execution")).toBeInTheDocument();
      expect(within(serviceRow("beacon")).getByText("Beacon")).toBeInTheDocument();
      expect(screen.getAllByText("active")).toHaveLength(2);
      expect(screen.getByText("Ethereum")).toHaveAttribute("href", "https://learn.valve.city/eth");
    });

    it("reads Node not running when both services are down", async () => {
      await renderWired({ execActive: false, beaconActive: false });

      expect(screen.getByText("Node not running")).toBeInTheDocument();
      expect(screen.getAllByText("down")).toHaveLength(2);
    });

    it("renders sync/peer figures on the execution and beacon cards", async () => {
      await renderWired({ execHead: 12345, refHead: 12345, beaconSlot: 999, beaconDistance: 0 });

      // Local head and Reference head both format to the same string here —
      // both dd's are expected.
      expect(screen.getAllByText("12,345")).toHaveLength(2);
      expect(screen.getByText("999")).toBeInTheDocument();
    });

    it("renders the storage card once disk usage loads, including the warn class above threshold", async () => {
      await renderWired({ diskUsedPct: 90 });

      expect(screen.getByText("90.0% used")).toBeInTheDocument();
      expect(screen.getByText(/Execution — /)).toBeInTheDocument();
      const storageHeading = screen.getByText("Storage");
      expect(storageHeading.closest(".card")).toHaveClass("card-warn");
    });

    it("shows a disk-usage error with a working Retry button", async () => {
      vi.mocked(api.listTargets).mockResolvedValue([WIRED_TARGET]);
      vi.mocked(api.getCatalog).mockResolvedValue(CATALOG);
      vi.mocked(api.getDiskUsage).mockRejectedValueOnce(new Error("du down")).mockResolvedValue(du());
      vi.mocked(api.getEndpoints).mockResolvedValue(endpoints());
      const stream = wireStream();

      renderSection();
      await screen.findByText("Connecting…");
      stream.push(snap());

      expect(await screen.findByText("du down")).toBeInTheDocument();
      fireEvent.click(screen.getByText("Retry"));

      await waitFor(() => expect(screen.getByText(/Execution — /)).toBeInTheDocument());
      expect(api.getDiskUsage).toHaveBeenCalledTimes(2);
    });

    it("renders reachable endpoints with copy buttons", async () => {
      await renderWired();

      expect(screen.getByText("http://127.0.0.1:8545")).toBeInTheDocument();
      expect(screen.getByText("http://127.0.0.1:5052")).toBeInTheDocument();
      expect(screen.getAllByText("Copy")).toHaveLength(2);
    });

    it("copies an endpoint URL to the clipboard and reverts the label after a delay", async () => {
      vi.useFakeTimers({ shouldAdvanceTime: true });
      const writeText = vi.fn().mockResolvedValue(undefined);
      Object.assign(navigator, { clipboard: { writeText } });

      await renderWired();
      const [copyBtn] = screen.getAllByText("Copy");
      fireEvent.click(copyBtn);

      await vi.waitFor(() => expect(screen.getByText("Copied!")).toBeInTheDocument());
      expect(writeText).toHaveBeenCalledWith("http://127.0.0.1:8545");

      await act(async () => {
        vi.advanceTimersByTime(1600);
      });
      expect(screen.getAllByText("Copy").length).toBeGreaterThan(0);
      vi.useRealTimers();
    });

    it("stops the stream on unmount", async () => {
      const stream = await renderWired();
      cleanup();
      expect(stream.stop).toHaveBeenCalledTimes(1);
    });
  });

  describe("service actions", () => {
    async function renderWired() {
      vi.mocked(api.listTargets).mockResolvedValue([WIRED_TARGET]);
      vi.mocked(api.getCatalog).mockResolvedValue(CATALOG);
      vi.mocked(api.getDiskUsage).mockResolvedValue(du());
      vi.mocked(api.getEndpoints).mockResolvedValue(endpoints());
      const stream = wireStream();

      renderSection();
      await screen.findByText("Connecting…");
      stream.push(snap());
      await waitFor(() => expect(screen.queryByText("Connecting…")).not.toBeInTheDocument());
      return stream;
    }

    it("calls serviceAction for the clicked service/action and disables that service's buttons while pending", async () => {
      let resolveAction: (() => void) | null = null;
      vi.mocked(api.serviceAction).mockReturnValue(
        new Promise((resolve) => {
          resolveAction = () => resolve({ Active: false });
        }),
      );

      await renderWired();
      const row = within(serviceRow("exec"));
      fireEvent.click(row.getByText("Stop"));

      expect(api.serviceAction).toHaveBeenCalledWith("t1", "exec", "stop");
      await waitFor(() => expect(row.getByText("Start")).toBeDisabled());
      expect(row.getByText("Restart")).toBeDisabled();
      expect(row.getByText("Clear…")).toBeDisabled();

      // Beacon's row is unaffected by exec's in-flight action.
      const beaconRow = within(serviceRow("beacon"));
      expect(beaconRow.getByText("Stop")).not.toBeDisabled();

      await act(async () => {
        resolveAction!();
        await Promise.resolve();
      });
      // Pending clears once the action resolves — Restart and Clear… go back
      // to enabled. Start stays disabled, but for a different reason now:
      // the snapshot (unchanged by the mocked response — active state is
      // never derived from an action's result) still reports exec active.
      await waitFor(() => expect(row.getByText("Restart")).not.toBeDisabled());
      expect(row.getByText("Clear…")).not.toBeDisabled();
      expect(row.getByText("Start")).toBeDisabled();
    });

    it("shows a formatted error message when an action fails", async () => {
      vi.mocked(api.serviceAction).mockRejectedValue(new Error("connection refused"));

      await renderWired();
      fireEvent.click(within(serviceRow("exec")).getByText("Stop"));

      expect(
        await screen.findByText("Execution stop failed: connection refused"),
      ).toBeInTheDocument();
    });

    it("Start is disabled while active and Stop is disabled while inactive", async () => {
      await renderWired();
      const row = within(serviceRow("exec"));
      expect(row.getByText("Start")).toBeDisabled();
      expect(row.getByText("Stop")).not.toBeDisabled();
    });
  });

  describe("clear service modal", () => {
    async function renderWired() {
      vi.mocked(api.listTargets).mockResolvedValue([WIRED_TARGET]);
      vi.mocked(api.getCatalog).mockResolvedValue(CATALOG);
      vi.mocked(api.getDiskUsage).mockResolvedValue(du());
      vi.mocked(api.getEndpoints).mockResolvedValue(endpoints());
      const stream = wireStream();

      renderSection();
      await screen.findByText("Connecting…");
      stream.push(snap());
      await waitFor(() => expect(screen.queryByText("Connecting…")).not.toBeInTheDocument());
      return stream;
    }

    it("opens on Clear…, gates the confirm button on typing the exact service id, and calls clearService", async () => {
      vi.mocked(api.clearService).mockResolvedValue({ status: "ok" });

      await renderWired();
      fireEvent.click(within(serviceRow("exec")).getByText("Clear…"));

      const dialog = await screen.findByRole("dialog");
      const confirmBtn = within(dialog).getByText("Clear and resync");
      expect(confirmBtn).toBeDisabled();

      const input = within(dialog).getByRole("textbox");
      fireEvent.change(input, { target: { value: "wrong" } });
      expect(confirmBtn).toBeDisabled();

      fireEvent.change(input, { target: { value: "exec" } });
      expect(confirmBtn).not.toBeDisabled();

      fireEvent.click(confirmBtn);

      await waitFor(() => expect(api.clearService).toHaveBeenCalledWith("t1", "exec"));
      await waitFor(() => expect(screen.queryByRole("dialog")).not.toBeInTheDocument());
    });

    it("shows a 'Clear failed: <message>' inline error (mirrors runClear's own prefix) and keeps the modal open", async () => {
      vi.mocked(api.clearService).mockRejectedValue(new Error("disk busy"));

      await renderWired();
      fireEvent.click(within(serviceRow("exec")).getByText("Clear…"));

      const dialog = await screen.findByRole("dialog");
      fireEvent.change(within(dialog).getByRole("textbox"), { target: { value: "exec" } });
      fireEvent.click(within(dialog).getByText("Clear and resync"));

      expect(await within(dialog).findByText("Clear failed: disk busy")).toBeInTheDocument();
      expect(within(dialog).getByText("Clear and resync")).not.toBeDisabled();
    });

    it("cancelling closes the modal without calling clearService", async () => {
      await renderWired();
      fireEvent.click(within(serviceRow("exec")).getByText("Clear…"));

      const dialog = await screen.findByRole("dialog");
      fireEvent.click(within(dialog).getByText("Cancel"));

      await waitFor(() => expect(screen.queryByRole("dialog")).not.toBeInTheDocument());
      expect(api.clearService).not.toHaveBeenCalled();
    });

    it("does not disable Cancel while a clear is in flight (dashboard.ts never disables it there either)", async () => {
      let resolveClear: (() => void) | null = null;
      vi.mocked(api.clearService).mockReturnValue(
        new Promise((resolve) => {
          resolveClear = () => resolve({ status: "ok" });
        }),
      );

      await renderWired();
      fireEvent.click(within(serviceRow("exec")).getByText("Clear…"));

      const dialog = await screen.findByRole("dialog");
      fireEvent.change(within(dialog).getByRole("textbox"), { target: { value: "exec" } });
      fireEvent.click(within(dialog).getByText("Clear and resync"));

      await waitFor(() => expect(within(dialog).getByText("Clearing…")).toBeInTheDocument());
      expect(within(dialog).getByText("Cancel")).not.toBeDisabled();

      await act(async () => {
        resolveClear!();
        await Promise.resolve();
      });
    });
  });
});
