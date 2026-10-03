import { describe, it, expect } from "vitest";
import type { ContainerStatus, ContainerView, DevnetConfig, DockerView } from "../../api";
import {
  actionLabel,
  devnetConfigError,
  devnetSummary,
  dockerBannerTitle,
  dockerOk,
  exitCodeLine,
  parseIntOr,
  provisionFinished,
  provisionLine,
  showNoEndpointsMessage,
  stateBadge,
  wipeConfirmValid,
} from "./servicesModel";

function status(overrides: Partial<ContainerStatus> = {}): ContainerStatus {
  return {
    ID: "id",
    ContainerName: "devnet",
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
    actions: null,
    wipeDiscards: "the devnet's chain data",
    restartsOnWipe: null,
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

describe("stateBadge", () => {
  it("maps every container state to its badge", () => {
    expect(stateBadge(view({ status: status({ State: "running" }) }))).toEqual({ text: "running", kind: "ok" });
    expect(stateBadge(view({ status: status({ State: "created-but-stopped" }) }))).toEqual({
      text: "stopped",
      kind: "warn",
    });
    expect(stateBadge(view({ status: status({ State: "not-created" }) }))).toEqual({
      text: "not created",
      kind: "neutral",
    });
    expect(stateBadge(view({ status: status({ State: "unknown" }) }))).toEqual({ text: "unknown", kind: "bad" });
  });
});

describe("exitCodeLine", () => {
  it("is null while running", () => {
    expect(exitCodeLine(view({ status: status({ State: "running", ExitCode: 1 }) }))).toBeNull();
  });

  it("is null when stopped with exit code 0", () => {
    expect(exitCodeLine(view({ status: status({ State: "created-but-stopped", ExitCode: 0 }) }))).toBeNull();
  });

  it("reports a plain non-zero exit code", () => {
    expect(exitCodeLine(view({ status: status({ State: "created-but-stopped", ExitCode: 1 }) }))).toBe(
      "It exited with code 1.",
    );
  });

  it("calls out 137 as a likely OOM kill", () => {
    expect(exitCodeLine(view({ status: status({ State: "created-but-stopped", ExitCode: 137 }) }))).toBe(
      "It exited with code 137 (137 is a kill — most often the machine ran out of memory).",
    );
  });
});

describe("showNoEndpointsMessage", () => {
  it("is true only when running with zero endpoints", () => {
    expect(showNoEndpointsMessage(view({ status: status({ State: "running" }), endpoints: [] }))).toBe(true);
    expect(showNoEndpointsMessage(view({ status: status({ State: "running" }), endpoints: null }))).toBe(true);
  });

  it("is false when stopped or not-created, even with zero endpoints", () => {
    expect(
      showNoEndpointsMessage(view({ status: status({ State: "created-but-stopped" }), endpoints: [] })),
    ).toBe(false);
    expect(showNoEndpointsMessage(view({ status: status({ State: "not-created" }), endpoints: [] }))).toBe(
      false,
    );
  });

  it("is false when there are endpoints", () => {
    expect(
      showNoEndpointsMessage(
        view({ status: status({ State: "running" }), endpoints: [{ label: "RPC", url: "http://x" }] }),
      ),
    ).toBe(false);
  });
});

describe("devnetSummary", () => {
  it("renders — when no devnet config is present", () => {
    expect(devnetSummary(view({ devnet: undefined }))).toBe("—");
  });

  it("renders the chain id, block time and both endpoints", () => {
    expect(devnetSummary(view({ devnet: devnet() }))).toBe(
      "Chain 1337 · a block every 2s · JSON-RPC on 127.0.0.1:8545 · WebSocket on 127.0.0.1:8546",
    );
  });
});

describe("devnetConfigError", () => {
  it("rejects matching HTTP/WS ports", () => {
    expect(devnetConfigError(devnet({ HTTPPort: 9000, WSPort: 9000 }))?.length).toBeGreaterThan(0);
  });

  it("accepts distinct ports", () => {
    expect(devnetConfigError(devnet({ HTTPPort: 8545, WSPort: 8546 }))).toBeNull();
  });
});

describe("wipeConfirmValid", () => {
  it("requires an exact (trimmed) match of the service id", () => {
    expect(wipeConfirmValid("devnet", "devnet")).toBe(true);
    expect(wipeConfirmValid("  devnet  ", "devnet")).toBe(true);
    expect(wipeConfirmValid("Devnet", "devnet")).toBe(false);
    expect(wipeConfirmValid("", "devnet")).toBe(false);
  });
});

describe("dockerOk / dockerBannerTitle", () => {
  function docker(overrides: Partial<DockerView> = {}): DockerView {
    return { present: true, reachable: true, flavor: "docker", ...overrides };
  }

  it("is ok only when present, reachable and hint-free", () => {
    expect(dockerOk(docker())).toBe(true);
    expect(dockerOk(docker({ reachable: false }))).toBe(false);
    expect(dockerOk(docker({ hint: "start docker" }))).toBe(false);
    expect(dockerOk(docker({ present: false }))).toBe(false);
  });

  it("titles the banner by whether docker is even installed", () => {
    expect(dockerBannerTitle(docker({ present: false }))).toBe("No docker engine on this machine");
    expect(dockerBannerTitle(docker({ present: true, reachable: false }))).toBe(
      "Docker is installed, but no engine answered",
    );
  });
});

describe("actionLabel", () => {
  it("labels create as 'Create devnet' for the devnet service", () => {
    expect(actionLabel("devnet", "create")).toBe("Create devnet");
  });

  it("passes through the plain label for non-create actions", () => {
    expect(actionLabel("devnet", "start")).toBe("Start");
    expect(actionLabel("devnet", "wipe")).toBe("Wipe…");
  });

  it("is empty for an unknown action id", () => {
    expect(actionLabel("devnet", "bogus")).toBe("");
  });
});

describe("parseIntOr", () => {
  it("parses a valid integer string", () => {
    expect(parseIntOr("8545", 0)).toBe(8545);
  });

  it("falls back on an unparseable value", () => {
    expect(parseIntOr("not-a-port", 8545)).toBe(8545);
    expect(parseIntOr("", 8545)).toBe(8545);
  });
});

describe("provisionLine / provisionFinished", () => {
  it("formats an in-progress line", () => {
    expect(provisionLine({ stepId: "preflight", line: "checking docker" })).toBe("preflight: checking docker");
  });

  it("formats a step with no line as done", () => {
    expect(provisionLine({ stepId: "preflight" })).toBe("preflight: done");
  });

  it("prioritizes an error over a line", () => {
    expect(provisionLine({ stepId: "run", line: "starting", err: "boom" })).toBe("run: boom");
  });

  it("is finished on any error, regardless of step", () => {
    expect(provisionFinished({ stepId: "preflight", err: "boom" })).toBe(true);
  });

  it("is finished when the final step reports done", () => {
    expect(provisionFinished({ stepId: "run", done: true })).toBe(true);
  });

  it("is not finished on an intermediate step, even if done", () => {
    expect(provisionFinished({ stepId: "preflight", done: true })).toBe(false);
  });

  it("is not finished on the final step until done", () => {
    expect(provisionFinished({ stepId: "run", done: false })).toBe(false);
  });
});
