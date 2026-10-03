import { describe, it, expect } from "vitest";
import type { Target, Catalog } from "../../api";
import { orderTargets, targetStatus, slugify } from "./targetsModel";

const CATALOG: Catalog = {
  networks: [
    {
      ChainID: 1,
      Name: "Ethereum",
      CheckpointURL: "",
      ExecClients: [],
      BeaconClients: [],
      LearnURL: "",
      SnapshotSizeTB: 0,
      SyncLabel: "",
      GenesisSyncLabel: "",
    },
  ],
  clients: [],
};

describe("orderTargets", () => {
  it("puts local targets first, preserving relative order otherwise", () => {
    const targets: Target[] = [
      { id: "ssh1", mode: "ssh" },
      { id: "local", mode: "local" },
      { id: "ssh2", mode: "ssh" },
    ];
    expect(orderTargets(targets).map((t) => t.id)).toEqual(["local", "ssh1", "ssh2"]);
  });

  it("leaves an all-SSH list untouched", () => {
    const targets: Target[] = [
      { id: "ssh1", mode: "ssh" },
      { id: "ssh2", mode: "ssh" },
    ];
    expect(orderTargets(targets).map((t) => t.id)).toEqual(["ssh1", "ssh2"]);
  });

  it("does not mutate the input array", () => {
    const targets: Target[] = [
      { id: "ssh1", mode: "ssh" },
      { id: "local", mode: "local" },
    ];
    const copy = [...targets];
    orderTargets(targets);
    expect(targets).toEqual(copy);
  });
});

describe("targetStatus", () => {
  it("returns cant-run for an unwired target that cannot run a node", () => {
    const t: Target = { id: "local", mode: "local" };
    expect(targetStatus(t, CATALOG, false, "darwin")).toEqual({ kind: "cant-run", hostOS: "darwin" });
  });

  it("returns not-set-up for an unwired target that CAN run a node", () => {
    const t: Target = { id: "ssh1", mode: "ssh" };
    expect(targetStatus(t, CATALOG, true, "linux")).toEqual({ kind: "not-set-up" });
  });

  it("returns wired with the catalog's network name when the chain is known", () => {
    const t: Target = {
      id: "t1",
      mode: "local",
      wire: { ChainID: 1, ExecID: "reth", BeaconID: "lighthouse", DataDir: "", JWTPath: "", Archive: false },
    };
    expect(targetStatus(t, CATALOG, true, "linux")).toEqual({
      kind: "wired",
      networkName: "Ethereum",
      execId: "reth",
      beaconId: "lighthouse",
      archive: false,
    });
  });

  it("falls back to 'chain <id>' when the chain isn't in the catalog", () => {
    const t: Target = {
      id: "t1",
      mode: "local",
      wire: { ChainID: 999, ExecID: "geth", BeaconID: "prysm", DataDir: "", JWTPath: "", Archive: true },
    };
    expect(targetStatus(t, CATALOG, true, "linux")).toEqual({
      kind: "wired",
      networkName: "chain 999",
      execId: "geth",
      beaconId: "prysm",
      archive: true,
    });
  });

  it("ignores canRunNode/hostOS once a target is wired", () => {
    const t: Target = {
      id: "t1",
      mode: "local",
      wire: { ChainID: 1, ExecID: "reth", BeaconID: "lighthouse", DataDir: "", JWTPath: "", Archive: false },
    };
    expect(targetStatus(t, CATALOG, false, "darwin").kind).toBe("wired");
  });
});

describe("slugify", () => {
  it("lowercases and dashes non-alphanumeric runs", () => {
    expect(slugify("My Server 01")).toBe("my-server-01");
  });

  it("trims leading and trailing dashes", () => {
    expect(slugify("--my.host--")).toBe("my-host");
  });

  it("collapses an IP into a dashed slug", () => {
    expect(slugify("203.0.113.10")).toBe("203-0-113-10");
  });

  it("falls back to 'target' when nothing alphanumeric survives", () => {
    expect(slugify("---")).toBe("target");
    expect(slugify("")).toBe("target");
  });
});
