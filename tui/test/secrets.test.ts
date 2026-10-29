// Whole-app leak scan: every screen is drawn against a fake server whose data
// and errors carry fixture secrets in the places the server may put them
// (URL userinfo, key-like path segments, query keys, error text, a bearer
// token), and no frame or activity line may show one.
import { expect, test } from "bun:test";
import { createTestRenderer } from "@opentui/core/testing";
import { mountApp } from "../src/shell/start.ts";
import { key } from "../src/shell/keys.ts";
import { fakeServer, json, sseResponse } from "./fake.ts";
import { fixture, TOKEN } from "./harness.ts";

const S = ["SECRETA1", "SECRETB2", "SECRETC3", "SECRETD4"];
const URL1 = `https://user:${S[0]}@rpc.example/key/${S[1]}?apikey=${S[2]}`;
const ERR = { error: `upstream ${URL1} said no; token ${"ab".repeat(32)} Bearer ${S[3]}`, code: "upstream_failed", hint: `check u:${S[3]}@h` };

/** salt puts URL1 into every string that holds a URL or free text the server writes. */
function salt(v: any): any {
  if (Array.isArray(v)) return v.map(salt);
  if (v && typeof v === "object") return Object.fromEntries(Object.entries(v).map(([k, x]) => [k, salt(x)]));
  if (typeof v === "string" && /^https?:|^ssh |found|disk tight|stale|unreachable|alerts/.test(v)) return `${v} ${URL1}`;
  return v;
}

test("no screen shows a fixture secret or the session token", async () => {
  const ok = (name: string) => () => json(salt(fixture(name)));
  const fake = fakeServer({
    "GET /api/health": () => json({ ok: true }),
    "GET /api/version": () => json({ version: "0.0.1" }),
    "GET /api/vault": ok("vaultview"),
    "GET /api/ui-prefs": () => json({ ...fixture("uiprefs"), drawer: "open" }),
    "GET /api/jobs/running": ok("runningjobs"),
    "GET /api/fleet/stream": () => sseResponse([`data: ${JSON.stringify(salt(fixture("fleet")))}\n\n`], true),
    "GET /api/jobs/stream": () => sseResponse([`data: ${JSON.stringify(salt(fixture("jobsinbox")))}\n\n`], true),
    "GET /api/targets": ok("targets"),
    "GET /api/controller": ok("controllerview"),
    "GET /api/settings": ok("settings"),
    "GET /api/catalog": ok("catalog"),
    "GET /api/targets/alpha/endpoints": ok("endpoints"),
    "GET /api/targets/alpha/firewall": ok("firewall"),
    "GET /api/fleet/alpha/disk": ok("diskview"),
    "GET /api/targets/alpha/modules": ok("modulesview"),
    "GET /api/gateways": () => json({ gateways: salt(fixture("gateways")) }),
    "GET /api/vpns": () => json([]),
    "GET /api/vpn-servers": () => json([{ id: "s1", targetId: "alpha", interface: "wg0", address: "10.0.0.1/24", listenPort: 1, publicKey: "P", endpoint: URL1, peers: [] }]),
    "GET /api/update": () => json({ latest: "v1", notes: URL1 }),
  });
  // Every other route answers with an error that carries secrets.
  const t = await createTestRenderer({ width: 140, height: 40 });
  const app = mountApp(t.renderer, {}, "0.0.1", { streamOpts: { delay: () => 5 }, runningEveryMs: 60_000, platform: "linux", run: async () => 0 });
  const orig = fake.routes;
  fake.routes = new Proxy(orig, { get: (o, k: string) => o[k] ?? (() => json(ERR, 502)) });
  app.attach({ socket: fake.socket, token: TOKEN, system: false });
  const frames: string[] = [];
  const snap = async () => {
    app.render();
    await t.renderOnce();
    frames.push(t.captureCharFrame());
  };
  await Bun.sleep(100);
  for (const id of ["fleet", "hosts", "modules", "services", "gateways", "vpn", "jobs", "vault", "keys", "settings", "setup", "connect"]) {
    app.go(id, id === "connect" ? { stay: true } : undefined);
    await Bun.sleep(40);
    await snap();
    for (const k of ["return", "1", "3", "4", "c", "g", "T", "u", "D", "d", "a", "A", "v", "s", "K", "t", "l", "h", "x"]) {
      if (app.modal) app.handleKey(key("escape"));
      app.handleKey(key(k));
      await Bun.sleep(5);
    }
    await Bun.sleep(40);
    if (app.modal) app.handleKey(key("escape"));
    await snap();
  }
  app.go("node", "alpha");
  for (const k of ["1", "2", "3", "4", "5"]) {
    app.handleKey(key(k));
    await Bun.sleep(30);
    await snap();
  }
  const all = [...frames, ...app.activity.all(), app.flash].join("\n");
  if (process.env.LEAKDBG) for (const l of all.split("\n")) if ([...S, TOKEN, "ab".repeat(32)].some((s) => l.includes(s))) console.log("LEAK", l);
  for (const s of [...S, TOKEN, "ab".repeat(32)]) expect([s, all.includes(s)]).toEqual([s, false]);
  expect(app.activity.all().some((l) => l.startsWith("FAIL") || l.includes(" FAIL "))).toBe(true); // errors did arrive
  app.quit();
  fake.stop();
}, 30_000);
