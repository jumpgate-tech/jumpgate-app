import { render, screen, cleanup, fireEvent, waitFor, act } from "@testing-library/react";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { describe, it, expect, vi, beforeEach, afterEach } from "vitest";
import type { ReactNode } from "react";
import * as api from "../../api";
import type { Catalog, SetupEvent, Target } from "../../api";
import { SetupWizard } from "./SetupWizard";

vi.mock("../../api", async () => {
  const actual = await vi.importActual<typeof import("../../api")>("../../api");
  return {
    ...actual,
    getCatalog: vi.fn(),
    listTargets: vi.fn(),
    getDiskFree: vi.fn(),
    startSetup: vi.fn(),
    streamSetup: vi.fn(),
  };
});

const CATALOG: Catalog = {
  networks: [
    {
      ChainID: 369,
      Name: "PulseChain",
      CheckpointURL: "https://checkpoint.example/369",
      ExecClients: ["reth", "erigon-pulse"],
      BeaconClients: ["lighthouse-pulse"],
      LearnURL: "https://learn.valve.city/rpc",
      SnapshotSizeTB: 2,
      SyncLabel: "~2h",
      GenesisSyncLabel: "~4d",
    },
    {
      ChainID: 943,
      Name: "PulseChain Testnet",
      CheckpointURL: "https://checkpoint.example/943",
      ExecClients: ["reth"],
      BeaconClients: ["lighthouse-pulse"],
      LearnURL: "https://learn.valve.city/rpc",
      SnapshotSizeTB: 0.1,
      SyncLabel: "~30m",
      GenesisSyncLabel: "~1d",
    },
    {
      ChainID: 1,
      Name: "Ethereum",
      CheckpointURL: "https://checkpoint.example/1",
      ExecClients: ["geth"],
      BeaconClients: ["lighthouse"],
      LearnURL: "https://learn.valve.city/rpc",
      SnapshotSizeTB: 5,
      SyncLabel: "~4h",
      GenesisSyncLabel: "~10d",
    },
  ],
  clients: [
    { id: "reth", kind: "exec", repo: "https://github.com/valve-tech/reth", pinVersion: "1", toolchain: "rust", learnUrl: "", snapshotSupported: true },
    { id: "erigon-pulse", kind: "exec", repo: "https://github.com/valve-tech/erigon-pulse", pinVersion: "1", toolchain: "go", learnUrl: "", snapshotSupported: false },
    { id: "geth", kind: "exec", repo: "https://github.com/ethereum/go-ethereum", pinVersion: "1", toolchain: "go", learnUrl: "", snapshotSupported: false },
    { id: "lighthouse-pulse", kind: "beacon", repo: "https://github.com/valve-tech/lighthouse-pulse", pinVersion: "1", toolchain: "rust", learnUrl: "", snapshotSupported: false },
    { id: "lighthouse", kind: "beacon", repo: "https://github.com/sigp/lighthouse", pinVersion: "1", toolchain: "rust", learnUrl: "", snapshotSupported: false },
  ],
};

function renderWizard(targetId = "t1") {
  const queryClient = new QueryClient({
    defaultOptions: { queries: { retry: false }, mutations: { retry: false } },
  });
  function wrapper({ children }: { children: ReactNode }) {
    return <QueryClientProvider client={queryClient}>{children}</QueryClientProvider>;
  }
  return render(<SetupWizard targetId={targetId} />, { wrapper });
}

// gotoClients advances from the default network step (chain 369 is
// preselected) to the clients step.
async function gotoClients(): Promise<void> {
  await screen.findByText("1. Choose a network");
  fireEvent.click(screen.getByText("Next: clients"));
  await screen.findByText("2. Choose your client pair");
}

// gotoMode advances from clients to mode, waiting for the disk probe
// wizard.ts's own goto-mode handler fires to settle.
async function gotoMode(): Promise<void> {
  await gotoClients();
  fireEvent.click(screen.getByText("Next: mode"));
  await screen.findByText("3. Choose sync mode");
  await waitFor(() => expect(api.getDiskFree).toHaveBeenCalled());
}

beforeEach(() => {
  vi.mocked(api.getCatalog).mockReset().mockResolvedValue(CATALOG);
  vi.mocked(api.listTargets).mockReset().mockResolvedValue([] as Target[]);
  vi.mocked(api.getDiskFree).mockReset().mockResolvedValue({ path: "/data", freeBytes: 3e12 });
  vi.mocked(api.startSetup).mockReset();
  vi.mocked(api.streamSetup).mockReset();
});
afterEach(cleanup);

describe("SetupWizard — loading/error", () => {
  it("shows a loading state while the catalog/targets fetch is in flight", () => {
    vi.mocked(api.getCatalog).mockReturnValue(new Promise(() => {}));
    renderWizard();
    expect(screen.getByText("Loading catalog…")).toBeInTheDocument();
  });

  it("shows a failed-to-load message on a rejected fetch", async () => {
    vi.mocked(api.getCatalog).mockRejectedValue(new Error("network down"));
    renderWizard();
    expect(await screen.findByText(/Failed to load: network down/)).toBeInTheDocument();
  });
});

describe("SetupWizard — step navigation", () => {
  it("walks network → clients → mode → review and back again", async () => {
    renderWizard();

    await screen.findByText("1. Choose a network");
    expect(screen.getByRole("button", { name: /^PulseChain \(chain 369\)/ })).toBeInTheDocument();

    fireEvent.click(screen.getByText("Next: clients"));
    await screen.findByText("2. Choose your client pair");

    fireEvent.click(screen.getByText("Next: mode"));
    await screen.findByText("3. Choose sync mode");
    await waitFor(() => expect(api.getDiskFree).toHaveBeenCalledWith("t1", "/var/lib/valve-node-app/369"));

    fireEvent.click(screen.getByText("Next: review"));
    await screen.findByText("4. Review");
    expect(screen.getByText("t1")).toBeInTheDocument();
    expect(screen.getByText("reth")).toBeInTheDocument();
    expect(screen.getByText("lighthouse-pulse")).toBeInTheDocument();

    fireEvent.click(screen.getByText("Back"));
    await screen.findByText("3. Choose sync mode");

    fireEvent.click(screen.getByText("Back"));
    await screen.findByText("2. Choose your client pair");

    fireEvent.click(screen.getByText("Back"));
    await screen.findByText("1. Choose a network");
  });

  it("switching network resets the client pair to the new network's first offering", async () => {
    renderWizard();
    await screen.findByText("1. Choose a network");

    fireEvent.click(screen.getByRole("button", { name: /Ethereum/ }));
    fireEvent.click(screen.getByText("Next: clients"));

    await screen.findByText("2. Choose your client pair");
    expect(screen.getByDisplayValue(/^geth/)).toBeInTheDocument();
    expect(screen.getByDisplayValue(/^lighthouse —/)).toBeInTheDocument();
  });
});

describe("SetupWizard — fit-check rendering", () => {
  it("reports both tiers fitting when free space comfortably covers archive", async () => {
    vi.mocked(api.getDiskFree).mockResolvedValue({ path: "/data", freeBytes: 3e12 });
    renderWizard();
    await gotoMode();

    expect(await screen.findByText(/archive fits/)).toBeInTheDocument();
    expect(screen.getByText(/full fits/)).toBeInTheDocument();
    const radios = document.querySelectorAll<HTMLInputElement>('input[name="mode"]');
    expect((radios[0] as HTMLInputElement).checked).toBe(true); // archive stays selected
  });

  it("downgrades to full and shows the switch note when only full fits", async () => {
    // 369's archive tier needs ~2.2e12 bytes (2TB * 1.1); full needs ~1.1e12.
    vi.mocked(api.getDiskFree).mockResolvedValue({ path: "/data", freeBytes: 1.5e12 });
    renderWizard();
    await gotoMode();

    expect(await screen.findByText(/switched to Full/)).toBeInTheDocument();
    const radios = document.querySelectorAll<HTMLInputElement>('input[name="mode"]');
    expect((radios[0] as HTMLInputElement).checked).toBe(false); // archive
    expect((radios[1] as HTMLInputElement).checked).toBe(true); // full
  });

  it("warns when neither tier fits", async () => {
    vi.mocked(api.getDiskFree).mockResolvedValue({ path: "/data", freeBytes: 1000 });
    renderWizard();
    await gotoMode();

    expect(await screen.findByText(/Neither full/)).toBeInTheDocument();
  });

  it("surfaces a probe failure instead of a fit verdict", async () => {
    vi.mocked(api.getDiskFree).mockRejectedValue(new Error("no such path"));
    renderWizard();
    await gotoMode();

    expect(await screen.findByText(/Couldn't read free space/)).toBeInTheDocument();
    expect(screen.getByText(/no such path/)).toBeInTheDocument();
  });
});

describe("SetupWizard — invalid selection blocks provisioning", () => {
  it("an invalid checkpoint URL shows an inline error and blocks Next: review", async () => {
    renderWizard();
    await gotoMode();

    const input = screen.getByLabelText(/Checkpoint URL/);
    fireEvent.change(input, { target: { value: "not-a-url" } });

    expect(await screen.findByText(/Enter an http\(s\) URL/)).toBeInTheDocument();

    fireEvent.click(screen.getByText("Next: review"));

    // Still on the mode step — never reached review.
    expect(screen.getByText("3. Choose sync mode")).toBeInTheDocument();
    expect(screen.queryByText("4. Review")).not.toBeInTheDocument();
  });

  it("fixing the field un-blocks navigation", async () => {
    renderWizard();
    await gotoMode();

    const input = screen.getByLabelText(/Checkpoint URL/);
    fireEvent.change(input, { target: { value: "not-a-url" } });
    expect(await screen.findByText(/Enter an http\(s\) URL/)).toBeInTheDocument();

    fireEvent.change(input, { target: { value: "https://checkpoint.example/custom" } });
    fireEvent.click(screen.getByText("Next: review"));

    expect(await screen.findByText("4. Review")).toBeInTheDocument();
  });
});

describe("SetupWizard — provisioning run", () => {
  function wireStream(): { push: (ev: SetupEvent) => void; stop: ReturnType<typeof vi.fn> } {
    const stop = vi.fn();
    let onEvent: ((ev: SetupEvent) => void) | null = null;
    vi.mocked(api.streamSetup).mockImplementation((_id, cb) => {
      onEvent = cb;
      return stop;
    });
    return { push: (ev: SetupEvent) => act(() => onEvent!(ev)), stop };
  }

  async function gotoReview(): Promise<void> {
    await gotoMode();
    fireEvent.click(screen.getByText("Next: review"));
    await screen.findByText("4. Review");
  }

  it("posts the wire config, moves to the run step, and streams progress to completion", async () => {
    vi.mocked(api.startSetup).mockResolvedValue({ status: "accepted" });
    const stream = wireStream();

    renderWizard();
    await gotoReview();

    fireEvent.click(screen.getByText("Start setup"));

    await waitFor(() =>
      expect(api.startSetup).toHaveBeenCalledWith("t1", {
        ChainID: 369,
        ExecID: "reth",
        BeaconID: "lighthouse-pulse",
        Archive: true,
      }),
    );

    expect(await screen.findByText("5. Running setup")).toBeInTheDocument();

    stream.push({ stepId: "preflight", line: "checking docker" });
    expect(await screen.findByText("checking docker")).toBeInTheDocument();

    stream.push({ stepId: "handshake", done: true });
    expect(await screen.findByText("Setup complete.")).toBeInTheDocument();
    expect(screen.getByText("Open the dashboard →")).toHaveAttribute("href", "#/dash/t1");

    // wizard.ts never closes the stream itself on completion.
    expect(stream.stop).not.toHaveBeenCalled();
  });

  it("shows a failed step's badge and error line without completing", async () => {
    vi.mocked(api.startSetup).mockResolvedValue({ status: "accepted" });
    const stream = wireStream();

    renderWizard();
    await gotoReview();
    fireEvent.click(screen.getByText("Start setup"));
    await screen.findByText("5. Running setup");

    stream.push({ stepId: "preflight", err: "docker unreachable" });

    expect(await screen.findByText("docker unreachable")).toBeInTheDocument();
    expect(screen.getByText("failed")).toBeInTheDocument();
    expect(screen.queryByText("Setup complete.")).not.toBeInTheDocument();
    expect(screen.getByText("Retry setup")).toBeInTheDocument();
  });

  it("shows the POST error inline and stays on review when starting fails outright", async () => {
    vi.mocked(api.startSetup).mockRejectedValue(new Error("connection refused"));

    renderWizard();
    await gotoReview();
    fireEvent.click(screen.getByText("Start setup"));

    expect(await screen.findByText("connection refused")).toBeInTheDocument();
    expect(screen.getByText("4. Review")).toBeInTheDocument();
    expect(api.streamSetup).not.toHaveBeenCalled();
  });

  it("a 409 (already running) attaches to the live stream instead of erroring", async () => {
    vi.mocked(api.startSetup).mockRejectedValue(new api.ApiError(409, "already running"));
    wireStream();

    renderWizard();
    await gotoReview();
    fireEvent.click(screen.getByText("Start setup"));

    expect(await screen.findByText("5. Running setup")).toBeInTheDocument();
    expect(api.streamSetup).toHaveBeenCalledWith("t1", expect.any(Function));
  });
});
