// Modules (F): the catalog; per box, its modules with on/off and a raw JSON
// config editor (msgboard, el-bootnode, cl-bootnode, watch, private-send,
// checkpointz). A change that starts setup follows the setup stream.
import * as api from "../api/client.ts";
import { isSetupEvent } from "../api/guards.ts";
import type { CatalogModule, ModulesView, SetModule } from "../api/types.ts";
import { maskText } from "../maskurl.ts";
import type { App, Screen } from "../shell/app.ts";
import type { Key } from "../shell/keys.ts";
import { sanitizeLine, wrap } from "../text.ts";
import { Pick } from "./pick.ts";

const clean = (s: string) => maskText(sanitizeLine(s));

export class ModulesScreen implements Screen {
  readonly id = "modules";
  readonly title = "Modules";
  private cat?: CatalogModule[];
  private boxPick = new Pick();
  private modPick = new Pick();
  private box: string | null = null;
  private view_?: ModulesView;
  private note = "";

  constructor(private readonly app: App) {}

  enter(): void {
    if (!this.app.state.conn) return;
    api.catalog(this.app.conn).then((c) => ((this.cat = c.modules), this.app.render()), (e) => this.app.fail("catalog", e));
  }

  crumbs(): string[] {
    return this.box ? [this.box] : [];
  }

  private boxes(): string[] {
    return this.app.state.fleet?.rows.map((r) => r.id) ?? [];
  }

  private load(id: string): void {
    api.listModules(this.app.conn, id).then((v) => ((this.view_ = v), this.app.render()), (e) => this.app.fail(`modules on ${id}`, e));
  }

  view(w: number): string[] {
    const out: string[] = ["catalog:"];
    out.push(...(this.cat ? this.cat.map((c) => `  ${c.id.padEnd(14)} ${clean(c.name).padEnd(16)} ${clean(c.placement).padEnd(8)} ${clean(c.summary)}`) : ["  n/a"]));
    out.push("");
    if (!this.box) {
      out.push("boxes:", ...(this.boxPick.lines(this.boxes()).length ? this.boxPick.lines(this.boxes()) : ["  none"]));
    } else {
      out.push("[space] on/off   [c] config as JSON   [esc] back");
      const v = this.view_;
      if (!v) out.push("  loading...");
      else {
        if (v.gateStale) out.push("  ~ the gate is out of date; values may be stale");
        if (v.message) out.push(`  ${clean(v.message)}`);
        const rows = v.modules.map(
          (s) =>
            `${s.id.padEnd(7)} ${s.active ? "active" : "off"}${s.served ? " served" : ""}${s.accepting ? " accepting" : ""} count ${s.count}` +
            `${s.note ? `  ${clean(s.note)}` : ""}${s.error ? `  FAIL ${clean(s.error)}` : ""}`,
        );
        out.push(...(rows.length ? this.modPick.lines(rows) : ["  no modules"]));
      }
    }
    if (this.note) out.push("", ...wrap(this.note, w));
    return out;
  }

  key(k: Key): boolean {
    if (!this.box) {
      if (this.boxPick.key(k, this.boxes().length)) return true;
      if (k.name === "return") {
        const id = this.boxPick.of(this.boxes());
        if (!id) return false;
        this.box = id;
        this.view_ = undefined;
        this.load(id);
        return true;
      }
      return false;
    }
    const mods = this.view_?.modules ?? [];
    if (this.modPick.key(k, mods.length)) return true;
    if (k.name === "escape") return (this.box = null), true;
    const mod = this.modPick.of(mods);
    if (!mod) return false;
    if (k.name === "space") return void this.toggle(this.box, mod.id, !mod.active), true;
    if (k.seq === "c") return void this.edit(this.box, mod.id, mod.active), true;
    return false;
  }

  hints(): string {
    return "up/down select" + (this.box ? "" : "  enter open");
  }

  private async toggle(id: string, mod: string, on: boolean): Promise<void> {
    if (!(await this.app.confirm(`Turn ${mod} ${on ? "on" : "off"} on ${id}?`))) return;
    await this.send(id, mod, { enabled: on });
  }

  private async edit(id: string, mod: string, active: boolean): Promise<void> {
    const raw = await this.app.askText(`${mod} on ${id} as JSON, e.g. {"enabled":${active}}`);
    if (raw === null) return;
    let body: SetModule;
    try {
      body = JSON.parse(raw);
      if (typeof body !== "object" || body === null || typeof body.enabled !== "boolean") throw new Error();
    } catch {
      this.note = 'FAIL not JSON with a boolean "enabled"';
      return this.app.render();
    }
    await this.send(id, mod, body);
  }

  private async send(id: string, mod: string, body: SetModule): Promise<void> {
    try {
      const r = await api.setModule(this.app.conn, id, mod, body);
      this.note = `${mod} on ${id}: ${sanitizeLine(r.status)}${r.failed?.length ? `; FAIL gateways ${r.failed.map((f) => `${f.id}: ${clean(f.error)}`).join(", ")}` : ""}`;
      this.app.log(this.note);
      if (r.status === "started") void this.followSetup(id);
      this.load(id);
    } catch (e) {
      this.app.fail(`${mod} on ${id}`, e);
    }
  }

  private async followSetup(id: string): Promise<void> {
    const ac = new AbortController();
    for await (const m of api.watchSetup(this.app.conn, id, ac.signal, this.app.opts.streamOpts)) {
      if (m.kind !== "event" || !isSetupEvent(m.data)) continue;
      const ev = m.data;
      if (ev.err) {
        this.app.fail(`setup ${id}`, { message: ev.err });
        break;
      }
      if (ev.line) this.app.log(`setup ${id}: ${ev.stepId} ${ev.line}`);
      if (ev.done && !ev.stepId) {
        this.app.log(`setup ${id}: done`);
        break;
      }
    }
    ac.abort();
  }
}
