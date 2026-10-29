// Gateways, RPC and customer keys (H). Replies show as key: value lines; a
// new customer key's raw value goes to the show-once modal and is dropped.
import * as api from "../api/client.ts";
import type { App, Screen } from "../shell/app.ts";
import type { Key } from "../shell/keys.ts";
import { sanitizeLine, wrap } from "../text.ts";
import { jsonLines, maskValue } from "./json.ts";
import { Pick } from "./pick.ts";

const NAME = /^[A-Za-z0-9._-]+$/;

const KEYS = [
  "[n] new  [g] get  [S]/[T]/[R] start/stop/restart  [G] config (JSON)  [P] provision  [W] wipe  [D] delete",
  "[M] module on/off  [c] trust cert  [v] TLS verify  [a] capabilities  [k] known set  [A] analytics  [t] traffic",
  "[K] customer keys  [+] new key  [x] revoke key   [l] chain list  [L] one chain   [d] docker  [s] start docker  [h] host",
  "[O] dismiss an orphaned container",
];

export class GatewaysScreen implements Screen {
  readonly id = "gateways";
  readonly title = "Gateways";
  private list?: api.GatewayList;
  private pick = new Pick();
  private out: string[] = [];

  constructor(private readonly app: App) {}

  enter(): void {
    this.load();
  }

  private load(): void {
    if (!this.app.state.conn) return;
    api.gateways(this.app.conn).then((l) => ((this.list = l), this.app.render()), (e) => this.app.fail("gateways", e));
  }

  view(w: number, h: number): string[] {
    const gws = this.list?.gateways ?? [];
    const rows = gws.map(
      (g) => `${g.id.padEnd(14)} ${maskValue(g.label ?? "").padEnd(14)} ${sanitizeLine(g.status?.State ?? "n/a").padEnd(10)} ${maskValue(g.baseUrl ?? "")}`,
    );
    const out = [...(this.list ? (rows.length ? this.pick.lines(rows) : ["no gateways"]) : ["loading..."])];
    for (const o of this.list?.orphans ?? [])
      out.push(`! orphaned container ${sanitizeLine(o.containerName)} on ${sanitizeLine(o.targetId)} (merged into ${sanitizeLine(o.mergedInto)})`);
    out.push("", ...KEYS);
    if (this.out.length) out.push("", ...this.out.flatMap((l) => wrap(l, w)));
    return out.slice(0, Math.max(h, 1));
  }

  key(k: Key): boolean {
    const gws = this.list?.gateways ?? [];
    if (this.pick.key(k, gws.length)) return true;
    const c = () => this.app.conn;
    const global: Record<string, () => Promise<void>> = {
      n: () => this.create(),
      l: () => this.call("chain list", () => api.chainlist(c())),
      L: () => this.num("chain id", (n) => this.call(`chain ${n}`, () => api.chain(c(), n))),
      d: () => this.call("docker", () => api.docker(c())),
      s: async () => {
        if (await this.app.confirm("Start Docker on this machine?")) await this.call("start docker", () => api.startDocker(c()));
      },
      h: () => this.call("host", () => api.host(c())),
      O: () => this.orphan(),
    };
    const g = this.pick.of(gws)?.id;
    const per: Record<string, () => Promise<void>> = g
      ? {
          g: () => this.call(g, () => api.gateway(c(), g)),
          S: () => this.call(`start ${g}`, () => api.gatewayAction(c(), g, "start")),
          T: () => this.typed(`Stop ${g}? Its RPC stops answering.`, g, `stop ${g}`, () => api.gatewayAction(c(), g, "stop")),
          R: () => this.call(`restart ${g}`, () => api.gatewayAction(c(), g, "restart")),
          G: () => this.json(`${g} config as JSON`, (v) => this.call(`config ${g}`, () => api.putGatewayConfig(c(), g, v))),
          P: () => this.confirmed(`Provision ${g}?`, `provision ${g}`, () => api.provisionGateway(c(), g)),
          W: () => this.wipe(g),
          D: () => this.typed(`Delete the gateway ${g}?`, g, `delete ${g}`, () => api.deleteGateway(c(), g)),
          M: () => this.module(g),
          c: () => this.confirmed(`Trust ${g}'s certificate on this machine? The OS may ask you.`, `trust cert ${g}`, () => api.trustCert(c(), g)),
          v: () => this.call(`TLS ${g}`, () => api.tlsVerify(c(), g)),
          a: () => this.call(`capabilities ${g}`, () => api.capabilities(c(), g)),
          k: () => this.num("chain id", (n) => this.call(`known set ${g} ${n}`, () => api.knownSet(c(), g, n))),
          A: () => this.call(`analytics ${g}`, () => api.analytics(c(), g)),
          t: () => this.call(`traffic ${g}`, () => api.traffic(c(), g)),
          K: () => this.call(`customer keys ${g}`, () => api.customerKeys(c(), g)),
          "+": () => this.newKey(g),
          x: () => this.revoke(g),
        }
      : {};
    const f = per[k.seq] ?? global[k.seq];
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
      this.app.log(`${what}: ok`);
    } catch (e) {
      this.app.fail(what, e);
    }
    this.app.render();
  }

  private async confirmed(q: string, what: string, p: () => Promise<unknown>): Promise<void> {
    if (!(await this.app.confirm(q))) return;
    await this.call(what, p);
    this.load();
  }

  private async typed(q: string, want: string, what: string, p: () => Promise<unknown>): Promise<void> {
    if (!(await this.app.confirmTyped(q, want))) return;
    await this.call(what, p);
    this.load();
  }

  private say(s: string): void {
    this.out = [s];
    this.app.render();
  }

  private async num(prompt: string, fn: (n: number) => Promise<void>): Promise<void> {
    const v = (await this.app.askText(prompt))?.trim();
    if (!v) return;
    if (!/^[0-9]{1,10}$/.test(v)) return this.say("FAIL a chain id is a number");
    await fn(Number(v));
  }

  private async json(prompt: string, fn: (v: unknown) => Promise<void>): Promise<void> {
    const raw = await this.app.askText(prompt);
    if (!raw?.trim()) return;
    let v: unknown;
    try {
      v = JSON.parse(raw);
    } catch {
      return this.say("FAIL not JSON");
    }
    await fn(v);
  }

  private async create(): Promise<void> {
    const id = (await this.app.askText("gateway id"))?.trim();
    if (!id) return;
    if (!NAME.test(id)) return this.say("FAIL the id is letters, digits, '.', '_' and '-'");
    const t = (await this.app.askText("box it runs on"))?.trim();
    if (!t) return;
    await this.call(`create ${id}`, () => api.createGateway(this.app.conn, id, t));
    this.load();
  }

  private async wipe(g: string): Promise<void> {
    if (!(await this.app.confirmTyped(`Wipe ${g}: its data is deleted.`, g))) return;
    await this.call(`wipe ${g}`, () => api.wipeGateway(this.app.conn, g, g));
  }

  private async module(g: string): Promise<void> {
    const mod = (await this.app.askText("module id"))?.trim();
    if (!mod) return;
    if (!NAME.test(mod)) return this.say("FAIL a module id is a plain name");
    const on = await this.app.confirm(`Turn ${mod} on for ${g}? (n turns it off)`);
    await this.call(`${mod} ${on ? "on" : "off"} for ${g}`, () => api.setGatewayModule(this.app.conn, g, mod, on));
  }

  private async newKey(g: string): Promise<void> {
    const label = await this.app.askText("label for the new customer key");
    if (label === null) return;
    try {
      const r = await api.createCustomerKey(this.app.conn, g, label.trim());
      this.app.log(`created customer key ${sanitizeLine(r.id)} on ${g}`);
      await this.app.showOnce(`customer key ${sanitizeLine(r.id)}: give it to the customer`, r.key);
      r.key = "";
    } catch (e) {
      this.app.fail(`create a key on ${g}`, e);
    }
  }

  private async revoke(g: string): Promise<void> {
    const id = (await this.app.askText("key id to revoke"))?.trim();
    if (!id) return;
    if (!NAME.test(id)) return this.say("FAIL a key id is a plain name");
    await this.typed(`Revoke ${id} on ${g}? Its calls start failing at once.`, id, `revoke ${id}`, () => api.revokeCustomerKey(this.app.conn, g, id));
  }

  private async orphan(): Promise<void> {
    const name = (await this.app.askText("orphaned container name to dismiss"))?.trim();
    if (!name) return;
    if (!(await this.app.confirm(`Forget the record of ${name}? The container keeps running; stop it on its box.`))) return;
    await this.call(`dismiss ${name}`, () => api.dismissOrphan(this.app.conn, name));
    this.load();
  }
}
