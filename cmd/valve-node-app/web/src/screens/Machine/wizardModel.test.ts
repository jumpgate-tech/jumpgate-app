import { describe, it, expect } from "vitest";
import type { Catalog, Network, SetupEvent } from "../../api";
import {
  approxSize,
  archiveTierTB,
  ARCHIVE_SIZE_BASIS,
  buildStartSetupRequest,
  checkFit,
  clientOptionLabel,
  clientProvider,
  clientRepoDisplay,
  defaultDataDir,
  defaultJwtPath,
  DEFAULT_BEACON_HTTP_PORT,
  DEFAULT_EXEC_HTTP_PORT,
  DEFAULT_EXEC_P2P_PORT,
  doneStepIds,
  erroredStepIds,
  evaluateFit,
  FIT_MARGIN,
  FULL_TIER_FRACTION,
  FULL_SIZE_BASIS,
  fullTierTB,
  hasModeFieldErrors,
  NETWORK_BADGE,
  NETWORK_ORDER,
  neitherFitsWarning,
  nonDefaultPorts,
  parseBindAddr,
  parsePort,
  portOverride,
  resolveClientId,
  runAllDone,
  runAnyError,
  sizeBasisNoteText,
  STEP_PLAN,
  stepErrorLine,
  stepLines,
  tierNeeds,
  validateCheckpointUrl,
  validateModeFields,
  WIZARD_STEPS,
  wizardStepClass,
} from "./wizardModel";

function net(overrides: Partial<Network> = {}): Network {
  return {
    ChainID: 369,
    Name: "PulseChain",
    CheckpointURL: "https://checkpoint.example/",
    ExecClients: ["reth", "erigon-pulse"],
    BeaconClients: ["lighthouse-pulse"],
    LearnURL: "https://learn.valve.city/rpc",
    SnapshotSizeTB: 2,
    SyncLabel: "~2h",
    GenesisSyncLabel: "~4d",
    ...overrides,
  };
}

function catalog(overrides: Partial<Catalog> = {}): Catalog {
  return {
    networks: [net()],
    clients: [
      { id: "reth", kind: "exec", repo: "https://github.com/valve-tech/reth", pinVersion: "1.0", toolchain: "rust", learnUrl: "", snapshotSupported: true },
      { id: "erigon-pulse", kind: "exec", repo: "https://short", pinVersion: "1.0", toolchain: "go", learnUrl: "", snapshotSupported: false },
      { id: "lighthouse-pulse", kind: "beacon", repo: "https://github.com/valve-tech/lighthouse-pulse", pinVersion: "1.0", toolchain: "rust", learnUrl: "", snapshotSupported: false },
    ],
    ...overrides,
  };
}

describe("constants — byte-exact with wizard.ts", () => {
  it("FIT_MARGIN is the server preflight's 10% headroom", () => {
    expect(FIT_MARGIN).toBe(1.1);
  });

  it("FULL_TIER_FRACTION is the unsourced half-of-archive placeholder", () => {
    expect(FULL_TIER_FRACTION).toBe(0.5);
  });

  it("size-basis labels are exact", () => {
    expect(ARCHIVE_SIZE_BASIS).toBe("Valve reth snapshot");
    expect(FULL_SIZE_BASIS).toBe("rough estimate");
  });

  it("port defaults match internal/catalog/units.go", () => {
    expect(DEFAULT_EXEC_HTTP_PORT).toBe(8545);
    expect(DEFAULT_BEACON_HTTP_PORT).toBe(5052);
    expect(DEFAULT_EXEC_P2P_PORT).toBe(30303);
  });

  it("network order and badges", () => {
    expect(NETWORK_ORDER).toEqual([369, 943, 1]);
    expect(NETWORK_BADGE).toEqual({ 369: "default", 943: "practise here first" });
  });

  it("STEP_PLAN is the fixed 7-step setup sequence, in order", () => {
    expect(STEP_PLAN.map((s) => s.id)).toEqual([
      "preflight",
      "toolchain",
      "install-exec",
      "install-beacon",
      "wire",
      "start",
      "handshake",
    ]);
  });

  it("WIZARD_STEPS is the 5-step progress rail", () => {
    expect(WIZARD_STEPS.map((s) => s.id)).toEqual(["network", "clients", "mode", "review", "run"]);
  });
});

describe("approxSize", () => {
  it("renders — for zero or negative", () => {
    expect(approxSize(0)).toBe("—");
    expect(approxSize(-1)).toBe("—");
  });

  it("renders decimal TB at or above 1", () => {
    expect(approxSize(1)).toBe("~1.0 TB");
    expect(approxSize(2.345)).toBe("~2.3 TB");
  });

  it("drops to rounded GB below 1 TB", () => {
    expect(approxSize(0.75)).toBe("~750 GB");
    expect(approxSize(0.001)).toBe("~1 GB");
  });
});

describe("archiveTierTB / fullTierTB / tierNeeds", () => {
  it("archive tier is the raw snapshot size; full tier is half of it", () => {
    const n = net({ SnapshotSizeTB: 10 });
    expect(archiveTierTB(n)).toBe(10);
    expect(fullTierTB(n)).toBe(5);
  });

  it("tierNeeds applies the 1.1x headroom margin in bytes", () => {
    const n = net({ SnapshotSizeTB: 10 });
    const needs = tierNeeds(n);
    expect(needs.archive).toBe(10 * 1e12 * 1.1);
    expect(needs.full).toBe(5 * 1e12 * 1.1);
    expect(needs.archive).toBe(1.1e13);
    expect(needs.full).toBe(5.5e12);
  });
});

describe("checkFit", () => {
  it("fits exactly at the margin threshold", () => {
    const n = net({ SnapshotSizeTB: 1 });
    const needs = tierNeeds(n);
    expect(checkFit(n, needs.archive).archiveFits).toBe(true);
    expect(checkFit(n, needs.archive - 1).archiveFits).toBe(false);
    expect(checkFit(n, needs.full).fullFits).toBe(true);
    expect(checkFit(n, needs.full - 1).fullFits).toBe(false);
  });
});

describe("evaluateFit", () => {
  it("downgrades archive to full when only full fits", () => {
    const n = net({ SnapshotSizeTB: 2 }); // archive needs 2.2e12, full needs 1.1e12
    const free = 1.5e12; // fits full, not archive
    const result = evaluateFit(n, free, true, "/data");
    expect(result.archive).toBe(false);
    expect(result.downgradeNote).toBe(
      "Not enough space at /data for archive (~2.0 TB, Valve reth snapshot) — switched to Full (~1.0 TB, rough estimate). Pick a location with more room to run archive.",
    );
  });

  it("does not downgrade when archive already fits", () => {
    const n = net({ SnapshotSizeTB: 1 });
    const needs = tierNeeds(n);
    const result = evaluateFit(n, needs.archive, true, "/data");
    expect(result.archive).toBe(true);
    expect(result.downgradeNote).toBeNull();
  });

  it("does not downgrade when neither tier fits (nothing to switch to)", () => {
    const n = net({ SnapshotSizeTB: 5 });
    const result = evaluateFit(n, 1, true, "/data");
    expect(result.archive).toBe(true);
    expect(result.downgradeNote).toBeNull();
  });

  it("never overrides a manual full pick", () => {
    const n = net({ SnapshotSizeTB: 2 });
    const result = evaluateFit(n, 1.5e12, false, "/data");
    expect(result.archive).toBe(false);
    expect(result.downgradeNote).toBeNull();
  });

  it("is a no-op with no network or no probe yet", () => {
    expect(evaluateFit(undefined, 100, true, "/data")).toEqual({ archive: true, downgradeNote: null });
    expect(evaluateFit(net(), null, true, "/data")).toEqual({ archive: true, downgradeNote: null });
  });
});

describe("neitherFitsWarning", () => {
  it("warns only when full doesn't fit either", () => {
    const n = net({ SnapshotSizeTB: 5 });
    expect(neitherFitsWarning(n, 1)).toContain("Neither full");
    expect(neitherFitsWarning(n, tierNeeds(n).full)).toBeNull();
  });
});

describe("sizeBasisNoteText", () => {
  it("names the network and both bases", () => {
    const text = sizeBasisNoteText(net({ Name: "PulseChain", SnapshotSizeTB: 2 }));
    expect(text).toContain("~2.0 TB is the measured size of Valve's reth snapshot for PulseChain");
    expect(text).toContain("not a measurement");
  });
});

describe("defaultDataDir / defaultJwtPath", () => {
  it("derives the default data dir from the chain id", () => {
    expect(defaultDataDir(369)).toBe("/var/lib/valve-node-app/369");
    expect(defaultDataDir(null)).toBe("");
  });

  it("derives the default jwt path from a data dir", () => {
    expect(defaultJwtPath("/var/lib/valve-node-app/369")).toBe("/var/lib/valve-node-app/369/jwt.hex");
  });
});

describe("client picker", () => {
  it("extracts the publishing org from a 4+-segment repo URL", () => {
    expect(clientProvider("https://github.com/valve-tech/reth")).toBe("valve-tech");
  });

  it("falls back to the whole repo string when it has too few segments", () => {
    expect(clientProvider("https://short")).toBe("https://short");
  });

  it("labels a known client as '<id> — <provider>'", () => {
    expect(clientOptionLabel("reth", catalog())).toBe("reth — valve-tech");
  });

  it("falls back to the bare id for an unknown client", () => {
    expect(clientOptionLabel("mystery", catalog())).toBe("mystery");
  });

  it("strips the scheme for display", () => {
    expect(clientRepoDisplay("https://github.com/valve-tech/reth")).toBe("github.com/valve-tech/reth");
  });

  it("resolveClientId keeps a valid current pick", () => {
    expect(resolveClientId("erigon-pulse", ["reth", "erigon-pulse"])).toBe("erigon-pulse");
  });

  it("resolveClientId falls back to the network's first client when invalid or null", () => {
    expect(resolveClientId(null, ["reth", "erigon-pulse"])).toBe("reth");
    expect(resolveClientId("geth", ["reth", "erigon-pulse"])).toBe("reth");
  });

  it("resolveClientId is null when the network offers nothing", () => {
    expect(resolveClientId("reth", [])).toBeNull();
  });
});

describe("validateCheckpointUrl", () => {
  it("accepts empty (network default)", () => {
    expect(validateCheckpointUrl("")).toBeNull();
  });

  it("accepts http(s) URLs", () => {
    expect(validateCheckpointUrl("http://x")).toBeNull();
    expect(validateCheckpointUrl("https://x")).toBeNull();
  });

  it("rejects other schemes or garbage", () => {
    expect(validateCheckpointUrl("ftp://x")).toMatch(/http\(s\) URL/);
    expect(validateCheckpointUrl("not a url")).toMatch(/http\(s\) URL/);
  });
});

describe("parseBindAddr", () => {
  it("accepts empty as loopback default", () => {
    expect(parseBindAddr("")).toEqual({});
  });

  it("accepts a valid IPv4 address", () => {
    expect(parseBindAddr("100.64.1.2")).toEqual({ addr: "100.64.1.2" });
  });

  it("rejects an out-of-range IPv4 octet", () => {
    expect(parseBindAddr("999.1.1.1").error).toMatch(/0–255/);
  });

  it("accepts a loose IPv6 literal", () => {
    expect(parseBindAddr("::1")).toEqual({ addr: "::1" });
  });

  it("rejects a hostname", () => {
    expect(parseBindAddr("my-host").error).toMatch(/valid IP address/);
  });
});

describe("parsePort", () => {
  it("accepts empty as unset", () => {
    expect(parsePort("")).toEqual({});
  });

  it("accepts a valid port", () => {
    expect(parsePort("8080")).toEqual({ port: 8080 });
    expect(parsePort("65535")).toEqual({ port: 65535 });
    expect(parsePort("1")).toEqual({ port: 1 });
  });

  it("rejects 0 and values above 65535", () => {
    expect(parsePort("0").error).toMatch(/1 and 65535/);
    expect(parsePort("70000").error).toMatch(/1 and 65535/);
  });

  it("rejects decimals, signs, and non-numeric input", () => {
    expect(parsePort("12.5").error).toMatch(/whole number/);
    expect(parsePort("-1").error).toMatch(/whole number/);
    expect(parsePort("abc").error).toMatch(/whole number/);
  });
});

describe("portOverride", () => {
  it("omits an invalid value", () => {
    expect(portOverride("abc", DEFAULT_EXEC_HTTP_PORT)).toBeUndefined();
  });

  it("omits a value equal to the default", () => {
    expect(portOverride("8545", DEFAULT_EXEC_HTTP_PORT)).toBeUndefined();
  });

  it("returns a valid, non-default value", () => {
    expect(portOverride("9000", DEFAULT_EXEC_HTTP_PORT)).toBe(9000);
  });
});

describe("validateModeFields / hasModeFieldErrors", () => {
  function fields(overrides: Partial<Parameters<typeof validateModeFields>[0]> = {}) {
    return {
      execHTTPPort: "",
      beaconHTTPPort: "",
      execP2PPort: "",
      rpcBindAddr: "",
      checkpoint: true,
      checkpointUrl: "",
      execSnapshot: false,
      snapshotKey: "",
      ...overrides,
    };
  }

  it("is error-free with all-blank optional fields", () => {
    const errors = validateModeFields(fields());
    expect(hasModeFieldErrors(errors)).toBe(false);
  });

  it("only validates the checkpoint URL while checkpoint sync is on", () => {
    expect(validateModeFields(fields({ checkpoint: true, checkpointUrl: "bad" })).checkpointUrlError).not.toBeNull();
    expect(validateModeFields(fields({ checkpoint: false, checkpointUrl: "bad" })).checkpointUrlError).toBeNull();
  });

  it("requires a snapshot key only when exec snapshot is opted in", () => {
    expect(validateModeFields(fields({ execSnapshot: true, snapshotKey: "" })).snapshotKeyError).toMatch(
      /snapshot key is required/,
    );
    expect(validateModeFields(fields({ execSnapshot: true, snapshotKey: "  " })).snapshotKeyError).toMatch(
      /snapshot key is required/,
    );
    expect(validateModeFields(fields({ execSnapshot: true, snapshotKey: "vk_abc" })).snapshotKeyError).toBeNull();
    expect(validateModeFields(fields({ execSnapshot: false, snapshotKey: "" })).snapshotKeyError).toBeNull();
  });

  it("surfaces a bad port as an error and flips hasModeFieldErrors", () => {
    const errors = validateModeFields(fields({ execHTTPPort: "abc" }));
    expect(errors.execHTTPPortError).not.toBeNull();
    expect(hasModeFieldErrors(errors)).toBe(true);
  });
});

describe("buildStartSetupRequest", () => {
  const base = {
    chainId: 369,
    execId: "reth",
    beaconId: "lighthouse-pulse",
    archive: true,
    dataDir: "",
    jwtPath: "",
    execHTTPPort: "",
    beaconHTTPPort: "",
    execP2PPort: "",
    rpcBindAddr: "",
    checkpoint: true,
    checkpointUrl: "",
    execSnapshot: false,
    snapshotKey: "",
  };

  it("sends only the required fields when everything else is left at default", () => {
    expect(buildStartSetupRequest(base)).toEqual({
      ChainID: 369,
      ExecID: "reth",
      BeaconID: "lighthouse-pulse",
      Archive: true,
    });
  });

  it("includes DataDir/JWTPath only when non-empty", () => {
    const req = buildStartSetupRequest({ ...base, dataDir: "/data", jwtPath: "/data/jwt.hex" });
    expect(req.DataDir).toBe("/data");
    expect(req.JWTPath).toBe("/data/jwt.hex");
  });

  it("includes only non-default, valid ports", () => {
    const req = buildStartSetupRequest({
      ...base,
      execHTTPPort: "8545", // equals default — omitted
      beaconHTTPPort: "6000", // non-default — included
      execP2PPort: "bogus", // invalid — omitted
    });
    expect(req.ExecHTTPPort).toBeUndefined();
    expect(req.BeaconHTTPPort).toBe(6000);
    expect(req.ExecP2PPort).toBeUndefined();
  });

  it("includes RPCBindAddr only when a valid non-empty address is set", () => {
    expect(buildStartSetupRequest({ ...base, rpcBindAddr: "100.64.1.2" }).RPCBindAddr).toBe("100.64.1.2");
    expect(buildStartSetupRequest({ ...base, rpcBindAddr: "" }).RPCBindAddr).toBeUndefined();
  });

  it("sets NoCheckpoint when checkpoint sync is off, and never sends a URL alongside it", () => {
    const req = buildStartSetupRequest({ ...base, checkpoint: false, checkpointUrl: "https://x" });
    expect(req.NoCheckpoint).toBe(true);
    expect(req.CheckpointURL).toBeUndefined();
  });

  it("sends CheckpointURL only when checkpoint sync is on and a URL was entered", () => {
    const req = buildStartSetupRequest({ ...base, checkpoint: true, checkpointUrl: "https://cp.example" });
    expect(req.NoCheckpoint).toBeUndefined();
    expect(req.CheckpointURL).toBe("https://cp.example");
  });

  it("sends ExecSnapshot+SnapshotKey only when opted in", () => {
    const req = buildStartSetupRequest({ ...base, execSnapshot: true, snapshotKey: "vk_abc" });
    expect(req.ExecSnapshot).toBe(true);
    expect(req.SnapshotKey).toBe("vk_abc");
    expect(buildStartSetupRequest(base).ExecSnapshot).toBeUndefined();
  });
});

describe("nonDefaultPorts", () => {
  it("is empty when every port is at its default", () => {
    expect(nonDefaultPorts({ execHTTPPort: "", beaconHTTPPort: "", execP2PPort: "" })).toEqual([]);
  });

  it("lists only the overridden ports, in exec-http/beacon-http/exec-p2p order", () => {
    expect(
      nonDefaultPorts({ execHTTPPort: "9000", beaconHTTPPort: "", execP2PPort: "31000" }),
    ).toEqual([
      { label: "exec HTTP", port: 9000 },
      { label: "exec p2p", port: 31000 },
    ]);
  });
});

describe("wizardStepClass", () => {
  it("classifies past/current/future relative to the current step", () => {
    expect(wizardStepClass("mode", "network")).toBe("past");
    expect(wizardStepClass("mode", "mode")).toBe("current");
    expect(wizardStepClass("mode", "review")).toBe("future");
  });
});

describe("doneStepIds / erroredStepIds / stepLines / stepErrorLine", () => {
  const events: SetupEvent[] = [
    { stepId: "preflight", line: "checking docker" },
    { stepId: "preflight", done: true },
    { stepId: "toolchain", line: "go 1.22" },
    { stepId: "install-exec", err: "download failed" },
  ];

  it("collects step ids that reported done", () => {
    expect(doneStepIds(events)).toEqual(new Set(["preflight"]));
  });

  it("collects step ids that reported an error", () => {
    expect(erroredStepIds(events)).toEqual(new Set(["install-exec"]));
  });

  it("collects a step's lines in arrival order", () => {
    expect(stepLines(events, "preflight")).toEqual(["checking docker"]);
    expect(stepLines(events, "toolchain")).toEqual(["go 1.22"]);
    expect(stepLines(events, "handshake")).toEqual([]);
  });

  it("caps lines at the last 5", () => {
    const many: SetupEvent[] = Array.from({ length: 8 }, (_, i) => ({ stepId: "wire", line: `line ${i}` }));
    expect(stepLines(many, "wire")).toEqual(["line 3", "line 4", "line 5", "line 6", "line 7"]);
  });

  it("returns the first error line for a step", () => {
    expect(stepErrorLine(events, "install-exec")).toBe("download failed");
    expect(stepErrorLine(events, "wire")).toBeUndefined();
  });
});

describe("runAllDone / runAnyError", () => {
  it("is not done with no events", () => {
    expect(runAllDone([])).toBe(false);
  });

  it("is done once every STEP_PLAN step has reported done", () => {
    const events: SetupEvent[] = STEP_PLAN.map((s) => ({ stepId: s.id, done: true }));
    expect(runAllDone(events)).toBe(true);
  });

  it("is done once the terminal handshake step alone reports done", () => {
    const events: SetupEvent[] = [{ stepId: "handshake", done: true }];
    expect(runAllDone(events)).toBe(true);
  });

  it("is not done on an intermediate step, even if done", () => {
    const events: SetupEvent[] = [{ stepId: "preflight", done: true }];
    expect(runAllDone(events)).toBe(false);
  });

  it("is not done when handshake ran but hasn't reported done", () => {
    const events: SetupEvent[] = [{ stepId: "handshake", line: "verifying" }];
    expect(runAllDone(events)).toBe(false);
  });

  it("reports any error regardless of which step it's on", () => {
    expect(runAnyError([{ stepId: "preflight", err: "boom" }])).toBe(true);
    expect(runAnyError([{ stepId: "preflight", line: "ok" }])).toBe(false);
  });
});
