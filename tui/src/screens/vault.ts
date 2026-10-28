// Vault (B, slice part): state, unlock with a masked passphrase, lock.
import * as api from "../api/client.ts";
import { ApiError } from "../api/http.ts";
import { redact, withSecret } from "../secret.ts";
import type { App, Screen } from "../shell/app.ts";
import type { Key } from "../shell/keys.ts";
import { maskText } from "../maskurl.ts";
import { na, sanitizeLine, wrap } from "../text.ts";

export class VaultScreen implements Screen {
  readonly id = "vault";
  readonly title = "Vault";
  private note = "";
  private busy = false;

  constructor(private readonly app: App) {}

  enter(): void {
    void this.app.refreshVault();
  }

  view(w: number): string[] {
    const v = this.app.state.vault;
    const lines = [
      `state  ${na(v?.state, sanitizeLine)}`,
      `slots  ${v ? v.slots.map(sanitizeLine).join(", ") || "none" : "n/a"}`,
    ];
    if (v?.reason) lines.push(`reason ${sanitizeLine(v.reason)}`);
    if (v?.plain?.length) lines.push(`plain  ${v.plain.map(sanitizeLine).join(", ")} (not in the vault yet)`);
    lines.push("", "[u] unlock   [l] lock");
    if (this.note) lines.push("", ...wrap(this.note, w));
    return lines;
  }

  key(k: Key): boolean {
    if (this.busy || !this.app.state.conn) return false;
    if (k.name === "u") return void this.unlock(), true;
    if (k.name === "l") return void this.lock(), true;
    return false;
  }

  hints(): string {
    return "u unlock  l lock";
  }

  private async unlock(): Promise<void> {
    const pass = await this.app.askSecret("vault passphrase");
    if (!pass) return;
    this.busy = true;
    try {
      const v = await withSecret(pass, (b) => api.unlock(this.app.conn, b));
      this.note = "Unlocked.";
      this.app.set({ vault: v });
      this.app.log("vault unlocked");
    } catch (e) {
      const ra = e instanceof ApiError && e.retryAfter ? ` (try again in ${e.retryAfter} s)` : "";
      this.note = `FAIL ${maskText(sanitizeLine(redact((e as Error).message)))}${ra}`;
      this.app.fail("unlock", e);
    } finally {
      this.busy = false;
      this.app.render();
    }
  }

  private async lock(): Promise<void> {
    if (!(await this.app.confirm("Lock the vault? Signing stops until it is unlocked again."))) return;
    try {
      const v = await api.lock(this.app.conn);
      this.note = "Locked.";
      this.app.set({ vault: v });
      this.app.log("vault locked");
    } catch (e) {
      this.note = `FAIL ${maskText(sanitizeLine(redact((e as Error).message)))}`;
      this.app.fail("lock", e);
    }
  }
}
