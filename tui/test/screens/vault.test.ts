import { afterEach, expect, test } from "bun:test";
import { SecretModal } from "../../src/shell/confirm.ts";
import { VaultScreen } from "../../src/screens/vault.ts";
import { json } from "../fake.ts";
import { mount, type Mounted } from "../harness.ts";

const PASS = "correct horse battery";
let m: Mounted | undefined;
afterEach(async () => {
  await m?.close();
  m = undefined;
});

const typeIt = async (s: string) => {
  for (const c of s) m!.app.handleKey({ name: c === " " ? "space" : c, seq: c, ctrl: false, shift: false, meta: false });
  return m!.frame();
};

test("u unlocks: the server gets the passphrase, the frame never shows it, the bytes are zeroed", async () => {
  let unlocked = false;
  m = await mount((app) => [new VaultScreen(app)], {
    routes: {
      "GET /api/vault": () => json({ state: unlocked ? "unlocked" : "locked", slots: ["passphrase"] }),
      "POST /api/vault/unlock": () => ((unlocked = true), json({ state: "unlocked", slots: ["passphrase"] })),
    },
  });
  await m.waitFor("state  locked");
  let handed: Uint8Array | null = null;
  const real = m.app.askSecret.bind(m.app);
  m.app.askSecret = async (p) => (handed = await real(p));
  await m.press("u");
  const modal = m.app.modal as SecretModal;
  const typed = await typeIt(PASS);
  expect(typed).toContain("*".repeat(PASS.length));
  expect(typed).not.toContain("horse");
  await m.press("return");
  const f = await m.waitFor("state  unlocked");
  const req = m.fake.requests.find((r) => r.path === "/api/vault/unlock")!;
  expect(Buffer.from(JSON.parse(req.body).passphrase, "base64").toString()).toBe(PASS);
  expect(modal.value).toBe("");
  expect([...handed!].every((b) => b === 0)).toBe(true);
  expect(f).not.toContain("horse");
  expect(m.app.activity.all().join("\n")).not.toContain("horse");
});

test("a wrong passphrase shows the server's message and the backoff", async () => {
  m = await mount((app) => [new VaultScreen(app)], {
    routes: {
      "POST /api/vault/unlock": () =>
        new Response(JSON.stringify({ error: "wrong passphrase", code: "bad_request" }), { status: 403, headers: { "retry-after": "4" } }),
    },
  });
  await m.waitFor("state  locked");
  await m.press("u");
  await typeIt("nope");
  await m.press("return");
  const f = await m.waitFor("wrong passphrase");
  expect(f).toContain("try again in 4 s");
  expect(f).not.toContain("nope");
});

test("escape cancels without a request", async () => {
  m = await mount((app) => [new VaultScreen(app)]);
  await m.waitFor("state  locked");
  await m.press("u");
  await typeIt("abc");
  await m.press("escape");
  expect(m.fake.requests.some((r) => r.path === "/api/vault/unlock")).toBe(false);
});

test("l locks after a confirm", async () => {
  m = await mount((app) => [new VaultScreen(app)], {
    routes: { "POST /api/vault/lock": () => json({ state: "locked", slots: [] }) },
  });
  await m.waitFor("state  locked");
  expect(await m.press("l")).toContain("Lock the vault?");
  await m.press("n");
  expect(m.fake.requests.some((r) => r.path === "/api/vault/lock")).toBe(false);
  await m.press("l", "y");
  await Bun.sleep(30);
  expect(m.fake.requests.some((r) => r.path === "/api/vault/lock")).toBe(true);
});
