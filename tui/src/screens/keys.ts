// Signers and keys (C): the controller key, test signing on a box, create,
// export, restart the service to load a new key.
import * as api from "../api/client.ts";
import type { AgentCheck, ControllerView } from "../api/types.ts";
import { maskText } from "../maskurl.ts";
import { withSecret } from "../secret.ts";
import type { App, Screen } from "../shell/app.ts";
import type { Key } from "../shell/keys.ts";
import { na, sanitizeLine, wrap } from "../text.ts";
import { Pick } from "./pick.ts";

export class KeysScreen implements Screen {
  readonly id = "keys";
  readonly title = "Keys";
  private ctl?: ControllerView;
  private check: Record<string, AgentCheck | string> = {};
  private pick = new Pick();
  private note = "";

  constructor(private readonly app: App) {}

  enter(): void {
    if (!this.app.state.conn) return;
    api.controller(this.app.conn).then((c) => ((this.ctl = c), this.app.render()), (e) => this.app.fail("controller", e));
  }

  private boxes(): string[] {
    return this.app.state.fleet?.rows.map((r) => r.id) ?? [];
  }

  view(w: number): string[] {
    const c = this.ctl;
    const s = (v?: string) => na(v || undefined, (x: string) => sanitizeLine(x));
    const out = [
      `address   ${s(c?.address)}`,
      `recorded  ${s(c?.recorded)}`,
      `store     ${s(c?.store)}`,
      `state     ${s(c?.state)}${c?.reason ? ` - ${maskText(sanitizeLine(c.reason))}` : ""}`,
      "",
      "boxes ([t] test signing on the selected one):",
    ];
    const ids = this.boxes();
    const lines = this.pick.lines(ids).map((l, i) => {
      const r = this.check[ids[i]!];
      if (!r) return l;
      if (typeof r === "string") return `${l}  FAIL ${r}`;
      return `${l}  signed: gate ${sanitizeLine(r.version)} chain ${r.chainId}${r.setUp ? "" : " (not set up)"}`;
    });
    out.push(...(lines.length ? lines : ["  none"]));
    out.push("", "[c] create the key   [e] export a keystore v3 file   [R] restart the service");
    if (this.note) out.push("", ...wrap(this.note, w));
    return out;
  }

  key(k: Key): boolean {
    if (this.pick.key(k, this.boxes().length)) return true;
    switch (k.seq) {
      case "t":
        return void this.test(), true;
      case "c":
        return void this.create(), true;
      case "e":
        return void this.export(), true;
      case "R":
        return void this.restart(), true;
    }
    return false;
  }

  hints(): string {
    return "up/down select";
  }

  private async test(): Promise<void> {
    const id = this.pick.of(this.boxes());
    if (!id) return;
    try {
      this.check[id] = await api.checkGate(this.app.conn, id);
      this.app.log(`test signing on ${id}: ok`);
    } catch (e) {
      this.check[id] = maskText(sanitizeLine((e as Error).message));
      this.app.fail(`test signing on ${id}`, e);
    }
  }

  private async create(): Promise<void> {
    if (!this.app.conn.system) {
      await this.app.run(["keys", "init"]);
      return this.enter();
    }
    const pass = await this.app.askSecret("new vault passphrase for the service");
    if (!pass) return;
    try {
      await withSecret(pass, (b) => api.systemInitKey(this.app.conn, b));
      this.note = "The service created its key. Unlock the vault, then restart the service with R.";
      this.app.log("the service created its controller key");
      this.enter();
    } catch (e) {
      this.app.fail("create the service key", e);
    }
  }

  private async export(): Promise<void> {
    const f = (await this.app.askText("keystore v3 file to write"))?.trim();
    if (!f) return;
    if (!(await this.app.confirm(`Write the controller key to ${f}? Anyone with the file and its passphrase holds the key.`))) return;
    await this.app.run(["keys", "export", "--keystore-v3", f]);
  }

  private async restart(): Promise<void> {
    if (!this.app.conn.system) {
      this.note = "Only the jumpgate service restarts over the API. For your own server: jumpgate stop, then start it from Connect.";
      return this.app.render();
    }
    if (!(await this.app.confirm("Restart the jumpgate service so it loads the new key?"))) return;
    try {
      await api.restart(this.app.conn);
      this.app.log("restarting the service");
      setTimeout(() => this.app.go("connect"), 1000);
    } catch (e) {
      this.app.fail("restart", e);
    }
  }
}
