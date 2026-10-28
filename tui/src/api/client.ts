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
export async function* pair(c: Conn, id: string, req: { sudo: boolean; installed?: string }, signal?: AbortSignal): AsyncGenerator<T.PairEvent> {
  const res = await open(c, "POST", tgt(id, "/pair"), req, signal);
  const p = parseSSE();
  const dec = new TextDecoder();
  const reader = res.body!.getReader();
  try {
    for (;;) {
      const { done, value } = await reader.read();
      if (done) return;
      for (const ev of p.push(dec.decode(value, { stream: true }))) {
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
