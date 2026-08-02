// Pure derivations for the Machines (Targets) screen — the parts of
// targets.ts's markup logic worth unit-testing on their own: how the target
// list orders (local first), what a target card's status line should say,
// and how a blank SSH "target name" field turns into an id.
import type * as api from "../../api";

// orderTargets puts local target(s) first, then SSH servers. JS's sort is
// stable, so this only ever moves local targets to the front — it never
// reorders within either group.
export function orderTargets(targets: api.Target[]): api.Target[] {
  return [...targets].sort((a, b) => (a.mode === "local" ? -1 : 0) - (b.mode === "local" ? -1 : 0));
}

// TargetStatus is what a target card's status line shows: a target that
// cannot run a node (no wire config, and either an SSH-can-always-run-Linux
// target that's simply unwired doesn't hit this case — only a LOCAL target on
// a non-viable host does), one that's wired but hasn't run setup yet, or a
// fully wired one naming its network and clients.
export type TargetStatus =
  | { kind: "cant-run"; hostOS: string }
  | { kind: "not-set-up" }
  | { kind: "wired"; networkName: string; execId: string; beaconId: string; archive: boolean };

// targetStatus mirrors targetCard's statusLine derivation in the legacy
// targets.ts: canRunNode/hostOS shape the line for a target that hasn't
// completed setup (saying WHY it can't, when it can't), but a wired target's
// line ignores them — it already ran setup, so what matters is what it's
// running, not whether the host could still start one from scratch.
export function targetStatus(
  t: api.Target,
  catalog: api.Catalog,
  canRunNode: boolean,
  hostOS: string,
): TargetStatus {
  const wire = t.wire;
  if (!wire && !canRunNode) return { kind: "cant-run", hostOS };
  if (!wire) return { kind: "not-set-up" };
  const net = catalog.networks.find((n) => n.ChainID === wire.ChainID);
  return {
    kind: "wired",
    networkName: net ? net.Name : `chain ${wire.ChainID}`,
    execId: wire.ExecID,
    beaconId: wire.BeaconID,
    archive: wire.Archive,
  };
}

// slugify turns a free-typed SSH host into a safe default target id when the
// operator leaves the "Target name" field blank — lowercased, non-alphanumeric
// runs collapsed to a single dash, leading/trailing dashes trimmed, and
// falling back to "target" for input that slugifies to nothing (e.g. "---").
export function slugify(s: string): string {
  return (
    s
      .toLowerCase()
      .replace(/[^a-z0-9]+/g, "-")
      .replace(/^-+|-+$/g, "") || "target"
  );
}
