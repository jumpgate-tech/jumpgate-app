// Setup (L) and server control (A, rest): the root-only installs run the
// jumpgate CLI, which owns the terminal for sudo; stop ends the server.
import * as api from "../api/client.ts";
import type { App, Screen } from "../shell/app.ts";
import type { Key } from "../shell/keys.ts";
import { wrap } from "../text.ts";

export class SetupScreen implements Screen {
  readonly id = "setup";
  readonly title = "Setup";
  private note = "";

  constructor(private readonly app: App) {}

  private get linux(): boolean {
    return this.app.platform === "linux";
  }

  view(w: number): string[] {
    const out: string[] = [];
    if (this.linux)
      out.push(
        "[i] install the systemd service (sudo)",
        "[s] set up the service: its vault key, unlock, pair this machine (sudo)",
        "[S] the same, self-only (this machine is the only box)",
        "",
      );
    else out.push("The systemd service and its setup are Linux only.", "");
    out.push("[o] open the web app in your browser", "[X] stop the server");
    if (this.note) out.push("", ...wrap(this.note, w));
    return out;
  }

  key(k: Key): boolean {
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
    return "";
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
      this.app.set({ conn: null, fleet: undefined, inbox: undefined, running: undefined, vault: undefined, serverVersion: undefined });
      this.app.go("connect");
    } catch (e) {
      this.app.fail("stop the server", e);
    }
  }
}
