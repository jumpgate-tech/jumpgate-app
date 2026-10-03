import { describe, it, expect } from "vitest";
import { computeFleetVerdict } from "./verdict";
import type { Target, Catalog } from "./api";

const CATALOG: Catalog = {
  networks: [{ ChainID: 1, Name: "Ethereum", CheckpointURL: "", ExecClients: [], BeaconClients: [], LearnURL: "", SnapshotSizeTB: 0, SyncLabel: "", GenesisSyncLabel: "" }],
  clients: [],
};

const wired = (id: string, chainId: number): Target => ({
  id,
  mode: "local",
  wire: { ChainID: chainId, ExecID: "reth", BeaconID: "lighthouse", DataDir: "", JWTPath: "", Archive: false },
});

describe("computeFleetVerdict", () => {
  it("is ok with no machines yet when the fleet is empty", () => {
    expect(computeFleetVerdict([], CATALOG)).toEqual({ level: "ok", sentence: "No machines yet.", machines: [] });
  });

  it("flags attention for every unwired machine by id", () => {
    const v = computeFleetVerdict([{ id: "t1", mode: "local" }, { id: "t2", mode: "local" }], CATALOG);
    expect(v).toEqual({ level: "attention", sentence: "2 machines still need setup.", machines: ["t1", "t2"] });
  });

  it("names each distinct chain once, falling back to 'chain <id>' for one the catalog doesn't know", () => {
    const v = computeFleetVerdict([wired("t1", 1), wired("t2", 1), wired("t3", 1337)], CATALOG);
    expect(v.level).toBe("ok");
    expect(v.sentence).toBe("All 3 machines healthy — Ethereum and chain 1337.");
    expect(v.machines).toEqual([]);
  });
});
