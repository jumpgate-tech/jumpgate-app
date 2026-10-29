import { afterEach, expect, test } from "bun:test";
import { ServicesScreen } from "../../src/screens/services.ts";
import { json, sseResponse } from "../fake.ts";
import { fixture, mount, say, typeText, type Mounted } from "../harness.ts";

let m: Mounted | undefined;
afterEach(async () => {
  await m?.close();
  m = undefined;
});

const ok = () => json({ status: "ok", url: "https://u:SECRET@h" });
const routes: Record<string, any> = {};
for (const r of [
  "GET /api/targets/alpha/containers",
  "GET /api/targets/alpha/containers/beacon",
  "GET /api/targets/alpha/containers/beacon/config",
  "PUT /api/targets/alpha/containers/beacon/config",
  "POST /api/targets/alpha/containers/beacon/restart",
  "POST /api/targets/alpha/containers/beacon/provision",
  "POST /api/targets/alpha/containers/beacon/wipe",
  "POST /api/targets/alpha/containers/devnet/reset",
  "POST /api/targets/alpha/services/beacon/clear",
  "GET /api/targets/alpha/disk",
  "GET /api/targets/alpha/du",
  "GET /api/targets/alpha/diagnostics",
  "GET /api/targets/alpha/diagnostics/latest",
  "POST /api/targets/alpha/intent/status.read",
  "POST /api/targets/alpha/setup",
])
  routes[r] = ok;
routes["GET /api/targets/alpha/setup/stream"] = () => sseResponse([`data: {"stepId":"","done":true}\n\n`], true);
routes["GET /api/targets/alpha/monitor/stream"] = () => sseResponse([`data: {"cpu":12,"mem":40}\n\n`], true);
routes["GET /api/targets/alpha/logs"] = () => json(fixture("loghits"));
routes["POST /api/targets/alpha/explain"] = () => json({ text: "a peer left", sentExcerpt: ["peer dropped"], provider: "anthropic", local: false });

async function open() {
  m = await mount((app) => [new ServicesScreen(app)], { routes, width: 140, height: 40 });
  m.app.go("services");
  await m.waitFor("alpha");
  await m.press("return", "2");
  await m.waitFor("Services › alpha › beacon");
}

const sent = (method: string, path: string) => m!.fake.requests.find((r) => r.method === method && r.path === path);

test("every action reaches its route; replies are masked", async () => {
  await open();
  for (const k of ["c", "C", "g", "T", "u", "D", "d"]) await m!.press(k);
  await m!.press("G");
  await typeText(m!, '{"a":1}');
  await m!.press("return");
  await m!.press("a");
  await typeText(m!, "restart");
  await m!.press("return", "y");
  await m!.press("P", "y", "X", "y");
  await m!.press("I");
  await typeText(m!, "status.read");
  await m!.press("return");
  await m!.press("w");
  await typeText(m!, '{"chainId":1}');
  await m!.press("return");
  await Bun.sleep(50);
  for (const r of Object.keys(routes).filter((r) => !/wipe|clear|monitor|logs|explain|setup\/stream/.test(r))) {
    const [method, path] = r.split(" ");
    expect(sent(method!, path!)).toBeTruthy();
  }
  expect(JSON.parse(sent("PUT", "/api/targets/alpha/containers/beacon/config")!.body)).toEqual({ a: 1 });
  expect(JSON.parse(sent("POST", "/api/targets/alpha/setup")!.body)).toEqual({ chainId: 1 });
  expect(await m!.frame()).not.toContain("SECRET");
});

test("wipe and clear need the service name typed back", async () => {
  await open();
  await m!.press("W");
  await say(m!, "exec");
  expect(await m!.frame()).toContain('that is not "beacon"');
  expect(sent("POST", "/api/targets/alpha/containers/beacon/wipe")).toBeUndefined();
  await say(m!, "beacon");
  await m!.press("K");
  await say(m!, "beacon");
  await Bun.sleep(30);
  expect(JSON.parse(sent("POST", "/api/targets/alpha/containers/beacon/wipe")!.body)).toEqual({ Confirm: "beacon" });
  expect(JSON.parse(sent("POST", "/api/targets/alpha/services/beacon/clear")!.body)).toEqual({ Confirm: "beacon" });
});

test("monitor shows current numbers; explain sends log lines after a confirm", async () => {
  await open();
  await m!.press("m");
  expect(await m!.waitFor("cpu: 12")).toContain("mem: 40");
  await m!.press("E", "y");
  expect(await m!.waitFor("a peer left")).toContain("explain alpha (anthropic)");
  expect(JSON.parse(sent("POST", "/api/targets/alpha/explain")!.body)).toEqual({ lines: ["peer dropped"] });
});
