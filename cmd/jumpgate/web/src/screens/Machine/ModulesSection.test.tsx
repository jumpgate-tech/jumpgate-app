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
    startSetup: vi.fn(),
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

function setup(
  t: Target,
  opts: { gateways?: GatewayView[]; gateStale?: boolean; fleet?: Target[]; catalog?: Catalog; statuses?: api.ModuleStatus[] } = {},
) {
  vi.mocked(api.listTargets).mockResolvedValue(opts.fleet ?? [t]);
  vi.mocked(api.getCatalog).mockResolvedValue(opts.catalog ?? CATALOG);
  vi.mocked(api.listModules).mockResolvedValue({
    modules: opts.statuses ?? [],
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
  vi.mocked(api.startSetup).mockReset();
});
afterEach(cleanup);

const ENODE = "enode://" + "ab12".repeat(32) + "@203.0.113.7:30303";

function bootnodeFleet(): Target[] {
  const box: Target = {
    id: "t1",
    mode: "ssh",
    ssh: { Host: "boot.example", User: "root", KeyPath: "/k" },
    modules: [{ id: "el-bootnode", chainId: 369 }],
  };
  const nodeA: Target = {
    id: "a",
    mode: "ssh",
    ssh: { Host: "a.example", User: "root", KeyPath: "/k" },
    wire: { ChainID: 369, ExecID: "reth", BeaconID: "lighthouse-pulse", DataDir: "/d", JWTPath: "/d/jwt", Archive: false },
  };
  const nodeB: Target = {
    id: "b",
    mode: "ssh",
    ssh: { Host: "b.example", User: "root", KeyPath: "/k" },
    wire: {
      ChainID: 369,
      ExecID: "go-pulse",
      BeaconID: "prysm-pulse",
      DataDir: "/d",
      JWTPath: "/d/jwt",
      Archive: false,
      Bootnodes: ["enr:-abc"],
    },
  };
  const otherChain: Target = {
    id: "c",
    mode: "ssh",
    ssh: { Host: "c.example", User: "root", KeyPath: "/k" },
    wire: { ChainID: 1, ExecID: "reth", BeaconID: "lighthouse", DataDir: "/d", JWTPath: "/d/jwt", Archive: false },
  };
  return [box, nodeA, nodeB, otherChain];
}

const BOOT_STATUS: api.ModuleStatus = {
  id: "el-bootnode",
  served: false,
  active: true,
  accepting: false,
  count: 0,
  size: 0,
  headBlock: 0,
  identity: ENODE,
};

describe("ModulesSection", () => {
  it("shows a running bootnode's enode with a copy action and its note", async () => {
    const writeText = vi.fn().mockResolvedValue(undefined);
    Object.defineProperty(navigator, "clipboard", { value: { writeText }, configurable: true });
    setup(bootnodeFleet()[0], {
      statuses: [{ ...BOOT_STATUS, note: "Set the box's public IP (ExtIP) on the bootnode." }],
    });
    await waitFor(() => expect(card("el-bootnode").querySelector("code.module-identity")?.textContent).toBe(ENODE));
    expect(card("el-bootnode").textContent).toContain("Set the box's public IP (ExtIP) on the bootnode.");
    fireEvent.click(screen.getByRole("button", { name: "Copy enode" }));
    await waitFor(() => expect(writeText).toHaveBeenCalledWith(ENODE));
    expect(screen.queryByRole("button", { name: /Use on my other/ })).toBeNull();
    expect(card("el-bootnode").textContent).toContain("Set the box's public IP before your other nodes can use this bootnode.");
  });

  it("puts a failed copy's hint on its own line", async () => {
    const writeText = vi.fn().mockRejectedValue(new Error("denied"));
    Object.defineProperty(navigator, "clipboard", { value: { writeText }, configurable: true });
    setup(bootnodeFleet()[0], { statuses: [BOOT_STATUS] });
    const btn = await screen.findByRole("button", { name: "Copy enode" });
    fireEvent.click(btn);
    await waitFor(() => expect(card("el-bootnode").querySelector(".module-copy-failed")?.textContent).toBe("Copy failed; select the text instead."));
    expect(btn.textContent).toBe("Copy");
  });

  it("offers no fleet action for an enode that advertises 0.0.0.0", async () => {
    const fleet = bootnodeFleet();
    setup(fleet[0], { fleet, statuses: [{ ...BOOT_STATUS, identity: ENODE.replace("203.0.113.7", "0.0.0.0") }] });
    await waitFor(() => expect(card("el-bootnode").querySelector("code.module-identity")).toBeTruthy());
    expect(screen.queryByRole("button", { name: /Use on my other/ })).toBeNull();
    expect(card("el-bootnode").textContent).toContain("Set the box's public IP before your other nodes can use this bootnode.");
  });

  it("lists a node's saved bootnodes and removes one, whatever runs them", async () => {
    const node = bootnodeFleet()[2];
    node.id = "t1";
    node.wire!.Bootnodes = ["enr:-abc", ENODE];
    vi.mocked(api.startSetup).mockResolvedValue({ status: "started" });
    setup(node);
    const list = await screen.findByRole("list", { name: "Bootnodes" });
    expect(list.textContent).toContain("enr:-abc");
    expect(list.textContent).toContain(ENODE);
    fireEvent.click(screen.getByRole("button", { name: `Remove ${ENODE}` }));
    // It restarts a client, so it asks first.
    const dialog = await screen.findByRole("dialog");
    expect(dialog.textContent).toContain("This restarts go-pulse on b.example.");
    expect(api.startSetup).not.toHaveBeenCalled();
    fireEvent.click(screen.getByRole("button", { name: "Remove bootnode" }));
    await waitFor(() => expect(api.startSetup).toHaveBeenCalledWith("t1", { ...node.wire, Bootnodes: ["enr:-abc"] }));
    expect(await screen.findByText("b.example: setup started, go-pulse restarts.")).toBeTruthy();
    // The removed entry is gone from the list at once, and a second
    // removal before the targets refetch does not send it back.
    await waitFor(() => expect(list.textContent).not.toContain(ENODE));
    vi.mocked(api.startSetup).mockRejectedValueOnce(new Error("busy"));
    fireEvent.click(screen.getByRole("button", { name: "Remove enr:-abc" }));
    fireEvent.click(await screen.findByRole("button", { name: "Remove bootnode" }));
    expect(await screen.findByText("b.example: busy")).toBeTruthy();
    expect(api.startSetup).toHaveBeenLastCalledWith("t1", { ...node.wire, Bootnodes: [] });
    // A failed removal keeps the entry listed.
    expect(list.textContent).toContain("enr:-abc");
  });

  it("stops using a bootnode on the nodes that list it", async () => {
    const fleet = bootnodeFleet();
    fleet[2].wire!.Bootnodes = ["enr:-abc", ENODE];
    vi.mocked(api.startSetup).mockResolvedValue({ status: "started" });
    setup(fleet[0], { fleet, statuses: [BOOT_STATUS] });
    fireEvent.click(await screen.findByRole("button", { name: "Stop using on my chain 369 nodes" }));
    const dialog = await screen.findByRole("dialog");
    expect(dialog.textContent).toContain("a.example: does not use this bootnode.");
    expect(dialog.textContent).toContain("Restarts go-pulse on b.example.");
    fireEvent.click(screen.getByRole("button", { name: "Stop using bootnode" }));
    await waitFor(() => expect(api.startSetup).toHaveBeenCalledTimes(1));
    expect(api.startSetup).toHaveBeenCalledWith("b", { ...fleet[2].wire, Bootnodes: ["enr:-abc"] });
    await waitFor(() => expect(dialog.textContent).toContain("b.example: setup started, go-pulse restarts."));
  });

  it("offers a running bootnode to the fleet's other nodes on its chain", async () => {
    const fleet = bootnodeFleet();
    vi.mocked(api.startSetup).mockResolvedValue({ status: "started" });
    setup(fleet[0], {
      fleet,
      statuses: [BOOT_STATUS],
      catalog: {
        ...CATALOG,
        networks: [{ ChainID: 369, Name: "PulseChain", CheckpointURL: "", ExecClients: [], BeaconClients: [], LearnURL: "", SnapshotSizeTB: 0, SyncLabel: "", GenesisSyncLabel: "" }],
      },
    });
    fireEvent.click(await screen.findByRole("button", { name: "Use on my other PulseChain nodes" }));
    const dialog = await screen.findByRole("dialog");
    expect(dialog.textContent).toContain("a.example");
    expect(dialog.textContent).toContain("b.example");
    expect(dialog.textContent).not.toContain("c.example");
    expect(dialog.textContent).toContain("Restarts reth on a.example.");
    fireEvent.click(screen.getByRole("button", { name: "Use bootnode" }));
    await waitFor(() => expect(api.startSetup).toHaveBeenCalledTimes(2));
    expect(api.startSetup).toHaveBeenCalledWith("a", { ...fleet[1].wire, Bootnodes: [ENODE] });
    expect(api.startSetup).toHaveBeenCalledWith("b", { ...fleet[2].wire, Bootnodes: ["enr:-abc", ENODE] });
    await waitFor(() => expect(dialog.textContent).toContain("a.example: setup started, reth restarts."));
    expect(dialog.textContent).toContain("b.example: setup started, go-pulse restarts.");
  });

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

  it("turns the watch on with its endpoints and webhook, and no chain", async () => {
    setup(target());
    await screen.findByText("Services on this box");
    fireEvent.click(card("watch").querySelector("button.module-toggle")!);
    fireEvent.change(await screen.findByLabelText("Endpoints to watch"), { target: { value: "gw=https://rpc.example\n" } });
    fireEvent.change(screen.getByLabelText("Webhook"), { target: { value: "http://ntfy.sh/t" } });
    expect(screen.getByText("The webhook must be an https:// URL.")).toBeTruthy();
    fireEvent.change(screen.getByLabelText("Webhook"), { target: { value: "https://ntfy.sh/t" } });
    expect(screen.getByText(/Off by default\. HTTPS only\./)).toBeTruthy();
    fireEvent.click(screen.getByRole("button", { name: "Turn on" }));
    await waitFor(() =>
      expect(api.setModule).toHaveBeenCalledWith("t1", "watch", {
        enabled: true,
        upstreams: ["gw=https://rpc.example"],
        webhook: "https://ntfy.sh/t",
      }),
    );
  });

  it("edits a running watch without ever showing or resending the saved webhook", async () => {
    const t = { ...target(), modules: [{ id: "watch", upstreams: ["gw=https://rpc.example"] }], watchWebhookSet: true };
    setup(t);
    await screen.findByText("Services on this box");
    const area = (await screen.findByLabelText("Endpoints to watch")) as HTMLTextAreaElement;
    expect(area.value).toBe("gw=https://rpc.example");
    expect((screen.getByLabelText("Webhook") as HTMLInputElement).value).toBe("");
    fireEvent.change(area, { target: { value: "gw=https://rpc.example\nb=https://b.example" } });
    fireEvent.click(screen.getByRole("button", { name: "Save watch settings" }));
    await waitFor(() =>
      expect(api.setModule).toHaveBeenCalledWith("t1", "watch", {
        enabled: true,
        upstreams: ["gw=https://rpc.example", "b=https://b.example"],
      }),
    );
    fireEvent.click(screen.getByLabelText(/Remove the saved webhook/));
    fireEvent.click(screen.getByRole("button", { name: "Save watch settings" }));
    await waitFor(() =>
      expect(api.setModule).toHaveBeenLastCalledWith("t1", "watch", {
        enabled: true,
        upstreams: ["gw=https://rpc.example", "b=https://b.example"],
        clearWebhook: true,
      }),
    );
  });
});

describe("Checkpointz", () => {
  const PULSE: Catalog = {
    ...CATALOG,
    networks: [{ ChainID: 369, Name: "PulseChain", CheckpointURL: "", ExecClients: [], BeaconClients: [], LearnURL: "", SnapshotSizeTB: 0, SyncLabel: "", GenesisSyncLabel: "" }],
  };
  const SERVING: api.ModuleStatus = {
    id: "checkpointz", instance: "checkpointz-369", served: true, active: true, accepting: false, count: 0, size: 0, headBlock: 4123,
  };
  const URL = "http://10.0.0.5:5555";

  function fleet(bindAddr?: string): Target[] {
    const wire = (exec: string, extra: Partial<api.WireConfig> = {}): api.WireConfig => ({
      ChainID: 369, ExecID: exec, BeaconID: "lighthouse-pulse", DataDir: "/d", JWTPath: "/d/jwt", Archive: false, ...extra,
    });
    return [
      {
        id: "t1", mode: "ssh", ssh: { Host: "cp.example", User: "root", KeyPath: "/k" }, wire: wire("reth"),
        modules: [{ id: "checkpointz", chainId: 369, bindAddr, upstreams: ["http://127.0.0.1:5052"] }],
      },
      { id: "a", mode: "ssh", ssh: { Host: "a.example", User: "root", KeyPath: "/k" }, wire: wire("reth") },
      { id: "b", mode: "ssh", ssh: { Host: "b.example", User: "root", KeyPath: "/k" }, wire: wire("go-pulse", { CheckpointURL: URL }) },
      { id: "c", mode: "ssh", ssh: { Host: "c.example", User: "root", KeyPath: "/k" }, wire: { ...wire("reth"), ChainID: 943 } },
    ];
  }

  it("sends a listen address when one is given", async () => {
    setup(target());
    await screen.findByText("Services on this box");
    fireEvent.click(card("checkpointz").querySelector("button.module-toggle")!);
    fireEvent.change(await screen.findByLabelText("Listen address"), { target: { value: "10.0.0.5" } });
    fireEvent.click(screen.getByRole("button", { name: "Turn on" }));
    await waitFor(() =>
      expect(api.setModule).toHaveBeenCalledWith("t1", "checkpointz", { enabled: true, chainId: 369, bindAddr: "10.0.0.5" }),
    );
  });

  it("sets the checkpoint URL on the fleet's other nodes on its chain, after a confirm", async () => {
    const f = fleet("10.0.0.5");
    vi.mocked(api.startSetup).mockResolvedValue({ status: "started" });
    setup(f[0], { fleet: f, statuses: [SERVING], catalog: PULSE });
    expect(await screen.findByText("Serving · finalized epoch 4123")).toBeTruthy();
    fireEvent.click(await screen.findByRole("button", { name: "Use as checkpoint for my other PulseChain nodes" }));
    const dialog = await screen.findByRole("dialog");
    expect(dialog.textContent).toContain(URL);
    expect(dialog.textContent).toContain("Restarts lighthouse-pulse on a.example.");
    expect(dialog.textContent).toContain("b.example: already uses this checkpoint.");
    expect(dialog.textContent).not.toContain("c.example");
    expect(api.startSetup).not.toHaveBeenCalled();
    fireEvent.click(screen.getByRole("button", { name: "Use checkpoint" }));
    await waitFor(() => expect(api.startSetup).toHaveBeenCalledTimes(1));
    expect(api.startSetup).toHaveBeenCalledWith("a", { ...f[1].wire, CheckpointURL: URL });
    await waitFor(() => expect(dialog.textContent).toContain("a.example: setup started, lighthouse-pulse restarts."));
  });

  it("offers nothing to other boxes from a loopback Checkpointz", async () => {
    const f = fleet();
    setup(f[0], { fleet: f, statuses: [SERVING], catalog: PULSE });
    await screen.findByText("Serving · finalized epoch 4123");
    expect(screen.queryByRole("button", { name: /Use as checkpoint/ })).toBeNull();
    expect(screen.getByText("This Checkpointz serves only this box. Turn it off and on with a private listen address to let your other boxes use it.")).toBeTruthy();
  });

  it("offers nothing until it serves a finalized checkpoint", async () => {
    const f = fleet("10.0.0.5");
    setup(f[0], { fleet: f, statuses: [{ ...SERVING, served: false }], catalog: PULSE });
    await screen.findByText("Running");
    expect(screen.queryByRole("button", { name: /Use as checkpoint/ })).toBeNull();
  });
});
