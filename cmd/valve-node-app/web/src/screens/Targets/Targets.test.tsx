import { render, screen, cleanup, fireEvent, waitFor, within } from "@testing-library/react";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { describe, it, expect, vi, beforeEach, afterEach } from "vitest";
import type { ReactNode } from "react";
import * as api from "../../api";
import type { Target, Catalog, Host } from "../../api";
import { Targets } from "./Targets";

vi.mock("../../api", async () => {
  const actual = await vi.importActual<typeof import("../../api")>("../../api");
  return {
    ...actual,
    listTargets: vi.fn(),
    getCatalog: vi.fn(),
    getHost: vi.fn(),
    addTarget: vi.fn(),
    deleteTarget: vi.fn(),
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
      LearnURL: "",
      SnapshotSizeTB: 0,
      SyncLabel: "",
      GenesisSyncLabel: "",
    },
  ],
  clients: [],
};

const LINUX_HOST: Host = { os: "linux", arch: "amd64" };
const MAC_HOST: Host = { os: "darwin", arch: "arm64" };

const WIRED_LOCAL: Target = {
  id: "local",
  mode: "local",
  wire: { ChainID: 1, ExecID: "reth", BeaconID: "lighthouse", DataDir: "", JWTPath: "", Archive: false },
};

const UNWIRED_SSH: Target = {
  id: "box1",
  mode: "ssh",
  ssh: { Host: "203.0.113.10", User: "root", KeyPath: "/home/me/.ssh/id_ed25519" },
};

function renderScreen() {
  const queryClient = new QueryClient({
    defaultOptions: { queries: { retry: false }, mutations: { retry: false } },
  });
  function wrapper({ children }: { children: ReactNode }) {
    return <QueryClientProvider client={queryClient}>{children}</QueryClientProvider>;
  }
  return render(<Targets />, { wrapper });
}

beforeEach(() => {
  vi.mocked(api.listTargets).mockReset();
  vi.mocked(api.getCatalog).mockReset();
  vi.mocked(api.getHost).mockReset();
  vi.mocked(api.addTarget).mockReset();
  vi.mocked(api.deleteTarget).mockReset();
});
afterEach(cleanup);

describe("Targets", () => {
  it("shows Loading… while targets/catalog/host are in flight", () => {
    vi.mocked(api.listTargets).mockReturnValue(new Promise(() => {}));
    vi.mocked(api.getCatalog).mockReturnValue(new Promise(() => {}));
    vi.mocked(api.getHost).mockReturnValue(new Promise(() => {}));

    renderScreen();

    expect(screen.getByText("Loading…")).toBeInTheDocument();
  });

  it("shows a failed-to-load error (String(err)) on rejection", async () => {
    vi.mocked(api.listTargets).mockRejectedValue(new Error("network down"));
    vi.mocked(api.getCatalog).mockResolvedValue(CATALOG);
    vi.mocked(api.getHost).mockResolvedValue(LINUX_HOST);

    renderScreen();

    expect(await screen.findByText("Failed to load machines: Error: network down")).toBeInTheDocument();
  });

  it("shows the empty state and both add options (local first) on a viable Linux host with no machines", async () => {
    vi.mocked(api.listTargets).mockResolvedValue([]);
    vi.mocked(api.getCatalog).mockResolvedValue(CATALOG);
    vi.mocked(api.getHost).mockResolvedValue(LINUX_HOST);

    renderScreen();

    expect(await screen.findByText("No machines yet — pick an option below.")).toBeInTheDocument();
    expect(screen.getByText("Add this machine")).toBeInTheDocument();
    expect(screen.getByText("Add a server")).toBeInTheDocument();
    // Local is viable, so it leads: "This machine" appears before "A server over SSH".
    const headings = screen.getAllByRole("heading", { level: 3 }).map((h) => h.textContent);
    expect(headings[0]).toMatch(/^This machine/);
  });

  it("on a non-Linux host, leads with SSH and shows the local option as a warned 'add anyway'", async () => {
    vi.mocked(api.listTargets).mockResolvedValue([]);
    vi.mocked(api.getCatalog).mockResolvedValue(CATALOG);
    vi.mocked(api.getHost).mockResolvedValue(MAC_HOST);

    renderScreen();

    await screen.findByText("No machines yet — pick an option below.");
    const headings = screen.getAllByRole("heading", { level: 3 }).map((h) => h.textContent);
    expect(headings[0]).toMatch(/^A server over SSH/);
    expect(screen.getByText("Add anyway — preview the wizard")).toBeInTheDocument();
    expect(screen.getByText(/darwin/)).toBeInTheDocument();
  });

  it("renders machine cards (local first) with their status badges and the fleet verdict", async () => {
    vi.mocked(api.listTargets).mockResolvedValue([UNWIRED_SSH, WIRED_LOCAL]);
    vi.mocked(api.getCatalog).mockResolvedValue(CATALOG);
    vi.mocked(api.getHost).mockResolvedValue(LINUX_HOST);

    renderScreen();

    await screen.findByText("local");
    const allH2 = screen.getAllByRole("heading", { level: 2 }).map((h) => h.textContent);
    const cardHeadings = allH2.filter((t) => t !== "Your machines" && t !== "Add a machine");
    expect(cardHeadings).toEqual(["local", "box1"]);
    expect(screen.getByText("Ethereum")).toBeInTheDocument();
    expect(screen.getByText("not set up")).toBeInTheDocument();
    expect(screen.getByText("root@203.0.113.10")).toBeInTheDocument();
    // Local already added: only the SSH option is offered.
    expect(screen.queryByText("Add this machine")).not.toBeInTheDocument();
    expect(screen.getByText("Add a server")).toBeInTheDocument();
  });

  it("adds this machine on 'Add this machine' click", async () => {
    vi.mocked(api.listTargets).mockResolvedValue([]);
    vi.mocked(api.getCatalog).mockResolvedValue(CATALOG);
    vi.mocked(api.getHost).mockResolvedValue(LINUX_HOST);
    vi.mocked(api.addTarget).mockResolvedValue({ id: "local", mode: "local" });

    renderScreen();
    await screen.findByText("Add this machine");

    fireEvent.click(screen.getByText("Add this machine"));

    await waitFor(() => expect(api.addTarget).toHaveBeenCalledWith({ id: "local", mode: "local" }));
  });

  it("shows an add-local error inline (err.message) without crashing", async () => {
    vi.mocked(api.listTargets).mockResolvedValue([]);
    vi.mocked(api.getCatalog).mockResolvedValue(CATALOG);
    vi.mocked(api.getHost).mockResolvedValue(LINUX_HOST);
    vi.mocked(api.addTarget).mockRejectedValue(new Error("already exists"));

    renderScreen();
    await screen.findByText("Add this machine");

    fireEvent.click(screen.getByText("Add this machine"));

    expect(await screen.findByText("already exists")).toBeInTheDocument();
  });

  it("toggles the SSH form open/closed and focuses the host field", async () => {
    vi.mocked(api.listTargets).mockResolvedValue([]);
    vi.mocked(api.getCatalog).mockResolvedValue(CATALOG);
    vi.mocked(api.getHost).mockResolvedValue(LINUX_HOST);

    renderScreen();
    await screen.findByText("Add a server");

    expect(screen.queryByText("Add server over SSH")).not.toBeInTheDocument();
    fireEvent.click(screen.getByText("Add a server"));

    expect(screen.getByText("Add server over SSH")).toBeInTheDocument();
    expect(screen.getByPlaceholderText("203.0.113.10")).toHaveFocus();

    fireEvent.click(screen.getByText("Cancel"));
    expect(screen.queryByText("Add server over SSH")).not.toBeInTheDocument();
  });

  it("validates required SSH fields before calling the API", async () => {
    vi.mocked(api.listTargets).mockResolvedValue([]);
    vi.mocked(api.getCatalog).mockResolvedValue(CATALOG);
    vi.mocked(api.getHost).mockResolvedValue(LINUX_HOST);

    renderScreen();
    await screen.findByText("Add a server");
    fireEvent.click(screen.getByText("Add a server"));

    fireEvent.click(screen.getByRole("button", { name: "Add server" }));

    expect(await screen.findByText("host, user, and key path are required")).toBeInTheDocument();
    expect(api.addTarget).not.toHaveBeenCalled();
  });

  it("rejects a non-positive port before calling the API", async () => {
    vi.mocked(api.listTargets).mockResolvedValue([]);
    vi.mocked(api.getCatalog).mockResolvedValue(CATALOG);
    vi.mocked(api.getHost).mockResolvedValue(LINUX_HOST);

    renderScreen();
    await screen.findByText("Add a server");
    fireEvent.click(screen.getByText("Add a server"));

    fireEvent.change(screen.getByPlaceholderText("203.0.113.10"), { target: { value: "203.0.113.10" } });
    fireEvent.change(screen.getByPlaceholderText("root"), { target: { value: "root" } });
    fireEvent.change(screen.getByPlaceholderText("/home/me/.ssh/id_ed25519"), {
      target: { value: "/home/me/.ssh/id_ed25519" },
    });
    fireEvent.change(screen.getByPlaceholderText("22"), { target: { value: "-1" } });

    fireEvent.click(screen.getByRole("button", { name: "Add server" }));

    expect(await screen.findByText("port must be a positive number")).toBeInTheDocument();
    expect(api.addTarget).not.toHaveBeenCalled();
  });

  it("adds an SSH server, defaulting the id via slugify when the name field is blank", async () => {
    vi.mocked(api.listTargets).mockResolvedValue([]);
    vi.mocked(api.getCatalog).mockResolvedValue(CATALOG);
    vi.mocked(api.getHost).mockResolvedValue(LINUX_HOST);
    vi.mocked(api.addTarget).mockResolvedValue({ ...UNWIRED_SSH });

    renderScreen();
    await screen.findByText("Add a server");
    fireEvent.click(screen.getByText("Add a server"));

    fireEvent.change(screen.getByPlaceholderText("203.0.113.10"), { target: { value: "203.0.113.10" } });
    fireEvent.change(screen.getByPlaceholderText("root"), { target: { value: "root" } });
    fireEvent.change(screen.getByPlaceholderText("/home/me/.ssh/id_ed25519"), {
      target: { value: "/home/me/.ssh/id_ed25519" },
    });

    fireEvent.click(screen.getByRole("button", { name: "Add server" }));

    await waitFor(() =>
      expect(api.addTarget).toHaveBeenCalledWith({
        id: "203-0-113-10",
        mode: "ssh",
        ssh: { Host: "203.0.113.10", User: "root", KeyPath: "/home/me/.ssh/id_ed25519" },
      }),
    );
    // The form closes on success.
    expect(screen.queryByText("Add server over SSH")).not.toBeInTheDocument();
  });

  it("removes a machine after confirming, and does nothing on cancel", async () => {
    vi.mocked(api.listTargets).mockResolvedValue([WIRED_LOCAL]);
    vi.mocked(api.getCatalog).mockResolvedValue(CATALOG);
    vi.mocked(api.getHost).mockResolvedValue(LINUX_HOST);
    vi.mocked(api.deleteTarget).mockResolvedValue(undefined);

    renderScreen();
    await screen.findByText("local");

    fireEvent.click(screen.getByText("Remove"));
    expect(screen.getByText("Remove machine")).toBeInTheDocument();
    expect(screen.getByText(/Remove "local"\?/)).toBeInTheDocument();

    fireEvent.click(screen.getByText("Cancel"));
    expect(screen.queryByText("Remove machine")).not.toBeInTheDocument();
    expect(api.deleteTarget).not.toHaveBeenCalled();

    fireEvent.click(screen.getByText("Remove"));
    const dialog = screen.getByRole("dialog");
    fireEvent.click(within(dialog).getByRole("button", { name: "Remove" }));

    await waitFor(() => expect(api.deleteTarget).toHaveBeenCalledWith("local"));
  });
});
