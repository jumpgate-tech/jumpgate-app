import { render, screen, cleanup, fireEvent, waitFor, within } from "@testing-library/react";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { describe, it, expect, vi, beforeEach, afterEach } from "vitest";
import type { ReactNode } from "react";
import * as api from "../../api";
import type { VpnServerView, VpnStatus, VpnView } from "../../api";
import { PrivateAccess } from "./PrivateAccess";

// Mock the api module the hooks call through — the screen itself only imports
// types from it, so overriding these functions is enough to drive every hook.
vi.mock("../../api", async () => {
  const actual = await vi.importActual<typeof import("../../api")>("../../api");
  return {
    ...actual,
    getVpnServers: vi.fn(),
    getVpns: vi.fn(),
    listTargets: vi.fn(),
    getVpnServerStatus: vi.fn(),
    getVpnStatus: vi.fn(),
    provisionVpnServer: vi.fn(),
    enrollVpnDevice: vi.fn(),
    revokeVpnDevice: vi.fn(),
    deleteVpnServer: vi.fn(),
    saveVpn: vi.fn(),
  };
});

const STATUS_DOWN: VpnStatus = {
  id: "wg0",
  up: false,
  interface: "jumpgate0",
  provider: "",
  addresses: [],
  peers: 0,
  handshaked: false,
  lastHandshake: 0,
};

function serverWith(overrides: Partial<VpnServerView>): VpnServerView {
  return {
    id: "wg0",
    targetId: "hetzner-fsn-1",
    interface: "jumpgate0",
    address: "10.9.0.1/24",
    listenPort: 51820,
    publicKey: "hV9zLmA4pQxN7dK2eRwYtB1cJ6sF0gU3iO8nM5vXaZk=",
    endpoint: "vpn.example.com:51820",
    peers: [],
    ...overrides,
  };
}

function renderScreen() {
  const queryClient = new QueryClient({
    defaultOptions: { queries: { retry: false }, mutations: { retry: false } },
  });
  function wrapper({ children }: { children: ReactNode }) {
    return <QueryClientProvider client={queryClient}>{children}</QueryClientProvider>;
  }
  return render(<PrivateAccess />, { wrapper });
}

beforeEach(() => {
  vi.mocked(api.getVpnServers).mockReset().mockResolvedValue([]);
  vi.mocked(api.getVpns).mockReset().mockResolvedValue([]);
  vi.mocked(api.listTargets).mockReset().mockResolvedValue([]);
  vi.mocked(api.getVpnServerStatus).mockReset().mockResolvedValue(STATUS_DOWN);
  vi.mocked(api.getVpnStatus).mockReset().mockResolvedValue(STATUS_DOWN);
  vi.mocked(api.provisionVpnServer).mockReset();
  vi.mocked(api.enrollVpnDevice).mockReset();
  vi.mocked(api.revokeVpnDevice).mockReset().mockResolvedValue(undefined);
  vi.mocked(api.deleteVpnServer).mockReset().mockResolvedValue(undefined);
  vi.mocked(api.saveVpn).mockReset();
});
afterEach(cleanup);

describe("PrivateAccess", () => {
  it("renders the two-mode entry picker", async () => {
    renderScreen();
    expect(await screen.findByText("Set up on a device")).toBeInTheDocument();
    expect(screen.getByText("Provide credentials")).toBeInTheDocument();
    expect(screen.getByText("recommended")).toBeInTheDocument();
    expect(screen.getByText("simplest")).toBeInTheDocument();
  });

  it("provisioning calls provisionVpnServer with the typed id", async () => {
    vi.mocked(api.provisionVpnServer).mockResolvedValue({
      server: serverWith({ id: "wg0" }),
      firewallHint: "ufw allow 51820/udp",
      endpointConfigured: true,
    });

    renderScreen();
    fireEvent.click(await screen.findByText("Set up on a device"));

    const nameInput = screen.getByPlaceholderText("wg0");
    fireEvent.change(nameInput, { target: { value: "wg0" } });
    fireEvent.click(screen.getByText("Provision server"));

    await waitFor(() =>
      expect(api.provisionVpnServer).toHaveBeenCalledWith(expect.objectContaining({ id: "wg0" })),
    );
  });

  it("a server with endpointConfigured:false (endpoint '') disables enroll and shows the endpoint input", async () => {
    vi.mocked(api.getVpnServers).mockResolvedValue([serverWith({ endpoint: "" })]);

    renderScreen();

    // The amber state and its host input are present.
    expect(await screen.findByPlaceholderText("vpn.example.com or 203.0.113.7")).toBeInTheDocument();
    expect(screen.getByText("Set endpoint")).toBeInTheDocument();
    expect(screen.getByText("endpoint not set")).toBeInTheDocument();
    // Enroll is disabled until an endpoint exists.
    expect(screen.getByText("Enroll")).toBeDisabled();
  });

  it("enrolling surfaces the returned config with Copy and Download", async () => {
    const config =
      "[Interface]\nPrivateKey = XXX\nAddress = 10.9.0.2/32\n\n[Peer]\nPublicKey = YYY\nAllowedIPs = 10.9.0.0/24";
    vi.mocked(api.getVpnServers).mockResolvedValue([serverWith({ endpoint: "vpn.example.com:51820" })]);
    vi.mocked(api.enrollVpnDevice).mockResolvedValue({
      name: "mikes-iphone",
      publicKey: "xK2mQ8sPub=",
      allowedIp: "10.9.0.2/32",
      config,
    });

    renderScreen();
    await screen.findByText("Enroll a device");

    fireEvent.change(screen.getByPlaceholderText("mikes-iphone"), { target: { value: "mikes-iphone" } });
    fireEvent.click(screen.getByText("Enroll"));

    await waitFor(() =>
      expect(api.enrollVpnDevice).toHaveBeenCalledWith(
        "wg0",
        expect.objectContaining({ name: "mikes-iphone", fullTunnel: false }),
      ),
    );

    // The one-time handoff shows the returned config plus Copy/Download.
    const handoff = (await screen.findByText(/PrivateKey = XXX/)).closest(".card") as HTMLElement;
    expect(within(handoff).getByText("Copy")).toBeInTheDocument();
    expect(within(handoff).getByText("Download")).toBeInTheDocument();
    expect(within(handoff).getByText("I saved it — close")).toBeInTheDocument();
  });

  it("revoke calls through after a confirmed window.confirm", async () => {
    const confirmSpy = vi.spyOn(window, "confirm").mockReturnValue(true);
    vi.mocked(api.getVpnServers).mockResolvedValue([
      serverWith({
        peers: [{ name: "old-ipad", publicKey: "pT5nR2yPublicKeyValue=", allowedIp: "10.9.0.4/32" }],
      }),
    ]);

    renderScreen();

    const revokeBtn = await screen.findByText("Revoke");
    fireEvent.click(revokeBtn);

    expect(confirmSpy).toHaveBeenCalled();
    await waitFor(() =>
      expect(api.revokeVpnDevice).toHaveBeenCalledWith("wg0", "pT5nR2yPublicKeyValue="),
    );
    confirmSpy.mockRestore();
  });

  it("does not revoke when window.confirm is declined", async () => {
    const confirmSpy = vi.spyOn(window, "confirm").mockReturnValue(false);
    vi.mocked(api.getVpnServers).mockResolvedValue([
      serverWith({
        peers: [{ name: "old-ipad", publicKey: "pT5nR2yPublicKeyValue=", allowedIp: "10.9.0.4/32" }],
      }),
    ]);

    renderScreen();
    fireEvent.click(await screen.findByText("Revoke"));

    expect(confirmSpy).toHaveBeenCalled();
    expect(api.revokeVpnDevice).not.toHaveBeenCalled();
    confirmSpy.mockRestore();
  });

  it("provide-credentials mode saves a BYO overlay", async () => {
    vi.mocked(api.saveVpn).mockResolvedValue({} as VpnView);

    renderScreen();
    fireEvent.click(await screen.findByText("Provide credentials"));

    fireEvent.change(screen.getByPlaceholderText("proton-nl-42"), { target: { value: "proton-nl-42" } });
    const textarea = screen.getByRole("textbox", { name: /WireGuard .conf/ });
    fireEvent.change(textarea, { target: { value: "[Interface]\nPrivateKey = zzz" } });
    fireEvent.click(screen.getByText("Add overlay"));

    await waitFor(() =>
      expect(api.saveVpn).toHaveBeenCalledWith(
        expect.objectContaining({ id: "proton-nl-42", config: "[Interface]\nPrivateKey = zzz" }),
      ),
    );
  });

  it("renders an existing BYO overlay with an up/down switch and Remove", async () => {
    vi.mocked(api.getVpns).mockResolvedValue([
      {
        id: "proton-nl-42",
        provider: "proton",
        interface: "jumpgate1",
        targetId: "",
        autostart: true,
        configured: true,
        valid: true,
        endpoints: ["nl-42.protonvpn.net:51820"],
        overlay: ["10.2.0.2/32"],
        peers: 1,
      },
    ]);

    renderScreen();

    const card = (await screen.findByText("proton-nl-42")).closest(".card") as HTMLElement;
    expect(within(card).getByText("nl-42.protonvpn.net:51820")).toBeInTheDocument();
    expect(within(card).getByRole("switch")).toBeInTheDocument();
    expect(within(card).getByText("Remove")).toBeInTheDocument();
  });
});
