import { afterEach, expect, test } from "bun:test";
import { GatewaysScreen } from "../../src/screens/gateways.ts";
import { json } from "../fake.ts";
import { mount, say, typeText, type Mounted } from "../harness.ts";

let m: Mounted | undefined;
afterEach(async () => {
  await m?.close();
  m = undefined;
});

const RAW = "jg_live_SECRETKEY0123456789";
const ok = () => json({ ok: true, url: "https://u:SECRET@h" });
const routes: Record<string, any> = {
  "GET /api/gateways": () => json({ gateways: [{ id: "g1", label: "main", status: { State: "running" }, baseUrl: "https://rpc.example/key/SECRET" }], orphans: [{ containerName: "old-gw", targetId: "alpha", mergedInto: "g1" }] }),
  "POST /api/gateways/g1/keys": () => json({ id: "k9", key: RAW }, 201),
};
for (const r of [
  "GET /api/gateways/g1", "POST /api/gateways", "DELETE /api/gateways/g1", "POST /api/gateways/g1/start", "POST /api/gateways/g1/stop",
  "POST /api/gateways/g1/restart", "PUT /api/gateways/g1/config", "POST /api/gateways/g1/provision", "POST /api/gateways/g1/wipe",
  "PUT /api/gateways/g1/modules/watch", "POST /api/gateways/g1/trust-cert", "GET /api/gateways/g1/tls/verify", "GET /api/gateways/g1/capabilities",
  "GET /api/gateways/g1/knownset/1", "GET /api/gateways/g1/analytics", "GET /api/gateways/g1/traffic", "GET /api/gateways/g1/keys",
  "POST /api/gateways/g1/keys/k9/revoke", "DELETE /api/orphans/old-gw", "GET /api/chainlist", "GET /api/chainlist/1", "GET /api/docker",
  "POST /api/docker/start", "GET /api/host",
])
  routes[r] = ok;

async function open() {
  m = await mount((app) => [new GatewaysScreen(app)], { routes, width: 160, height: 50 });
  m.app.go("gateways");
  await m.waitFor("g1");
}
const sent = (method: string, path: string) => m!.fake.requests.find((r) => r.method === method && r.path === path);

test("lists gateways and orphans with masked URLs", async () => {
  await open();
  const f = await m!.frame();
  expect(f).toContain("orphaned container old-gw on alpha");
  expect(f).not.toContain("SECRET");
});

test("every gateway action reaches its route", async () => {
  await open();
  const input = async (...vals: string[]) => {
    for (const v of vals) {
      await typeText(m!, v);
      await m!.press("return");
    }
  };
  for (const k of ["g", "S", "R", "v", "a", "A", "t", "K", "l", "d", "h"]) await m!.press(k);
  await m!.press("T");
  await say(m!, "g1");
  await m!.press("P", "y", "c", "y", "s", "y");
  await m!.press("n"); await input("g2", "alpha");
  await m!.press("G"); await input('{"networks":[]}');
  await m!.press("W"); await say(m!, "g1");
  await m!.press("M"); await input("watch"); await m!.press("y");
  await m!.press("k"); await input("1");
  await m!.press("L"); await input("1");
  await m!.press("x"); await input("k9"); await say(m!, "k9");
  await m!.press("O"); await input("old-gw"); await m!.press("y");
  await m!.press("D"); await say(m!, "g1");
  await Bun.sleep(50);
  for (const r of Object.keys(routes).filter((r) => !r.endsWith(" /api/gateways") && !r.includes("POST /api/gateways/g1/keys"))) {
    const [method, path] = r.split(" ");
    expect([r, !!sent(method!, path!)]).toEqual([r, true]);
  }
  expect(JSON.parse(sent("POST", "/api/gateways")!.body)).toEqual({ id: "g2", placement: { targetId: "alpha" } });
  expect(JSON.parse(sent("POST", "/api/gateways/g1/wipe")!.body)).toEqual({ Confirm: "g1" });
  expect(JSON.parse(sent("PUT", "/api/gateways/g1/modules/watch")!.body)).toEqual({ enabled: true });
  expect(await m!.frame()).not.toContain("SECRET");
});

test("a new customer key is shown once, then gone from frames and activity", async () => {
  await open();
  await m!.press("+");
  await typeText(m!, "acme");
  await m!.press("return");
  expect(await m!.waitFor(RAW)).toContain("shown once");
  const after = await m!.press("q");
  expect(after).not.toContain(RAW);
  expect(m!.app.activity.all().join("\n")).not.toContain(RAW);
  expect(m!.app.isClosed).toBe(false);
});
