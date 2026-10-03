import { render, screen, cleanup, fireEvent, waitFor } from "@testing-library/react";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { MemoryRouter, Routes, Route, useNavigate } from "react-router-dom";
import { describe, it, expect, vi, beforeEach, afterEach } from "vitest";
import type { ReactNode } from "react";
import * as api from "../../api";
import type { GatewayAnalytics, GatewayView, Latency } from "../../api";
import { Analytics } from "./Analytics";

vi.mock("../../api", async () => {
  const actual = await vi.importActual<typeof import("../../api")>("../../api");
  return {
    ...actual,
    getGateways: vi.fn(),
    getGatewayAnalytics: vi.fn(),
  };
});

const STATUS = {
  ID: "gw-1",
  ContainerName: "erpc-gw-1",
  State: "running" as const,
  Image: "erpc:latest",
  ImageID: "sha256:abc",
  ExitCode: 0,
  Platform: "linux/amd64",
  EnginePlatform: "linux/amd64",
  Emulated: false,
  Detail: "",
};

function makeGateway(id: string, label = id): GatewayView {
  return {
    id,
    label,
    containerName: `erpc-${id}`,
    placement: { targetId: "local", backend: "docker" },
    status: STATUS,
    docker: { present: true, reachable: true, flavor: "docker" },
    baseUrl: "http://localhost:4000",
    tls: { enabled: false, status: STATUS },
    networks: [],
    actions: [],
    wipeDiscards: "the container and its config",
    config: { ProjectID: id, BindAddr: "0.0.0.0", Port: 4000, Networks: [] },
  };
}

const EMPTY_LATENCY: Latency = { count: 0, mean: null, buckets: [] };

const ANALYTICS_ON: GatewayAnalytics = {
  enabled: true,
  at: "2026-08-01T12:00:00Z",
  since: "2026-08-01T10:00:00Z",
  networks: [
    {
      chainId: 1,
      name: "Ethereum",
      received: 100,
      answered: 90,
      unattributed: 5,
      failed: 5,
      methods: [
        {
          method: "eth_call",
          count: 90,
          mean: 0.02,
          buckets: [
            { le: "0.05", count: 80 },
            { le: "0.5", count: 90 },
            { le: "5", count: 90 },
            { le: "30", count: 90 },
            { le: "+Inf", count: 90 },
          ],
        },
      ],
      endpoints: [
        {
          upstream: "https://rpc1.example.com",
          count: 90,
          mean: 0.02,
          buckets: [
            { le: "0.05", count: 80 },
            { le: "0.5", count: 90 },
            { le: "5", count: 90 },
            { le: "30", count: 90 },
            { le: "+Inf", count: 90 },
          ],
        },
      ],
      cached: { count: 5, mean: 0.0004, buckets: [] },
      failedLatency: { count: 5, mean: 30, buckets: [] },
    },
  ],
  endpoints: [
    {
      upstream: "https://rpc1.example.com",
      chainId: 1,
      configured: true,
      requests: 500,
      errors: [{ class: "timeout", severity: "warn", method: "eth_call", count: 3 }],
      scored: true,
      score: 0.987,
      position: 0,
      primarySwitches: 2,
      excludedSeconds: 0,
      headLag: 0,
      finalizationLag: 0,
      latestBlock: 123,
    },
  ],
};

function renderAt(id = "gw-1") {
  const queryClient = new QueryClient({
    defaultOptions: { queries: { retry: false }, mutations: { retry: false } },
  });
  function wrapper({ children }: { children: ReactNode }) {
    return (
      <QueryClientProvider client={queryClient}>
        <MemoryRouter initialEntries={[`/analytics/${id}`]}>{children}</MemoryRouter>
      </QueryClientProvider>
    );
  }
  return render(
    <Routes>
      <Route path="/analytics/:id" element={<Analytics />} />
    </Routes>,
    { wrapper },
  );
}

beforeEach(() => {
  vi.mocked(api.getGateways).mockReset();
  vi.mocked(api.getGatewayAnalytics).mockReset();
});
afterEach(cleanup);

describe("Analytics", () => {
  it("shows Loading… while the gateway list is in flight", () => {
    vi.mocked(api.getGateways).mockReturnValue(new Promise(() => {}));

    renderAt();

    expect(screen.getByText("Loading…")).toBeInTheDocument();
  });

  it("shows a generic top-level error when the gateway list fails to load", async () => {
    vi.mocked(api.getGateways).mockRejectedValue(new Error("network down"));

    renderAt();

    expect(await screen.findByText("Analytics")).toBeInTheDocument();
    expect(screen.getByText("network down")).toBeInTheDocument();
    expect(screen.getByText("← Back to RPC")).toHaveAttribute("href", "#/rpc");
    // No gateway-specific header — nothing about the gateway is confirmed yet.
    expect(screen.queryByText(/Analytics:/)).not.toBeInTheDocument();
  });

  it("shows a not-found message when no gateway matches the id", async () => {
    vi.mocked(api.getGateways).mockResolvedValue({ gateways: [makeGateway("gw-2")], targets: [], sources: [], presets: [] });

    renderAt("gw-1");

    expect(await screen.findByText(/No gateway called/)).toBeInTheDocument();
    expect(screen.getByText(/gw-1/)).toBeInTheDocument();
  });

  it("shows the gateway header + 'reading counters' while the first poll is in flight", async () => {
    vi.mocked(api.getGateways).mockResolvedValue({ gateways: [makeGateway("gw-1", "My Gateway")], targets: [], sources: [], presets: [] });
    vi.mocked(api.getGatewayAnalytics).mockReturnValue(new Promise(() => {}));

    renderAt();

    expect(await screen.findByText("Analytics: My Gateway")).toBeInTheDocument();
    expect(screen.getByText(/Reading the gateway's counters/)).toBeInTheDocument();
  });

  it("shows a generic top-level error (no gateway header) when the very first poll fails", async () => {
    vi.mocked(api.getGateways).mockResolvedValue({ gateways: [makeGateway("gw-1")], targets: [], sources: [], presets: [] });
    vi.mocked(api.getGatewayAnalytics).mockRejectedValue(new Error("could not reach the gateway"));

    renderAt();

    expect(await screen.findByText("could not reach the gateway")).toBeInTheDocument();
    expect(screen.queryByText(/Analytics:/)).not.toBeInTheDocument();
  });

  it("shows a counters-off card when the gateway isn't counting requests", async () => {
    vi.mocked(api.getGateways).mockResolvedValue({ gateways: [makeGateway("gw-1")], targets: [], sources: [], presets: [] });
    vi.mocked(api.getGatewayAnalytics).mockResolvedValue({ enabled: false, at: "now", since: "", networks: [], endpoints: [] });

    renderAt();

    expect(await screen.findByText(/not counting its own requests/)).toBeInTheDocument();
  });

  it("shows a could-not-be-read card when the scrape carries an error", async () => {
    vi.mocked(api.getGateways).mockResolvedValue({ gateways: [makeGateway("gw-1")], targets: [], sources: [], presets: [] });
    vi.mocked(api.getGatewayAnalytics).mockResolvedValue({
      enabled: true,
      at: "now",
      since: "s1",
      networks: [],
      endpoints: [],
      error: "dial tcp: connection refused",
    });

    renderAt();

    expect(await screen.findByText("The gateway's counters could not be read.")).toBeInTheDocument();
    expect(screen.getByText("dial tcp: connection refused")).toBeInTheDocument();
  });

  it("renders both sections from a populated scrape: stats, latency table, and the endpoint's selection info", async () => {
    vi.mocked(api.getGateways).mockResolvedValue({ gateways: [makeGateway("gw-1", "My Gateway")], targets: [], sources: [], presets: [] });
    vi.mocked(api.getGatewayAnalytics).mockResolvedValue(ANALYTICS_ON);

    renderAt();

    expect(await screen.findByText("Ethereum")).toBeInTheDocument();
    expect(screen.getByText("100")).toBeInTheDocument(); // received
    expect(screen.getByText("measuring rate…")).toBeInTheDocument(); // only one reading so far
    // "eth_call" also shows up as the (un-expanded) error detail row's plain
    // text method column; scope to the <code> cell that names the method row.
    expect(screen.getByText("eth_call", { selector: "code" })).toBeInTheDocument();
    // The endpoint's URL appears three times: the client section's
    // per-endpoint latency table, the gateway section's own row, and the
    // (always-rendered, CSS-hidden-until-expanded) error detail row's
    // "Errors the gateway saw when it called <upstream>" line.
    expect(screen.getAllByText("https://rpc1.example.com")).toHaveLength(3);
    expect(screen.getByText("preferred")).toHaveClass("badge-ok");
    expect(screen.getByText("score 0.987")).toBeInTheDocument();
    expect(screen.getByText("2 switches")).toBeInTheDocument();
  });

  it("expands an endpoint row with errors on click", async () => {
    vi.mocked(api.getGateways).mockResolvedValue({ gateways: [makeGateway("gw-1")], targets: [], sources: [], presets: [] });
    vi.mocked(api.getGatewayAnalytics).mockResolvedValue(ANALYTICS_ON);

    renderAt();

    const rows = await screen.findAllByText("https://rpc1.example.com");
    const tr = rows.map((r) => r.closest("tr")).find((t) => t?.classList.contains("an-endpoint"))!;
    expect(tr).toHaveClass("expandable");
    expect(tr).not.toHaveClass("expanded");

    fireEvent.click(tr);
    expect(tr).toHaveClass("expanded");
    expect(screen.getByText("timeout")).toBeInTheDocument();
  });

  it("shows the network-quiet note instead of latency tables when a chain has seen no clients", async () => {
    vi.mocked(api.getGateways).mockResolvedValue({ gateways: [makeGateway("gw-1")], targets: [], sources: [], presets: [] });
    vi.mocked(api.getGatewayAnalytics).mockResolvedValue({
      ...ANALYTICS_ON,
      networks: [
        {
          chainId: 1,
          name: "Ethereum",
          received: 0,
          answered: 0,
          unattributed: 0,
          failed: 0,
          methods: [],
          endpoints: [],
          cached: EMPTY_LATENCY,
          failedLatency: EMPTY_LATENCY,
        },
      ],
    });

    renderAt();

    expect(await screen.findByText(/No client has called this chain/)).toBeInTheDocument();
  });

  it("resets the rate history when navigating to a different gateway's analytics without unmounting", async () => {
    // React Router keeps the same Analytics instance mounted across a
    // same-route param change (/analytics/gw-1 -> /analytics/gw-2), so this
    // exercises the [gid] reset effect for real rather than relying on a
    // fresh mount to hide a stale-history bug. Without the reset, gw-2's
    // single reading would sit alongside gw-1's leftover one and (received
    // going 10 -> 500) would compute a bogus rate instead of "measuring
    // rate…".
    vi.mocked(api.getGateways).mockResolvedValue({
      gateways: [makeGateway("gw-1", "First"), makeGateway("gw-2", "Second")],
      targets: [],
      sources: [],
      presets: [],
    });
    function scrape(received: number, since: string): GatewayAnalytics {
      return {
        enabled: true,
        at: "now",
        since,
        networks: [
          { chainId: 1, name: "Ethereum", received, answered: received, unattributed: 0, failed: 0, methods: [], endpoints: [], cached: EMPTY_LATENCY, failedLatency: EMPTY_LATENCY },
        ],
        endpoints: [],
      };
    }
    vi.mocked(api.getGatewayAnalytics).mockImplementation((gid: string) =>
      Promise.resolve(gid === "gw-1" ? scrape(10, "s1") : scrape(500, "s2")),
    );

    function Harness() {
      const navigate = useNavigate();
      return (
        <>
          <button onClick={() => navigate("/analytics/gw-2")}>go to gw-2</button>
          <Routes>
            <Route path="/analytics/:id" element={<Analytics />} />
          </Routes>
        </>
      );
    }
    const queryClient = new QueryClient({ defaultOptions: { queries: { retry: false } } });
    render(
      <QueryClientProvider client={queryClient}>
        <MemoryRouter initialEntries={["/analytics/gw-1"]}>
          <Harness />
        </MemoryRouter>
      </QueryClientProvider>,
    );

    await screen.findByText("Analytics: First");
    await waitFor(() => expect(screen.getByText("measuring rate…")).toBeInTheDocument());

    fireEvent.click(screen.getByText("go to gw-2"));

    await screen.findByText("Analytics: Second");
    await waitFor(() => expect(screen.getByText("measuring rate…")).toBeInTheDocument());
    expect(screen.queryByText(/req\/s/)).not.toBeInTheDocument();
  });
});
