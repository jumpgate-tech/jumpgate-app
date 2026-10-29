import { afterEach, expect, test } from "bun:test";
import { VaultScreen } from "../../src/screens/vault.ts";
import { json } from "../fake.ts";
import { fixture, mount, recorder, typeText, type Mounted } from "../harness.ts";

let m: Mounted | undefined;
afterEach(async () => {
  await m?.close();
  m = undefined;
});

const SECRET = "relay-SECRET-token-value";

async function vault(platform = "linux", code = 0) {
  const rec = recorder(code);
  m = await mount((app) => [new VaultScreen(app)], {
    app: { run: rec.run, platform },
    routes: {
      "GET /api/settings": () => json(fixture("settings")),
      "PUT /api/settings": (_r, x) => json({ ...fixture("settings"), ...JSON.parse(x.body) }),
      "PUT /api/vault/secrets/relay-token": () => new Response(null, { status: 204 }),
    },
  });
  await m.waitFor("state  locked");
  return rec;
}

test("each shell-out runs a fixed argv with no secret on it", async () => {
  const rec = await vault();
  await m!.press("i");
  await m!.press("m", "y");
  await m!.press("b");
  await typeText(m!, "/tmp/v.bak");
  await m!.press("return");
  await m!.press("R");
  await typeText(m!, "/tmp/v.bak");
  await m!.press("return", "y");
  await m!.press("k", "y");
  await m!.press("K", "y");
  await m!.press("s", "y");
  await m!.press("S", "y");
  await Bun.sleep(20);
  expect(rec.calls).toEqual([
    { args: ["vault", "init"] },
    { args: ["vault", "migrate"] },
    { args: ["vault", "backup", "/tmp/v.bak"] },
    { args: ["vault", "restore", "/tmp/v.bak"] },
    { args: ["vault", "add-recovery"], pause: true },
    { args: ["vault", "rekdf"] },
    { args: ["vault", "seal", "--tpm2"] },
    { args: ["vault", "seal", "--allow-host-key"] },
  ]);
  expect(m!.app.activity.all().join("\n")).toContain("jumpgate vault init: exit 0");
});

test("seal is hidden off Linux", async () => {
  const rec = await vault("darwin");
  const f = await m!.press("s");
  expect(f).not.toContain("[s] seal");
  expect(rec.calls).toEqual([]);
});

test("a failing child shows its exit code", async () => {
  await vault("linux", 4);
  await m!.press("K", "y");
  await m!.waitFor(() => m!.app.activity.all().some((l) => l.endsWith("jumpgate vault rekdf: exit 4 FAIL")));
});

test("idle lock minutes go through PUT /api/settings, validated to 0-1440", async () => {
  await vault();
  await m!.press("t");
  await typeText(m!, "9999");
  await m!.press("return");
  expect(m!.fake.requests.some((r) => r.method === "PUT" && r.path === "/api/settings")).toBe(false);
  expect(await m!.frame()).toContain("0 to 1440");
  await m!.press("t");
  await m!.press("backspace", "backspace");
  await typeText(m!, "45");
  await m!.press("return");
  await m!.waitFor(() => m!.fake.requests.some((r) => r.method === "PUT" && r.path === "/api/settings"));
  expect(JSON.parse(m!.fake.requests.find((r) => r.method === "PUT")!.body)).toEqual({ vaultIdleLockMinutes: 45 });
});

test("put secret: masked, sent base64, zeroed, never drawn", async () => {
  await vault();
  await m!.press("p");
  await typeText(m!, "relay-token");
  await m!.press("return");
  const f = await typeText(m!, SECRET);
  expect(f).not.toContain("SECRET");
  await m!.press("return");
  await m!.waitFor(() => m!.fake.requests.some((r) => r.path === "/api/vault/secrets/relay-token"));
  const req = m!.fake.requests.find((r) => r.path === "/api/vault/secrets/relay-token")!;
  expect(Buffer.from(JSON.parse(req.body).value, "base64").toString()).toBe(SECRET);
  const after = await m!.frame();
  expect(after).not.toContain("SECRET");
  expect(m!.app.activity.all().join("\n")).not.toContain("SECRET");
});

test("put secret refuses an unknown name", async () => {
  await vault();
  await m!.press("p");
  await typeText(m!, "other");
  await m!.press("return");
  expect(await m!.frame()).toContain("relay-token or admin-token");
  expect(m!.app.modal).toBeNull();
});
