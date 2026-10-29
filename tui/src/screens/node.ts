// Node detail (E, slice part): five plain sections switched with 1-5:
// status, disk, endpoints, firewall, logs. s/S/r start, stop or restart the
// selected service (e/b pick exec or beacon) after a confirm.
import * as api from "../api/client.ts";
import type { CheckItem, DiskView, Endpoints, LogHit, NodeStatus } from "../api/types.ts";
import { maskText, maskURL } from "../maskurl.ts";
import { isLogHit, isLogHits, isNodeStatus } from "../api/guards.ts";
import type { App, Screen } from "../shell/app.ts";
import type { Key } from "../shell/keys.ts";
import { na, sanitizeLine } from "../text.ts";

const SECTIONS = ["status", "disk", "endpoints", "firewall", "logs"] as const;
type Section = (typeof SECTIONS)[number];
const LOG_MAX = 500;
const gb = (b: number) => `${(b / 1e9).toFixed(1)} GB`;

export function statusLines(s: NodeStatus | undefined, stale: boolean): string[] {
  if (!s) return ["status n/a"];
  const p = stale ? "~" : "";
  const c = (name: string, x: NodeStatus["exec"]) =>
    `${name.padEnd(7)} ${p}${x.state.padEnd(11)} head ${p}${x.head}  peers ${p}${x.peers}  ${x.active ? "active" : "inactive"}`;
  return [
    `overall ${p}${s.overall}`,
    c("exec", s.exec),
    c("beacon", s.beacon),
    `lag     ${s.headLag === undefined ? "n/a" : p + s.headLag}`,
    `at      ${sanitizeLine(s.at)}`,
  ];
}

export function diskLines(d: DiskView | undefined): string[] {
  if (!d) return ["disk n/a"];
  return [
    `exec      ${gb(d.execBytes)}`,
    `beacon    ${gb(d.beaconBytes)}`,
    `free      ${gb(d.freeBytes)}`,
    `expected  ${gb(d.expectedExecBytes + d.expectedBeaconBytes)} (${sanitizeLine(d.expectedLabel)})`,
    `fit       ${d.fit}`,
    `full in   ${d.daysToFull === undefined ? "n/a" : `${d.daysToFull.toFixed(0)} days`}`,
    `at        ${sanitizeLine(d.at)}`,
  ];
}

export function endpointLines(e: Endpoints | undefined): string[] {
  if (!e) return ["endpoints n/a"];
  const r = (ok: boolean) => (ok ? "reachable" : "FAIL unreachable");
  return [
    `exec    ${sanitizeLine(maskURL(e.ExecHTTP))}  ${r(e.ExecReachable)}`,
    `beacon  ${sanitizeLine(maskURL(e.BeaconHTTP))}  ${r(e.BeaconReachable)}`,
    `chain   ${e.ChainIDMatches ? "matches" : "FAIL does not match"}`,
    `access  ${sanitizeLine(e.Access)}`,
    ...(e.TunnelHint ? [`tunnel  ${maskText(sanitizeLine(e.TunnelHint))}`] : []),
  ];
}

export function firewallLines(items: CheckItem[] | undefined): string[] {
  if (!items) return ["firewall n/a"];
  if (!items.length) return ["no checks"];
  return items.flatMap((i) => [
    `[${i.Status.padEnd(7)}] ${sanitizeLine(i.Title)}${i.Detail ? ` - ${maskText(sanitizeLine(i.Detail))}` : ""}`,
    ...(i.Fix && i.Status !== "pass" ? [`          fix: ${maskText(sanitizeLine(i.Fix))}`] : []),
  ]);
}

export const logLine = (h: LogHit) => `${sanitizeLine(h.at).slice(11, 19)} ${sanitizeLine(h.unit)} ${sanitizeLine(h.severity)} ${sanitizeLine(h.line)}`;

export class NodeScreen implements Screen {
  readonly id = "node";
  readonly parent = "fleet";
  title = "Node";
  target = "";
  section: Section = "status";
  service: "exec" | "beacon" = "exec";
  status?: NodeStatus;
  statusStale = false;
  disk?: DiskView;
  endpoints?: Endpoints;
  firewall?: CheckItem[];
  logs: LogHit[] = [];
  private ac: AbortController | null = null;
  private err: Partial<Record<Section, string>> = {};

  constructor(private readonly app: App) {}

  enter(arg?: unknown): void {
    if (typeof arg !== "string") return;
    this.leave();
    this.target = arg;
    this.title = `Node ${sanitizeLine(arg)}`;
    this.status = this.disk = this.endpoints = this.firewall = undefined;
    this.logs = [];
    this.logBack = 0;
    this.err = {};
    const c = this.app.conn;
    const ac = new AbortController();
    this.ac = ac;
    const id = arg;
    const load = <T>(sec: Section, p: Promise<T>, set: (v: T) => void) =>
      p.then(
        (v) => {
          if (this.ac === ac) set(v), this.app.render();
        },
        (e) => {
          if (this.ac === ac) (this.err[sec] = String(e?.message ?? e)), this.app.fail(`${sec} ${id}`, e);
        },
      );
    load("disk", api.disk(c, id), (v) => (this.disk = v));
    load("endpoints", api.endpoints(c, id), (v) => (this.endpoints = v));
    load("firewall", api.firewall(c, id), (v) => (this.firewall = v));
    void (async () => {
      for await (const m of api.watchStatus(c, id, ac.signal, this.app.opts.streamOpts)) {
        if (m.kind === "event" && m.event === "message") {
          if (isNodeStatus(m.data)) (this.status = m.data), (this.statusStale = false);
        }
        else if (m.kind === "state" && (m.state === "retrying" || m.state === "failed")) this.statusStale = true;
        this.app.render();
      }
    })();
    void (async () => {
      for await (const m of api.watchLogs(c, id, ac.signal, 200, this.app.opts.streamOpts)) {
        if (m.kind !== "event") continue;
        if (m.event === "reset") this.logs = isLogHits(m.data) ? m.data : [];
        else if (m.event === "message" && isLogHit(m.data)) this.logs.push(m.data);
        if (this.logs.length > LOG_MAX) this.logs.splice(0, this.logs.length - LOG_MAX);
        if (this.section === "logs") this.app.render();
      }
    })();
  }

  leave(): void {
    this.ac?.abort();
    this.ac = null;
  }

  crumbs(): string[] {
    return [this.target, this.service];
  }

  /** tabs is the section row; each tab is "N name", the open one marked [ ]. */
  private tabs(): string {
    return SECTIONS.map((s, i) => (s === this.section ? `[${i + 1} ${s}]` : ` ${i + 1} ${s} `)).join(" ");
  }

  /** logBack is how far the logs are scrolled up from the newest line. */
  logBack = 0;

  view(_w: number, h: number): string[] {
    const head = [this.tabs(), ""];
    const e = this.err[this.section];
    let body: string[];
    switch (this.section) {
      case "status":
        body = statusLines(this.status, this.statusStale);
        break;
      case "disk":
        body = [...diskLines(this.disk), "", "[d] measure now"];
        break;
      case "endpoints":
        body = endpointLines(this.endpoints);
        break;
      case "firewall":
        body = firewallLines(this.firewall);
        break;
      case "logs": {
        const room = Math.max(1, h - head.length);
        this.logBack = Math.min(this.logBack, Math.max(0, this.logs.length - room));
        const end = this.logs.length - this.logBack;
        body = this.logs.length ? this.logs.slice(Math.max(0, end - room), end).map(logLine) : ["no log lines yet"];
        break;
      }
    }
    if (e) body = [`FAIL ${maskText(sanitizeLine(e))}`, ...body.filter((l) => !l.endsWith("n/a"))];
    return [...head, ...body];
  }

  click(line: number, col: number): boolean {
    if (line !== 0) return false;
    // Each tab is 1 + its text + 1 wide, joined by one space.
    let x = 0;
    for (let i = 0; i < SECTIONS.length; i++) {
      const wd = `${i + 1} ${SECTIONS[i]}`.length + 2;
      if (col >= x && col < x + wd) return (this.section = SECTIONS[i]!), true;
      x += wd + 1;
    }
    return false;
  }

  key(k: Key): boolean {
    const n = Number(k.name);
    if (n >= 1 && n <= SECTIONS.length) return (this.section = SECTIONS[n - 1]!), true;
    if (k.name === "escape") return this.app.go("fleet"), true;
    if (k.name === "e") return (this.service = "exec"), true;
    if (k.name === "b") return (this.service = "beacon"), true;
    if (this.section === "logs" && k.name === "up") return (this.logBack += 1), true;
    if (this.section === "logs" && k.name === "down") return (this.logBack = Math.max(0, this.logBack - 1)), true;
    if (k.name === "d" && this.section === "disk") return void this.measure(), true;
    if (k.seq === "s") return void this.act("start"), true;
    if (k.seq === "S") return void this.act("stop"), true;
    if (k.seq === "r") return void this.act("restart"), true;
    return false;
  }

  hints(): string {
    return `1-5 section  e/b service  s start  S stop  r restart${this.section === "logs" ? "  up/down scroll" : ""}  esc back`;
  }

  private async measure(): Promise<void> {
    this.app.log(`measuring disk on ${this.target}`);
    try {
      this.disk = await api.measureDisk(this.app.conn, this.target);
      this.app.render();
    } catch (e) {
      this.app.fail(`measure disk ${this.target}`, e);
    }
  }

  private async act(action: api.ServiceAction): Promise<void> {
    const id = this.target;
    const svc = this.service;
    if (!(await this.app.confirm(`${action} the ${svc} service on ${id}?`))) return;
    try {
      const r = await api.serviceAction(this.app.conn, id, svc, action);
      this.app.log(`${action} ${svc} on ${id}: ${na(r?.Active, (a: boolean) => (a ? "active" : "inactive"))}`);
    } catch (e) {
      this.app.fail(`${action} ${svc} on ${id}`, e);
    }
  }
}
