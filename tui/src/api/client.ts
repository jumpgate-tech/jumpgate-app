// One typed function per server route the dashboard calls. Every path here
// has its pattern in ROUTES (src/api/routes.ts), which CI checks against Go.
import { open, request, type Conn } from "./http.ts";
import { parseSSE, stream, type SSEMsg, type StreamOpts } from "./sse.ts";
import type * as T from "./types.ts";

const e = encodeURIComponent;
const tgt = (id: string, rest = "") => `/api/targets/${e(id)}${rest}`;
const flt = (id: string, rest = "") => `/api/fleet/${e(id)}${rest}`;
const job = (target: string, id: string, rest = "") => `/api/jobs/${e(target)}/${e(id)}${rest}`;

const B64 = new TextEncoder().encode("ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789+/");

/**
 * secretBody builds `{"<field>":"<base64 of value>"}` as bytes, never as a
 * string, so the caller can zero it. Go decodes a []byte field from base64.
 */
export function secretBody(field: string, value: Uint8Array, extra: Record<string, string> = {}): Uint8Array {
  const lead = Object.entries(extra).map(([k, v]) => `${JSON.stringify(k)}:${JSON.stringify(v)},`).join("");
  const pre = new TextEncoder().encode(`{${lead}"${field}":"`);
  const post = new TextEncoder().encode(`"}`);
  const n = Math.ceil(value.length / 3) * 4;
  const out = new Uint8Array(pre.length + n + post.length);
  out.set(pre, 0);
  let o = pre.length;
  for (let i = 0; i < value.length; i += 3) {
    const a = value[i]!;
    const b = i + 1 < value.length ? value[i + 1]! : 0;
    const c = i + 2 < value.length ? value[i + 2]! : 0;
    out[o++] = B64[a >> 2]!;
    out[o++] = B64[((a & 3) << 4) | (b >> 4)]!;
    out[o++] = i + 1 < value.length ? B64[((b & 15) << 2) | (c >> 6)]! : 61;
    out[o++] = i + 2 < value.length ? B64[c & 63]! : 61;
  }
  out.set(post, o);
  return out;
}

export const health = (c: Conn) => request<{ ok: boolean }>(c, "GET", "/api/health");
export const version = (c: Conn) => request<{ version: string }>(c, "GET", "/api/version");

export const vault = (c: Conn) => request<T.VaultView>(c, "GET", "/api/vault");

/** unlock sends pass to the server; pass and the encoded body are zeroed before it returns. */
export async function unlock(c: Conn, pass: Uint8Array): Promise<T.VaultView> {
  const body = secretBody("passphrase", pass);
  try {
    return await request<T.VaultView>(c, "POST", "/api/vault/unlock", body);
  } finally {
    body.fill(0);
    pass.fill(0);
  }
}
export const lock = (c: Conn) => request<T.VaultView>(c, "POST", "/api/vault/lock");

export const fleet = (c: Conn) => request<T.Fleet>(c, "GET", "/api/fleet");
export const watchFleet = (c: Conn, s: AbortSignal, o?: StreamOpts) => stream<T.Fleet>(c, "/api/fleet/stream", s, o);
export const fleetOne = (c: Conn, id: string) => request<T.FleetRow>(c, "GET", flt(id));
export const targets = (c: Conn) => request<T.TargetView[]>(c, "GET", "/api/targets");
export const watchStatus = (c: Conn, id: string, s: AbortSignal, o?: StreamOpts) =>
  stream<T.NodeStatus>(c, flt(id, "/status/stream"), s, o);
export const disk = (c: Conn, id: string) => request<T.DiskView>(c, "GET", flt(id, "/disk"));
export const measureDisk = (c: Conn, id: string) => request<T.DiskView>(c, "POST", flt(id, "/disk/measure"));
export const endpoints = (c: Conn, id: string) => request<T.Endpoints>(c, "GET", tgt(id, "/endpoints"));
export const firewall = (c: Conn, id: string) => request<T.CheckItem[]>(c, "GET", tgt(id, "/firewall"));
export const logs = (c: Conn, id: string, n = 200) => request<T.LogHit[]>(c, "GET", tgt(id, `/logs?n=${n}`));
/** watchLogs: the first "reset" event carries LogHit[]; later events are one LogHit. */
export const watchLogs = (c: Conn, id: string, s: AbortSignal, backlog = 200, o?: StreamOpts) =>
  stream<T.LogHit | T.LogHit[]>(c, tgt(id, `/logs/stream?backlog=${backlog}`), s, o);
export type ServiceAction = "start" | "stop" | "restart";
export const serviceAction = (c: Conn, id: string, svc: string, action: ServiceAction) =>
  request<T.ServiceResult>(c, "POST", tgt(id, `/services/${e(svc)}/${e(action)}`));

export const jobs = (c: Conn) => request<T.JobsInbox>(c, "GET", "/api/jobs");
export const watchJobs = (c: Conn, s: AbortSignal, o?: StreamOpts) => stream<T.JobsInbox>(c, "/api/jobs/stream", s, o);
export const runningJobs = (c: Conn) => request<T.RunningJobs>(c, "GET", "/api/jobs/running");
export const jobDetail = (c: Conn, target: string, id: string) => request<T.JobDetail>(c, "GET", job(target, id));
export const watchJobOutput = (c: Conn, target: string, id: string, s: AbortSignal, o?: StreamOpts) =>
  stream<T.JobOutputLine[]>(c, job(target, id, "/output/stream"), s, o);
export const decide = (c: Conn, target: string, id: string, d: T.JobDecision) =>
  request<T.JobView>(c, "POST", job(target, id, "/decide"), d);
export const adopt = (c: Conn, target: string, id: string) => request<T.JobView>(c, "POST", job(target, id, "/adopt"));

export const prefs = (c: Conn) => request<T.UIPrefs>(c, "GET", "/api/ui-prefs");
export const savePrefs = (c: Conn, p: T.UIPrefs) => request<T.UIPrefs>(c, "PUT", "/api/ui-prefs", p);

export type { SSEMsg };

export const settings = (c: Conn) => request<T.Settings>(c, "GET", "/api/settings");
export type SettingsUpdateT = T.SettingsUpdate;
export const saveSettings = (c: Conn, u: T.SettingsUpdate) => request<T.Settings>(c, "PUT", "/api/settings", u);

/** putVaultSecret stores value as name; value and the body are zeroed before it returns. */
export async function putVaultSecret(c: Conn, name: "relay-token" | "admin-token", value: Uint8Array): Promise<void> {
  const body = secretBody("value", value);
  try {
    await request(c, "PUT", `/api/vault/secrets/${e(name)}`, body);
  } finally {
    body.fill(0);
    value.fill(0);
  }
}

export const controller = (c: Conn) => request<T.ControllerView>(c, "GET", "/api/controller");
export const checkGate = (c: Conn, id: string) => request<T.AgentCheck>(c, "POST", flt(id, "/check"));
export const restart = (c: Conn) => request(c, "POST", "/api/restart");

/** systemInitKey has the jumpgate service create its controller key in a new vault; pass is zeroed. */
export async function systemInitKey(c: Conn, pass: Uint8Array): Promise<void> {
  const body = secretBody("passphrase", pass, { store: "vault" });
  try {
    await request(c, "POST", "/api/system/init-key", body);
  } finally {
    body.fill(0);
    pass.fill(0);
  }
}

export const probeHostKeys = (c: Conn, ssh: T.SSHView) => request<T.HostKeyProbe>(c, "POST", "/api/hostkeys/probe", { ssh });
export const confirmHostKey = (c: Conn, probeId: string, fingerprint: string) =>
  request(c, "POST", "/api/hostkeys/confirm", { probeId, fingerprint });
export const recordedHostKeys = (c: Conn, hostPort: string) =>
  request<T.RecordedHostKeys>(c, "GET", `/api/hostkeys?hostPort=${e(hostPort)}`);
export const forgetHostKey = (c: Conn, hostPort: string, fingerprint: string) =>
  request(c, "POST", "/api/hostkeys/forget", { hostPort, fingerprint });
export const addTarget = (c: Conn, id: string, ssh: T.SSHView) => request(c, "POST", "/api/targets", { id, mode: "ssh", ssh });
export const removeTarget = (c: Conn, id: string) => request(c, "DELETE", tgt(id));
export const pairingInfo = (c: Conn) => request<T.PairingInfo>(c, "GET", "/api/pairing-info");
export const upgradeGate = (c: Conn, id: string) => request<T.GateUpgrade>(c, "POST", tgt(id, "/agent/upgrade"));
export const sshCommand = (c: Conn, id: string) => request<T.SSHCommand>(c, "GET", flt(id, "/ssh"));
export const startJob = (c: Conn, target: string, kind: string, inputs?: unknown) =>
  request<T.JobView>(c, "POST", `/api/jobs/${e(target)}`, inputs === undefined ? { kind } : { kind, inputs });

/** pair starts pairing id; the reply is an SSE stream of PairEvent that ends with done or error. */
export async function* pair(c: Conn, id: string, req: { sudo: boolean; installed?: string }, signal?: AbortSignal, max?: number): AsyncGenerator<T.PairEvent> {
  const res = await open(c, "POST", tgt(id, "/pair"), req, signal);
  const p = parseSSE(max);
  const dec = new TextDecoder();
  const reader = res.body!.getReader();
  try {
    for (;;) {
      const { done, value } = await reader.read();
      if (done) return;
      const evs = p.push(dec.decode(value, { stream: true }));
      if (p.overflowed) {
        await reader.cancel().catch(() => {});
        yield { error: "the pairing stream sent an event larger than 1 MiB; it was dropped", code: "too_large" };
        return;
      }
      for (const ev of evs) {
        try {
          yield JSON.parse(ev.data) as T.PairEvent;
        } catch {
          /* skipped */
        }
      }
    }
  } finally {
    reader.releaseLock();
  }
}

export const catalog = (c: Conn) => request<{ modules: T.CatalogModule[] }>(c, "GET", "/api/catalog");
export const listModules = (c: Conn, id: string) => request<T.ModulesView>(c, "GET", tgt(id, "/modules"));
export const setModule = (c: Conn, id: string, mod: string, body: T.SetModule) =>
  request<T.SetModuleResult>(c, "PUT", tgt(id, `/modules/${e(mod)}`), body);
export const watchSetup = (c: Conn, id: string, s: AbortSignal, o?: StreamOpts) => stream<T.SetupEvent>(c, tgt(id, "/setup/stream"), s, o);

// Node services and containers (E, rest). Their replies are shown as JSON.
const ctr = (id: string, svc: string, rest = "") => tgt(id, `/containers/${e(svc)}${rest}`);
export const targetDisk = (c: Conn, id: string) => request<unknown>(c, "GET", tgt(id, "/disk"));
export const diskUsage = (c: Conn, id: string) => request<unknown>(c, "GET", tgt(id, "/du"));
export const containers = (c: Conn, id: string) => request<unknown>(c, "GET", tgt(id, "/containers"));
export const container = (c: Conn, id: string, svc: string) => request<unknown>(c, "GET", ctr(id, svc));
export const containerConfig = (c: Conn, id: string, svc: string) => request<unknown>(c, "GET", ctr(id, svc, "/config"));
export const putContainerConfig = (c: Conn, id: string, svc: string, cfg: unknown) => request<unknown>(c, "PUT", ctr(id, svc, "/config"), cfg);
export const containerAction = (c: Conn, id: string, svc: string, action: string) => request<unknown>(c, "POST", ctr(id, svc, `/${e(action)}`));
export const provisionContainer = (c: Conn, id: string, svc: string) => request<unknown>(c, "POST", ctr(id, svc, "/provision"));
/** wipe and clear need the service's name typed back as confirm, as the server checks. */
export const wipeContainer = (c: Conn, id: string, svc: string, confirm: string) => request<unknown>(c, "POST", ctr(id, svc, "/wipe"), { Confirm: confirm });
export const resetContainer = (c: Conn, id: string, svc: string) => request<unknown>(c, "POST", ctr(id, svc, "/reset"));
export const clearService = (c: Conn, id: string, svc: string, confirm: string) =>
  request<unknown>(c, "POST", tgt(id, `/services/${e(svc)}/clear`), { Confirm: confirm });
export const startSetup = (c: Conn, id: string, wire: unknown) => request<unknown>(c, "POST", tgt(id, "/setup"), wire);
export const watchMonitor = (c: Conn, id: string, s: AbortSignal, o?: StreamOpts) => stream<unknown>(c, tgt(id, "/monitor/stream"), s, o);
export const diagnostics = (c: Conn, id: string) => request<unknown>(c, "GET", tgt(id, "/diagnostics"));
export const diagnosticsLatest = (c: Conn, id: string) => request<unknown>(c, "GET", tgt(id, "/diagnostics/latest"));
export const explain = (c: Conn, id: string, lines: string[]) => request<{ text: string; sentExcerpt: string[]; provider: string; local: boolean }>(c, "POST", tgt(id, "/explain"), { lines });
export const intent = (c: Conn, id: string, kind: string, payload: unknown = {}) => request<unknown>(c, "POST", tgt(id, `/intent/${e(kind)}`), payload);

/**
 * jsonWithSecretString builds `{...fields, "<field>": "<value as a JSON string>"}`
 * as bytes, escaping value byte by byte, so a secret that the server reads as a
 * string (a WireGuard config) never becomes a JS string here.
 */
export function jsonWithSecretString(fields: Record<string, unknown>, field: string, value: Uint8Array): Uint8Array {
  const enc = new TextEncoder();
  const lead = Object.entries(fields).map(([k, v]) => `${JSON.stringify(k)}:${JSON.stringify(v)},`).join("");
  const pre = enc.encode(`{${lead}${JSON.stringify(field)}:"`);
  const post = enc.encode(`"}`);
  const out = new Uint8Array(pre.length + value.length * 6 + post.length);
  out.set(pre);
  let o = pre.length;
  const hex = enc.encode("0123456789abcdef");
  for (const b of value) {
    if (b === 0x22 || b === 0x5c) (out[o++] = 0x5c), (out[o++] = b);
    else if (b < 0x20) {
      out.set([0x5c, 0x75, 0x30, 0x30, hex[b >> 4]!, hex[b & 15]!], o);
      o += 6;
    } else out[o++] = b;
  }
  out.set(post, o);
  o += post.length;
  const body = out.slice(0, o);
  out.fill(0);
  return body;
}

// VPN (I).
const vpn = (id: string, rest = "") => `/api/vpns/${e(id)}${rest}`;
const vsrv = (id: string, rest = "") => `/api/vpn-servers/${e(id)}${rest}`;
export const vpns = (c: Conn) => request<T.VPNView[]>(c, "GET", "/api/vpns");
/** saveVPN sends the config bytes (they hold a private key) and zeroes them. */
export async function saveVPN(c: Conn, v: { id: string; provider: string; interface: string; targetId: string; autostart: boolean }, config: Uint8Array): Promise<unknown> {
  const body = jsonWithSecretString(v, "config", config);
  try {
    return await request(c, "POST", "/api/vpns", body);
  } finally {
    body.fill(0);
    config.fill(0);
  }
}
export const deleteVPN = (c: Conn, id: string) => request(c, "DELETE", vpn(id));
export const vpnUpDown = (c: Conn, id: string, up: boolean) => request<unknown>(c, "POST", vpn(id, up ? "/up" : "/down"));
export const vpnStatus = (c: Conn, id: string) => request<unknown>(c, "GET", vpn(id, "/status"));
export const vpnServers = (c: Conn) => request<T.VPNServerView[]>(c, "GET", "/api/vpn-servers");
export const vpnServer = (c: Conn, id: string) => request<T.VPNServerView>(c, "GET", vsrv(id));
export const createVPNServer = (c: Conn, req: Record<string, unknown>) => request<unknown>(c, "POST", "/api/vpn-servers", req);
export const deleteVPNServer = (c: Conn, id: string) => request(c, "DELETE", vsrv(id));
export const vpnServerUpDown = (c: Conn, id: string, up: boolean) => request<unknown>(c, "POST", vsrv(id, up ? "/up" : "/down"));
export const vpnServerStatus = (c: Conn, id: string) => request<unknown>(c, "GET", vsrv(id, "/status"));
export const setVPNEndpoint = (c: Conn, id: string, endpointHost: string) => request<unknown>(c, "POST", vsrv(id, "/endpoint"), { endpointHost });
/** addPeer's reply holds the device's private key in config: show it once, never keep it. */
export const addPeer = (c: Conn, id: string, req: { name: string; fullTunnel: boolean }) =>
  request<{ name: string; publicKey: string; allowedIp: string; config: string }>(c, "POST", vsrv(id, "/peers"), req);
export const removePeer = (c: Conn, id: string, publicKey: string) => request(c, "POST", vsrv(id, "/peers/remove"), { publicKey });

// Gateways, RPC and customer keys (H).
const gw = (gid: string, rest = "") => `/api/gateways/${e(gid)}${rest}`;
export interface GatewayList {
  gateways: { id: string; label?: string; status?: { State?: string }; baseUrl?: string; tier?: string }[];
  orphans?: { containerName: string; targetId: string; mergedInto: string }[];
}
export const gateways = (c: Conn) => request<GatewayList>(c, "GET", "/api/gateways");
export const gateway = (c: Conn, gid: string) => request<unknown>(c, "GET", gw(gid));
export const createGateway = (c: Conn, id: string, targetId: string) => request<unknown>(c, "POST", "/api/gateways", { id, placement: { targetId } });
export const deleteGateway = (c: Conn, gid: string) => request(c, "DELETE", gw(gid));
export const gatewayAction = (c: Conn, gid: string, action: "start" | "stop" | "restart") => request<unknown>(c, "POST", gw(gid, `/${action}`));
export const putGatewayConfig = (c: Conn, gid: string, cfg: unknown) => request<unknown>(c, "PUT", gw(gid, "/config"), cfg);
export const provisionGateway = (c: Conn, gid: string) => request<unknown>(c, "POST", gw(gid, "/provision"));
export const wipeGateway = (c: Conn, gid: string, confirm: string) => request<unknown>(c, "POST", gw(gid, "/wipe"), { Confirm: confirm });
export const setGatewayModule = (c: Conn, gid: string, mod: string, enabled: boolean) => request<unknown>(c, "PUT", gw(gid, `/modules/${e(mod)}`), { enabled });
export const trustCert = (c: Conn, gid: string) => request<unknown>(c, "POST", gw(gid, "/trust-cert"));
export const tlsVerify = (c: Conn, gid: string) => request<unknown>(c, "GET", gw(gid, "/tls/verify"));
export const capabilities = (c: Conn, gid: string) => request<unknown>(c, "GET", gw(gid, "/capabilities"));
export const knownSet = (c: Conn, gid: string, chainId: number) => request<unknown>(c, "GET", gw(gid, `/knownset/${chainId}`));
export const analytics = (c: Conn, gid: string) => request<unknown>(c, "GET", gw(gid, "/analytics"));
export const traffic = (c: Conn, gid: string) => request<unknown>(c, "GET", gw(gid, "/traffic"));
export const customerKeys = (c: Conn, gid: string) => request<unknown>(c, "GET", gw(gid, "/keys"));
/** createCustomerKey's reply holds the raw key once: show it once, never keep it. */
export const createCustomerKey = (c: Conn, gid: string, label: string) => request<{ id: string; key: string }>(c, "POST", gw(gid, "/keys"), { label });
export const revokeCustomerKey = (c: Conn, gid: string, keyId: string) => request<unknown>(c, "POST", gw(gid, `/keys/${e(keyId)}/revoke`));
export const dismissOrphan = (c: Conn, name: string) => request(c, "DELETE", `/api/orphans/${e(name)}`);
export const chainlist = (c: Conn) => request<unknown>(c, "GET", "/api/chainlist");
export const chain = (c: Conn, chainId: number) => request<unknown>(c, "GET", `/api/chainlist/${chainId}`);
export const docker = (c: Conn) => request<unknown>(c, "GET", "/api/docker");
export const startDocker = (c: Conn) => request<unknown>(c, "POST", "/api/docker/start");
export const host = (c: Conn) => request<unknown>(c, "GET", "/api/host");

/** saveAIKey sets the AI provider key, write-only; the bytes and the body are zeroed. */
export async function saveAIKey(c: Conn, key: Uint8Array): Promise<T.Settings> {
  const body = jsonWithSecretString({}, "aiKey", key);
  try {
    return await request<T.Settings>(c, "PUT", "/api/settings", body);
  } finally {
    body.fill(0);
    key.fill(0);
  }
}
export const update = (c: Conn, refresh = false) => request<unknown>(c, "GET", `/api/update${refresh ? "?refresh=1" : ""}`);
export const shutdown = (c: Conn) => request(c, "POST", "/api/shutdown");
