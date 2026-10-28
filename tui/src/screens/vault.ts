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

  private idle?: number;

  private get linux(): boolean {
    return this.app.platform === "linux";
  }

  enter(): void {
    void this.app.refreshVault();
    if (this.app.state.conn)
      api.settings(this.app.conn).then((s) => ((this.idle = s.vaultIdleLockMinutes), this.app.render()), (e) => this.app.fail("settings", e));
  }

  view(w: number): string[] {
    const v = this.app.state.vault;
    const lines = [
      `state  ${na(v?.state, sanitizeLine)}`,
      `slots  ${v ? v.slots.map(sanitizeLine).join(", ") || "none" : "n/a"}`,
    ];
    if (v?.reason) lines.push(`reason ${sanitizeLine(v.reason)}`);
    if (v?.plain?.length) lines.push(`plain  ${v.plain.map(sanitizeLine).join(", ")} (not in the vault yet)`);
    const idle = this.idle === undefined ? "n/a" : this.idle === 0 ? "off" : `${this.idle} min`;
    lines.push(`idle lock ${idle}`);
    lines.push("", "[u] unlock   [l] lock   [t] idle lock minutes   [p] put relay-token/admin-token");
    lines.push("[i] init   [m] migrate plain secrets   [b] backup   [R] restore");
    lines.push("[k] add recovery code   [K] raise the KDF cost");
    if (this.linux) lines.push("[s] seal --tpm2   [S] seal --allow-host-key");
    lines.push("", "The CLI actions take over the terminal and ask for passphrases there.");
    if (this.note) lines.push("", ...wrap(this.note, w));
    return lines;
  }

  key(k: Key): boolean {
    if (this.busy || !this.app.state.conn) return false;
    if (k.name === "u") return void this.unlock(), true;
    if (k.name === "l") return void this.lock(), true;
    switch (k.seq) {
      case "t":
        return void this.idleLock(), true;
      case "p":
        return void this.put(), true;
      case "i":
        return void this.app.run(["vault", "init"]), true;
      case "m":
        return void this.cli("Move the plain secrets into the vault?", ["vault", "migrate"]), true;
      case "b":
        return void this.withPath("backup to file", (f) => this.app.run(["vault", "backup", f])), true;
      case "R":
        return void this.withPath("restore from file", (f) => this.cli(`Replace the vault with ${f}?`, ["vault", "restore", f])), true;
      case "k":
        return void this.cli("Add a recovery code? It prints once on the terminal: copy it before you press Enter.", ["vault", "add-recovery"], true), true;
      case "K":
        return void this.cli("Raise the vault's argon2id cost? It re-encrypts the vault.", ["vault", "rekdf"]), true;
      case "s":
        if (this.linux) return void this.cli("Seal the vault key with the TPM (systemd-creds, needs sudo)?", ["vault", "seal", "--tpm2"]), true;
        return false;
      case "S":
        if (this.linux) return void this.cli("Seal the vault key with the host key (systemd-creds, needs sudo)?", ["vault", "seal", "--allow-host-key"]), true;
        return false;
    }
    return false;
  }

  hints(): string {
    return "u unlock  l lock  t idle  p put  i/m/b/R/k/K" + (this.linux ? "/s/S" : "");
  }

  private async cli(question: string, args: string[], pause = false): Promise<void> {
    if (!(await this.app.confirm(question))) return;
    await this.app.run(args, pause ? { pause } : {});
  }

  private async withPath(prompt: string, fn: (f: string) => Promise<unknown>): Promise<void> {
    const f = (await this.app.askText(prompt))?.trim();
    if (!f) return;
    await fn(f);
  }

  private async idleLock(): Promise<void> {
    const v = await this.app.askText("idle lock minutes (0 = off)", this.idle === undefined ? "" : String(this.idle));
    if (v === null) return;
    const n = Number(v.trim());
    if (!/^\d+$/.test(v.trim()) || n > 1440) {
      this.note = "FAIL idle lock is 0 to 1440 minutes";
      return this.app.render();
    }
    try {
      const s = await api.saveSettings(this.app.conn, { vaultIdleLockMinutes: n });
      this.idle = s.vaultIdleLockMinutes;
      this.note = `Idle lock set to ${n} min.`;
      this.app.log(`vault idle lock ${n} min`);
    } catch (e) {
      this.app.fail("idle lock", e);
    }
  }

  private async put(): Promise<void> {
    const name = (await this.app.askText("secret name (relay-token or admin-token)"))?.trim();
    if (!name) return;
    if (name !== "relay-token" && name !== "admin-token") {
      this.note = "FAIL the name is relay-token or admin-token";
      return this.app.render();
    }
    const value = await this.app.askSecret(name);
    if (!value) return;
    try {
      await withSecret(value, (b) => api.putVaultSecret(this.app.conn, name, b));
      this.note = `Stored ${name}.`;
      this.app.log(`stored ${name} in the vault`);
    } catch (e) {
      this.note = `FAIL ${maskText(sanitizeLine(redact((e as Error).message)))}`;
      this.app.fail(`store ${name}`, e);
    }
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
