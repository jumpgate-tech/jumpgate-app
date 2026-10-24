import { render, screen, cleanup, fireEvent, waitFor } from "@testing-library/react";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { describe, it, expect, vi, afterEach, beforeEach } from "vitest";
import type { ReactNode } from "react";
import * as api from "../../api";
import type { Catalog, GatewayView } from "../../api";
import { GatewayModules } from "./GatewayModules";

vi.mock("../../api", async () => {
  const actual = await vi.importActual<typeof import("../../api")>("../../api");
  return { ...actual, getCatalog: vi.fn(), setGatewayModule: vi.fn(), streamSetup: vi.fn() };
});

const CATALOG: Catalog = {
  networks: [],
  clients: [],
  modules: [
    {
      id: "private-send",
      name: "Private send",
      summary: "Sends transactions privately.",
      siteUrl: "",
      docsStep: "private-send-enable",
      placement: "gateway",
      resources: { ramBytes: 0, diskBytes: 0 },
      applies: [
        { chainId: 1, client: "", ok: true, beta: false },
        { chainId: 369, client: "", ok: false, beta: false },
      ],
      methods: [],
    },
  ],
};

const PLS_REASON = "No private send route exists on PulseChain.";

// withReasons is CATALOG with the Applies rows the real entry produces.
function withReasons(): Catalog {
  return {
    ...CATALOG,
    modules: [
      {
        ...CATALOG.modules![0],
        applies: [
          { chainId: 1, client: "", ok: true, beta: false },
          { chainId: 369, client: "", ok: false, beta: false, reason: PLS_REASON },
          { chainId: 943, client: "", ok: false, beta: false, reason: PLS_REASON },
          { chainId: 560048, client: "", ok: false, beta: false, reason: "No private send route exists on Hoodi." },
        ],
      },
    ],
  };
}

function gw(chains: number[], modules: string[] = []): GatewayView {
  return {
    id: "gw1",
    networks: chains.map((c) => ({ chainId: c })),
    config: { ProjectID: "p", BindAddr: "", Port: 4000, Networks: null, Modules: modules.map((id) => ({ id })) },
  } as unknown as GatewayView;
}

function renderIt(view: GatewayView) {
  const qc = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  const wrapper = ({ children }: { children: ReactNode }) => <QueryClientProvider client={qc}>{children}</QueryClientProvider>;
  return render(<GatewayModules gw={view} />, { wrapper });
}

beforeEach(() => {
  vi.mocked(api.getCatalog).mockResolvedValue(CATALOG);
  vi.mocked(api.setGatewayModule).mockResolvedValue({ status: "started", targetId: "box1" });
  vi.mocked(api.streamSetup).mockReturnValue(() => {});
});
afterEach(cleanup);

describe("GatewayModules", () => {
  it("shows a gateway module where one of the gateway's chains applies, and toggles it", async () => {
    renderIt(gw([369, 1]));
    fireEvent.click(await screen.findByRole("button", { name: "Turn on Private send" }));
    await waitFor(() => expect(api.setGatewayModule).toHaveBeenCalledWith("gw1", "private-send", true));
    expect(api.streamSetup).toHaveBeenCalledWith("box1", expect.any(Function));
  });

  it("hides the card when no chain applies", async () => {
    const { container } = renderIt(gw([369]));
    await waitFor(() => expect(api.getCatalog).toHaveBeenCalled());
    expect(container.querySelector("[data-module]")).toBeNull();
  });

  it("shows private send enabled on a gateway serving chain 1", async () => {
    vi.mocked(api.getCatalog).mockResolvedValue(withReasons());
    renderIt(gw([1]));
    const btn = await screen.findByRole("button", { name: "Turn on Private send" });
    expect(btn).not.toBeDisabled();
    expect(screen.queryByText(PLS_REASON)).toBeNull();
  });

  it("shows private send disabled with the PulseChain reason on a 369-only gateway", async () => {
    vi.mocked(api.getCatalog).mockResolvedValue(withReasons());
    renderIt(gw([369]));
    const btn = await screen.findByRole("button", { name: "Turn on Private send" });
    expect(btn).toBeDisabled();
    expect(screen.getByText(PLS_REASON)).toBeInTheDocument();
  });

  it("turns an enabled module off", async () => {
    renderIt(gw([1], ["private-send"]));
    fireEvent.click(await screen.findByRole("button", { name: "Turn off Private send" }));
    await waitFor(() => expect(api.setGatewayModule).toHaveBeenCalledWith("gw1", "private-send", false));
  });
});
