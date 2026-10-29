// VPN (I): client overlays and VPN servers. A client's WireGuard config holds
// a private key, so it is read from a file as bytes and zeroed after the
// save. A new peer's config is shown once and never kept.
import { readFileSync } from "node:fs";
import * as api from "../api/client.ts";
import type { VPNServerView, VPNView } from "../api/types.ts";
import type { App, Screen } from "../shell/app.ts";
import type { Key } from "../shell/keys.ts";
import { sanitizeLine, wrap } from "../text.ts";
import { jsonLines, maskValue } from "./json.ts";
import { Pick } from "./pick.ts";

const NAME = /^[A-Za-z0-9._-]+$/;

export class VPNScreen implements Screen {
  readonly id = "vpn";
  readonly title = "VPN";
  mode: "clients" | "servers" = "clients";
  private clients?: VPNView[];
  private servers?: VPNServerView[];
  private pick = new Pick();
  private out: string[] = [];

  constructor(
    private readonly app: App,
    private readonly readFile: (path: string) => Uint8Array = (p) => readFileSync(p),
  ) {}

  enter(): void {
    this.load();
  }

  private load(): void {
    if (!this.app.state.conn) return;
    const c = this.app.conn;
    api.vpns(c).then((v) => ((this.clients = v), this.app.render()), (e) => this.app.fail("vpn clients", e));
    api.vpnServers(c).then((v) => ((this.servers = v), this.app.render()), (e) => this.app.fail("vpn servers", e));
  }

  crumbs(): string[] {
    return [this.mode];
  }

  view(w: number): string[] {
    const out = [`[v] show ${this.mode === "clients" ? "servers" : "clients"}`, ""];
    if (this.mode === "clients") {
      const rows = (this.clients ?? []).map(
        (v) => `${v.id.padEnd(14)} ${sanitizeLine(v.provider).padEnd(16)} ${sanitizeLine(v.interface).padEnd(8)} on ${sanitizeLine(v.targetId || "this machine")}${v.autostart ? "  autostart" : ""}`,
      );
      out.push(...(this.clients ? (rows.length ? this.pick.lines(rows) : ["no client overlays"]) : ["n/a"]));
      out.push("", "[n] new (config from a file)   [U] up   [d] down   [s] status   [D] delete");
    } else {
      const rows = (this.servers ?? []).map(
        (s) => `${s.id.padEnd(14)} ${sanitizeLine(s.interface).padEnd(8)} ${sanitizeLine(s.address).padEnd(16)} :${s.listenPort}  ${maskValue(s.endpoint || "no endpoint")}  peers ${s.peers.length}`,
      );
      out.push(...(this.servers ? (rows.length ? this.pick.lines(rows) : ["no VPN servers"]) : ["n/a"]));
      out.push("", "[n] create   [U] up   [d] down   [s] status   [e] endpoint   [p] add peer   [r] remove peer   [D] delete");
      const sel = this.pick.of(this.servers ?? []);
      if (sel?.peers.length) out.push("", "peers:", ...sel.peers.map((p) => `  ${sanitizeLine(p.name).padEnd(14)} ${sanitizeLine(p.allowedIp).padEnd(16)} ${sanitizeLine(p.publicKey)}`));
    }
    if (this.out.length) out.push("", ...this.out.flatMap((l) => wrap(l, w)));
    return out;
  }

  key(k: Key): boolean {
    const list = this.mode === "clients" ? (this.clients ?? []) : (this.servers ?? []);
    if (this.pick.key(k, list.length)) return true;
    if (k.seq === "v") {
      this.mode = this.mode === "clients" ? "servers" : "clients";
      this.pick.sel = 0;
      return true;
    }
    if (k.seq === "n") return void (this.mode === "clients" ? this.newClient() : this.newServer()), true;
    const sel = this.pick.of(list as { id: string }[]);
    if (!sel) return false;
    const id = sel.id;
    const c = () => this.app.conn;
    const srv = this.mode === "servers";
    const acts: Record<string, () => Promise<void>> = {
      U: () => this.call(`up ${id}`, () => (srv ? api.vpnServerUpDown(c(), id, true) : api.vpnUpDown(c(), id, true))),
      d: () => this.call(`down ${id}`, () => (srv ? api.vpnServerUpDown(c(), id, false) : api.vpnUpDown(c(), id, false))),
      s: () => this.call(`status ${id}`, () => (srv ? api.vpnServerStatus(c(), id) : api.vpnStatus(c(), id))),
      D: async () => {
        if (!(await this.app.confirmTyped(`Delete the VPN ${srv ? "server" : "overlay"} ${id}?`, id))) return;
        await this.call(`delete ${id}`, () => (srv ? api.deleteVPNServer(c(), id) : api.deleteVPN(c(), id)));
      },
    };
    if (srv) {
      acts.e = () => this.endpoint(id);
      acts.p = () => this.addPeer(id);
      acts.r = () => this.removePeer(sel as VPNServerView);
    }
    const f = acts[k.seq];
    if (!f) return false;
    void f();
    return true;
  }

  hints(): string {
    return "up/down select";
  }

  private async call(what: string, p: () => Promise<unknown>): Promise<void> {
    try {
      const v = await p();
      this.out = [sanitizeLine(what), ...jsonLines(v)];
      this.app.log(`vpn ${what}: ok`);
    } catch (e) {
      this.app.fail(`vpn ${what}`, e);
    }
    this.load();
  }

  private async newClient(): Promise<void> {
    const id = (await this.app.askText("overlay id"))?.trim();
    if (!id) return;
    if (!NAME.test(id)) return this.say("FAIL the id is letters, digits, '.', '_' and '-'");
    const provider = (await this.app.askText("provider label", "bring-your-own"))?.trim();
    if (!provider) return;
    const iface = (await this.app.askText("interface", "wg0"))?.trim();
    if (!iface) return;
    const targetId = (await this.app.askText("box to run it on (blank: this machine)"))?.trim() ?? "";
    const path = (await this.app.askText("path to the WireGuard .conf file"))?.trim();
    if (!path) return;
    let config: Uint8Array;
    try {
      // The Buffer itself, not a copy, so zeroing it clears the only copy we made.
      config = this.readFile(path);
    } catch {
      return this.say(`FAIL cannot read ${sanitizeLine(path)}`);
    }
    try {
      await api.saveVPN(this.app.conn, { id, provider, interface: iface, targetId, autostart: false }, config);
      this.say(`saved ${id}; the config was sent and dropped`);
      this.app.log(`vpn saved ${id}`);
    } catch (e) {
      this.app.fail(`vpn save ${id}`, e);
    } finally {
      config.fill(0);
    }
    this.load();
  }

  private async newServer(): Promise<void> {
    const id = (await this.app.askText("server id"))?.trim();
    if (!id) return;
    if (!NAME.test(id)) return this.say("FAIL the id is letters, digits, '.', '_' and '-'");
    const targetId = (await this.app.askText("box to run it on"))?.trim();
    if (!targetId) return;
    const host = (await this.app.askText("public host devices dial (blank: derive from the box)"))?.trim();
    if (host === undefined) return;
    await this.call(`create ${id}`, () => api.createVPNServer(this.app.conn, { id, targetId, ...(host ? { endpointHost: host } : {}) }));
  }

  private async endpoint(id: string): Promise<void> {
    const host = (await this.app.askText("public host or domain devices dial"))?.trim();
    if (!host) return;
    await this.call(`endpoint ${id}`, () => api.setVPNEndpoint(this.app.conn, id, host));
  }

  private async addPeer(id: string): Promise<void> {
    const name = (await this.app.askText("device name"))?.trim();
    if (!name) return;
    if (!NAME.test(name)) return this.say("FAIL the name is letters, digits, '.', '_' and '-'");
    const full = await this.app.confirm("Route all of the device's traffic through the tunnel? (n: only the VPN's network)");
    try {
      const r = await api.addPeer(this.app.conn, id, { name, fullTunnel: full });
      this.app.log(`vpn ${id}: added peer ${name} (${r.allowedIp})`);
      await this.app.showOnce(`config for ${name}: copy it into the device's WireGuard app`, r.config);
      r.config = "";
    } catch (e) {
      this.app.fail(`vpn add peer ${name}`, e);
    }
    this.load();
  }

  private async removePeer(s: VPNServerView): Promise<void> {
    const name = (await this.app.askText("peer name to remove"))?.trim();
    if (!name) return;
    const p = s.peers.find((x) => x.name === name);
    if (!p) return this.say(`FAIL no peer named ${sanitizeLine(name)}`);
    if (!(await this.app.confirmTyped(`Remove ${name} from ${s.id}? The device loses access.`, name))) return;
    await this.call(`remove peer ${name}`, () => api.removePeer(this.app.conn, s.id, p.publicKey));
  }

  private say(s: string): void {
    this.out = [s];
    this.app.render();
  }
}
