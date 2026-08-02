import { render, screen, cleanup, waitFor, fireEvent } from "@testing-library/react";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { describe, it, expect, vi, beforeEach, afterEach } from "vitest";
import type { ReactNode } from "react";
import * as api from "../../api";
import type { GatewayAnalytics } from "../../api";
import { Panel } from "./Panel";
import { makeGateway } from "./fixtures";

vi.mock("../../api", async () => {
  const actual = await vi.importActual<typeof import("../../api")>("../../api");
  return {
    ...actual,
    getGateways: vi.fn(),
    getGatewayAnalytics: vi.fn(),
    getGatewayCapabilities: vi.fn(),
    verifyGatewayTls: vi.fn(),
  };
});

const EMPTY_ANALYTICS: GatewayAnalytics = {
  enabled: true,
  at: "now",
  since: "then",
  networks: [],
  endpoints: [],
};

function wrapper({ children }: { children: ReactNode }) {
  const qc = new QueryClient({
    defaultOptions: { queries: { retry: false }, mutations: { retry: false } },
  });
  return <QueryClientProvider client={qc}>{children}</QueryClientProvider>;
}

beforeEach(() => {
  vi.mocked(api.getGateways).mockReset();
  vi.mocked(api.getGatewayAnalytics).mockReset().mockResolvedValue(EMPTY_ANALYTICS);
  vi.mocked(api.getGatewayCapabilities).mockReset().mockResolvedValue({ at: "now", endpoints: [] });
});
afterEach(cleanup);

describe("Panel", () => {
  it("shows the one-click setup hero when the fleet has no gateway", async () => {
    vi.mocked(api.getGateways).mockResolvedValue({ gateways: [], targets: [], sources: [], presets: [] });
    render(<Panel />, { wrapper });
    expect(await screen.findByText("Set up my endpoint")).toBeInTheDocument();
  });

  it("renders the network list for the primary (local) gateway", async () => {
    vi.mocked(api.getGateways).mockResolvedValue({
      gateways: [makeGateway()],
      targets: [],
      sources: [],
      presets: [],
    });
    render(<Panel />, { wrapper });
    expect(await screen.findByText("Ethereum")).toBeInTheDocument();
    expect(screen.getByText("PulseChain")).toBeInTheDocument();
    expect(screen.getByText("Running")).toBeInTheDocument();
  });

  it("drills from the list into a network's detail view on row click", async () => {
    vi.mocked(api.getGateways).mockResolvedValue({
      gateways: [makeGateway()],
      targets: [],
      sources: [],
      presets: [],
    });
    render(<Panel />, { wrapper });
    fireEvent.click(await screen.findByText("Ethereum"));
    // The network detail shows the gateway URL band and a Remove action.
    expect(await screen.findByText("https://valve.local/1")).toBeInTheDocument();
    expect(screen.getByText("Remove network")).toBeInTheDocument();
    // Capabilities probe fires lazily on entering the detail view.
    await waitFor(() => expect(api.getGatewayCapabilities).toHaveBeenCalledWith("default", false));
  });

  it("opens the settings sheet from the gear", async () => {
    vi.mocked(api.getGateways).mockResolvedValue({
      gateways: [makeGateway()],
      targets: [],
      sources: [],
      presets: [],
    });
    render(<Panel />, { wrapper });
    await screen.findByText("Ethereum");
    fireEvent.click(screen.getByLabelText("Settings"));
    expect(await screen.findByText("Appearance")).toBeInTheDocument();
    // fixture lists wipe → Danger zone shows.
    expect(screen.getByText("Danger zone")).toBeInTheDocument();
  });
});
