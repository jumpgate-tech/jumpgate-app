// Pure helpers for the panel. No DOM, no api calls — unit-tested in panelModel.test.ts.

import type { GatewayConfig, GatewayNetwork, GatewayUpstream, GatewayView, Latency, NetworkAnalytics } from "./api";

// endpointNameFromUrl derives a friendly default name from an endpoint URL:
// the second-level domain label ("publicnode" from rpc.publicnode.com), or the
// bare host for localhost / IPs, or "endpoint" if the URL won't parse.
export function endpointNameFromUrl(endpoint: string): string {
  let host: string;
  try {
    host = new URL(endpoint).hostname;
  } catch {
    return "endpoint";
  }
  if (!host) return "endpoint";
  if (host === "localhost" || /^[0-9.]+$/.test(host) || /^\[.*\]$/.test(host)) return host;
  const parts = host.split(".").filter(Boolean);
  if (parts.length <= 1) return host;
  // second-level label: publicnode.com → "publicnode"; infura.io → "infura".
  return parts[parts.length - 2];
}

export type PowerTone = "on" | "off" | "blocked";
export interface MasterState { tone: PowerTone; label: string; sub: string; actions: string[]; blocked?: string; }

// masterState maps a gateway to the master power button. tone drives the color
// (green/red/grey); actions/blocked come straight from the server so the button
// never offers an impossible transition.
export function masterState(gw: GatewayView | null): MasterState {
  if (!gw) return { tone: "off", label: "Not set up", sub: "Press to set up your endpoint", actions: [] };
  const actions = gw.actions ?? [];
  if (gw.blocked) return { tone: "blocked", label: "Unavailable", sub: gw.blocked, actions, blocked: gw.blocked };
  const count = gw.networks?.length ?? 0;
  if (gw.status.State === "running") {
    return { tone: "on", label: "Running", sub: `${count} network${count === 1 ? "" : "s"} served`, actions };
  }
  return { tone: "off", label: "Stopped", sub: count ? `${count} network${count === 1 ? "" : "s"} configured` : "Press to start", actions };
}

export type HealthClass = "stable" | "occasional" | "frequent" | "off";

// healthClass turns coarse signals into the dot's motion. Stillness = health, so
// a serviceable endpoint with rare slow requests is "stable" (no animation).
// Motion frequency tracks the slow-request rate. Thresholds: <10% stable,
// 10–40% occasional, >40% frequent. Not-serviceable while running is "frequent".
export function healthClass(input: { running: boolean; serviceable: boolean; slowRate?: number }): HealthClass {
  if (!input.running) return "off";
  if (!input.serviceable) return "frequent";
  const r = input.slowRate ?? 0;
  if (r > 0.4) return "frequent";
  if (r >= 0.1) return "occasional";
  return "stable";
}

// SLOW_LATENCY_LE is the bucket boundary the dots call "slow". eRPC's
// histograms carry a FIXED bucket set — 0.05s / 0.5s / 5s / 30s / +Inf (see
// internal/metrics's own Latency doc) — so there is no bucket for an
// arbitrary threshold like 1s; asking for one would just never match. 0.5s
// is the nearest real boundary: under it is ordinary RPC latency, over it is
// a request an operator would actually notice.
const SLOW_LATENCY_LE = "0.5";

// slowRateFromLatency reads the fraction of one cumulative histogram's
// requests that finished slower than SLOW_LATENCY_LE. Buckets are
// cumulative (Prometheus-style): the count at a given `le` is every request
// that finished within `le` seconds, so subtracting it from the histogram's
// total count leaves exactly the slow tail.
//
// Returns undefined — "no opinion", not "healthy" — when there's nothing to
// read: no buckets yet, zero requests counted, or (defensively) a bucket set
// that doesn't include SLOW_LATENCY_LE. Callers fall back to a cruder signal
// rather than let undefined be misread as "fast".
export function slowRateFromLatency(l: Latency | null | undefined): number | undefined {
  if (!l || l.count <= 0 || !l.buckets || l.buckets.length === 0) return undefined;
  const bucket = l.buckets.find((b) => b.le === SLOW_LATENCY_LE);
  if (!bucket) return undefined;
  const slow = l.count - bucket.count;
  return Math.max(0, Math.min(1, slow / l.count));
}

// combineLatency folds a list of per-method (or per-upstream) histograms
// into one combined histogram, summing counts bucket-by-bucket at matching
// `le`s. Valid because every histogram on the wire shares eRPC's one fixed
// bucket set — summing cumulative counts at the same bound is itself a valid
// cumulative count.
function combineLatency(list: readonly Latency[] | null | undefined): Latency | null {
  if (!list || list.length === 0) return null;
  let count = 0;
  const byLe = new Map<string, number>();
  for (const l of list) {
    count += l.count;
    for (const b of l.buckets ?? []) byLe.set(b.le, (byLe.get(b.le) ?? 0) + b.count);
  }
  return { count, mean: null, buckets: [...byLe.entries()].map(([le, c]) => ({ le, count: c })) };
}

// networkSlowRate is the network dot's live signal: the fraction of this
// network's client-observed requests — folded across every RPC method it
// served — that finished slower than SLOW_LATENCY_LE.
//
// Falls back to failed/received (cruder, but still a real symptom) when the
// histograms haven't filled in — e.g. right after a (re)start, or a chain
// with too little traffic to have landed in a bucket yet.
export function networkSlowRate(na: NetworkAnalytics): number | undefined {
  const fromBuckets = slowRateFromLatency(combineLatency(na.methods));
  if (fromBuckets !== undefined) return fromBuckets;
  if (na.received > 0) return Math.max(0, Math.min(1, na.failed / na.received));
  return undefined;
}

// endpointSlowRate is the per-upstream signal for one endpoint's own dot:
// this network's EndpointLatency row for that upstream ID — the same
// client-observed histogram narrowed to the requests eRPC actually routed
// there. No received/failed fallback at this level (NetworkAnalytics only
// counts those per-network) — an upstream with no histogram yet reports
// undefined, same as "no opinion" anywhere else in this module.
export function endpointSlowRate(na: NetworkAnalytics | null | undefined, upstreamId: string): number | undefined {
  const el = na?.endpoints?.find((e) => e.upstream === upstreamId);
  return slowRateFromLatency(el ?? null);
}

export interface CapCell { key: string; label: string; lit: boolean; hot: boolean; }
const CAP_ORDER: { key: string; label: string; hot?: boolean }[] = [
  { key: "http", label: "HTTP" }, { key: "ws", label: "WS" },
  { key: "archive", label: "Archive", hot: true }, { key: "trace", label: "Trace" },
];

// capabilityCells folds probed capability statuses into the fixed-order meter.
// A cell is "lit" when supported; "hot" marks the standout (archive) when lit.
export function capabilityCells(statuses: Record<string, string>): CapCell[] {
  return CAP_ORDER.map(({ key, label, hot }) => {
    const lit = statuses[key] === "supported";
    return { key, label, lit, hot: !!hot && lit };
  });
}

// The four helpers below are the panel's only writers of GatewayConfig. Each
// returns a NEW config — the input is never mutated — so a caller can hold
// onto the config it read while building the one it's about to PUT.

// withNetwork adds a network at chainId, or — if one is already configured for
// that chain — replaces its upstreams wholesale.
export function withNetwork(cfg: GatewayConfig, chainId: number, upstreams: GatewayUpstream[]): GatewayConfig {
  const networks = cfg.Networks ?? [];
  const idx = networks.findIndex((n) => n.ChainID === chainId);
  const network: GatewayNetwork = { ChainID: chainId, Upstreams: upstreams };
  const nextNetworks = idx === -1 ? [...networks, network] : networks.map((n, i) => (i === idx ? network : n));
  return { ...cfg, Networks: nextNetworks };
}

// withoutNetwork drops the network at chainId, if one is configured.
export function withoutNetwork(cfg: GatewayConfig, chainId: number): GatewayConfig {
  const networks = cfg.Networks ?? [];
  return { ...cfg, Networks: networks.filter((n) => n.ChainID !== chainId) };
}

// withUpstream adds or replaces (by ID) one upstream on chainId's network. If
// chainId has no network configured yet, one is created holding just this
// upstream — the panel's "add an endpoint" affordance doesn't require the
// operator to have added the chain first.
export function withUpstream(cfg: GatewayConfig, chainId: number, up: GatewayUpstream): GatewayConfig {
  const networks = cfg.Networks ?? [];
  const idx = networks.findIndex((n) => n.ChainID === chainId);
  if (idx === -1) {
    return { ...cfg, Networks: [...networks, { ChainID: chainId, Upstreams: [up] }] };
  }
  const network = networks[idx];
  const upIdx = network.Upstreams.findIndex((u) => u.ID === up.ID);
  const nextUpstreams =
    upIdx === -1 ? [...network.Upstreams, up] : network.Upstreams.map((u, i) => (i === upIdx ? up : u));
  const nextNetwork: GatewayNetwork = { ...network, Upstreams: nextUpstreams };
  return { ...cfg, Networks: networks.map((n, i) => (i === idx ? nextNetwork : n)) };
}

// withoutUpstream removes one upstream by ID from chainId's network. The
// network itself is left in place, even with zero upstreams left — only
// withoutNetwork removes a network.
export function withoutUpstream(cfg: GatewayConfig, chainId: number, upstreamId: string): GatewayConfig {
  const networks = cfg.Networks ?? [];
  const idx = networks.findIndex((n) => n.ChainID === chainId);
  if (idx === -1) return { ...cfg, Networks: networks };
  const network = networks[idx];
  const nextNetwork: GatewayNetwork = { ...network, Upstreams: network.Upstreams.filter((u) => u.ID !== upstreamId) };
  return { ...cfg, Networks: networks.map((n, i) => (i === idx ? nextNetwork : n)) };
}
