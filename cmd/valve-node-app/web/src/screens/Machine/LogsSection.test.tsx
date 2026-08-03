import { render, screen, cleanup, fireEvent, waitFor, within } from "@testing-library/react";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { describe, it, expect, vi, beforeEach, afterEach } from "vitest";
import type { ReactNode } from "react";
import * as api from "../../api";
import type { Target, Catalog, Hit } from "../../api";
import { LogsSection } from "./LogsSection";

vi.mock("../../api", async () => {
  const actual = await vi.importActual<typeof import("../../api")>("../../api");
  return {
    ...actual,
    listTargets: vi.fn(),
    getCatalog: vi.fn(),
    getLogs: vi.fn(),
    streamLogs: vi.fn(),
    explain: vi.fn(),
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

function hit(overrides: Partial<Hit> = {}): Hit {
  return {
    unit: "exec",
    line: "boom",
    at: "2026-08-01T12:00:00Z",
    signature: "sig",
    severity: "info",
    explain: "",
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
  return render(<LogsSection targetId={targetId} />, { wrapper });
}

beforeEach(() => {
  vi.mocked(api.listTargets).mockReset();
  vi.mocked(api.getCatalog).mockReset();
  vi.mocked(api.getLogs).mockReset();
  vi.mocked(api.streamLogs).mockReset();
  vi.mocked(api.explain).mockReset();
  localStorage.clear();
});
afterEach(cleanup);

describe("LogsSection", () => {
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
    expect(api.getLogs).not.toHaveBeenCalled();
  });

  it("shows a setup-wizard prompt when the target has no wire config", async () => {
    vi.mocked(api.listTargets).mockResolvedValue([{ id: "t1", mode: "local" }]);
    vi.mocked(api.getCatalog).mockResolvedValue(CATALOG);

    renderSection();

    expect(await screen.findByText(/hasn't completed setup yet/)).toBeInTheDocument();
    expect(api.getLogs).not.toHaveBeenCalled();
  });

  it("shows a logs error when getLogs rejects", async () => {
    vi.mocked(api.listTargets).mockResolvedValue([WIRED_TARGET]);
    vi.mocked(api.getCatalog).mockResolvedValue(CATALOG);
    vi.mocked(api.getLogs).mockRejectedValue(new Error("logs down"));

    renderSection();

    expect(await screen.findByText("Failed to load logs: logs down")).toBeInTheDocument();
  });

  it("renders the live tail, the error feed count, and the network footer link", async () => {
    vi.mocked(api.listTargets).mockResolvedValue([WIRED_TARGET]);
    vi.mocked(api.getCatalog).mockResolvedValue(CATALOG);
    vi.mocked(api.getLogs).mockResolvedValue([
      hit({ line: "starting up", severity: "info" }),
      hit({ line: "boom", severity: "error", unit: "beacon" }),
    ]);
    vi.mocked(api.streamLogs).mockReturnValue(vi.fn());

    renderSection();

    expect(await screen.findByText("starting up")).toBeInTheDocument();
    // "boom" is an error-severity hit, so it renders once in the live tail
    // and once more (reversed) in the error feed panel.
    expect(screen.getAllByText("boom")).toHaveLength(2);
    expect(screen.getByText("1")).toBeInTheDocument(); // error feed badge count
    expect(screen.getByText("Ethereum")).toHaveAttribute("href", "https://learn.valve.city/eth");
  });

  it("shows 'No errors seen yet' when nothing in the tail is error/critical", async () => {
    vi.mocked(api.listTargets).mockResolvedValue([WIRED_TARGET]);
    vi.mocked(api.getCatalog).mockResolvedValue(CATALOG);
    vi.mocked(api.getLogs).mockResolvedValue([hit({ line: "all good", severity: "info" })]);
    vi.mocked(api.streamLogs).mockReturnValue(vi.fn());

    renderSection();

    expect(await screen.findByText("No errors seen yet.")).toBeInTheDocument();
  });

  it("appends a live streamed hit to the tail", async () => {
    vi.mocked(api.listTargets).mockResolvedValue([WIRED_TARGET]);
    vi.mocked(api.getCatalog).mockResolvedValue(CATALOG);
    vi.mocked(api.getLogs).mockResolvedValue([hit({ line: "seed" })]);
    let onHit: ((h: Hit) => void) | null = null;
    vi.mocked(api.streamLogs).mockImplementation((_id, cb) => {
      onHit = cb;
      return vi.fn();
    });

    renderSection();
    await screen.findByText("seed");

    onHit!(hit({ line: "live-tail-hit" }));

    expect(await screen.findByText("live-tail-hit")).toBeInTheDocument();
  });

  it("stops the stream on unmount", async () => {
    vi.mocked(api.listTargets).mockResolvedValue([WIRED_TARGET]);
    vi.mocked(api.getCatalog).mockResolvedValue(CATALOG);
    vi.mocked(api.getLogs).mockResolvedValue([hit()]);
    const stop = vi.fn();
    vi.mocked(api.streamLogs).mockReturnValue(stop);

    const { unmount } = renderSection();
    await screen.findByText("boom");

    unmount();
    expect(stop).toHaveBeenCalledTimes(1);
  });

  describe("Explain with AI", () => {
    async function renderWired(errorLine = "kaboom") {
      vi.mocked(api.listTargets).mockResolvedValue([WIRED_TARGET]);
      vi.mocked(api.getCatalog).mockResolvedValue(CATALOG);
      vi.mocked(api.getLogs).mockResolvedValue([hit({ line: errorLine, severity: "error" })]);
      vi.mocked(api.streamLogs).mockReturnValue(vi.fn());
      renderSection();
      // The error hit renders once in the live tail and once more in the
      // error feed panel — wait for both before interacting.
      await waitFor(() => expect(screen.getAllByText(errorLine)).toHaveLength(2));
    }

    it("shows the consent modal with the candidate excerpt on first click", async () => {
      await renderWired();

      fireEvent.click(screen.getByText("Explain with AI"));

      const dialog = await screen.findByRole("dialog");
      expect(within(dialog).getByText("Send logs to your AI provider?")).toBeInTheDocument();
      expect(within(dialog).getByText("kaboom")).toBeInTheDocument();
      expect(api.explain).not.toHaveBeenCalled();
    });

    it("cancelling the consent modal does not call explain", async () => {
      await renderWired();
      fireEvent.click(screen.getByText("Explain with AI"));
      await screen.findByText("Send logs to your AI provider?");

      fireEvent.click(screen.getByText("Cancel"));

      await waitFor(() => expect(screen.queryByText("Send logs to your AI provider?")).not.toBeInTheDocument());
      expect(api.explain).not.toHaveBeenCalled();
    });

    it("proceeding sets the localStorage consent and shows the explanation", async () => {
      await renderWired();
      vi.mocked(api.explain).mockResolvedValue({ text: "It broke because X.", sentExcerpt: ["kaboom"] });

      fireEvent.click(screen.getByText("Explain with AI"));
      await screen.findByText("Send logs to your AI provider?");
      fireEvent.click(screen.getByText("Send to AI provider"));

      expect(await screen.findByText("It broke because X.")).toBeInTheDocument();
      expect(api.explain).toHaveBeenCalledWith("t1", ["kaboom"]);
      expect(localStorage.getItem("valve-node-app.explain-consent")).toBe("1");
    });

    it("skips the consent modal on a later click once consent was already given", async () => {
      localStorage.setItem("valve-node-app.explain-consent", "1");
      await renderWired();
      vi.mocked(api.explain).mockResolvedValue({ text: "Explained.", sentExcerpt: ["kaboom"] });

      fireEvent.click(screen.getByText("Explain with AI"));

      expect(screen.queryByText("Send logs to your AI provider?")).not.toBeInTheDocument();
      expect(await screen.findByText("Explained.")).toBeInTheDocument();
      expect(api.explain).toHaveBeenCalledWith("t1", ["kaboom"]);
    });

    it("shows a no-provider message on a 409", async () => {
      localStorage.setItem("valve-node-app.explain-consent", "1");
      await renderWired();
      vi.mocked(api.explain).mockRejectedValue(new api.ApiError(409, "no provider configured"));

      fireEvent.click(screen.getByText("Explain with AI"));

      expect(await screen.findByText("No AI provider configured")).toBeInTheDocument();
    });

    it("shows a generic error message on any other failure", async () => {
      localStorage.setItem("valve-node-app.explain-consent", "1");
      await renderWired();
      vi.mocked(api.explain).mockRejectedValue(new Error("boom"));

      fireEvent.click(screen.getByText("Explain with AI"));

      expect(await screen.findByText("Explain failed")).toBeInTheDocument();
      expect(screen.getByText("boom")).toBeInTheDocument();
    });
  });
});
