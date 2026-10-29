// Node services and containers (E, rest), one box at a time. Every reply is
// shown as plain key: value lines. Destructive actions need the service's
// name typed back, as the server itself checks.
import * as api from "../api/client.ts";
import { isSetupEvent } from "../api/guards.ts";
import type { App, Screen } from "../shell/app.ts";
import type { Key } from "../shell/keys.ts";
import { sanitizeLine, wrap } from "../text.ts";
import { jsonLines, maskValue } from "./json.ts";
import { Pick } from "./pick.ts";

const SERVICES = ["exec", "beacon", "devnet"] as const;

const KEYS = [
  "[1-3] service   [c] containers   [C] container status   [g] config   [G] set config (JSON)",
  "[a] container action   [P] provision   [W] wipe   [X] reset devnet   [K] clear service state",
  "[T] disk   [u] du   [w] setup wizard (wire JSON)   [m] monitor   [D] diagnostics   [d] latest",
  "[E] explain recent logs (AI)   [I] run an intent",
];

export class ServicesScreen implements Screen {
  readonly id = "services";
  readonly title = "Services";
  private pick = new Pick();
  private box: string | null = null;
  svc: (typeof SERVICES)[number] = "exec";
  private out: string[] = [];
  private head = "";
  private mon: AbortController | null = null;

  constructor(private readonly app: App) {}

  leave(): void {
    this.stopMonitor();
  }

  private stopMonitor(): void {
    this.mon?.abort();
    this.mon = null;
  }

  private boxes(): string[] {
    return this.app.state.fleet?.rows.map((r) => r.id) ?? [];
  }

  view(w: number, h: number): string[] {
    if (!this.box) {
      const l = this.pick.lines(this.boxes());
      return ["boxes ([enter] open):", ...(l.length ? l : ["  none"])];
    }
    const top = [`${sanitizeLine(this.box)}  service: ${this.svc}`, ...KEYS, ""];
    const body = this.head ? [this.head, ...this.out] : this.out;
    return [...top, ...body.flatMap((l) => wrap(l, w)).slice(0, Math.max(1, h - top.length))];
  }

  private show(head: string, v: unknown): void {
    this.head = sanitizeLine(head);
    this.out = jsonLines(v);
    this.app.render();
  }

  private async call(head: string, p: () => Promise<unknown>): Promise<void> {
    try {
      this.show(head, await p());
      this.app.log(`${head}: ok`);
    } catch (e) {
      this.app.fail(head, e);
    }
  }

  key(k: Key): boolean {
    if (!this.box) {
      if (this.pick.key(k, this.boxes().length)) return true;
      if (k.name === "return") {
        const id = this.pick.of(this.boxes());
        if (!id) return false;
        this.box = id;
        this.out = [];
        this.head = "";
        return true;
      }
      return false;
    }
    if (k.name === "escape") {
      this.stopMonitor();
      this.box = null;
      return true;
    }
    const n = Number(k.name);
    if (n >= 1 && n <= SERVICES.length) return (this.svc = SERVICES[n - 1]!), true;
    const f = this.actions(this.box)[k.seq];
    if (!f) return false;
    void f();
    return true;
  }

  hints(): string {
    return this.box ? "1-3 service  esc back  (actions listed above)" : "enter open a box";
  }

  private actions(id: string): Record<string, () => Promise<void>> {
    const c = () => this.app.conn;
    const svc = this.svc;
    return {
      c: () => this.call(`containers on ${id}`, () => api.containers(c(), id)),
      C: () => this.call(`${svc} on ${id}`, () => api.container(c(), id, svc)),
      g: () => this.call(`${svc} config on ${id}`, () => api.containerConfig(c(), id, svc)),
      G: () => this.json(`${svc} config on ${id} as JSON`, (v) => this.call(`set ${svc} config on ${id}`, () => api.putContainerConfig(c(), id, svc, v))),
      a: async () => {
        const action = (await this.app.askText(`action for ${svc} (start, stop, restart)`))?.trim();
        if (!action) return;
        if (!/^[a-z-]+$/.test(action)) return this.show("FAIL", "the action is a plain word");
        if (!(await this.app.confirm(`${action} ${svc} on ${id}?`))) return;
        await this.call(`${action} ${svc} on ${id}`, () => api.containerAction(c(), id, svc, action));
      },
      P: async () => {
        if (await this.app.confirm(`Provision ${svc} on ${id}?`)) await this.call(`provision ${svc} on ${id}`, () => api.provisionContainer(c(), id, svc));
      },
      W: () => this.typed(`Wipe ${svc} on ${id}: its data is deleted.`, svc, (t) => this.call(`wipe ${svc} on ${id}`, () => api.wipeContainer(c(), id, svc, t))),
      X: async () => {
        if (await this.app.confirm(`Reset the devnet on ${id}? Its chain starts over.`)) await this.call(`reset devnet on ${id}`, () => api.resetContainer(c(), id, "devnet"));
      },
      K: () => this.typed(`Clear ${svc}'s state on ${id}: it resyncs from scratch.`, svc, (t) => this.call(`clear ${svc} on ${id}`, () => api.clearService(c(), id, svc, t))),
      T: () => this.call(`disk on ${id}`, () => api.targetDisk(c(), id)),
      u: () => this.call(`du on ${id}`, () => api.diskUsage(c(), id)),
      w: () => this.json(`wire config for ${id} as JSON`, (v) => this.setup(id, v)),
      m: () => this.monitor(id),
      D: () => this.call(`diagnostics on ${id}`, () => api.diagnostics(c(), id)),
      d: () => this.call(`latest diagnostics on ${id}`, () => api.diagnosticsLatest(c(), id)),
      E: () => this.explain(id),
      I: async () => {
        const kind = (await this.app.askText("intent kind (e.g. status.read)"))?.trim();
        if (!kind) return;
        if (!/^[a-z0-9.-]+$/.test(kind)) return this.show("FAIL", "the kind is a dotted name");
        await this.call(`intent ${kind} on ${id}`, () => api.intent(c(), id, kind));
      },
    };
  }

  private async json(prompt: string, fn: (v: unknown) => Promise<void>): Promise<void> {
    const raw = await this.app.askText(prompt);
    if (raw === null || !raw.trim()) return;
    let v: unknown;
    try {
      v = JSON.parse(raw);
    } catch {
      return this.show("FAIL", "not JSON");
    }
    await fn(v);
  }

  private async typed(what: string, name: string, fn: (typed: string) => Promise<void>): Promise<void> {
    if (!(await this.app.confirmTyped(what, name))) return;
    await fn(name);
  }

  private async setup(id: string, wire: unknown): Promise<void> {
    try {
      await api.startSetup(this.app.conn, id, wire);
      this.app.log(`setup ${id}: started`);
      const ac = new AbortController();
      for await (const m of api.watchSetup(this.app.conn, id, ac.signal, this.app.opts.streamOpts)) {
        if (m.kind !== "event" || !isSetupEvent(m.data)) continue;
        if (m.data.err) {
          this.app.fail(`setup ${id}`, { message: m.data.err });
          break;
        }
        if (m.data.line) this.app.log(`setup ${id}: ${m.data.stepId} ${m.data.line}`);
        if (m.data.done && !m.data.stepId) {
          this.app.log(`setup ${id}: done`);
          break;
        }
      }
      ac.abort();
    } catch (e) {
      this.app.fail(`setup ${id}`, e);
    }
  }

  private async monitor(id: string): Promise<void> {
    this.stopMonitor();
    const ac = new AbortController();
    this.mon = ac;
    for await (const m of api.watchMonitor(this.app.conn, id, ac.signal, this.app.opts.streamOpts)) {
      if (m.kind === "event") this.show(`monitor ${id} (latest; m again restarts, esc stops)`, m.data);
      else if (m.state === "retrying") this.head = `monitor ${id} ~ (reconnecting)`;
    }
  }

  private async explain(id: string): Promise<void> {
    if (!(await this.app.confirm(`Send ${id}'s recent log lines to the AI provider in Settings to explain them? Known secrets are redacted first.`))) return;
    try {
      const hits = await api.logs(this.app.conn, id, 50);
      const r = await api.explain(this.app.conn, id, hits.map((h) => h.line));
      this.head = `explain ${id} (${sanitizeLine(r.provider)}${r.local ? ", local" : ""})`;
      this.out = [...r.text.split("\n").map(maskValue), "", `sent ${r.sentExcerpt.length} lines`];
      this.app.render();
    } catch (e) {
      this.app.fail(`explain ${id}`, e);
    }
  }
}
