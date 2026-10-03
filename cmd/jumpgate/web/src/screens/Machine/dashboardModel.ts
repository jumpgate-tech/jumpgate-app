// Pure derivations for the Dashboard section — kept apart from
// DashboardSection.tsx so they're testable without rendering. Mirrors
// dashboard.ts's own constants and inline helpers: the top-of-page status
// badge, the exponentially-smoothed exec block rate, the shared sync-ETA
// calculation, the per-card sync status badges, the disk-usage warning
// threshold, and the service label map.
import type { ServiceActionKind, ServiceID, Snapshot } from "../../api";
import { fmtDuration } from "../../ui";

// highDiskUsagePct is the threshold above which the disk/storage card
// switches from its normal styling to a warning one — mirrors dashboard.ts's
// own constant of the same name.
export const highDiskUsagePct = 85;

export const SERVICE_LABEL: Record<ServiceID, string> = { exec: "Execution", beacon: "Beacon" };

export type BadgeKind = "ok" | "bad" | "warn" | "neutral";

export interface StatusBadge {
  text: string;
  kind: BadgeKind;
}

// topStatus summarizes the whole node in one glance, ahead of the card grid:
// both services down reads as not-running (regardless of per-service sync
// state); otherwise either side still syncing wins over "synced" so the
// headline never claims synced while catching up. Mirrors topStatusBadge.
export function topStatus(snap: Snapshot): StatusBadge {
  if (!snap.execActive && !snap.beaconActive) return { text: "Node not running", kind: "bad" };
  if (snap.execSyncing || snap.beaconDistance > 0) return { text: "Syncing", kind: "warn" };
  return { text: "Running · synced", kind: "ok" };
}

// nextBlocksPerSec mirrors updateRate's exponential smoothing (0.7 old / 0.3
// new) so one slow/fast tick doesn't swing the ETA wildly. `prev` is the
// snapshot before this one (null on the very first tick, in which case the
// rate is left unchanged); a non-positive delta (clock skew, a counter
// reset) also leaves the rate unchanged rather than corrupting it.
export function nextBlocksPerSec(
  prev: Snapshot | null,
  curr: Snapshot,
  prevRate: number | null,
): number | null {
  if (!prev) return prevRate;
  const deltaSeconds = (new Date(curr.at).getTime() - new Date(prev.at).getTime()) / 1000;
  const deltaBlocks = curr.execHead - prev.execHead;
  if (deltaSeconds > 0 && deltaBlocks >= 0) {
    const rate = deltaBlocks / deltaSeconds;
    return prevRate === null ? rate : prevRate * 0.7 + rate * 0.3;
  }
  return prevRate;
}

export interface SyncETA {
  lag: number | null;
  eta: string;
}

// syncETA computes the execution head's lag behind the reference head and a
// human ETA at the current smoothed rate. Shared by the Execution-sync card
// and the Storage card, so both surface the same rate-based estimate instead
// of drifting out of sync with each other. Mirrors dashboard.ts's syncETA.
export function syncETA(snap: Snapshot, execBlocksPerSec: number | null): SyncETA {
  const hasRef = snap.refHead > 0;
  const lag = hasRef ? snap.refHead - snap.execHead : null;
  const eta =
    lag !== null && lag > 0 && execBlocksPerSec && execBlocksPerSec > 0
      ? fmtDuration(lag / execBlocksPerSec)
      : lag !== null && lag <= 0
        ? "caught up"
        : "—";
  return { lag, eta };
}

export function execSyncStatus(snap: Snapshot): StatusBadge {
  if (!snap.execActive) return { text: "stopped", kind: "bad" };
  if (snap.execSyncing) return { text: "syncing", kind: "warn" };
  if (snap.execHead === 0) return { text: "no data", kind: "neutral" };
  return { text: "synced", kind: "ok" };
}

export function beaconSyncStatus(snap: Snapshot): StatusBadge {
  if (!snap.beaconActive) return { text: "stopped", kind: "bad" };
  if (snap.beaconSlot === 0) return { text: "no data", kind: "neutral" };
  if (snap.beaconDistance === 0) return { text: "synced", kind: "ok" };
  return { text: "syncing", kind: "warn" };
}

export function diskWarn(snap: Snapshot): boolean {
  return snap.diskUsedPct >= highDiskUsagePct;
}

// servicePct mirrors storageCard's `used > 0 ? min((used/expected)*100, 100)
// : 0` per-service current-vs-expected meter fill.
export function servicePct(usedBytes: number, expectedBytes: number): number {
  return expectedBytes > 0 ? Math.min((usedBytes / expectedBytes) * 100, 100) : 0;
}

// storageAdvancing mirrors storageCard's `advancing`: the estimated-time-
// remaining line only shows while the exec head is actually behind AND has a
// meaningful rate to estimate from — a stalled or already-synced node has no
// "time remaining" to speak of.
export function storageAdvancing(lag: number | null, execBlocksPerSec: number | null): boolean {
  return lag !== null && lag > 0 && execBlocksPerSec !== null && execBlocksPerSec > 0;
}

// serviceActionDisabled mirrors serviceRow's per-button disabled logic: any
// in-flight action for this service disables every button for it; beyond
// that, Start is disabled while already active and Stop is disabled while
// already inactive (Restart has no state-based disable).
export function serviceActionDisabled(
  kind: ServiceActionKind,
  active: boolean,
  busy: ServiceActionKind | null,
): boolean {
  if (busy !== null) return true;
  if (kind === "start") return active;
  if (kind === "stop") return !active;
  return false;
}

// clearConfirmValid mirrors openClearModal's typed-confirm gate: the input
// must equal the service id exactly (after trimming surrounding whitespace).
export function clearConfirmValid(input: string, svc: ServiceID): boolean {
  return input.trim() === svc;
}
