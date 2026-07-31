// The fleet verdict: one sentence, above everything — "is anything wrong?"
// answered without making the operator assemble it from parts.
//
// THE ONE RULE: only render a claim that is derived from MEASURED / KNOWN
// state and can be acted on. A verdict that can be wrong is worse than the
// screens it replaces (docs/superpowers/specs/2026-07-28-ui-direction-notes.md:
// "A confident wrong headline is worse than the nine screens it replaces").
//
// That rule is why this module is deliberately narrow. It draws ONLY on state
// the `Target`/`Catalog` types actually carry:
//
//   - `Target.wire` is the persisted node config. Its ABSENCE is a fact — the
//     operator added a machine and has not finished setup — not an inference
//     about how a listener looks. That is the one attention signal here.
//   - `Catalog.networks` maps a wired target's ChainID to a human name, so the
//     healthy line names what the fleet is for in the operator's terms.
//
// What it deliberately does NOT do, and why (the doc's cautionary tales):
//
//   - No redundancy verdict. Distinguishing "a chain with one node = a gap"
//     from "a chain that is single BY DESIGN" needs `KnownSetSize` (0 ⇒ single
//     by design, e.g. devnet 1337). That signal lives on the gateway's
//     `NetworkView`, NOT on `Catalog` — so from (targets, catalog) alone the
//     data cannot tell a gap from a devnet, and painting a single node amber
//     would repeat the exact amber-on-devnet mistake the doc calls out. So we
//     don't make the claim. If a caller later threads gateway state through,
//     redundancy can be added under the same rule (only flag a gap when
//     KnownSetSize > 1).
//   - No liveness/sync claim. This module never touches the monitor stream, so
//     "healthy" here is a fleet-level statement that every machine is set up
//     and nothing configured is flagged — not that each node is synced with
//     peers. Measured per-machine health is a separate, later signal.

import * as api from "./api";
import { badge, escapeHtml } from "./ui";

export interface FleetVerdict {
  level: "ok" | "attention";
  // One sentence: what needs attention across the fleet, or that nothing does.
  // Self-contained and unit-testable on its own — the render layer adds links,
  // not meaning.
  sentence: string;
  // The target ids the verdict implicates, for click-through. Empty when the
  // verdict is "ok" (nothing to act on).
  machines: string[];
}

// computeFleetVerdict is pure: no I/O, no clock, no DOM — it maps the two
// pieces of known fleet state to a single verdict, so it can be unit-tested by
// feeding it plain objects.
export function computeFleetVerdict(targets: api.Target[], catalog: api.Catalog): FleetVerdict {
  if (targets.length === 0) {
    return { level: "ok", sentence: "No machines yet.", machines: [] };
  }

  // A machine added but not yet wired is the one thing these types let us
  // assert truthfully AND act on: the fix is to run setup. We state it as a
  // configuration fact ("needs setup"), never as a health failure.
  const unwired = targets.filter((t) => !t.wire);
  if (unwired.length > 0) {
    const ids = unwired.map((t) => t.id);
    const sentence =
      ids.length === 1
        ? "1 machine still needs setup."
        : `${ids.length} machines still need setup.`;
    return { level: "attention", sentence, machines: ids };
  }

  // Every machine is wired. Name the distinct chains the fleet is configured
  // for — each wired target carries a ChainID, and the catalog turns it into a
  // network name (falling back to `chain <id>` exactly as the machine cards do,
  // which is what keeps a devnet like 1337 named plainly rather than guessed).
  const networks = catalog.networks ?? [];
  const chainName = (chainId: number): string => {
    const net = networks.find((n) => n.ChainID === chainId);
    return net ? net.Name : `chain ${chainId}`;
  };
  const names = distinct(targets.map((t) => chainName(t.wire!.ChainID)));
  const plural = targets.length === 1 ? "machine" : "machines";
  return {
    level: "ok",
    sentence: `All ${targets.length} ${plural} healthy — ${humanList(names)}.`,
    machines: [],
  };
}

// renderVerdictLine paints the verdict as one prominent line: a status pill
// (green for ok, amber for attention) followed by the sentence, and — when the
// verdict implicates machines — a link to each so the operator can act without
// hunting. Minimal DOM, existing classes only; no framework.
export function renderVerdictLine(root: HTMLElement, v: FleetVerdict): void {
  // Implicated machines link to their setup route, which is exactly what an
  // un-set-up machine needs. (The machine-page collapse keeps #/setup alive as
  // a redirect, so the link stays valid once the routes merge.)
  const links = v.machines.length
    ? ` <span class="verdict-machines">${v.machines
        .map((id) => `<a href="#/setup/${encodeURIComponent(id)}">${escapeHtml(id)}</a>`)
        .join(" ")}</span>`
    : "";

  root.innerHTML = `
    <div class="verdict-line verdict-${v.level}">
      ${badge(v.level === "ok" ? "OK" : "Attention", v.level === "ok" ? "ok" : "warn")}
      <strong class="verdict-sentence">${escapeHtml(v.sentence)}</strong>${links}
    </div>
  `;
}

// distinct preserves first-seen order while dropping repeats — two machines on
// the same chain should name that chain once.
function distinct(items: string[]): string[] {
  return [...new Set(items)];
}

// humanList joins names the way a sentence would: "a", "a and b",
// "a, b and c".
function humanList(items: string[]): string {
  if (items.length <= 1) return items[0] ?? "";
  if (items.length === 2) return `${items[0]} and ${items[1]}`;
  return `${items.slice(0, -1).join(", ")} and ${items[items.length - 1]}`;
}
