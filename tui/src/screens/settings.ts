// Settings and prefs (J): server settings one field at a time, the AI key
// write-only, the drawer pref, the update check.
import * as api from "../api/client.ts";
import type { DrawerPref, Settings } from "../api/types.ts";
import { withSecret } from "../secret.ts";
import type { App, Screen } from "../shell/app.ts";
import type { Key } from "../shell/keys.ts";
import { na, sanitizeLine, wrap } from "../text.ts";
import { jsonLines } from "./json.ts";

const DRAWERS: DrawerPref[] = ["auto", "open", "min"];

export class SettingsScreen implements Screen {
  readonly id = "settings";
  readonly title = "Settings";
  private s?: Settings & Record<string, unknown>;
  private upd: string[] = [];
  private note = "";

  constructor(private readonly app: App) {}

  enter(): void {
    if (!this.app.state.conn) return;
    api.settings(this.app.conn).then((s) => ((this.s = s as never), this.app.render()), (e) => this.app.fail("settings", e));
    void this.check(false);
  }

  view(w: number): string[] {
    const s = this.s;
    const str = (v: unknown) => na(v, (x: unknown) => sanitizeLine(String(x)));
    const out = [
      `AI provider       ${str(s?.aiProvider || undefined)}`,
      `AI key            ${s ? (s.aiKeySet ? "set" : "not set") : "n/a"}`,
      `idle lock         ${s ? `${s.vaultIdleLockMinutes} min` : "n/a"}`,
      `update notices    ${s ? (s.updateNotifyEnabled ? "on" : "off") : "n/a"}`,
      `drawer            ${this.app.state.drawer}`,
      "",
      "[p] AI provider   [k] AI key (write-only)   [i] idle lock   [d] drawer auto/open/min   [r] check for updates",
    ];
    if (s?.aiDisclosure) out.push("", ...wrap(`AI: ${sanitizeLine(String(s.aiDisclosure))}`, w));
    if (this.upd.length) out.push("", "update:", ...this.upd);
    if (this.note) out.push("", ...wrap(this.note, w));
    return out;
  }

  key(k: Key): boolean {
    const f: Record<string, () => Promise<void>> = {
      p: () => this.provider(),
      k: () => this.aiKey(),
      i: () => this.idle(),
      d: () => this.drawer(),
      r: () => this.check(true),
    };
    const g = f[k.seq];
    if (!g || !this.app.state.conn) return false;
    void g();
    return true;
  }

  hints(): string {
    return "";
  }

  private async save(u: api.SettingsUpdateT, what: string): Promise<void> {
    try {
      this.s = (await api.saveSettings(this.app.conn, u)) as never;
      this.note = `Saved ${what}.`;
      this.app.log(`settings: ${what}`);
    } catch (e) {
      this.app.fail(`save ${what}`, e);
    }
    this.app.render();
  }

  private async provider(): Promise<void> {
    const v = (await this.app.askText("AI provider (blank: none)", this.s?.aiProvider ?? ""))?.trim();
    if (v === undefined) return;
    if (v && !/^[a-z0-9-]+$/.test(v)) return void ((this.note = "FAIL a provider is a plain name"), this.app.render());
    await this.save({ aiProvider: v }, "the AI provider");
  }

  private async aiKey(): Promise<void> {
    const key = await this.app.askSecret("AI provider key");
    if (!key) return;
    try {
      this.s = (await withSecret(key, (b) => api.saveAIKey(this.app.conn, b))) as never;
      this.note = "Saved the AI key.";
      this.app.log("settings: the AI key");
    } catch (e) {
      this.app.fail("save the AI key", e);
    }
    this.app.render();
  }

  private async idle(): Promise<void> {
    const v = (await this.app.askText("idle lock minutes (0 = off)"))?.trim();
    if (!v) return;
    if (!/^\d+$/.test(v) || Number(v) > 1440) return void ((this.note = "FAIL idle lock is 0 to 1440 minutes"), this.app.render());
    await this.save({ vaultIdleLockMinutes: Number(v) }, "the idle lock");
  }

  private async drawer(): Promise<void> {
    const { prefs } = this.app.state;
    if (!prefs) return;
    const next = DRAWERS[(DRAWERS.indexOf(this.app.state.drawer) + 1) % DRAWERS.length]!;
    try {
      const p = await api.savePrefs(this.app.conn, { ...prefs, drawer: next });
      this.app.set({ prefs: p, drawer: p.drawer });
    } catch (e) {
      this.app.fail("save the drawer setting", e);
    }
  }

  private async check(refresh: boolean): Promise<void> {
    try {
      this.upd = jsonLines(await api.update(this.app.conn, refresh)).map((l) => `  ${l}`);
    } catch (e) {
      this.upd = [];
      if (refresh) this.app.fail("update check", e);
    }
    this.app.render();
  }
}
