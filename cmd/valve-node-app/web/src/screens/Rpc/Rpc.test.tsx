import { render, screen, cleanup, waitFor, fireEvent, act, within } from "@testing-library/react";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { describe, it, expect, vi, beforeEach, afterEach } from "vitest";
import type { ReactNode } from "react";
import * as api from "../../api";
import type { GatewayView, GatewaysResponse, SetupEvent } from "../../api";
import { Rpc } from "./Rpc";

vi.mock("../../api", async () => {
  const actual = await vi.importActual<typeof import("../../api")>("../../api");
  return {
    ...actual,
    getGateways: vi.fn(),
    getGatewayTraffic: vi.fn(),
    getGatewayCapabilities: vi.fn(),
    getHost: vi.fn(),
    verifyGatewayTls: vi.fn(),
    trustGatewayCert: vi.fn(),
    gatewayAction: vi.fn(),
    putGatewayConfig: vi.fn(),
    provisionGateway: vi.fn(),
    streamSetup: vi.fn(),
    knownSet: vi.fn(),
    createGateway: vi.fn(),
  };
});

const STATUS = {
  ID: "c1",
  ContainerName: "erpc-default",
  State: "running" as const,
  Image: "erpc:latest",
  ImageID: "sha256:abc",
  ExitCode: 0,
  Platform: "linux/arm64",
  EnginePlatform: "linux/arm64",
  Emulated: false,
  Detail: "",
};

function makeGateway(): GatewayView {
  return {
    id: "default",
    label: "default",
    containerName: "erpc-default",
    placement: { targetId: "local", backend: "docker" },
    status: STATUS,
    docker: { present: true, reachable: true, flavor: "docker" },
    baseUrl: "https://valve.local",
    tls: {
      enabled: true,
      hostname: "valve.local",
      url: "https://valve.local",
      certSource: "internal",
      effectiveCertSource: "internal",
      rootCaPath: "/root.crt",
      suggestedHostname: "valve.local",
      status: STATUS,
    },
    networks: [
      {
        chainId: 1,
        name: "Ethereum",
        url: "https://valve.local/main/evm/1",
        path: "/main/evm/1",
        upstreams: [
          { id: "u1", kind: "external", endpoint: "wss://a", label: "a", local: true, recentOnly: false, actions: [] },
          { id: "u2", kind: "external", endpoint: "https://b", label: "b", local: false, recentOnly: false, actions: [] },
        ],
        knownSetSize: 2,
        serviceable: true,
      },
    ],
    actions: ["stop", "restart", "recreate", "wipe"],
    wipeDiscards: "the container and its config",
    config: {
      ProjectID: "main",
      BindAddr: "127.0.0.1",
      Port: 4000,
      Networks: [
        {
          ChainID: 1,
          Upstreams: [
            { ID: "u1", Kind: "external", Endpoint: "wss://a", Local: true, RecentOnly: false },
            { ID: "u2", Kind: "external", Endpoint: "https://b", Local: false, RecentOnly: false },
          ],
        },
      ],
      TLS: { Enabled: true, Hostname: "valve.local", CertSource: "internal", CertFile: "", KeyFile: "", HTTPSPort: 443, BindAddr: "", ImageRef: "" },
    },
  };
}

function response(over: Partial<GatewaysResponse> = {}): GatewaysResponse {
  return {
    gateways: [makeGateway()],
    targets: [{ id: "local", mode: "local", hasDevnet: false, hasNode: false }],
    sources: [],
    presets: [
      { chainId: 369, name: "PulseChain", devnet: false },
      { chainId: 1, name: "Ethereum", devnet: false },
    ],
    orphans: [],
    ...over,
  };
}

function wrapper({ children }: { children: ReactNode }) {
  const qc = new QueryClient({ defaultOptions: { queries: { retry: false }, mutations: { retry: false } } });
  return <QueryClientProvider client={qc}>{children}</QueryClientProvider>;
}

beforeEach(() => {
  vi.mocked(api.getGateways).mockReset();
  vi.mocked(api.getGatewayTraffic).mockReset().mockResolvedValue({ enabled: true, at: "now", since: "then", networks: [] });
  vi.mocked(api.getGatewayCapabilities).mockReset().mockResolvedValue({ at: "now", endpoints: [] });
  vi.mocked(api.getHost).mockReset().mockResolvedValue({ os: "darwin", arch: "arm64" });
  vi.mocked(api.verifyGatewayTls).mockReset();
  vi.mocked(api.trustGatewayCert).mockReset();
  vi.mocked(api.gatewayAction).mockReset().mockResolvedValue({ status: STATUS });
  vi.mocked(api.putGatewayConfig).mockReset().mockResolvedValue(makeGateway());
  vi.mocked(api.provisionGateway).mockReset();
  vi.mocked(api.streamSetup).mockReset();
  vi.mocked(api.knownSet).mockReset();
  vi.mocked(api.createGateway).mockReset();
});
afterEach(cleanup);

describe("Rpc — list states", () => {
  it("shows the no-machines empty state", async () => {
    vi.mocked(api.getGateways).mockResolvedValue(response({ gateways: [], targets: [] }));
    render(<Rpc />, { wrapper });
    expect(await screen.findByText(/No machines yet\./)).toBeInTheDocument();
  });

  it("shows the no-gateway empty state and an add-a-gateway button when a machine exists", async () => {
    vi.mocked(api.getGateways).mockResolvedValue(response({ gateways: [] }));
    render(<Rpc />, { wrapper });
    expect(await screen.findByText(/No gateway yet\./)).toBeInTheDocument();
    expect(screen.getByRole("button", { name: /Add a gateway/ })).toBeInTheDocument();
  });

  it("surfaces a read error", async () => {
    vi.mocked(api.getGateways).mockRejectedValue(new Error("nope"));
    render(<Rpc />, { wrapper });
    expect(await screen.findByText(/Could not read the gateways: nope/)).toBeInTheDocument();
  });

  it("renders the gateway, its chain and its dialable URL", async () => {
    vi.mocked(api.getGateways).mockResolvedValue(response());
    render(<Rpc />, { wrapper });
    expect(await screen.findByText("Ethereum")).toBeInTheDocument();
    expect(screen.getByText("evm:1")).toBeInTheDocument();
    expect(screen.getByText("https://valve.local/main/evm/1")).toBeInTheDocument();
    expect(screen.getByText("healthy")).toBeInTheDocument();
  });
});

describe("Rpc — create a gateway", () => {
  it("opens the form and fires createGateway", async () => {
    vi.mocked(api.getGateways).mockResolvedValue(response({ gateways: [] }));
    vi.mocked(api.createGateway).mockResolvedValue(makeGateway());
    render(<Rpc />, { wrapper });
    fireEvent.click(await screen.findByRole("button", { name: /Add a gateway/ }));
    fireEvent.click(await screen.findByRole("button", { name: "Create gateway" }));
    await waitFor(() =>
      expect(api.createGateway).toHaveBeenCalledWith(
        expect.objectContaining({ id: "default", placement: { targetId: "local", backend: "docker" } }),
      ),
    );
  });
});

describe("Rpc — lifecycle", () => {
  it("fires the restart action from Manage gateway", async () => {
    vi.mocked(api.getGateways).mockResolvedValue(response());
    render(<Rpc />, { wrapper });
    fireEvent.click(await screen.findByRole("button", { name: /Manage gateway/ }));
    fireEvent.click(await screen.findByRole("button", { name: "Restart" }));
    await waitFor(() => expect(api.gatewayAction).toHaveBeenCalledWith("default", "restart"));
  });

  it("streams provisioning progress on Re-create", async () => {
    vi.mocked(api.getGateways).mockResolvedValue(response());
    vi.mocked(api.provisionGateway).mockResolvedValue({ status: "accepted", targetId: "local" });
    let onEvent: ((ev: SetupEvent) => void) | null = null;
    vi.mocked(api.streamSetup).mockImplementation((_id, cb) => {
      onEvent = cb;
      return vi.fn();
    });
    render(<Rpc />, { wrapper });
    fireEvent.click(await screen.findByRole("button", { name: /Manage gateway/ }));
    fireEvent.click(await screen.findByRole("button", { name: "Re-create (apply config)" }));
    await waitFor(() => expect(api.streamSetup).toHaveBeenCalledWith("local", expect.any(Function)));
    act(() => onEvent!({ stepId: "preflight", line: "checking ports" }));
    expect(await screen.findByText(/preflight: checking ports/)).toBeInTheDocument();
  });
});

describe("Rpc — known-set picker", () => {
  it("reads valve's set and saves the selection", async () => {
    vi.mocked(api.getGateways).mockResolvedValue(response());
    vi.mocked(api.knownSet).mockResolvedValue({
      endpoints: [{ url: "https://new.example", provider: "prov", websocket: false, archive: true, alreadyAdded: false }],
      usingDefaultKey: true,
    });
    render(<Rpc />, { wrapper });
    fireEvent.click(await screen.findByRole("button", { name: "Details" }));
    fireEvent.click(await screen.findByRole("button", { name: "+ Endpoint" }));
    fireEvent.click(await screen.findByRole("button", { name: "Add valve's set…" }));
    await waitFor(() => expect(api.knownSet).toHaveBeenCalledWith("default", 1));
    fireEvent.click(await screen.findByRole("button", { name: "Add 1" }));
    await waitFor(() => expect(api.putGatewayConfig).toHaveBeenCalled());
    const cfg = vi.mocked(api.putGatewayConfig).mock.calls[0][1];
    expect(cfg.Networks?.find((n) => n.ChainID === 1)?.Upstreams.some((u) => u.Endpoint === "https://new.example")).toBe(true);
  });
});

describe("Rpc — TLS verify + trust", () => {
  it("runs the live verification and shows the report", async () => {
    vi.mocked(api.getGateways).mockResolvedValue(response());
    vi.mocked(api.verifyGatewayTls).mockResolvedValue({
      at: new Date().toISOString(),
      url: "https://valve.local",
      hostname: "valve.local",
      address: "127.0.0.1",
      assertions: [{ id: "handshake", title: "TLS handshake", why: "", status: "pass", detail: "ok" }],
      ok: true,
      subscriptionsOk: true,
      summary: "HTTPS is serving correctly",
    });
    render(<Rpc />, { wrapper });
    fireEvent.click(await screen.findByRole("button", { name: /Manage gateway/ }));
    fireEvent.click(await screen.findByRole("button", { name: "Settings" }));
    fireEvent.click(await screen.findByRole("button", { name: "Verify HTTPS now" }));
    await waitFor(() => expect(api.verifyGatewayTls).toHaveBeenCalledWith("default"));
    expect(await screen.findByText("HTTPS is serving correctly")).toBeInTheDocument();
    expect(screen.getByText("TLS handshake")).toBeInTheDocument();
  });

  it("installs the cert on this machine in one click", async () => {
    vi.mocked(api.getGateways).mockResolvedValue(response());
    vi.mocked(api.trustGatewayCert).mockResolvedValue({ ok: true, message: "Installed into the system keychain." });
    render(<Rpc />, { wrapper });
    const trustBtn = await screen.findByRole("button", { name: "Trust on this machine" });
    fireEvent.click(trustBtn);
    await waitFor(() => expect(api.trustGatewayCert).toHaveBeenCalledWith("default"));
    expect(await screen.findByText("Trusted — reload your wallet or browser.")).toBeInTheDocument();
  });

  it("renders cert-trust exactly once, at the gateway level, even with several chains", async () => {
    const gw = makeGateway();
    gw.networks = [
      ...(gw.networks ?? []),
      {
        chainId: 369,
        name: "PulseChain",
        url: "https://valve.local/main/evm/369",
        path: "/main/evm/369",
        upstreams: [
          { id: "u3", kind: "external", endpoint: "wss://c", label: "c", local: true, recentOnly: false, actions: [] },
        ],
        knownSetSize: 1,
        serviceable: true,
      },
    ];
    vi.mocked(api.getGateways).mockResolvedValue(response({ gateways: [gw] }));
    render(<Rpc />, { wrapper });
    expect(await screen.findByText("PulseChain")).toBeInTheDocument();
    // One control for the whole gateway, not one per chain — the base URL and
    // its trust control are host-level; a chain is only a path underneath it.
    expect(screen.getAllByRole("button", { name: "Trust on this machine" })).toHaveLength(1);
    expect(screen.queryByText(/Your wallet must trust this gateway's certificate first/)).not.toBeInTheDocument();
  });

  it("shows a failed trust's ranCommand as a full-width block, with a copy button", async () => {
    vi.mocked(api.getGateways).mockResolvedValue(response());
    vi.mocked(api.trustGatewayCert).mockResolvedValue({
      ok: false,
      message: "Could not install automatically.",
      ranCommand: 'sudo security add-trusted-cert -d -r trustRoot -k /Library/Keychains/System.keychain "/root.crt"',
    });
    render(<Rpc />, { wrapper });
    fireEvent.click(await screen.findByRole("button", { name: "Trust on this machine" }));
    await waitFor(() => expect(api.trustGatewayCert).toHaveBeenCalledWith("default"));
    expect(await screen.findByText("Could not install automatically.")).toBeInTheDocument();
    const cmd = await screen.findByText(
      'sudo security add-trusted-cert -d -r trustRoot -k /Library/Keychains/System.keychain "/root.crt"',
    );
    expect(cmd.tagName).toBe("CODE");
    expect(cmd.className).toContain("strip-cmd");
    expect(screen.getByRole("button", { name: "Copy command" })).toBeInTheDocument();
  });

  it("offers a calm Try again on a failed trust that re-runs the trust action", async () => {
    vi.mocked(api.getGateways).mockResolvedValue(response());
    // First attempt fails (detached-launch osascript, no GUI prompt); the retry
    // succeeds because the operator trusted the root by hand in the meantime.
    vi.mocked(api.trustGatewayCert)
      .mockResolvedValueOnce({
        ok: false,
        message: "macOS needs a GUI login session to prompt for authorization.",
        ranCommand: "sudo security add-trusted-cert -d -r trustRoot -k /Library/Keychains/System.keychain '/root.crt'",
      })
      .mockResolvedValueOnce({ ok: true, message: "Already trusted." });
    render(<Rpc />, { wrapper });
    fireEvent.click(await screen.findByRole("button", { name: "Trust on this machine" }));
    await waitFor(() => expect(api.trustGatewayCert).toHaveBeenCalledTimes(1));
    // The failure reads as fixable, with an explicit retry beside the command.
    const retry = await screen.findByRole("button", { name: "Try again" });
    fireEvent.click(retry);
    await waitFor(() => expect(api.trustGatewayCert).toHaveBeenCalledTimes(2));
    expect(api.trustGatewayCert).toHaveBeenLastCalledWith("default");
    expect(await screen.findByText("Trusted — reload your wallet or browser.")).toBeInTheDocument();
  });
});

describe("Rpc — config edit", () => {
  it("saves an edited listen port", async () => {
    vi.mocked(api.getGateways).mockResolvedValue(response());
    render(<Rpc />, { wrapper });
    fireEvent.click(await screen.findByRole("button", { name: /Manage gateway/ }));
    fireEvent.click(await screen.findByRole("button", { name: "Settings" }));
    const portInput = (await screen.findByText("Listen port")).parentElement!.querySelector("input")!;
    fireEvent.change(portInput, { target: { value: "5000" } });
    fireEvent.click(screen.getByRole("button", { name: "Save settings" }));
    await waitFor(() => expect(api.putGatewayConfig).toHaveBeenCalled());
    expect(vi.mocked(api.putGatewayConfig).mock.calls[0][1].Port).toBe(5000);
  });
});

describe("Rpc — orphans", () => {
  it("shows a loose orphan banner with its docker rm command", async () => {
    vi.mocked(api.getGateways).mockResolvedValue(
      response({ gateways: [], orphans: [{ containerName: "erpc-old", targetId: "box9", mergedInto: "default" }] }),
    );
    render(<Rpc />, { wrapper });
    const banner = (await screen.findByText(/erpc-old is still running on box9/)).closest(".strip")!;
    expect(within(banner as HTMLElement).getByText("docker rm -f erpc-old")).toBeInTheDocument();
  });
});
