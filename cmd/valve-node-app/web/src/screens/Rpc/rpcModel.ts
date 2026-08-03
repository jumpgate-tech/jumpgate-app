// Pure helpers for the eRPC screen — no DOM, no api calls, no JSX. Everything
// the screen DERIVES rather than fetches lives here so it can be unit-tested in
// rpcModel.test.ts, exactly as panelModel.ts is for the panel. The container
// and the presentational components read these and render; they never re-derive
// a verdict, a redundancy count or a config transform of their own.
import type {
  ChainlistEndpoint,
  EndpointCapabilities,
  GatewayConfig,
  GatewayTraffic,
  GatewayUpstream,
  GatewayView,
  NetworkView,
  TargetSummary,
  TlsVerification,
  UpstreamKind,
  UpstreamView,
} from "../../api";

// DEVNET_CHAIN_ID is reth's --dev genesis id (catalog.DevnetChainID): the one
// chain this screen renders LAST and de-emphasised, offered only as an opt-in.
export const DEVNET_CHAIN_ID = 1337;

// FINAL_STEP is the id every gateway setup plan ends on. The setup stream has
// no terminal frame, so this is what tells the progress log it is finished.
export const FINAL_STEP = "run";

// CAP_ORDER is the capability set every endpoint row renders, in order, whether
// or not the endpoint has an opinion about each. CAP_TAGS are the short forms.
export const CAP_ORDER = ["http", "ws", "archive", "trace"] as const;
export const CAP_TAGS: Record<string, string> = {
  http: "HTTP",
  ws: "WS",
  archive: "ARCHIVE",
  trace: "TRACE",
};

// Tone is the only colour vocabulary the page has: ok = answering, warn =
// degraded or single-path, bad = down.
export type Tone = "ok" | "warn" | "bad";

// Attention is one line of the strip: what is wrong and the exact thing to run.
// "note" carries no colour — it is information, not a state claim.
export interface Attention {
  tone: Tone | "note";
  text: string;
  cmd?: string;
}

// Verdict is a chain's one derived sentence. It is STRUCTURED rather than
// pre-rendered html (as the legacy screen carried it) so it can be asserted in
// a unit test; the component turns the kind + data into the sentence + <code>.
export type VerdictKind =
  | "no-endpoint"
  | "not-serviceable"
  | "no-websocket"
  | "single"
  | "no-local"
  | "some-unusable"
  | "ok";

export interface Verdict {
  tone: Tone;
  kind: VerdictKind;
  schemes?: string[];
  broken?: number;
  total?: number;
  usable?: number;
  why?: string;
}

const NO_WS_WHY =
  "eRPC infers WebSocket from the endpoint's scheme and has no separate setting, " +
  "so a chain configured entirely with http:// or https:// upstreams refuses every " +
  "eth_subscribe — even where the same host would accept a wss:// connection. That " +
  "is why an endpoint below can be tagged WS and this still be true.";

// isWebSocketUpstream reads the SCHEME, and that is the whole of it: eRPC infers
// the capability from the endpoint it was given.
export function isWebSocketUpstream(u: UpstreamView): boolean {
  return /^wss?:\/\//i.test((u.endpoint ?? "").trim());
}

// configuredSchemes lists the distinct schemes a chain's upstreams are dialed
// with, so the no-WebSocket sentence can name what they ARE.
export function configuredSchemes(ups: UpstreamView[]): string[] {
  const seen = new Set<string>();
  for (const u of ups) {
    const m = /^([a-z][a-z0-9+.-]*):\/\//i.exec((u.endpoint ?? "").trim());
    if (m) seen.add(m[1].toLowerCase());
  }
  return [...seen].sort();
}

// chainVerdict is the one sentence, DERIVED — never written per chain. The
// rules are ordered by what is ALREADY failing before what merely might.
export function chainVerdict(n: NetworkView): Verdict {
  const ups = n.upstreams ?? [];
  if (ups.length === 0) return { tone: "bad", kind: "no-endpoint" };
  if (!n.serviceable) return { tone: "bad", kind: "not-serviceable" };
  if (!ups.some(isWebSocketUpstream)) {
    return { tone: "warn", kind: "no-websocket", schemes: configuredSchemes(ups), why: NO_WS_WHY };
  }
  if (ups.length === 1) return { tone: "warn", kind: "single" };
  if (!ups.some((u) => u.local)) return { tone: "warn", kind: "no-local" };
  const broken = ups.filter((u) => !!u.problem);
  if (broken.length > 0) {
    return {
      tone: "warn",
      kind: "some-unusable",
      broken: broken.length,
      total: ups.length,
      usable: ups.length - broken.length,
    };
  }
  return { tone: "ok", kind: "ok", total: ups.length };
}

// healthWord collapses a tone to the one word the chain head shows: only "ok"
// is genuinely nothing-to-see-here; warn and bad both mean look.
export function healthWord(tone: Tone): "healthy" | "attention" {
  return tone === "ok" ? "healthy" : "attention";
}

// orderedNetworks puts the devnet last without disturbing the real chains'
// order — a stable partition, not a sort.
export function orderedNetworks(nets: NetworkView[]): NetworkView[] {
  const real = nets.filter((n) => n.chainId !== DEVNET_CHAIN_ID);
  const dev = nets.filter((n) => n.chainId === DEVNET_CHAIN_ID);
  return [...real, ...dev];
}

// Redundancy is the count made into a mark. See the legacy redundancyBar
// comment: setSize is the ENTRY count for the chain (0 when valve has measured
// nothing), and past it the label states the count in the open rather than
// reading as an arithmetic bug.
export interface Redundancy {
  total: number;
  filled: number;
  label: string;
  title: string;
}

export function redundancy(count: number, setSize: number): Redundancy {
  const known = setSize > 0;
  const total = known ? setSize : count;
  const filled = Math.min(count, total);
  const over = known && count > setSize;
  const label = !known ? `${count}` : over ? `${count} (set is ${setSize})` : `${count} of ${setSize}`;
  const ups = `${count} upstream${count === 1 ? "" : "s"} configured`;
  const title = known
    ? `${ups}${over ? `, ${count - setSize} beyond the set` : ""}. valve's set for this chain is ${setSize}.`
    : `${ups}. valve has not measured a set for this chain, so there is nothing to count it against.`;
  return { total, filled, label, title };
}

// statusOf resolves one capability, including "http", which is not in the
// capability list at all — it is synthesised from Endpoint.Reachable.
export function statusOf(e: EndpointCapabilities | undefined, key: string): string | undefined {
  if (!e) return undefined;
  if (key === "http") {
    if (e.unprobeable) return "inconclusive";
    return e.reachable ? "supported" : "unsupported";
  }
  return (e.capabilities ?? []).find((c) => c.key === key)?.status;
}

// chainLacksEntirely is what turns a grey absence red: a capability missing
// from EVERY probed endpoint on the chain is a hole in the path, not one
// endpoint's shortfall.
export function chainLacksEntirely(chainEndpoints: (EndpointCapabilities | undefined)[], key: string): boolean {
  const probed = chainEndpoints.filter((e): e is EndpointCapabilities => !!e && !e.unprobeable);
  return probed.length > 0 && probed.every((e) => statusOf(e, key) === "unsupported");
}

// capTagClass maps a capability status (+ whether the chain lacks it entirely)
// to the tag's css class, distinguishing "unsupported here" (grey) from
// "missing chain-wide" (red).
export function capTagClass(status: string, missing: boolean): string {
  if (status === "unsupported") return missing ? "cap missing" : "cap off";
  if (status === "inconclusive") return "cap unknown";
  if (status === "inconsistent") return "cap mixed";
  return "cap";
}

// capTagTitle is the evidence behind a tag, not just the verdict.
export function capTagTitle(e: EndpointCapabilities, key: string): string {
  const cap = (e.capabilities ?? []).find((c) => c.key === key);
  const tag = CAP_TAGS[key] ?? key.toUpperCase();
  if (cap?.detail) return `${cap.label}: ${cap.detail}`;
  if (key === "http" && e.reachDetail) return `Answers JSON-RPC over HTTP: ${e.reachDetail}`;
  return `${tag}: no verdict`;
}

// --- traffic share -------------------------------------------------------

export type ShareCell =
  | { kind: "reading" }
  | { kind: "unreadable" }
  | { kind: "off" }
  | { kind: "none" }
  | { kind: "bar"; pct: number; tickPct: number; fill: "" | "ok" | "warn"; title: string; unconfigured: boolean };

// shareCellModel is the join of one endpoint's measured share against the
// routing configuration's intent. undefined traffic = still reading; null =
// the counters could not be read.
export function shareCellModel(
  traffic: GatewayTraffic | null | undefined,
  loading: boolean,
  chainId: number,
  upstreamId: string,
  local: boolean,
): ShareCell {
  if (loading) return { kind: "reading" };
  if (!traffic) return { kind: "unreadable" };
  if (!traffic.enabled) return { kind: "off" };
  const net = (traffic.networks ?? []).find((x) => x.chainId === chainId);
  const s = (net?.upstreams ?? []).find((u) => u.upstream === upstreamId);
  if (!s || !net || net.attributed === 0) return { kind: "none" };
  const pct = Math.round(s.actual * 100);
  const tickPct = Math.round(s.intended * 100);
  const fill: "" | "ok" | "warn" = s.diverged ? (local ? "warn" : "") : "ok";
  const title =
    `${s.succeeded.toLocaleString()} of ${net.attributed.toLocaleString()} answered requests` +
    ` · routing intends ${tickPct}%` +
    (s.unconfigured ? " · this endpoint is no longer in the saved configuration" : "");
  return { kind: "bar", pct, tickPct, fill, title, unconfigured: !!s.unconfigured };
}

// --- tls / trust ---------------------------------------------------------

// internalCaPath returns the root a caller must trust ONLY when the gateway is
// fronted by Caddy's own CA (the self-signed case), so the inline hint and the
// operator note ask the same question of the same fact.
export function internalCaPath(gw: GatewayView): string | null {
  const t = gw.tls;
  if (t?.enabled && t.rootCaPath && t.effectiveCertSource === "internal") return t.rootCaPath;
  return null;
}

// targetModeOf is how a gateway's placement machine was registered — what
// decides whether one-click "Trust on this machine" makes sense (local only).
export function targetModeOf(targets: TargetSummary[] | null | undefined, targetId: string): string {
  return (targets ?? []).find((t) => t.id === targetId)?.mode ?? "";
}

export function osLabel(os: string): string {
  switch (os) {
    case "darwin":
      return "macOS";
    case "windows":
      return "Windows";
    case "linux":
      return "Linux";
    default:
      return os || "this device";
  }
}

// manualTrustCommand is the copy-paste fallback that installs the gateway's
// root into a trust store, for the OS of the device that will open the URL.
export function manualTrustCommand(os: string, certPath: string, gid: string): string {
  switch (os) {
    case "darwin":
      return `sudo security add-trusted-cert -d -r trustRoot -k /Library/Keychains/System.keychain "${certPath}"`;
    case "windows":
      return `certutil -addstore -f ROOT "${certPath}"`;
    case "linux":
    default:
      return `sudo cp "${certPath}" /usr/local/share/ca-certificates/valve-node-app-${gid}.crt && sudo update-ca-certificates`;
  }
}

// --- attention strip -----------------------------------------------------

// tlsAttention is what the HTTPS front is ACTUALLY doing, as opposed to what
// was configured. verification is the live check this screen last ran, falling
// back to the server's own last one.
export function tlsAttention(gw: GatewayView, verification: TlsVerification | null | undefined): Attention[] {
  const t = gw.tls;
  if (!t?.enabled) return [];
  const out: Attention[] = [];
  if (t.fallback) out.push({ tone: "warn", text: t.fallback });
  if (t.error) {
    out.push({ tone: "warn", text: `HTTPS front: ${t.error}` });
  } else if (t.status?.State !== "running") {
    out.push({
      tone: "warn",
      text:
        `The HTTPS front is ${t.status?.State ?? "unknown"}, so nothing answers on ` +
        `${t.url ?? "its https URL"} even if the gateway itself is up.`,
      cmd: t.containerName ? `docker start ${t.containerName}` : undefined,
    });
  }
  const v = verification ?? t.verification ?? null;
  if (v && (!v.ok || !v.subscriptionsOk)) {
    out.push({
      tone: v.ok ? "warn" : "bad",
      text: `${v.summary} Checked ${new Date(v.at).toLocaleString()} — open Settings for the full check.`,
    });
  }
  if (v?.expiryWarning) out.push({ tone: "warn", text: v.expiryWarning });
  return out;
}

// attentionLines is the ONE surface a gateway uses to say what is wrong: the
// read error, the block, the warnings, the TLS state, then the last action
// error — the five old banners folded into one strip.
export function attentionLines(
  gw: GatewayView,
  verification: TlsVerification | null | undefined,
  actionErr: string | null,
): Attention[] {
  const lines: Attention[] = [];
  if (gw.error) {
    lines.push({ tone: "bad", text: `This gateway could not be read: ${gw.error}${gw.hint ? ` — ${gw.hint}` : ""}` });
  }
  if (gw.blocked) lines.push({ tone: "warn", text: gw.blocked });
  for (const wmsg of gw.warnings ?? []) lines.push({ tone: "warn", text: wmsg });
  lines.push(...tlsAttention(gw, verification));
  if (actionErr) lines.push({ tone: "bad", text: actionErr });
  return lines;
}

// manageStatus is the collapsed "Manage gateway" line's one-glance summary.
export function manageStatus(gw: GatewayView, orphanCount: number): string {
  const bits: string[] = [];
  if (gw.status.State !== "running") bits.push("gateway not running");
  if (orphanCount > 0) bits.push(`${orphanCount} leftover container${orphanCount === 1 ? "" : "s"}`);
  if (bits.length === 0) return "container, settings, certificate";
  return bits.join(" · ");
}

// --- discovery -----------------------------------------------------------

// defaultSelection pre-ticks the three fastest live endpoints, except one slot
// goes to the fastest live wss:// candidate when the feed offers one — a chain
// with only http upstreams cannot serve eth_subscribe.
export function defaultSelection(live: ChainlistEndpoint[]): Set<string> {
  const byLatency = [...live].sort((a, b) => (a.latencyMs ?? 1e9) - (b.latencyMs ?? 1e9));
  const picked = byLatency.slice(0, 3);
  const ws = byLatency.find((e) => e.url.startsWith("wss://") || e.url.startsWith("ws://"));
  if (ws && !picked.some((e) => e.url === ws.url)) {
    if (picked.length === 3) picked.pop();
    picked.push(ws);
  }
  return new Set(picked.map((e) => e.url));
}

// --- add-a-gateway guard -------------------------------------------------

// canAddGatewayOn reports whether a machine has no gateway yet. A machine hosts
// one managed eRPC, so the picker must not offer a target already placed.
export function canAddGatewayOn(targetId: string, gateways: GatewayView[]): boolean {
  return !gateways.some((g) => g.placement?.targetId === targetId);
}

// --- config transforms ---------------------------------------------------

// storedConfig is a deep copy of what the SERVER holds — the thing an edit must
// be applied to. Every transform below returns a fresh config; the input view
// is never mutated.
export function storedConfig(gw: GatewayView): GatewayConfig {
  return {
    ...gw.config,
    Networks: (gw.config.Networks ?? []).map((n) => ({
      ChainID: n.ChainID,
      Upstreams: n.Upstreams.map((u) => ({ ...u })),
    })),
  };
}

// withoutChain drops a chain entirely.
export function withoutChain(cfg: GatewayConfig, chainId: number): GatewayConfig {
  return { ...cfg, Networks: (cfg.Networks ?? []).filter((x) => x.ChainID !== chainId) };
}

// withoutEndpoint removes one upstream by its VIEW id (u.id), matched against
// the stored config the same way the legacy row key was: ID, or a positional
// fallback for an unnamed upstream.
export function withoutEndpoint(cfg: GatewayConfig, chainId: number, upstreamId: string): GatewayConfig {
  const nets = (cfg.Networks ?? []).map((n) => {
    if (n.ChainID !== chainId) return n;
    const idx = n.Upstreams.findIndex((u, i) => (u.ID || `${chainId}-${i}`) === upstreamId);
    if (idx < 0) return n;
    return { ...n, Upstreams: n.Upstreams.filter((_, i) => i !== idx) };
  });
  return { ...cfg, Networks: nets };
}

// appendUpstream pushes one upstream onto a chain's network, creating the
// network if it does not exist yet. Returns a fresh config.
function appendUpstream(cfg: GatewayConfig, chainId: number, upstream: GatewayUpstream): GatewayConfig {
  const nets = (cfg.Networks ?? []).map((n) => ({ ChainID: n.ChainID, Upstreams: [...n.Upstreams] }));
  const existing = nets.find((n) => n.ChainID === chainId);
  if (existing) existing.Upstreams.push(upstream);
  else nets.push({ ChainID: chainId, Upstreams: [upstream] });
  return { ...cfg, Networks: nets };
}

// withManagedUpstream adds a managed node/devnet upstream referencing a machine
// in the fleet.
export function withManagedUpstream(
  cfg: GatewayConfig,
  chainId: number,
  kind: UpstreamKind,
  targetId: string,
): GatewayConfig {
  const upstream: GatewayUpstream = {
    ID: `${kind === "managed-devnet" ? "devnet" : "node"}-${targetId}`,
    Kind: kind,
    TargetID: targetId,
    Endpoint: "",
    Local: true,
    RecentOnly: false,
  };
  return appendUpstream(cfg, chainId, upstream);
}

// withDevnetUpstream mirrors catalog.GatewayForDevnet — the devnet as the
// preferred (and only) upstream for its own chain.
export function withDevnetUpstream(cfg: GatewayConfig, chainId: number, targetId: string): GatewayConfig {
  const upstream: GatewayUpstream = {
    ID: "devnet",
    Kind: "managed-devnet",
    TargetID: targetId,
    Endpoint: "",
    Local: true,
    RecentOnly: false,
  };
  return appendUpstream(cfg, chainId, upstream);
}

// withExternalUpstreams adds one or more external URLs in one write, continuing
// the public-<chain>-<n> id scheme from the highest suffix already present and
// skipping URLs already configured.
export function withExternalUpstreams(
  cfg: GatewayConfig,
  chainId: number,
  urls: string[],
  recentOnly = false,
): GatewayConfig {
  const nets = (cfg.Networks ?? []).map((n) => ({ ChainID: n.ChainID, Upstreams: [...n.Upstreams] }));
  let net = nets.find((n) => n.ChainID === chainId);
  if (!net) {
    net = { ChainID: chainId, Upstreams: [] };
    nets.push(net);
  }
  let next = 1;
  for (const u of net.Upstreams) {
    const m = /^public-\d+-(\d+)$/.exec(u.ID ?? "");
    if (m) next = Math.max(next, Number(m[1]) + 1);
  }
  for (const url of urls) {
    if (net.Upstreams.some((u) => u.Endpoint === url)) continue;
    net.Upstreams.push({
      ID: `public-${chainId}-${next++}`,
      Kind: "external",
      Endpoint: url,
      Local: false,
      RecentOnly: recentOnly,
    });
  }
  return { ...cfg, Networks: nets };
}

// pruneEmptyNetworks drops networks with no upstream, which the server's config
// route legitimately rejects (eRPC refuses a network with none).
export function pruneEmptyNetworks(cfg: GatewayConfig): GatewayConfig {
  return { ...cfg, Networks: (cfg.Networks ?? []).filter((n) => n.Upstreams.length > 0) };
}

// placeholderNetwork builds the client-side view of a chain just added with no
// endpoint yet — held until the server echoes it back once an endpoint lands.
export function placeholderNetwork(chainId: number, name: string, projectId: string): NetworkView {
  return {
    chainId,
    name,
    path: `/${projectId}/evm/${chainId}`,
    upstreams: [],
    knownSetSize: 0,
    serviceable: false,
    warnings: ["This network has no endpoint yet, so it is not saved on the gateway until you add one."],
  };
}

// mergePendingNetworks folds client-side placeholder chains into the server's
// list, dropping any the server has since echoed back.
export function mergePendingNetworks(
  server: NetworkView[],
  pending: number[],
  presets: { chainId: number; name: string }[],
  projectId: string,
): NetworkView[] {
  const present = new Set(server.map((n) => n.chainId));
  const extras = pending
    .filter((cid) => !present.has(cid))
    .map((cid) => placeholderNetwork(cid, presets.find((p) => p.chainId === cid)?.name ?? `Chain ${cid}`, projectId));
  return [...server, ...extras];
}

// shortTime renders an ISO timestamp as local time, falling back to the raw
// string rather than "Invalid Date".
export function shortTime(iso: string): string {
  const d = new Date(iso);
  return Number.isNaN(d.getTime()) ? iso : d.toLocaleString();
}
