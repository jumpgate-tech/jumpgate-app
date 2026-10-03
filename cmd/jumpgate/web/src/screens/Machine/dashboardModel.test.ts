import { describe, it, expect } from "vitest";
import type { Snapshot } from "../../api";
import {
  beaconSyncStatus,
  clearConfirmValid,
  diskWarn,
  execSyncStatus,
  highDiskUsagePct,
  nextBlocksPerSec,
  serviceActionDisabled,
  servicePct,
  storageAdvancing,
  syncETA,
  topStatus,
} from "./dashboardModel";

function snap(overrides: Partial<Snapshot> = {}): Snapshot {
  return {
    at: "2026-08-01T12:00:00.000Z",
    execSyncing: false,
    execHead: 100,
    refHead: 100,
    beaconSlot: 100,
    beaconDistance: 0,
    execPeers: 5,
    beaconPeers: 5,
    diskUsedPct: 10,
    execActive: true,
    beaconActive: true,
    ...overrides,
  };
}

describe("topStatus", () => {
  it("reads not-running when both services are down, regardless of sync state", () => {
    expect(topStatus(snap({ execActive: false, beaconActive: false, execSyncing: true }))).toEqual({
      text: "Node not running",
      kind: "bad",
    });
  });

  it("reads syncing when exec is syncing", () => {
    expect(topStatus(snap({ execSyncing: true }))).toEqual({ text: "Syncing", kind: "warn" });
  });

  it("reads syncing when the beacon has distance", () => {
    expect(topStatus(snap({ beaconDistance: 5 }))).toEqual({ text: "Syncing", kind: "warn" });
  });

  it("reads running/synced otherwise", () => {
    expect(topStatus(snap())).toEqual({ text: "Running · synced", kind: "ok" });
  });
});

describe("nextBlocksPerSec", () => {
  it("leaves the rate unchanged on the very first tick (no previous snapshot)", () => {
    expect(nextBlocksPerSec(null, snap(), null)).toBeNull();
    expect(nextBlocksPerSec(null, snap(), 3)).toBe(3);
  });

  it("computes a fresh rate when there is no prior rate", () => {
    const prev = snap({ at: "2026-08-01T12:00:00.000Z", execHead: 100 });
    const curr = snap({ at: "2026-08-01T12:00:10.000Z", execHead: 150 });
    expect(nextBlocksPerSec(prev, curr, null)).toBe(5);
  });

  it("exponentially smooths against a prior rate (0.7 old / 0.3 new)", () => {
    const prev = snap({ at: "2026-08-01T12:00:00.000Z", execHead: 100 });
    const curr = snap({ at: "2026-08-01T12:00:10.000Z", execHead: 200 });
    // new instantaneous rate = 10; prior rate = 4 -> 4*0.7 + 10*0.3 = 5.8
    expect(nextBlocksPerSec(prev, curr, 4)).toBeCloseTo(5.8);
  });

  it("leaves the rate unchanged on a non-positive delta (clock skew or a counter reset)", () => {
    const prev = snap({ at: "2026-08-01T12:00:10.000Z", execHead: 200 });
    const curr = snap({ at: "2026-08-01T12:00:00.000Z", execHead: 100 });
    expect(nextBlocksPerSec(prev, curr, 4)).toBe(4);
  });
});

describe("syncETA", () => {
  it("reports unavailable lag when there is no reference head", () => {
    expect(syncETA(snap({ refHead: 0, execHead: 0 }), null)).toEqual({ lag: null, eta: "—" });
  });

  it("reports caught up when lag is zero or negative", () => {
    expect(syncETA(snap({ refHead: 100, execHead: 100 }), 5)).toEqual({ lag: 0, eta: "caught up" });
    expect(syncETA(snap({ refHead: 100, execHead: 150 }), 5)).toEqual({ lag: -50, eta: "caught up" });
  });

  it("computes a duration ETA when lagging with a positive rate", () => {
    const { lag, eta } = syncETA(snap({ refHead: 200, execHead: 100 }), 10);
    expect(lag).toBe(100);
    expect(eta).toBe("~10s");
  });

  it("shows — when lagging but the rate is unknown or zero", () => {
    expect(syncETA(snap({ refHead: 200, execHead: 100 }), null).eta).toBe("—");
    expect(syncETA(snap({ refHead: 200, execHead: 100 }), 0).eta).toBe("—");
  });
});

describe("execSyncStatus", () => {
  it("reads stopped when inactive", () => {
    expect(execSyncStatus(snap({ execActive: false }))).toEqual({ text: "stopped", kind: "bad" });
  });
  it("reads syncing when active and syncing", () => {
    expect(execSyncStatus(snap({ execSyncing: true }))).toEqual({ text: "syncing", kind: "warn" });
  });
  it("reads no data when active, not syncing, and head is zero", () => {
    expect(execSyncStatus(snap({ execHead: 0 }))).toEqual({ text: "no data", kind: "neutral" });
  });
  it("reads synced otherwise", () => {
    expect(execSyncStatus(snap())).toEqual({ text: "synced", kind: "ok" });
  });
});

describe("beaconSyncStatus", () => {
  it("reads stopped when inactive", () => {
    expect(beaconSyncStatus(snap({ beaconActive: false }))).toEqual({ text: "stopped", kind: "bad" });
  });
  it("reads no data when active with a zero slot", () => {
    expect(beaconSyncStatus(snap({ beaconSlot: 0 }))).toEqual({ text: "no data", kind: "neutral" });
  });
  it("reads synced when distance is zero", () => {
    expect(beaconSyncStatus(snap())).toEqual({ text: "synced", kind: "ok" });
  });
  it("reads syncing when distance is positive", () => {
    expect(beaconSyncStatus(snap({ beaconDistance: 3 }))).toEqual({ text: "syncing", kind: "warn" });
  });
});

describe("diskWarn", () => {
  it("is false below the threshold and true at/above it", () => {
    expect(diskWarn(snap({ diskUsedPct: highDiskUsagePct - 1 }))).toBe(false);
    expect(diskWarn(snap({ diskUsedPct: highDiskUsagePct }))).toBe(true);
    expect(diskWarn(snap({ diskUsedPct: 99 }))).toBe(true);
  });
});

describe("servicePct", () => {
  it("is zero when there is no expected size", () => {
    expect(servicePct(500, 0)).toBe(0);
  });
  it("computes a percentage, capped at 100", () => {
    expect(servicePct(50, 200)).toBe(25);
    expect(servicePct(500, 200)).toBe(100);
  });
});

describe("storageAdvancing", () => {
  it("is true only when lagging with a positive known rate", () => {
    expect(storageAdvancing(100, 5)).toBe(true);
    expect(storageAdvancing(0, 5)).toBe(false);
    expect(storageAdvancing(-5, 5)).toBe(false);
    expect(storageAdvancing(100, null)).toBe(false);
    expect(storageAdvancing(100, 0)).toBe(false);
    expect(storageAdvancing(null, 5)).toBe(false);
  });
});

describe("serviceActionDisabled", () => {
  it("disables every action while any action for the service is in flight", () => {
    expect(serviceActionDisabled("start", false, "restart")).toBe(true);
    expect(serviceActionDisabled("stop", true, "start")).toBe(true);
    expect(serviceActionDisabled("restart", true, "stop")).toBe(true);
  });
  it("disables Start while already active, Stop while already inactive", () => {
    expect(serviceActionDisabled("start", true, null)).toBe(true);
    expect(serviceActionDisabled("start", false, null)).toBe(false);
    expect(serviceActionDisabled("stop", false, null)).toBe(true);
    expect(serviceActionDisabled("stop", true, null)).toBe(false);
  });
  it("never state-disables Restart", () => {
    expect(serviceActionDisabled("restart", true, null)).toBe(false);
    expect(serviceActionDisabled("restart", false, null)).toBe(false);
  });
});

describe("clearConfirmValid", () => {
  it("requires an exact (trimmed) match of the service id", () => {
    expect(clearConfirmValid("exec", "exec")).toBe(true);
    expect(clearConfirmValid("  exec  ", "exec")).toBe(true);
    expect(clearConfirmValid("Exec", "exec")).toBe(false);
    expect(clearConfirmValid("beacon", "exec")).toBe(false);
    expect(clearConfirmValid("", "exec")).toBe(false);
  });
});
