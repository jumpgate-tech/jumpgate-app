import { afterEach, expect, test } from "bun:test";
import { KeysScreen } from "../../src/screens/keys.ts";
import { secretBody } from "../../src/api/client.ts";
import { json } from "../fake.ts";
import { fixture, mount, recorder, typeText, type Mounted } from "../harness.ts";

let m: Mounted | undefined;
afterEach(async () => {
  await m?.close();
  m = undefined;
});

const routes = {
  "GET /api/controller": () => json(fixture("controllerview")),
  "POST /api/fleet/alpha/check": () => json(fixture("agentcheck")),
  "POST /api/system/init-key": () => new Response(null, { status: 204 }),
  "POST /api/restart": () => new Response(null, { status: 202 }),
};

async function keys(system = false) {
  const rec = recorder();
  m = await mount((app) => [new KeysScreen(app)], { routes, app: { run: rec.run } });
  if (system) m.app.state.conn!.system = true;
  m.app.go("keys");
  await m.waitFor("address   0xabc");
  await m.waitFor("alpha");
  return rec;
}

test("shows the controller and test-signs on the selected box", async () => {
  await keys();
  await m!.press("t");
  const f = await m!.waitFor("signed: gate 0.6.0");
  expect(f).toContain("store     vault");
});

test("a user server creates its key and exports through the CLI", async () => {
  const rec = await keys();
  await m!.press("c");
  await m!.press("e");
  await typeText(m!, "/tmp/k.json");
  await m!.press("return", "y");
  await Bun.sleep(10);
  expect(rec.calls).toEqual([{ args: ["keys", "init"] }, { args: ["keys", "export", "--keystore-v3", "/tmp/k.json"] }]);
  expect(await m!.press("R")).toContain("Only the jumpgate service restarts");
});

test("the service creates its key over the API with a masked passphrase", async () => {
  const rec = await keys(true);
  await m!.press("c");
  const f = await typeText(m!, "SECRETpass");
  expect(f).not.toContain("SECRET");
  await m!.press("return");
  await m!.waitFor(() => m!.fake.requests.some((r) => r.path === "/api/system/init-key"));
  const body = JSON.parse(m!.fake.requests.find((r) => r.path === "/api/system/init-key")!.body);
  expect(body.store).toBe("vault");
  expect(Buffer.from(body.passphrase, "base64").toString()).toBe("SECRETpass");
  expect(rec.calls).toEqual([]);
  await m!.press("R", "y");
  await m!.waitFor(() => m!.fake.requests.some((r) => r.path === "/api/restart"));
});

test("secretBody puts extra fields first", () => {
  expect(new TextDecoder().decode(secretBody("passphrase", new TextEncoder().encode("ab"), { store: "vault" }))).toBe('{"store":"vault","passphrase":"YWI="}');
});
