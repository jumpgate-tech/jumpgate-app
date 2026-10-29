// Setup (L): the first-run checklist, derived from the live state, then
// server control (A, rest). Enter on a step opens its screen and presses
// its key there: the guide reuses the existing actions. The root-only
// installs run the jumpgate CLI, which owns the terminal for sudo; stop
// ends the server.
import * as api from "../api/client.ts";
import type { ModulesView } from "../api/types.ts";
import { guideSteps, nextStep, type Step } from "../shell/guide.ts";
import type { App, Screen } from "../shell/app.ts";
import { key, type Key } from "../shell/keys.ts";
import { wrap } from "../text.ts";
import { Pick } from "./pick.ts";

export class SetupScreen implements Screen {
  readonly id = "setup";
  readonly title = "Setup";
  private note = "";
  private pick = new Pick();
  private modules?: Record<string, ModulesView>;
  /** moved: the user picked a step; until then the selection follows the next step. */
  private moved = false;

  constructor(private readonly app: App) {}

  private get linux(): boolean {
    return this.app.platform === "linux";
  }

  enter(): void {
    this.moved = false;
    void this.loadModules();
  }

  /** loadModules reads each paired box's modules for the optional steps. */
  private async loadModules(): Promise<void> {
    if (!this.app.state.conn) return;
    const rows = (this.app.state.fleet?.rows ?? []).filter((r) => r.agent).slice(0, 8);
    const out: Record<string, ModulesView> = {};
    await Promise.all(rows.map((r) => api.listModules(this.app.conn, r.id).then((v) => void (out[r.id] = v), () => {})));
    this.modules = out;
    this.app.render();
  }

  steps(): Step[] {
    return guideSteps({ ...this.app.guideFacts(), modules: this.modules });
  }

  view(w: number): string[] {
    const steps = this.steps();
    const next = nextStep(steps);
    if (!this.moved) this.pick.sel = Math.max(0, next ? steps.indexOf(next) : 0);
    const req = steps.filter((s) => !s.optional);
    const row = (s: Step, n: number) => `${s.done ? "done" : "todo"}  ${n}. ${s.title}${s === next ? "   <- next" : ""}`;
    const lines = this.pick.lines([...req.map((s, i) => row(s, i + 1)), ...steps.filter((s) => s.optional).map((s, i) => row(s, req.length + i + 1))]);
    // The optional steps follow a blank line and a label; the pick drew them in one run.
    const out = [...lines.slice(0, req.length), "", "optional:", ...lines.slice(req.length)];
    const sel = this.pick.of(steps);
    if (sel) out.push("", ...wrap(`why: ${sel.why}`, w), ...wrap(`do:  ${sel.how}`, w), sel.go ? "[enter] go there" : "");
    out.push("", "server:");
    if (this.linux)
      out.push(
        "[i] install the systemd service (sudo)",
        "[s] set up the service: its vault key, unlock, pair this machine (sudo)",
        "[S] the same, self-only (this machine is the only box)",
      );
    else out.push("The systemd service and its setup are Linux only.");
    out.push("[o] open the web app in your browser", "[X] stop the server");
    if (this.note) out.push("", ...wrap(this.note, w));
    return out;
  }

  /** click on a step selects it; a double click runs it. */
  click(line: number, _col: number, double: boolean): boolean {
    const steps = this.steps();
    const req = steps.filter((s) => !s.optional).length;
    const i = line < req ? line : line >= req + 2 && line < steps.length + 2 ? line - 2 : -1;
    if (i < 0) return false;
    this.pick.sel = i;
    this.moved = true;
    if (double) this.run(steps[i]!);
    return true;
  }

  /** run opens the step's screen and presses its key there. */
  private run(s: Step): void {
    if (!s.go) return;
    this.app.go(s.go.screen);
    if (s.go.key && !s.done) this.app.current.key?.(key(s.go.key.toLowerCase(), { seq: s.go.key, shift: s.go.key !== s.go.key.toLowerCase() }));
  }

  key(k: Key): boolean {
    const steps = this.steps();
    if (this.pick.key(k, steps.length)) return (this.moved = true), true;
    if (k.name === "return") {
      const s = this.pick.of(steps);
      if (s) this.run(s);
      return true;
    }
    const f: Record<string, () => Promise<void>> = {
      o: async () => void (await this.app.run(["open"])),
      X: () => this.stop(),
    };
    if (this.linux) {
      f.i = () => this.cli("Install the jumpgate systemd service? sudo asks for your password.", ["setup", "install-service"]);
      f.s = () => this.cli("Set up the jumpgate service? sudo and the new vault passphrase are asked on the terminal.", ["setup", "server"]);
      f.S = () => this.cli("Set up the jumpgate service as self-only?", ["setup", "server", "--self-only"]);
    }
    const g = f[k.seq];
    if (!g) return false;
    void g();
    return true;
  }

  hints(): string {
    return "up/down step";
  }

  private async cli(q: string, args: string[]): Promise<void> {
    if (await this.app.confirm(q)) await this.app.run(args);
  }

  private async stop(): Promise<void> {
    if (!this.app.state.conn) return;
    if (!(await this.app.confirm("Stop the jumpgate server? Running jobs pause until it starts again."))) return;
    try {
      await api.shutdown(this.app.conn);
      this.app.log("server stopped");
      this.app.detach();
      this.app.set({ conn: null, fleet: undefined, inbox: undefined, running: undefined, vault: undefined, controller: undefined, serverVersion: undefined });
      this.app.go("connect");
    } catch (e) {
      this.app.fail("stop the server", e);
    }
  }
}
