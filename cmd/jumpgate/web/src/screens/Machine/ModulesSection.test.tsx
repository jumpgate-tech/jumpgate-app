import { render, screen, cleanup, fireEvent, waitFor } from "@testing-library/react";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { describe, it, expect, vi, beforeEach, afterEach } from "vitest";
import type { ReactNode } from "react";
import * as api from "../../api";
import type { Catalog, CatalogModule, GatewayView, Target } from "../../api";
import { ModulesSection } from "./ModulesSection";

vi.mock("../../api", async () => {
  const actual = await vi.importActual<typeof import("../../api")>("../../api");
  return {
    ...actual,
    listTargets: vi.fn(),
    getCatalog: vi.fn(),
    listModules: vi.fn(),
    setModule: vi.fn(),
    streamMonitor: vi.fn(),
    streamSetup: vi.fn(),
    getGateways: vi.fn(),
  };
});

const REASON = "The message board needs reth (valve fork). This node runs go-pulse.";

function mod(id: string, placement: CatalogModule["placement"], name: string, applies: CatalogModule["applies"]): CatalogModule {
  return {
    id,
    name,
    summary: `${name} summary`,
    siteUrl: id === "msgboard" ? "https://msgboard.xyz" : "",
    docsStep: `${id}-enable`,
    placement,
    resources: { ramBytes: 256_000_000, diskBytes: 0 },
    applies,
    methods: [],
    public: id === "msgboard" ? ["read", "post"] : [],
  };
}

const CATALOG: Catalog = {
  networks: [],
  clients: [],
  modules: [
    mod("msgboard", "node", "Message board", [
      { chainId: 369, client: "reth", ok: true, beta: false },
      { chainId: 369, client: "go-pulse", ok: false, beta: false, reason: REASON },
    ]),
    mod("el-bootnode", "box", "EL bootnode", [{ chainId: 369, client: "", ok: true, beta: false }]),
    mod("checkpointz", "box", "Checkpointz", [{ chainId: 369, client: "", ok: true, beta: false }]),
    mod("watch", "box", "Health watch", [{ chainId: 0, client: "", ok: true, beta: false }]),
  ],
};

function target(exec = "reth", modules?: api.NodeModule[]): Target {
  return {
    id: "t1",
    mode: "local",
    wire: { ChainID: 369, ExecID: exec, BeaconID: "lighthouse", DataDir: "/d", JWTPath: "/d/jwt", Archive: false, Modules: modules },
  };
}

function gateway(acme: boolean): GatewayView {
  return {
    id: "gw1",
    // The TLS below is left contradicting nothing on purpose; the UI reads tier.
    tier: acme ? "public" : "private",
    config: {
      ProjectID: "p",
      BindAddr: "",
      Port: 4000,
      TLS: acme
        ? { Enabled: true, CertSource: "acme", Hostname: "rpc.example", CertFile: "", KeyFile: "", HTTPSPort: 443, BindAddr: "", ImageRef: "" }
        : null,
      Networks: [{ ChainID: 369, Upstreams: [{ ID: "u", Kind: "managed-node", TargetID: "t1", Endpoint: "", Local: true, RecentOnly: false }] }],
    },
  } as unknown as GatewayView;
}

function setup(t: Target, opts: { gateways?: GatewayView[]; gateStale?: boolean } = {}) {
  vi.mocked(api.listTargets).mockResolvedValue([t]);
  vi.mocked(api.getCatalog).mockResolvedValue(CATALOG);
  vi.mocked(api.listModules).mockResolvedValue({
    modules: [],
    gateStale: !!opts.gateStale,
    message: opts.gateStale ? "Status needs a newer gate on this box" : "",
  });
  vi.mocked(api.getGateways).mockResolvedValue({ gateways: opts.gateways ?? [] } as unknown as api.GatewaysResponse);
  vi.mocked(api.streamMonitor).mockReturnValue(() => {});
  vi.mocked(api.streamSetup).mockReturnValue(() => {});
  vi.mocked(api.setModule).mockResolvedValue({ status: "started" });
  const qc = new QueryClient({ defaultOptions: { queries: { retry: false }, mutations: { retry: false } } });
  const wrapper = ({ children }: { children: ReactNode }) => <QueryClientProvider client={qc}>{children}</QueryClientProvider>;
  return render(<ModulesSection targetId="t1" />, { wrapper });
}

function card(id: string): HTMLElement {
  const el = document.querySelector<HTMLElement>(`[data-module="${id}"]`);
  if (!el) throw new Error(`no card ${id}`);
  return el;
}

beforeEach(() => {
  vi.mocked(api.setModule).mockReset();
});
afterEach(cleanup);

describe("ModulesSection", () => {
  it("shows the restart line before setModule is called", async () => {
    setup(target());
    await screen.findByText("Message board");
    fireEvent.click(card("msgboard").querySelector("button.module-toggle")!);
    expect(await screen.findByText("This restarts reth on this machine. The beacon keeps running.")).toBeTruthy();
    expect(api.setModule).not.toHaveBeenCalled();
    fireEvent.click(screen.getByRole("button", { name: "Turn on" }));
    await waitFor(() => expect(api.setModule).toHaveBeenCalledWith("t1", "msgboard", { enabled: true, public: "" }));
  });

  it("disables the toggle with the catalog reason verbatim", async () => {
    setup(target("go-pulse"));
    await screen.findByText(REASON);
    expect((card("msgboard").querySelector("button.module-toggle") as HTMLButtonElement).disabled).toBe(true);
  });

  it("asks a box module for port and ExtIP only when it declares them", async () => {
    setup(target());
    await screen.findByText("Services on this box");

    fireEvent.click(card("el-bootnode").querySelector("button.module-toggle")!);
    expect(await screen.findByLabelText("Port")).toBeTruthy();
    expect(screen.getByLabelText("External IP")).toBeTruthy();
    fireEvent.change(screen.getByLabelText("External IP"), { target: { value: "203.0.113.7" } });
    fireEvent.click(screen.getByRole("button", { name: "Turn on" }));
    await waitFor(() =>
      expect(api.setModule).toHaveBeenCalledWith("t1", "el-bootnode", { enabled: true, chainId: 369, extIp: "203.0.113.7" }),
    );
    cleanup();

    setup(target());
    await screen.findByText("Services on this box");
    fireEvent.click(card("checkpointz").querySelector("button.module-toggle")!);
    expect(await screen.findByLabelText("Port")).toBeTruthy();
    expect(screen.queryByLabelText("External IP")).toBeNull();
    fireEvent.click(screen.getByRole("button", { name: "Cancel" }));

    fireEvent.click(card("watch").querySelector("button.module-toggle")!);
    await screen.findByRole("button", { name: "Turn on" });
    expect(screen.queryByLabelText("Port")).toBeNull();
  });

  it("shows the Public radio only behind a Public-tier gateway", async () => {
    setup(target("reth", [{ id: "msgboard" }]), { gateways: [gateway(false)] });
    await screen.findByText("Message board");
    expect(screen.queryByLabelText("Public reads")).toBeNull();
    cleanup();

    setup(target("reth", [{ id: "msgboard" }]), { gateways: [gateway(true)] });
    expect(await screen.findByLabelText("Public reads")).toBeTruthy();
    const warning = "Anyone can post through your node. Posts cost the sender proof of work, not you.";
    expect(screen.queryByText(warning)).toBeNull();
    fireEvent.click(screen.getByLabelText("Public reads and posting"));
    expect(screen.getByText(warning)).toBeTruthy();
    await waitFor(() => expect(api.setModule).toHaveBeenCalledWith("t1", "msgboard", { enabled: true, public: "post" }));
  });

  it("shows no Public radio when the Public-tier gateway routes to another machine", async () => {
    const gw = gateway(true);
    gw.config.Networks![0].Upstreams![0].TargetID = "other";
    setup(target("reth", [{ id: "msgboard" }]), { gateways: [gw] });
    await screen.findByText("Message board");
    await waitFor(() => expect(api.getGateways).toHaveBeenCalled());
    expect(screen.queryByLabelText("Public reads")).toBeNull();
  });

  it("puts the Public radio back on the saved value when setModule fails", async () => {
    setup(target("reth", [{ id: "msgboard" }]), { gateways: [gateway(true)] });
    vi.mocked(api.setModule).mockRejectedValue(new Error("boom"));
    const local = (await screen.findByLabelText("Local only")) as HTMLInputElement;
    expect(local.checked).toBe(true);
    fireEvent.click(screen.getByLabelText("Public reads and posting"));
    await waitFor(() => expect(api.setModule).toHaveBeenCalled());
    await waitFor(() => expect(local.checked).toBe(true));
    expect((screen.getByLabelText("Public reads and posting") as HTMLInputElement).checked).toBe(false);
  });

  it("shows no Public radio for a module that offers no public exposure", async () => {
    setup(target("reth", [{ id: "msgboard" }]), { gateways: [gateway(true)] });
    cleanup();
    vi.mocked(api.getCatalog).mockResolvedValue({
      ...CATALOG,
      modules: CATALOG.modules!.map((m) => (m.id === "msgboard" ? { ...m, public: [] } : m)),
    });
    const qc = new QueryClient({ defaultOptions: { queries: { retry: false } } });
    render(
      <QueryClientProvider client={qc}>
        <ModulesSection targetId="t1" />
      </QueryClientProvider>,
    );
    await screen.findByText("Message board");
    await waitFor(() => expect(api.getGateways).toHaveBeenCalled());
    expect(screen.queryByLabelText("Public reads")).toBeNull();
  });

  it("always lets an on module be turned off, even when it no longer applies", async () => {
    setup(target("go-pulse", [{ id: "msgboard" }]));
    const btn = await screen.findByRole("button", { name: "Turn off Message board" });
    expect((btn as HTMLButtonElement).disabled).toBe(false);
    fireEvent.click(btn);
    await waitFor(() => expect(api.setModule).toHaveBeenCalledWith("t1", "msgboard", { enabled: false }));
  });

  it("says the gate is too old and keeps the toggle usable", async () => {
    setup(target("reth", [{ id: "msgboard" }]), { gateStale: true });
    await screen.findByText("Message board");
    await waitFor(() => expect(card("msgboard").textContent).toContain("Status needs a newer gate on this box"));
    expect((card("msgboard").querySelector("button.module-toggle") as HTMLButtonElement).disabled).toBe(false);
  });
});
