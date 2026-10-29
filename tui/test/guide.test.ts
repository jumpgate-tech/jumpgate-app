// The setup guide: its steps from each state, its progress on the status
// strip, the first-run jump to it, and that its steps run existing actions.
import { afterEach, expect, test } from "bun:test";
import type { ControllerView, Fleet, ModulesView, VaultView } from "../src/api/types.ts";
import { FleetScreen } from "../src/screens/fleet.ts";
import { HostsScreen } from "../src/screens/hosts.ts";
import { SetupScreen } from "../src/screens/setup.ts";
import { VaultScreen } from "../src/screens/vault.ts";
import { guideProgress, guideSteps, nextStep, type GuideFacts } from "../src/shell/guide.ts";
import { statusLine } from "../src/shell/statusbar.ts";
import { json } from "./fake.ts";
import { fixture, mount, recorder, type Mounted } from "./harness.ts";

let m: Mounted | undefined;
afterEach(async () => {
  await m?.close();
  m = undefined;
});

const vault = (state: string, slots: string[] = ["passphrase"]): VaultView => ({ state, slots });
const ctl = (state: string, recorded = "0xabc"): ControllerView => ({ state, recorded, address: recorded, store: "vault" });
const healthy = (): Fleet => {
  const f = fixture("fleet");
  const r = f.rows[0];
  Object.assign(r, { error: null, statusError: null, diskError: null, firewallError: null, alertCount: 0, stale: { status: false, disk: false, firewall: false } });
  return f;
};
const facts = (o: Partial<GuideFacts> = {}): GuideFacts => ({
  connected: true,
  vault: vault("unlocked", ["passphrase", "passphrase"]),
  controller: ctl("ok"),
  fleet: healthy(),
  platform: "linux",
  ...o,
});
const step = (f: GuideFacts, id: string) => guideSteps(f).find((s) => s.id === id)!;
const done = (f: GuideFacts) => Object.fromEntries(guideSteps(f).map((s) => [s.id, s.done]));

test("a fresh install: nothing but the server is done; the vault is next", () => {
  const f = facts({ vault: vault("none", []), controller: ctl("missing", ""), fleet: { at: "", intervalSeconds: 30, rows: [] } });
  expect(done(f)).toEqual({ server: true, vault: false, recovery: false, key: false, box: false, paired: false, seal: false, module: false, alerts: false });
  expect(guideProgress(guideSteps(f))).toEqual({ done: 1, total: 6 });
  expect(nextStep(guideSteps(f))!.id).toBe("vault");
  expect(step(f, "vault").go).toEqual({ screen: "vault", key: "i" });
  expect(step(f, "recovery").how).toContain("Create the vault first");
  expect(step(f, "key").go).toEqual({ screen: "keys", key: "c" });
  expect(step(f, "box").go).toEqual({ screen: "hosts", key: "a" });
});

test("not connected: the server step is next and starts it from Connect", () => {
  const f: GuideFacts = { connected: false, platform: "darwin" };
  const s = guideSteps(f);
  expect(nextStep(s)!.id).toBe("server");
  expect(step(f, "server").go).toEqual({ screen: "connect", key: "s" });
  expect(guideProgress(s).done).toBe(0);
});

test("the vault step follows its state", () => {
  expect(step(facts({ vault: vault("locked") }), "vault")).toMatchObject({ done: false, go: { screen: "vault", key: "u" } });
  expect(step(facts({ vault: vault("unusable") }), "vault")).toMatchObject({ done: false, go: { screen: "vault", key: "R" } });
  expect(step(facts(), "vault").done).toBe(true);
});

test("the recovery code is a second passphrase slot", () => {
  expect(step(facts({ vault: vault("unlocked", ["passphrase"]) }), "recovery")).toMatchObject({ done: false, go: { screen: "vault", key: "k" } });
  expect(step(facts(), "recovery").done).toBe(true);
});

test("the controller key: ok, or locked but recorded, is present", () => {
  expect(step(facts({ controller: ctl("locked") }), "key").done).toBe(true);
  expect(step(facts({ controller: ctl("locked", "") }), "key").done).toBe(false);
  expect(step(facts({ controller: ctl("mismatch") }), "key")).toMatchObject({ done: false, go: { screen: "keys" } });
  expect(step(facts({ controller: undefined }), "key").done).toBe(false);
});

test("a box added but not paired, then paired but unhealthy, then healthy", () => {
  const unpaired = healthy();
  Object.assign(unpaired.rows[0]!, { link: "ssh_only", agent: undefined });
  expect(done(facts({ fleet: unpaired }))).toMatchObject({ box: true, paired: false });
  expect(step(facts({ fleet: unpaired }), "paired").go!.screen).toBe("hosts");
  const sick = fixture("fleet"); // paired, but with alerts
  expect(step(facts({ fleet: sick }), "paired")).toMatchObject({ done: false, go: { screen: "alerts" } });
  expect(step(facts(), "paired").done).toBe(true);
});

test("everything required done: 6/6, the next step is an optional one", () => {
  const s = guideSteps(facts());
  expect(guideProgress(s)).toEqual({ done: 6, total: 6 });
  expect(nextStep(s)!.id).toBe("seal");
  expect(nextStep(s)!.optional).toBe(true);
});

test("the optional steps: seal by slot, a module and watch from the modules read", () => {
  expect(step(facts({ vault: vault("unlocked", ["passphrase", "passphrase", "systemd-creds"]) }), "seal").done).toBe(true);
  expect(step(facts({ platform: "darwin" }), "seal").go).toEqual({ screen: "vault" });
  expect(step(facts(), "module").how).toBe("Not checked yet.");
  const mods = (on: string[]): Record<string, ModulesView> => ({
    alpha: { gateStale: false, message: "", modules: ["msgboard", "watch"].map((id) => ({ id, active: on.includes(id), served: false, accepting: false, count: 0 })) as never },
  });
  expect(done(facts({ modules: mods([]) }))).toMatchObject({ module: false, alerts: false });
  expect(done(facts({ modules: mods(["msgboard"]) }))).toMatchObject({ module: true, alerts: false });
  expect(done(facts({ modules: mods(["watch"]) }))).toMatchObject({ module: true, alerts: true });
});

test("no step text says agent or shows a secret", () => {
  for (const f of [facts(), facts({ vault: vault("none", []), controller: ctl("missing", "") }), { connected: false, platform: "linux" }])
    for (const s of guideSteps(f)) expect(`${s.title} ${s.why} ${s.how}`).not.toMatch(/\bagents?\b|0x[0-9a-f]{40}/i);
});

test("the status strip shows setup N/6 until it is done", () => {
  const base = { connected: true, myVersion: "1", serverVersion: "1" };
  expect(statusLine({ ...base, setup: { done: 3, total: 6 } })).toEndWith(" | setup 3/6");
  expect(statusLine({ ...base, setup: { done: 6, total: 6 } })).not.toContain("setup");
  expect(statusLine(base)).not.toContain("setup");
});

test("an incomplete setup opens the guide once, the next step selected", async () => {
  m = await mount((app) => [new FleetScreen(app), new VaultScreen(app), new SetupScreen(app)], {
    routes: { "GET /api/settings": () => json(fixture("settings")) },
  });
  const f = await m.waitFor("setup 3/6");
  expect(m.app.current.id).toBe("setup");
  expect(f).toMatch(/> todo {2}2\. Vault created and unlocked {3}<- next/);
  expect(f).toContain("do:  Vault: press u");
  // It does not open itself again.
  m.app.go("fleet");
  m.app.set({ vault: fixture("vaultview") });
  expect(m.app.current.id).toBe("fleet");
});

test("a complete setup stays on Fleet and the strip drops the count", async () => {
  m = await mount((app) => [new FleetScreen(app), new SetupScreen(app)], {
    routes: {
      "GET /api/vault": () => json(vault("unlocked", ["passphrase", "passphrase"])),
      "GET /api/fleet/stream": () => new Response(`data: ${JSON.stringify(healthy())}\n\n`, { headers: { "content-type": "text/event-stream" } }),
    },
  });
  await m.waitFor(() => !!(m!.app.state.fleet && m!.app.state.controller && m!.app.state.vault));
  const f = await m.frame();
  expect(m.app.current.id).toBe("fleet");
  expect(f).not.toContain("setup ");
});

test("enter on a step opens its screen and runs its existing action", async () => {
  m = await mount((app) => [new FleetScreen(app), new VaultScreen(app), new SetupScreen(app)], {
    routes: { "GET /api/settings": () => json(fixture("settings")) },
  });
  await m.waitFor("setup 3/6");
  const f = await m.press("return");
  expect(m.app.current.id).toBe("vault");
  expect(f).toContain("vault passphrase:"); // the unlock prompt, from the Vault screen's u
  await m.press("escape");
});

test("a step's action can be clicked: select, then [enter] go there", async () => {
  const rec = recorder();
  m = await mount((app) => [new FleetScreen(app), new VaultScreen(app), new HostsScreen(app), new SetupScreen(app)], {
    app: { run: rec.run },
    routes: { "GET /api/settings": () => json(fixture("settings")), "GET /api/targets": () => json(fixture("targets")) },
  });
  let f = await m.waitFor("setup 3/6");
  const rows = f.split("\n");
  const y = rows.findIndex((l) => l.includes("3. Recovery code added"));
  const x = rows[y]!.indexOf("3. Recovery");
  await m.t.mockMouse.click(x, y);
  f = await m.frame();
  expect(f).toMatch(/> todo {2}3\. Recovery code added/);
  expect(f).toContain("do:  Vault: press k");
  const by = f.split("\n").findIndex((l) => l.includes("[enter] go there"));
  await m.t.mockMouse.click(f.split("\n")[by]!.indexOf("[enter]") + 1, by);
  f = await m.frame();
  expect(m.app.current.id).toBe("vault");
  expect(f).toContain("Add a recovery code?"); // the Vault screen's own confirm
});
