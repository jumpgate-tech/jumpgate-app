// Pure model for the Modules section (Machine screen), the wizard's Modules
// group and the gateway screen's module cards. Every placement renders
// through moduleCards, and whether a module applies is READ from the
// catalog's Applies rows — never re-derived here (spec D11): ok=false with no
// reason hides the card, a reason disables it and is shown verbatim.
import type { Catalog, CatalogModule, CatalogModuleApplies, GatewayNetwork, ModulePlacement, ModuleStatus, Snapshot } from "../../api";

export type ModuleCardState = "hidden" | "disabled" | "off" | "on";

export interface ModuleCard {
  id: string;
  name: string;
  summary: string;
  state: ModuleCardState;
  reason?: string;
  beta: boolean;
  status?: string;
  siteUrl?: string;
  docsStep: string;
  resources: string;
  restartLine?: string;
  // identity is a running box module's enode:// or enr:- record; note is
  // the probe's advice about it.
  identity?: string;
  note?: string;
}

export interface ModuleCtx {
  chainId: number;
  // client is the node's exec client; it selects the node-module Applies row.
  client?: string;
  enabled: string[];
  statuses: ModuleStatus[];
  // restartedAt (ms) is when this screen last restarted the exec client for
  // a module change, for D8's "Starting" grace.
  restartedAt?: number;
  gateStale: boolean;
  snapshot?: Snapshot | null;
  // box is the machine's name, for the D4 restart line.
  box?: string;
  // now (ms) is injectable for tests.
  now?: number;
}

export const GATE_STALE = "Status needs a newer gate on this box";
export const POST_WARNING = "Anyone can post through your node. Posts cost the sender proof of work, not you.";
export const PUBLIC_OPTIONS: { value: "" | "read" | "post"; label: string }[] = [
  { value: "", label: "Local only" },
  { value: "read", label: "Public reads" },
  { value: "post", label: "Public reads and posting" },
];

const STARTING_GRACE_MS = 60_000;

// appliesRow finds the catalog's verdict for this placement and context. A
// box module bound to no chain carries a chainId 0 row, used when the box's
// own chain has none.
function appliesRow(m: CatalogModule, place: ModulePlacement, ctx: ModuleCtx): CatalogModuleApplies | undefined {
  if (place === "node") return m.applies.find((a) => a.chainId === ctx.chainId && a.client === (ctx.client ?? ""));
  return (
    m.applies.find((a) => a.chainId === ctx.chainId && a.client === "") ??
    m.applies.find((a) => a.chainId === 0 && a.client === "")
  );
}

export function moduleCards(cat: Catalog, place: ModulePlacement, ctx: ModuleCtx): ModuleCard[] {
  return (cat.modules ?? [])
    .filter((m) => m.placement === place)
    .map((m) => {
      const row = appliesRow(m, place, ctx);
      const on = ctx.enabled.includes(m.id);
      // A module that is on always shows as on, so it can be turned off even
      // when the catalog no longer lets it be turned on here.
      let state: ModuleCardState;
      if (on) state = "on";
      else if (row?.ok) state = "off";
      else if (row?.reason) state = "disabled";
      else state = "hidden";
      const card: ModuleCard = {
        id: m.id,
        name: m.name,
        summary: m.summary,
        state,
        beta: !!row?.ok && row.beta,
        docsStep: m.docsStep,
        resources: resourcesLine(m.resources),
      };
      if (!row?.ok && row?.reason) card.reason = row.reason;
      if (m.siteUrl) card.siteUrl = m.siteUrl;
      if (place === "node" && ctx.client) {
        card.restartLine = `This restarts ${ctx.client} on ${ctx.box ?? "this box"}. The beacon keeps running.`;
      }
      if (state === "on" || state === "off") {
        const s = ctx.statuses.find((x) => x.id === m.id);
        const status = statusLine(m, s, ctx);
        if (status) card.status = status;
        if (m.placement === "box" && s?.active && !ctx.gateStale) {
          if (s.identity) card.identity = s.identity;
          if (s.note) card.note = s.note;
        }
      }
      return card;
    });
}

const STATE_RANK: Record<ModuleCardState, number> = { hidden: 0, disabled: 1, off: 2, on: 2 };

// gatewayModuleCards is moduleCards for a gateway, which serves several
// chains: a gateway module acts on the chains it applies to (plan D-G), so a
// card takes the best verdict across the gateway's chains — usable on any
// chain beats a reason, and a reason beats hidden.
export function gatewayModuleCards(cat: Catalog, chainIds: number[], enabled: string[]): ModuleCard[] {
  const best = new Map<string, ModuleCard>();
  const order: string[] = [];
  for (const chainId of chainIds.length ? chainIds : [-1]) {
    for (const c of moduleCards(cat, "gateway", { chainId, enabled, statuses: [], gateStale: false })) {
      const prev = best.get(c.id);
      if (!prev) order.push(c.id);
      if (!prev || STATE_RANK[c.state] > STATE_RANK[prev.state]) best.set(c.id, c);
    }
  }
  return order.map((id) => best.get(id)!);
}

// statusLine is the one status line (spec D8 order for node modules; box
// modules report their unit). Off and not observed is the empty string.
export function statusLine(m: CatalogModule, s: ModuleStatus | undefined, ctx: ModuleCtx): string {
  if (ctx.gateStale) return GATE_STALE;
  const on = ctx.enabled.includes(m.id);
  if (m.placement === "box") {
    if (!on && !s) return "";
    if (s?.error) return s.error;
    if (s?.active) return "Running";
    return "Stopped";
  }
  if (m.placement !== "node") return "";
  const snap = ctx.snapshot;
  if (!on) return s?.served ? "Served but not set by jumpgate" : "";
  if (snap && !snap.execActive) return "Node stopped";
  if (!s?.served) {
    const now = ctx.now ?? Date.now();
    if (ctx.restartedAt !== undefined && now - ctx.restartedAt < STARTING_GRACE_MS) return "Starting";
    return s?.error || "Not served yet";
  }
  if (snap?.execSyncing) return "Waiting for sync";
  if (snap && snap.execPeers === 0) return "No peers";
  if (!s.accepting) return "Starting";
  return `Accepting · ${s.count.toLocaleString("en-US")} messages · ${decimalBytes(s.size)}`;
}

// decimalBytes is "3.1 MB"-style (SI units, one decimal under 10).
function decimalBytes(n: number): string {
  const units = ["B", "kB", "MB", "GB", "TB"];
  let v = n;
  let u = 0;
  while (v >= 1000 && u < units.length - 1) {
    v /= 1000;
    u++;
  }
  const digits = u === 0 || v >= 10 ? 0 : 1;
  return `${v.toFixed(digits).replace(/\.0$/, "")} ${units[u]}`;
}

export function resourcesLine(r: { ramBytes: number; diskBytes: number }): string {
  const parts: string[] = [];
  if (r.ramBytes > 0) parts.push(`${decimalBytes(r.ramBytes)} RAM`);
  if (r.diskBytes > 0) parts.push(`${decimalBytes(r.diskBytes)} disk`);
  return parts.length ? `about ${parts.join(" · ")}` : "";
}

// lowerName turns "Message board" into "message board" for mid-sentence use,
// leaving acronyms ("EL bootnode") alone.
function lowerName(name: string): string {
  return /^[A-Z][a-z]/.test(name) ? name[0].toLowerCase() + name.slice(1) : name;
}

// droppedOnClientChange lists the enabled node modules the catalog says the
// newly chosen client cannot run, with the D10 notice for each. The wizard
// removes them from the submitted wire.
export function droppedOnClientChange(
  cat: Catalog,
  chainId: number,
  client: string,
  enabled: string[],
): { id: string; notice: string }[] {
  const out: { id: string; notice: string }[] = [];
  for (const m of cat.modules ?? []) {
    if (m.placement !== "node" || !enabled.includes(m.id)) continue;
    const row = m.applies.find((a) => a.chainId === chainId && a.client === client);
    if (row?.ok) continue;
    out.push({ id: m.id, notice: `The ${lowerName(m.name)} will be turned off: ${client} does not support it.` });
  }
  return out;
}

// BOX_FIELDS is which settings a box module asks for when it is turned on.
// Modules not listed ask for none.
export type BoxField = "port" | "extIp";
const BOX_FIELDS: Record<string, BoxField[]> = {
  "el-bootnode": ["port", "extIp"],
  "cl-bootnode": ["port", "extIp"],
  checkpointz: ["port"],
};

export const BOX_FIELD_DEFAULTS: Record<string, { port?: number }> = {
  "el-bootnode": { port: 30301 },
  "cl-bootnode": { port: 9200 },
  checkpointz: { port: 5555 },
};

export function boxFields(id: string): BoxField[] {
  return BOX_FIELDS[id] ?? [];
}

// publicGatewayRefs lists the Public-tier gateways that route to this node —
// the condition for the Public radio. The tier is the server's
// (catalog.GatewayTier on /api/gateways), never re-derived here.
export function publicGatewayRefs(
  gateways: { id: string; tier?: string; config: { Networks: GatewayNetwork[] | null } }[],
  targetId: string,
): string[] {
  return gateways
    .filter(
      (g) =>
        g.tier === "public" &&
        (g.config.Networks ?? []).some((n) =>
          n.Upstreams.some((u) => u.Kind === "managed-node" && u.TargetID === targetId),
        ),
    )
    .map((g) => g.id);
}

// publicOptions is the Public radio for a module: "Local only" plus the
// exposure choices the catalog says it offers. Empty means no radio.
export function publicOptions(m: CatalogModule): typeof PUBLIC_OPTIONS {
  const offered = m.public ?? [];
  if (offered.length === 0) return [];
  return PUBLIC_OPTIONS.filter((o) => o.value === "" || offered.includes(o.value));
}
