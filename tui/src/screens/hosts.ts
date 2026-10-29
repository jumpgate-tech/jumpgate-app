// Hosts (D, rest): the recorded boxes and what you can do to them: add over
// SSH (probe, confirm the fingerprint, add, pair), add this machine, invite,
// upgrade the gate, remove, forget a host key, ssh, rename legacy names.
import * as api from "../api/client.ts";
import type { PairingInfo, SSHView, TargetView } from "../api/types.ts";
import { maskText } from "../maskurl.ts";
import { checkSSHArgv, localSSH } from "../sshargv.ts";
import { shq } from "../shq.ts";
import type { App, Screen } from "../shell/app.ts";
import type { Key } from "../shell/keys.ts";
import { sanitizeLine, wrap } from "../text.ts";
import { Pick } from "./pick.ts";

const NAME = /^[A-Za-z0-9._-]+$/;
const RELEASE = /^v[0-9]+\.[0-9]+\.[0-9]+$/;
const ADDRESS = /^0x[0-9a-fA-F]{40}$/;

/** inviteLines formats `jumpgate hosts invite NAME` exactly as the CLI does. */
export function inviteLines(version: string, info: PairingInfo, name: string): string[] {
  if (!RELEASE.test(version)) return [`FAIL this controller is a development build (${version}); an invite needs a released version`];
  if (info.scope === "self") return ["FAIL this controller is self-only (self_only)"];
  if (!ADDRESS.test(info.controller)) return ["FAIL the server gave no valid controller address"];
  if (!NAME.test(name)) return [`FAIL ${sanitizeLine(name)} is not a host name`];
  const key = info.transportKey.trim();
  if (!key || /[\r\n]/.test(key)) return ["FAIL the server gave no usable transport key"];
  return [
    "on the box, as root:",
    "curl -fsSL https://jumpgate.app/install.sh | sh -s -- --agent \\",
    `  --version ${shq(version)} --controller ${shq(info.controller)} --transport-key ${shq(key)}`,
    `then: jumpgate hosts add ${shq(name)} --ssh jumpgate@HOST --installed <address printed on the box>`,
  ];
}

/**
 * inviteCommand is the invite's install command as one line, or a FAIL line.
 * It holds only public data: the version, the controller's address and its
 * tunnel public key.
 */
export function inviteCommand(version: string, info: PairingInfo, name: string): { cmd: string } | { fail: string } {
  const l = inviteLines(version, info, name);
  if (l.length < 4) return { fail: l[0]! };
  return { cmd: `${l[1]!.replace(/ \\$/, "")} ${l[2]!.trim()}` };
}

/** parseSSH reads user@host[:port]. */
export function parseSSH(s: string): SSHView | null {
  const m = /^([^@\s]+)@([^:\s]+)(?::(\d+))?$/.exec(s.trim());
  if (!m) return null;
  return { Host: m[2]!, User: m[1]!, Port: m[3] ? Number(m[3]) : 22 };
}

/** linkLabel names a target's link for people: the gate, never "agent". */
export const linkLabel = (l: string) =>
  ({ agent: "gate", agent_local: "gate, local", ssh_only: "ssh only", local_only: "local only" })[l] ?? sanitizeLine(l);

const hostPort = (t: TargetView) => (t.ssh ? `${t.ssh.Host}:${t.ssh.Port || 22}` : "");

export class HostsScreen implements Screen {
  readonly id = "hosts";
  readonly title = "Hosts";
  private targets?: TargetView[];
  private pick = new Pick();
  private panel: string[] = [];
  /** cmd is the invite command shown outside the panes, for copying. */
  private cmd: string | null = null;

  constructor(private readonly app: App) {}

  enter(): void {
    this.load();
  }

  private load(): void {
    if (!this.app.state.conn) return;
    api.targets(this.app.conn).then((t) => ((this.targets = t), this.app.render()), (e) => this.app.fail("targets", e));
  }

  private sel(): TargetView | undefined {
    return this.pick.of(this.targets ?? []);
  }

  view(w: number): string[] {
    if (!this.targets) return ["Loading boxes..."];
    const rows = this.targets.map((t) => {
      const where = t.ssh ? `${t.ssh.User}@${t.ssh.Host}:${t.ssh.Port || 22}` : t.mode;
      return `${t.id.padEnd(14)} ${linkLabel(t.link).padEnd(12)} ${t.agent ? "paired" : "not paired"}  ${where}${t.thisMachine ? "  (this machine)" : ""}`;
    });
    const out = rows.length ? this.pick.lines(rows) : ["No boxes yet."];
    out.push(
      "",
      "[a] add over SSH   [L] add this machine   [i] invite   [u] upgrade the gate",
      "[D] remove   [f] forget a host key   [x] ssh   [M] rename legacy names",
    );
    if (this.panel.length) out.push("", ...this.panel.flatMap((l) => wrap(l, w)));
    return out;
  }

  outside(): string | null {
    return this.cmd;
  }

  key(k: Key): boolean {
    if (this.pick.key(k, this.targets?.length ?? 0)) return true;
    const t = this.sel();
    const act: Record<string, () => Promise<void>> = {
      a: () => this.add(),
      L: () => this.addLocal(),
      i: () => this.invite(),
    };
    const cmd = this.cmd;
    if (cmd) act.c = () => this.app.copy(cmd, "invite command");
    if (t) {
      act.u = () => this.upgrade(t);
      act.D = () => this.remove(t);
      act.f = () => this.forget(t);
      act.x = () => this.ssh(t);
      act.M = () => this.migrate(t);
    }
    const f = act[k.seq];
    if (!f) return false;
    void f();
    return true;
  }

  hints(): string {
    return this.cmd ? "up/down select  c copy the invite" : "up/down select";
  }

  private async add(): Promise<void> {
    const c = this.app.conn;
    const name = (await this.app.askText("box name"))?.trim();
    if (!name) return;
    if (!NAME.test(name)) return this.say(`FAIL ${name} is not a box name (letters, digits, '.', '_' and '-')`);
    const addr = await this.app.askText("ssh as user@host[:port]");
    const ssh = addr ? parseSSH(addr) : null;
    if (!ssh) return this.say("FAIL want user@host or user@host:port");
    const jump = (await this.app.askText("jump host user@host[:port] (blank for none)"))?.trim();
    if (jump === undefined) return;
    if (jump) {
      const j = parseSSH(jump);
      if (!j) return this.say("FAIL want the jump host as user@host[:port]");
      ssh.jump = j;
    }
    const keyPath = (await this.app.askText("ssh key file on this machine (blank for the keys in SSH_AUTH_SOCK)"))?.trim();
    if (keyPath === undefined) return;
    if (keyPath) ssh.KeyPath = keyPath;
    try {
      const probe = await api.probeHostKeys(c, ssh);
      for (const hop of probe.hops) {
        if (hop.state === "confirmed") continue;
        if (hop.state === "mismatch") return this.say(`FAIL ${hop.hostPort} presents a key that contradicts the one on record. Stop and find out why.`);
        if (hop.state === "unreachable" || !hop.probeId || !hop.fingerprint)
          return this.say(`FAIL ${hop.hostPort}: ${maskText(sanitizeLine(hop.error ?? "unreachable"))}`);
        const ok = await this.app.confirmTyped(
          `${hop.hostPort} presents ${hop.keyType ?? "a key"} ${hop.fingerprint}. Compare it with the box (ssh-keygen -lf /etc/ssh/ssh_host_*_key.pub). Trust it?`,
          "yes",
        );
        if (!ok) return this.say("Not added: the host key was not confirmed.");
        await api.confirmHostKey(c, hop.probeId, hop.fingerprint);
      }
      await api.addTarget(c, name, ssh);
      this.app.log(`added ${name}`);
      const installed = await this.app.askText("installed address from `jumpgate agent setup` on the box (blank to install over ssh)");
      if (installed === null) return this.load();
      await this.pairBox(name, { sudo: ssh.User !== "root", ...(installed.trim() ? { installed: installed.trim() } : {}) });
    } catch (e) {
      this.app.fail(`add ${name}`, e);
    }
    this.load();
  }

  private async pairBox(name: string, req: { sudo: boolean; installed?: string }): Promise<void> {
    for await (const ev of api.pair(this.app.conn, name, req)) {
      if (ev.error) {
        this.app.fail(`pair ${name}`, { message: ev.error, hint: ev.hint });
        return;
      }
      if (ev.line || ev.step) this.app.log(`pair ${name}: ${ev.step ?? ""} ${ev.line ?? ""}`.trim());
      if (ev.done) this.app.log(`paired ${name}: gate ${ev.agent ?? "n/a"}`);
    }
  }

  private async addLocal(): Promise<void> {
    const name = (await this.app.askText("name for this machine"))?.trim();
    if (!name) return;
    if (!NAME.test(name)) return this.say(`FAIL ${name} is not a box name`);
    await this.app.run(["hosts", "add", name, "--local"]);
    this.load();
  }

  private async invite(): Promise<void> {
    const name = (await this.app.askText("name for the new box"))?.trim();
    if (!name) return;
    if (!NAME.test(name)) return this.say(`FAIL ${name} is not a box name`);
    try {
      const info = await api.pairingInfo(this.app.conn);
      const r = inviteCommand(this.app.state.serverVersion ?? "n/a", info, name);
      if ("fail" in r) return this.say(r.fail);
      this.cmd = r.cmd;
      this.panel = [
        "on the box, as root, run the command below the panes. [c] copy",
        `then: jumpgate hosts add ${shq(name)} --ssh jumpgate@HOST --installed <address printed on the box>`,
      ].map(sanitizeLine);
      this.app.render();
    } catch (e) {
      this.app.fail("pairing info", e);
    }
  }

  private async upgrade(t: TargetView): Promise<void> {
    try {
      const v = await api.upgradeGate(this.app.conn, t.id);
      this.app.log(v.upgraded ? `gate on ${t.id}: ${v.from} -> ${v.to}` : `gate on ${t.id} is already ${v.to}`);
    } catch (e) {
      this.app.fail(`upgrade the gate on ${t.id}`, e);
    }
  }

  private async remove(t: TargetView): Promise<void> {
    if (!(await this.app.confirmTyped(`Remove ${t.id} from this controller? The box itself is not touched.`, t.id))) return;
    try {
      await api.removeTarget(this.app.conn, t.id);
      this.app.log(`removed ${t.id}`);
    } catch (e) {
      this.app.fail(`remove ${t.id}`, e);
    }
    this.load();
  }

  private async forget(t: TargetView): Promise<void> {
    const hp = hostPort(t);
    if (!hp) return this.say(`${t.id} has no ssh address.`);
    try {
      const rec = await api.recordedHostKeys(this.app.conn, hp);
      const mine = rec.keys.filter((k) => k.store === "jumpgate");
      if (!mine.length) return this.say(`No confirmed keys for ${hp} that jumpgate can forget (OpenSSH's known_hosts is never edited).`);
      for (const k of mine) {
        if (!(await this.app.confirm(`Forget ${k.keyType} ${k.fingerprint} for ${hp}? The next connect asks you to confirm it again.`))) continue;
        await api.forgetHostKey(this.app.conn, hp, k.fingerprint);
        this.app.log(`forgot ${k.fingerprint} for ${hp}`);
      }
    } catch (e) {
      this.app.fail(`host keys for ${hp}`, e);
    }
  }

  private async ssh(t: TargetView): Promise<void> {
    try {
      const cmd = await api.sshCommand(this.app.conn, t.id);
      const bad = checkSSHArgv(cmd.argv);
      if (bad) return this.app.fail(`ssh ${t.id}`, { message: `refusing the server's ssh command: ${bad}` });
      const ssh = this.app.opts.bins?.ssh;
      if (!ssh) return this.app.fail(`ssh ${t.id}`, { message: "ssh was not found on an absolute PATH entry; set JUMPGATE_SSH" });
      await this.app.runArgv(localSSH(cmd.argv, ssh), `ssh ${t.id}`);
    } catch (e) {
      this.app.fail(`ssh ${t.id}`, e);
    }
  }

  private async migrate(t: TargetView): Promise<void> {
    if (!(await this.app.confirm(`Rename the legacy names on ${t.id}? This stops the node while it runs.`))) return;
    try {
      const j = await api.startJob(this.app.conn, t.id, "node.migrate-names");
      this.app.log(`started ${t.id}/${j.id}: follow it on Jobs`);
    } catch (e) {
      this.app.fail(`rename legacy names on ${t.id}`, e);
    }
  }

  private say(s: string): void {
    this.cmd = null;
    this.panel = [s];
    this.app.render();
  }
}
