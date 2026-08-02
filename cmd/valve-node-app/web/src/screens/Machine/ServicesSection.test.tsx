import { render, screen, cleanup, fireEvent, waitFor, within, act } from "@testing-library/react";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { describe, it, expect, vi, beforeEach, afterEach } from "vitest";
import type { ReactNode } from "react";
import * as api from "../../api";
import type { ContainersResponse, ContainerStatus, ContainerView, DevnetConfig, SetupEvent, WipeResult } from "../../api";
import { ServicesSection } from "./ServicesSection";

vi.mock("../../api", async () => {
  const actual = await vi.importActual<typeof import("../../api")>("../../api");
  return {
    ...actual,
    getContainers: vi.fn(),
    containerAction: vi.fn(),
    provisionContainer: vi.fn(),
    streamSetup: vi.fn(),
    putContainerConfig: vi.fn(),
    wipeContainer: vi.fn(),
  };
});

function status(overrides: Partial<ContainerStatus> = {}): ContainerStatus {
  return {
    ID: "id",
    ContainerName: "valve-devnet",
    State: "not-created",
    Image: "",
    ImageID: "",
    ExitCode: 0,
    Platform: "",
    EnginePlatform: "",
    Emulated: false,
    Detail: "",
    ...overrides,
  };
}

function devnet(overrides: Partial<DevnetConfig> = {}): DevnetConfig {
  return {
    ChainID: 1337,
    BlockTime: "2s",
    ImageRef: "reth:dev",
    ContainerName: "valve-devnet",
    BindAddr: "127.0.0.1",
    HTTPPort: 8545,
    WSPort: 8546,
    Platform: "",
    ...overrides,
  };
}

function view(overrides: Partial<ContainerView> = {}): ContainerView {
  return {
    id: "devnet",
    label: "Devnet",
    containerName: "valve-devnet",
    configured: true,
    status: status(),
    endpoints: null,
    actions: ["create"],
    wipeDiscards: "the devnet's chain data",
    restartsOnWipe: null,
    devnet: devnet(),
    ...overrides,
  };
}

function containersResponse(overrides: Partial<ContainersResponse> = {}): ContainersResponse {
  return {
    docker: { present: true, reachable: true, flavor: "docker", serverVersion: "24.0" },
    services: [view()],
    ...overrides,
  };
}

function renderSection(targetId = "t1") {
  const queryClient = new QueryClient({
    defaultOptions: { queries: { retry: false }, mutations: { retry: false } },
  });
  function wrapper({ children }: { children: ReactNode }) {
    return <QueryClientProvider client={queryClient}>{children}</QueryClientProvider>;
  }
  return render(<ServicesSection targetId={targetId} />, { wrapper });
}

beforeEach(() => {
  vi.mocked(api.getContainers).mockReset();
  vi.mocked(api.containerAction).mockReset();
  vi.mocked(api.provisionContainer).mockReset();
  vi.mocked(api.streamSetup).mockReset();
  vi.mocked(api.putContainerConfig).mockReset();
  vi.mocked(api.wipeContainer).mockReset();
});
afterEach(cleanup);

describe("ServicesSection", () => {
  it("shows Loading… while the containers fetch is in flight", () => {
    vi.mocked(api.getContainers).mockReturnValue(new Promise(() => {}));

    renderSection();

    expect(screen.getByText("Loading…")).toBeInTheDocument();
  });

  it("shows a failed-to-read error when the fetch rejects", async () => {
    vi.mocked(api.getContainers).mockRejectedValue(new Error("network down"));

    renderSection();

    expect(await screen.findByText(/Could not read this machine's services/)).toBeInTheDocument();
    expect(screen.getByText(/network down/)).toBeInTheDocument();
  });

  it("shows the docker-unreachable banner when the engine did not answer", async () => {
    vi.mocked(api.getContainers).mockResolvedValue(
      containersResponse({ docker: { present: true, reachable: false, flavor: "", hint: "start docker" } }),
    );

    renderSection();

    expect(await screen.findByText("Docker is installed, but no engine answered")).toBeInTheDocument();
    expect(screen.getByText("start docker")).toBeInTheDocument();
  });

  it("shows the docker-ok line when reachable", async () => {
    vi.mocked(api.getContainers).mockResolvedValue(containersResponse());

    renderSection();

    expect(await screen.findByText(/Docker: docker 24.0 · reachable/)).toBeInTheDocument();
  });

  it("renders the card's label, badge, blurb and stat list", async () => {
    vi.mocked(api.getContainers).mockResolvedValue(
      containersResponse({ services: [view({ status: status({ State: "running", Image: "reth:dev" }) })] }),
    );

    renderSection();

    expect(await screen.findByText("Devnet")).toBeInTheDocument();
    expect(screen.getByText("running")).toBeInTheDocument();
    expect(screen.getByText(/throwaway chain that runs entirely on this machine/)).toBeInTheDocument();
    expect(screen.getByText("valve-devnet")).toBeInTheDocument();
    expect(screen.getByText("reth:dev")).toBeInTheDocument();
  });

  it("reports a non-zero exit code, calling out 137 as a likely OOM kill", async () => {
    vi.mocked(api.getContainers).mockResolvedValue(
      containersResponse({
        services: [view({ status: status({ State: "created-but-stopped", ExitCode: 137 }) })],
      }),
    );

    renderSection();

    expect(await screen.findByText(/It exited with code 137/)).toBeInTheDocument();
    expect(screen.getByText(/most often the machine ran out of memory/)).toBeInTheDocument();
  });

  it("shows a no-endpoints message only while running", async () => {
    vi.mocked(api.getContainers).mockResolvedValue(
      containersResponse({ services: [view({ status: status({ State: "running" }), endpoints: [] })] }),
    );

    renderSection();

    expect(await screen.findByText(/No endpoint to show/)).toBeInTheDocument();
  });

  it("renders endpoint rows with a working copy button", async () => {
    vi.useFakeTimers({ shouldAdvanceTime: true });
    const writeText = vi.fn().mockResolvedValue(undefined);
    Object.assign(navigator, { clipboard: { writeText } });
    vi.mocked(api.getContainers).mockResolvedValue(
      containersResponse({
        services: [
          view({
            status: status({ State: "running" }),
            endpoints: [{ label: "JSON-RPC", url: "http://127.0.0.1:8545" }],
          }),
        ],
      }),
    );

    renderSection();

    expect(await screen.findByText("http://127.0.0.1:8545")).toBeInTheDocument();
    fireEvent.click(screen.getByText("Copy"));

    await vi.waitFor(() => expect(screen.getByText("Copied!")).toBeInTheDocument());
    expect(writeText).toHaveBeenCalledWith("http://127.0.0.1:8545");

    await act(async () => {
      vi.advanceTimersByTime(1600);
    });
    expect(screen.getByText("Copy")).toBeInTheDocument();
    vi.useRealTimers();
  });

  describe("start/stop/restart actions", () => {
    it("calls containerAction and disables every action button while pending", async () => {
      let resolveAction: (() => void) | null = null;
      vi.mocked(api.containerAction).mockReturnValue(
        new Promise((resolve) => {
          resolveAction = () => resolve({ status: status({ State: "running" }) });
        }),
      );
      vi.mocked(api.getContainers).mockResolvedValue(
        containersResponse({
          services: [view({ status: status({ State: "running" }), actions: ["stop", "restart", "wipe"] })],
        }),
      );

      renderSection();
      await screen.findByText("Devnet");
      fireEvent.click(screen.getByText("Stop"));

      expect(api.containerAction).toHaveBeenCalledWith("t1", "devnet", "stop");
      await waitFor(() => expect(screen.getByText("Restart")).toBeDisabled());
      expect(screen.getByText("Wipe…")).toBeDisabled();

      await act(async () => {
        resolveAction!();
        await Promise.resolve();
      });
      // Always reloads afterward, regardless of outcome.
      await waitFor(() => expect(api.getContainers).toHaveBeenCalledTimes(2));
    });

    it("shows a formatted error message on failure", async () => {
      vi.mocked(api.containerAction).mockRejectedValue(new Error("connection refused"));
      vi.mocked(api.getContainers).mockResolvedValue(
        containersResponse({ services: [view({ status: status({ State: "running" }), actions: ["stop"] })] }),
      );

      renderSection();
      await screen.findByText("Devnet");
      fireEvent.click(screen.getByText("Stop"));

      expect(await screen.findByText("stop failed: connection refused")).toBeInTheDocument();
    });
  });

  describe("provisioning (create)", () => {
    function wireStream(): { push: (ev: SetupEvent) => void; stop: ReturnType<typeof vi.fn> } {
      const stop = vi.fn();
      let onEvent: ((ev: SetupEvent) => void) | null = null;
      vi.mocked(api.streamSetup).mockImplementation((_id, cb) => {
        onEvent = cb;
        return stop;
      });
      return { push: (ev: SetupEvent) => act(() => onEvent!(ev)), stop };
    }

    it("posts, streams progress lines, and reloads + re-enables the button once the final step is done", async () => {
      vi.mocked(api.provisionContainer).mockResolvedValue({ status: "accepted" });
      vi.mocked(api.getContainers).mockResolvedValue(containersResponse());
      const stream = wireStream();

      renderSection();
      await screen.findByText("Devnet");
      const createBtn = screen.getByText("Create devnet");
      fireEvent.click(createBtn);

      expect(api.provisionContainer).toHaveBeenCalledWith("t1", "devnet");
      // The button's label becomes a spinner while busy — assert on the same
      // node's disabled state rather than re-querying by (now gone) text.
      await waitFor(() => expect(createBtn).toBeDisabled());

      stream.push({ stepId: "preflight", line: "checking docker" });
      expect(await screen.findByText("preflight: checking docker")).toBeInTheDocument();

      stream.push({ stepId: "run", line: "started", done: true });

      await waitFor(() => expect(createBtn).not.toBeDisabled());
      expect(stream.stop).toHaveBeenCalledTimes(1);
      await waitFor(() => expect(api.getContainers).toHaveBeenCalledTimes(2));
    });

    it("shows the generic provisioning-failed message on a step error", async () => {
      vi.mocked(api.provisionContainer).mockResolvedValue({ status: "accepted" });
      vi.mocked(api.getContainers).mockResolvedValue(containersResponse());
      const stream = wireStream();

      renderSection();
      await screen.findByText("Devnet");
      fireEvent.click(screen.getByText("Create devnet"));
      await waitFor(() => expect(api.streamSetup).toHaveBeenCalled());

      stream.push({ stepId: "preflight", err: "docker unreachable" });

      expect(await screen.findByText("Provisioning failed — see the log below.")).toBeInTheDocument();
      expect(screen.getByText("preflight: docker unreachable")).toBeInTheDocument();
    });
  });

  describe("wipe", () => {
    it("gates the confirm button on typing the exact service id, then calls wipeContainer and shows the result", async () => {
      const wipeResult: WipeResult = {
        report: {
          ID: "id",
          ContainerName: "valve-devnet",
          ContainerRemoved: true,
          VolumesRemoved: ["devnet-data"],
          VolumesAbsent: null,
          Recreated: false,
          Cascaded: null,
          CascadeSkipped: null,
        },
        status: status({ State: "not-created" }),
      };
      vi.mocked(api.wipeContainer).mockResolvedValue(wipeResult);
      vi.mocked(api.getContainers).mockResolvedValue(
        containersResponse({ services: [view({ status: status({ State: "running" }), actions: ["wipe"] })] }),
      );

      renderSection();
      await screen.findByText("Devnet");
      fireEvent.click(screen.getByText("Wipe…"));

      const dialog = await screen.findByRole("dialog");
      const confirmBtn = within(dialog).getByText("Wipe devnet");
      expect(confirmBtn).toBeDisabled();

      const input = within(dialog).getByRole("textbox");
      fireEvent.change(input, { target: { value: "wrong" } });
      expect(confirmBtn).toBeDisabled();

      fireEvent.change(input, { target: { value: "devnet" } });
      expect(confirmBtn).not.toBeDisabled();
      fireEvent.click(confirmBtn);

      await waitFor(() => expect(api.wipeContainer).toHaveBeenCalledWith("t1", "devnet"));
      expect(await screen.findByText("Devnet wiped")).toBeInTheDocument();
      expect(screen.getByText("Container removed.")).toBeInTheDocument();
      expect(screen.getByText("Volume devnet-data deleted.")).toBeInTheDocument();

      fireEvent.click(screen.getByText("Close"));
      await waitFor(() => expect(screen.queryByRole("dialog")).not.toBeInTheDocument());
      await waitFor(() => expect(api.getContainers).toHaveBeenCalledTimes(2));
    });

    it("shows cascaded restart labels resolved from other services' labels", async () => {
      const wipeResult: WipeResult = {
        report: {
          ID: "id",
          ContainerName: "valve-devnet",
          ContainerRemoved: true,
          VolumesRemoved: null,
          VolumesAbsent: null,
          Recreated: false,
          Cascaded: ["devnet"],
          CascadeSkipped: null,
        },
        status: status(),
      };
      vi.mocked(api.wipeContainer).mockResolvedValue(wipeResult);
      vi.mocked(api.getContainers).mockResolvedValue(
        containersResponse({ services: [view({ status: status({ State: "running" }), actions: ["wipe"] })] }),
      );

      renderSection();
      await screen.findByText("Devnet");
      fireEvent.click(screen.getByText("Wipe…"));
      const dialog = await screen.findByRole("dialog");
      fireEvent.change(within(dialog).getByRole("textbox"), { target: { value: "devnet" } });
      fireEvent.click(within(dialog).getByText("Wipe devnet"));

      expect(await screen.findByText(/Restarted in front of it: Devnet/)).toBeInTheDocument();
    });

    it("shows an inline error and keeps the confirm modal open on failure", async () => {
      vi.mocked(api.wipeContainer).mockRejectedValue(new Error("disk busy"));
      vi.mocked(api.getContainers).mockResolvedValue(
        containersResponse({ services: [view({ status: status({ State: "running" }), actions: ["wipe"] })] }),
      );

      renderSection();
      await screen.findByText("Devnet");
      fireEvent.click(screen.getByText("Wipe…"));
      const dialog = await screen.findByRole("dialog");
      fireEvent.change(within(dialog).getByRole("textbox"), { target: { value: "devnet" } });
      fireEvent.click(within(dialog).getByText("Wipe devnet"));

      expect(await within(dialog).findByText("Wipe failed: disk busy")).toBeInTheDocument();
      expect(within(dialog).getByText("Wipe devnet")).not.toBeDisabled();
    });

    it("cancelling closes the modal without calling wipeContainer, and reloads", async () => {
      vi.mocked(api.getContainers).mockResolvedValue(
        containersResponse({ services: [view({ status: status({ State: "running" }), actions: ["wipe"] })] }),
      );

      renderSection();
      await screen.findByText("Devnet");
      fireEvent.click(screen.getByText("Wipe…"));
      const dialog = await screen.findByRole("dialog");
      fireEvent.click(within(dialog).getByText("Cancel"));

      await waitFor(() => expect(screen.queryByRole("dialog")).not.toBeInTheDocument());
      expect(api.wipeContainer).not.toHaveBeenCalled();
      await waitFor(() => expect(api.getContainers).toHaveBeenCalledTimes(2));
    });
  });

  describe("config editor", () => {
    it("shows the collapsed summary, then a pre-filled form on Edit", async () => {
      vi.mocked(api.getContainers).mockResolvedValue(containersResponse());

      renderSection();
      expect(await screen.findByText(/Chain 1337 · a block every 2s/)).toBeInTheDocument();

      fireEvent.click(screen.getByText("Edit"));

      expect(screen.getByDisplayValue("2s")).toBeInTheDocument();
      expect(screen.getByDisplayValue("8545")).toBeInTheDocument();
      expect(screen.getByDisplayValue("8546")).toBeInTheDocument();
      expect(screen.getByDisplayValue("127.0.0.1")).toBeInTheDocument();
    });

    it("saves edited fields and shows the plain 'Saved.' note when the service is not running", async () => {
      vi.mocked(api.putContainerConfig).mockResolvedValue({ id: "devnet", configured: true, devnet: devnet() });
      vi.mocked(api.getContainers).mockResolvedValue(containersResponse());

      renderSection();
      await screen.findByText(/Chain 1337/);
      fireEvent.click(screen.getByText("Edit"));

      fireEvent.change(screen.getByDisplayValue("2s"), { target: { value: "5s" } });
      fireEvent.click(screen.getByText("Save configuration"));

      await waitFor(() =>
        expect(api.putContainerConfig).toHaveBeenCalledWith("t1", "devnet", { ...devnet(), BlockTime: "5s" }),
      );
      expect(await screen.findByText("Saved.")).toBeInTheDocument();
    });

    it("shows the re-create note when saving while the container is running", async () => {
      vi.mocked(api.putContainerConfig).mockResolvedValue({ id: "devnet", configured: true, devnet: devnet() });
      vi.mocked(api.getContainers).mockResolvedValue(
        containersResponse({ services: [view({ status: status({ State: "running" }) })] }),
      );

      renderSection();
      await screen.findByText(/Chain 1337/);
      fireEvent.click(screen.getByText("Edit"));
      fireEvent.click(screen.getByText("Save configuration"));

      expect(await screen.findByText(/press.*Re-create \(apply config\).*to put them into effect/)).toBeInTheDocument();
    });

    it("blocks a save with matching HTTP/WS ports client-side, without calling the API", async () => {
      vi.mocked(api.getContainers).mockResolvedValue(containersResponse());

      renderSection();
      await screen.findByText(/Chain 1337/);
      fireEvent.click(screen.getByText("Edit"));

      fireEvent.change(screen.getByDisplayValue("8546"), { target: { value: "8545" } });
      fireEvent.click(screen.getByText("Save configuration"));

      expect(await screen.findByText(/cannot share a port/)).toBeInTheDocument();
      expect(api.putContainerConfig).not.toHaveBeenCalled();
    });
  });
});
