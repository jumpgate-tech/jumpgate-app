import { act, renderHook, waitFor } from "@testing-library/react";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { describe, it, expect, vi, beforeEach } from "vitest";
import type { ReactNode } from "react";
import * as api from "../api";
import type { ContainersResponse, ContainerStatus, ContainerView, DevnetConfig, SetupEvent, WipeResult } from "../api";
import { useContainerOps, useContainers, useSaveContainerConfig, useWipeContainer } from "./services";

vi.mock("../api", async () => {
  const actual = await vi.importActual<typeof import("../api")>("../api");
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
    ...overrides,
  };
}

function containersResponse(overrides: Partial<ContainersResponse> = {}): ContainersResponse {
  return {
    docker: { present: true, reachable: true, flavor: "docker" },
    services: [view()],
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

function wrapper({ children }: { children: ReactNode }) {
  const queryClient = new QueryClient({
    defaultOptions: { queries: { retry: false }, mutations: { retry: false } },
  });
  return <QueryClientProvider client={queryClient}>{children}</QueryClientProvider>;
}

beforeEach(() => {
  vi.mocked(api.getContainers).mockReset();
  vi.mocked(api.containerAction).mockReset();
  vi.mocked(api.provisionContainer).mockReset();
  vi.mocked(api.streamSetup).mockReset();
  vi.mocked(api.putContainerConfig).mockReset();
  vi.mocked(api.wipeContainer).mockReset();
});

describe("useContainers", () => {
  it("fetches the containers list", async () => {
    vi.mocked(api.getContainers).mockResolvedValue(containersResponse());

    const { result } = renderHook(() => useContainers("t1"), { wrapper });

    await waitFor(() => expect(result.current.isSuccess).toBe(true));
    expect(api.getContainers).toHaveBeenCalledWith("t1");
    expect(result.current.data?.services[0].id).toBe("devnet");
  });
});

describe("useContainerOps.run", () => {
  it("marks the service busy, calls containerAction, then always reloads", async () => {
    let resolveAction: (() => void) | null = null;
    vi.mocked(api.containerAction).mockReturnValue(
      new Promise((resolve) => {
        resolveAction = () => resolve({ status: status({ State: "running" }) });
      }),
    );
    vi.mocked(api.getContainers).mockResolvedValue(containersResponse());

    const { result } = renderHook(() => useContainerOps("t1"), { wrapper });

    let runPromise!: Promise<void>;
    act(() => {
      runPromise = result.current.run("devnet", "start");
    });
    await waitFor(() => expect(result.current.busy.devnet).toBe("start"));

    resolveAction!();
    await act(async () => {
      await runPromise;
    });

    expect(api.containerAction).toHaveBeenCalledWith("t1", "devnet", "start");
    expect(result.current.busy.devnet).toBeNull();
    expect(result.current.error.devnet).toBeNull();
  });

  it("ignores a second call for a service already in flight", async () => {
    let resolveAction: (() => void) | null = null;
    vi.mocked(api.containerAction).mockReturnValue(
      new Promise((resolve) => {
        resolveAction = () => resolve({ status: status() });
      }),
    );
    vi.mocked(api.getContainers).mockResolvedValue(containersResponse());

    const { result } = renderHook(() => useContainerOps("t1"), { wrapper });

    let first!: Promise<void>;
    act(() => {
      first = result.current.run("devnet", "start");
    });
    await waitFor(() => expect(result.current.busy.devnet).toBe("start"));

    const second = result.current.run("devnet", "stop");

    resolveAction!();
    await act(async () => {
      await Promise.all([first, second]);
    });

    expect(api.containerAction).toHaveBeenCalledTimes(1);
    expect(api.containerAction).toHaveBeenCalledWith("t1", "devnet", "start");
  });

  it("formats a failed action and still clears busy", async () => {
    vi.mocked(api.containerAction).mockRejectedValue(new Error("connection refused"));
    vi.mocked(api.getContainers).mockResolvedValue(containersResponse());

    const { result } = renderHook(() => useContainerOps("t1"), { wrapper });

    await act(async () => {
      await result.current.run("devnet", "stop");
    });

    expect(result.current.error.devnet).toBe("stop failed: connection refused");
    expect(result.current.busy.devnet).toBeNull();
  });

  it("appends the server's hint to a failed action's message", async () => {
    vi.mocked(api.containerAction).mockRejectedValue(new api.ApiError(503, "unreachable", "start Docker Desktop"));
    vi.mocked(api.getContainers).mockResolvedValue(containersResponse());

    const { result } = renderHook(() => useContainerOps("t1"), { wrapper });

    await act(async () => {
      await result.current.run("devnet", "start");
    });

    expect(result.current.error.devnet).toBe("start failed: unreachable — start Docker Desktop");
  });
});

describe("useContainerOps.provision", () => {
  function wireStream(): { push: (ev: SetupEvent) => void; stop: ReturnType<typeof vi.fn> } {
    const stop = vi.fn();
    let onEvent: ((ev: SetupEvent) => void) | null = null;
    vi.mocked(api.streamSetup).mockImplementation((_id, cb) => {
      onEvent = cb;
      return stop;
    });
    return { push: (ev: SetupEvent) => act(() => onEvent!(ev)), stop };
  }

  it("posts, opens the setup stream, and accumulates progress lines", async () => {
    vi.mocked(api.provisionContainer).mockResolvedValue({ status: "accepted" });
    vi.mocked(api.getContainers).mockResolvedValue(containersResponse());
    const stream = wireStream();

    const { result } = renderHook(() => useContainerOps("t1"), { wrapper });

    await act(async () => {
      await result.current.provision("devnet");
    });

    expect(api.provisionContainer).toHaveBeenCalledWith("t1", "devnet");
    expect(api.streamSetup).toHaveBeenCalledWith("t1", expect.any(Function));
    expect(result.current.busy.devnet).toBe("create");
    expect(result.current.activity.devnet).toEqual(["starting…"]);

    stream.push({ stepId: "preflight", line: "checking docker" });
    expect(result.current.activity.devnet).toEqual(["preflight: checking docker"]);
    expect(result.current.busy.devnet).toBe("create");
  });

  it("clears busy, reloads, and stops the stream when the final step reports done", async () => {
    vi.mocked(api.provisionContainer).mockResolvedValue({ status: "accepted" });
    vi.mocked(api.getContainers).mockResolvedValue(containersResponse());
    const stream = wireStream();

    const queryClient = new QueryClient({ defaultOptions: { queries: { retry: false }, mutations: { retry: false } } });
    function localWrapper({ children }: { children: ReactNode }) {
      return <QueryClientProvider client={queryClient}>{children}</QueryClientProvider>;
    }
    const listResult = renderHook(() => useContainers("t1"), { wrapper: localWrapper });
    await waitFor(() => expect(listResult.result.current.isSuccess).toBe(true));
    expect(api.getContainers).toHaveBeenCalledTimes(1);

    const { result } = renderHook(() => useContainerOps("t1"), { wrapper: localWrapper });
    await act(async () => {
      await result.current.provision("devnet");
    });

    stream.push({ stepId: "run", line: "started", done: true });

    await waitFor(() => expect(result.current.busy.devnet).toBeNull());
    expect(stream.stop).toHaveBeenCalledTimes(1);
    expect(result.current.error.devnet).toBeNull();
    // one reload after the initial fetch
    await waitFor(() => expect(api.getContainers).toHaveBeenCalledTimes(2));
  });

  it("surfaces a step error as the generic 'see the log below' message and still reloads", async () => {
    vi.mocked(api.provisionContainer).mockResolvedValue({ status: "accepted" });
    vi.mocked(api.getContainers).mockResolvedValue(containersResponse());
    const stream = wireStream();

    const queryClient = new QueryClient({ defaultOptions: { queries: { retry: false }, mutations: { retry: false } } });
    function localWrapper({ children }: { children: ReactNode }) {
      return <QueryClientProvider client={queryClient}>{children}</QueryClientProvider>;
    }
    const listResult = renderHook(() => useContainers("t1"), { wrapper: localWrapper });
    await waitFor(() => expect(listResult.result.current.isSuccess).toBe(true));

    const { result } = renderHook(() => useContainerOps("t1"), { wrapper: localWrapper });
    await act(async () => {
      await result.current.provision("devnet");
    });

    stream.push({ stepId: "preflight", err: "docker unreachable" });

    await waitFor(() => expect(result.current.busy.devnet).toBeNull());
    expect(result.current.error.devnet).toBe("Provisioning failed — see the log below.");
    expect(result.current.activity.devnet).toEqual(["preflight: docker unreachable"]);
    await waitFor(() => expect(api.getContainers).toHaveBeenCalledTimes(2));
  });

  it("on a POST failure, clears busy and sets an error WITHOUT reloading or opening a stream", async () => {
    vi.mocked(api.provisionContainer).mockRejectedValue(new Error("already provisioning"));
    vi.mocked(api.getContainers).mockResolvedValue(containersResponse());

    const { result } = renderHook(() => useContainerOps("t1"), { wrapper });
    await act(async () => {
      await result.current.provision("devnet");
    });

    expect(result.current.busy.devnet).toBeNull();
    expect(result.current.error.devnet).toBe("already provisioning");
    expect(result.current.activity.devnet).toEqual([]);
    expect(api.streamSetup).not.toHaveBeenCalled();
    // getContainers was never called at all here (no query mounted, no reload).
    expect(api.getContainers).not.toHaveBeenCalled();
  });

  it("run() and provision() share the same busy slot — one blocks the other", async () => {
    let resolveAction: (() => void) | null = null;
    vi.mocked(api.containerAction).mockReturnValue(
      new Promise((resolve) => {
        resolveAction = () => resolve({ status: status() });
      }),
    );
    vi.mocked(api.getContainers).mockResolvedValue(containersResponse());

    const { result } = renderHook(() => useContainerOps("t1"), { wrapper });

    let runPromise!: Promise<void>;
    act(() => {
      runPromise = result.current.run("devnet", "start");
    });
    await waitFor(() => expect(result.current.busy.devnet).toBe("start"));

    await act(async () => {
      await result.current.provision("devnet");
    });
    expect(api.provisionContainer).not.toHaveBeenCalled();

    resolveAction!();
    await act(async () => {
      await runPromise;
    });
  });

  it("closes the stream on unmount", async () => {
    vi.mocked(api.provisionContainer).mockResolvedValue({ status: "accepted" });
    vi.mocked(api.getContainers).mockResolvedValue(containersResponse());
    const stream = wireStream();

    const { result, unmount } = renderHook(() => useContainerOps("t1"), { wrapper });
    await act(async () => {
      await result.current.provision("devnet");
    });

    unmount();
    expect(stream.stop).toHaveBeenCalledTimes(1);
  });
});

describe("useSaveContainerConfig", () => {
  it("saves and invalidates the containers query on success", async () => {
    vi.mocked(api.putContainerConfig).mockResolvedValue({ id: "devnet", configured: true, devnet: devnet() });
    vi.mocked(api.getContainers).mockResolvedValue(containersResponse());

    const listResult = renderHook(() => useContainers("t1"), { wrapper });
    await waitFor(() => expect(listResult.result.current.isSuccess).toBe(true));

    // NOTE: separate wrapper instances would use separate QueryClients, so
    // this test only exercises the mutation resolving; invalidation across
    // the SAME client is covered by ServicesSection's integration test.
    const { result } = renderHook(() => useSaveContainerConfig("t1"), { wrapper });
    result.current.mutate({ svc: "devnet", config: devnet() });

    await waitFor(() => expect(result.current.isSuccess).toBe(true));
    expect(api.putContainerConfig).toHaveBeenCalledWith("t1", "devnet", devnet());
  });

  it("surfaces a rejection as mutation error", async () => {
    vi.mocked(api.putContainerConfig).mockRejectedValue(new Error("bad config"));

    const { result } = renderHook(() => useSaveContainerConfig("t1"), { wrapper });
    result.current.mutate({ svc: "devnet", config: devnet() });

    await waitFor(() => expect(result.current.isError).toBe(true));
  });
});

describe("useWipeContainer", () => {
  it("calls wipeContainer and resolves with the report", async () => {
    const wipeResult: WipeResult = {
      report: {
        ID: "id",
        ContainerName: "valve-devnet",
        ContainerRemoved: true,
        VolumesRemoved: ["vol"],
        VolumesAbsent: null,
        Recreated: false,
        Cascaded: null,
        CascadeSkipped: null,
      },
      status: status({ State: "not-created" }),
    };
    vi.mocked(api.wipeContainer).mockResolvedValue(wipeResult);

    const { result } = renderHook(() => useWipeContainer("t1"), { wrapper });
    result.current.mutate("devnet");

    await waitFor(() => expect(result.current.isSuccess).toBe(true));
    expect(api.wipeContainer).toHaveBeenCalledWith("t1", "devnet");
    expect(result.current.data).toEqual(wipeResult);
  });

  it("does NOT auto-invalidate the containers query on success", async () => {
    vi.mocked(api.getContainers).mockResolvedValue(containersResponse());
    vi.mocked(api.wipeContainer).mockResolvedValue({
      report: {
        ID: "id",
        ContainerName: "valve-devnet",
        ContainerRemoved: true,
        VolumesRemoved: null,
        VolumesAbsent: null,
        Recreated: false,
        Cascaded: null,
        CascadeSkipped: null,
      },
      status: status(),
    });

    const queryClient = new QueryClient({ defaultOptions: { queries: { retry: false }, mutations: { retry: false } } });
    function localWrapper({ children }: { children: ReactNode }) {
      return <QueryClientProvider client={queryClient}>{children}</QueryClientProvider>;
    }

    const listResult = renderHook(() => useContainers("t1"), { wrapper: localWrapper });
    await waitFor(() => expect(listResult.result.current.isSuccess).toBe(true));
    expect(api.getContainers).toHaveBeenCalledTimes(1);

    const { result } = renderHook(() => useWipeContainer("t1"), { wrapper: localWrapper });
    await act(async () => {
      await result.current.mutateAsync("devnet");
    });

    expect(api.getContainers).toHaveBeenCalledTimes(1);
  });
});
