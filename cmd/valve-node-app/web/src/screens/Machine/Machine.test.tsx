import { render, screen, cleanup, fireEvent, waitFor } from "@testing-library/react";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { MemoryRouter, Routes, Route } from "react-router-dom";
import { describe, it, expect, vi, beforeEach, afterEach } from "vitest";
import type { ReactNode } from "react";
import * as api from "../../api";
import type { Target, Catalog } from "../../api";
import { Machine } from "./Machine";

vi.mock("../../api", async () => {
  const actual = await vi.importActual<typeof import("../../api")>("../../api");
  return {
    ...actual,
    listTargets: vi.fn(),
    getCatalog: vi.fn(),
  };
});

// The four section components are exercised by their own tests; here we mock
// them to plain markers so we can assert WHEN each is mounted (its real
// counterpart opens an SSE stream on mount, so "mounted" == "streaming").
vi.mock("./SetupWizard", () => ({
  SetupWizard: ({ targetId }: { targetId: string }) => <div>setup-mounted:{targetId}</div>,
}));
vi.mock("./DashboardSection", () => ({
  DashboardSection: ({ targetId }: { targetId: string }) => <div>dashboard-mounted:{targetId}</div>,
}));
vi.mock("./LogsSection", () => ({
  LogsSection: ({ targetId }: { targetId: string }) => <div>logs-mounted:{targetId}</div>,
}));
vi.mock("./ServicesSection", () => ({
  ServicesSection: ({ targetId }: { targetId: string }) => <div>services-mounted:{targetId}</div>,
}));

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
    Archive: true,
  },
};

const UNWIRED_TARGET: Target = { id: "t1", mode: "local" };

function renderAt(id = "t1") {
  const queryClient = new QueryClient({
    defaultOptions: { queries: { retry: false }, mutations: { retry: false } },
  });
  function wrapper({ children }: { children: ReactNode }) {
    return (
      <QueryClientProvider client={queryClient}>
        <MemoryRouter initialEntries={[`/machine/${id}`]}>{children}</MemoryRouter>
      </QueryClientProvider>
    );
  }
  return render(
    <Routes>
      <Route path="/machine/:id" element={<Machine />} />
      <Route path="/targets" element={<div>targets-list</div>} />
    </Routes>,
    { wrapper },
  );
}

// sectionBody returns the <div data-section-body="key"> container for a section.
function sectionBody(key: string): HTMLElement {
  return document.querySelector<HTMLElement>(`[data-section-body="${key}"]`)!;
}
function sectionHead(title: string): HTMLElement {
  return screen.getByRole("button", { name: new RegExp(title) });
}

beforeEach(() => {
  vi.mocked(api.listTargets).mockReset();
  vi.mocked(api.getCatalog).mockReset();
});
afterEach(cleanup);

describe("Machine", () => {
  it("renders the machine id as the page header even while loading", () => {
    vi.mocked(api.listTargets).mockReturnValue(new Promise(() => {}));
    vi.mocked(api.getCatalog).mockReturnValue(new Promise(() => {}));

    renderAt("t1");

    expect(screen.getByRole("heading", { level: 1, name: "t1" })).toBeInTheDocument();
    expect(screen.getByText("Loading…")).toBeInTheDocument();
  });

  it("shows a failed-to-load error when the target/catalog fetch rejects", async () => {
    vi.mocked(api.listTargets).mockRejectedValue(new Error("network down"));
    vi.mocked(api.getCatalog).mockResolvedValue(CATALOG);

    renderAt("t1");

    expect(await screen.findByText(/Failed to load machine: network down/)).toBeInTheDocument();
  });

  it("redirects to the targets list when no target matches the id", async () => {
    vi.mocked(api.listTargets).mockResolvedValue([]);
    vi.mocked(api.getCatalog).mockResolvedValue(CATALOG);

    renderAt("nope");

    expect(await screen.findByText("targets-list")).toBeInTheDocument();
  });

  describe("once the machine loads", () => {
    async function renderWired(target: Target = WIRED_TARGET) {
      vi.mocked(api.listTargets).mockResolvedValue([target]);
      vi.mocked(api.getCatalog).mockResolvedValue(CATALOG);
      renderAt("t1");
      // Wait for the shell (sections) to appear.
      await screen.findByText("Setup");
    }

    it("renders the four sections in order with the machine.ts titles", async () => {
      await renderWired();

      const titles = Array.from(document.querySelectorAll(".machine-section-title")).map(
        (el) => el.textContent,
      );
      expect(titles).toEqual(["Setup", "Dashboard", "Logs", "Devnet"]);
    });

    it("renders the wired header status: network, both clients, and the archive badge", async () => {
      await renderWired();

      expect(screen.getByText("Ethereum")).toBeInTheDocument();
      expect(screen.getByText("reth")).toBeInTheDocument();
      expect(screen.getByText("lighthouse")).toBeInTheDocument();
      expect(screen.getByText("archive")).toBeInTheDocument();
    });

    it("renders a 'not set up' header status for an unwired machine", async () => {
      await renderWired(UNWIRED_TARGET);

      // "not set up" appears both in the header and the Setup section status.
      expect(screen.getAllByText("not set up").length).toBeGreaterThanOrEqual(1);
      expect(screen.queryByText("Ethereum")).not.toBeInTheDocument();
    });

    it("mounts NO section content by default — every section starts collapsed", async () => {
      await renderWired();

      expect(screen.queryByText(/setup-mounted/)).not.toBeInTheDocument();
      expect(screen.queryByText(/dashboard-mounted/)).not.toBeInTheDocument();
      expect(screen.queryByText(/logs-mounted/)).not.toBeInTheDocument();
      expect(screen.queryByText(/services-mounted/)).not.toBeInTheDocument();
      // Each body container exists but is hidden.
      expect(sectionBody("setup")).toHaveAttribute("hidden");
    });

    it("mounts a section's content (and thus its stream) only when it is first expanded", async () => {
      await renderWired();

      fireEvent.click(sectionHead("Dashboard"));

      expect(await screen.findByText("dashboard-mounted:t1")).toBeInTheDocument();
      expect(sectionBody("dashboard")).not.toHaveAttribute("hidden");
      // The other sections' content stays unmounted — no stream opened for them.
      expect(screen.queryByText(/setup-mounted/)).not.toBeInTheDocument();
      expect(screen.queryByText(/logs-mounted/)).not.toBeInTheDocument();
      expect(screen.queryByText(/services-mounted/)).not.toBeInTheDocument();
    });

    it("keeps a section mounted after it is collapsed again (only hides its body)", async () => {
      await renderWired();

      const head = sectionHead("Logs");
      fireEvent.click(head);
      expect(await screen.findByText("logs-mounted:t1")).toBeInTheDocument();
      expect(head).toHaveAttribute("aria-expanded", "true");

      // Collapse it — the content stays in the tree (stream still running),
      // the body is merely hidden. Mirrors machine.ts toggling bodyEl.hidden.
      fireEvent.click(head);
      await waitFor(() => expect(head).toHaveAttribute("aria-expanded", "false"));
      expect(screen.getByText("logs-mounted:t1")).toBeInTheDocument();
      expect(sectionBody("logs")).toHaveAttribute("hidden");
    });

    it("toggles the section card's open class and caret with expand/collapse", async () => {
      await renderWired();

      const head = sectionHead("Setup");
      const card = document.querySelector('[data-section-card="setup"]')!;
      expect(card).not.toHaveClass("open");

      fireEvent.click(head);
      await waitFor(() => expect(card).toHaveClass("open"));

      fireEvent.click(head);
      await waitFor(() => expect(card).not.toHaveClass("open"));
    });
  });
});
