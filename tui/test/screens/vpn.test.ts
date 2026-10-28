import { afterEach, expect, test } from "bun:test";
import { mkdtempSync, writeFileSync } from "node:fs";
import { tmpdir } from "node:os";
import { join } from "node:path";
import { jsonWithSecretString } from "../../src/api/client.ts";
import { VPNScreen } from "../../src/screens/vpn.ts";
import { json } from "../fake.ts";
import { mount, typeText, type Mounted } from "../harness.ts";

let m: Mounted | undefined;
afterEach(async () => {
  await m?.close();
  m = undefined;
});

const CONF = '[Interface]\nPrivateKey = SECRETPRIVKEY=\nAddress = 10.8.0.2/32\n';
const PEERCONF = "[Interface]\nPrivateKey = PEERSECRET=\n[Peer]\nEndpoint = vpn.example:51820";
const server = { id: "s1", targetId: "alpha", interface: "wg1", address: "10.9.0.1/24", listenPort: 51820, publicKey: "PUB", endpoint: "vpn.example:51820", peers: [{ name: "phone", publicKey: "PPUB", allowedIp: "10.9.0.2/32" }] };

async function vpn() {
  m = await mount((app) => [new VPNScreen(app)], {
    routes: {
      "GET /api/vpns": () => json([{ id: "c1", provider: "bring-your-own", interface: "wg0", targetId: "", autostart: false, configured: true }]),
      "POST /api/vpns": () => json({ id: "c2" }),
      "POST /api/vpns/c1/up": () => json({ up: true }),
      "GET /api/vpns/c1/status": () => json({ id: "c1", up: true, peers: 1 }),
      "DELETE /api/vpns/c1": () => new Response(null, { status: 204 }),
      "GET /api/vpn-servers": () => json([server]),
      "POST /api/vpn-servers": () => json({ server }),
      "POST /api/vpn-servers/s1/down": () => json({}),
      "GET /api/vpn-servers/s1/status": () => json({ up: true }),
      "POST /api/vpn-servers/s1/endpoint": () => json({}),
      "POST /api/vpn-servers/s1/peers": () => json({ name: "laptop", publicKey: "LPUB", allowedIp: "10.9.0.3/32", config: PEERCONF }),
      "POST /api/vpn-servers/s1/peers/remove": () => json({}),
      "DELETE /api/vpn-servers/s1": () => new Response(null, { status: 204 }),
    },
    width: 140,
    height: 40,
  });
  m.app.go("vpn");
  await m.waitFor("c1");
}
const sent = (method: string, path: string) => m!.fake.requests.find((r) => r.method === method && r.path === path);

test("a client config is read from a file, sent once, and never drawn", async () => {
  const dir = mkdtempSync(join(tmpdir(), "jgvpn-"));
  const f = join(dir, "wg.conf");
  writeFileSync(f, CONF, { mode: 0o600 });
  await vpn();
  await m!.press("n");
  for (const v of ["c2", "", "", "", f]) {
    if (v) await typeText(m!, v);
    await m!.press("return");
  }
  await m!.waitFor(() => !!sent("POST", "/api/vpns"));
  const body = JSON.parse(sent("POST", "/api/vpns")!.body);
  expect(body).toEqual({ id: "c2", provider: "bring-your-own", interface: "wg0", targetId: "", autostart: false, config: CONF });
  expect(await m!.frame()).not.toContain("SECRET");
  expect(m!.app.activity.all().join("\n")).not.toContain("SECRET");
});

test("client up, status and delete", async () => {
  await vpn();
  await m!.press("U", "s");
  await m!.press("D", "y");
  await m!.waitFor(() => !!sent("DELETE", "/api/vpns/c1"));
  expect(sent("POST", "/api/vpns/c1/up")).toBeTruthy();
  expect(sent("GET", "/api/vpns/c1/status")).toBeTruthy();
});

test("servers: create, down, status, endpoint, remove peer, delete", async () => {
  await vpn();
  await m!.press("v");
  await m!.waitFor("phone");
  await m!.press("n");
  await typeText(m!, "s2");
  await m!.press("return");
  await typeText(m!, "alpha");
  await m!.press("return", "return");
  await m!.press("d", "s", "e");
  await typeText(m!, "vpn2.example");
  await m!.press("return");
  await m!.press("r");
  await typeText(m!, "phone");
  await m!.press("return", "y");
  await m!.press("D", "y");
  await m!.waitFor(() => !!sent("DELETE", "/api/vpn-servers/s1"));
  expect(JSON.parse(sent("POST", "/api/vpn-servers")!.body)).toEqual({ id: "s2", targetId: "alpha" });
  expect(JSON.parse(sent("POST", "/api/vpn-servers/s1/endpoint")!.body)).toEqual({ endpointHost: "vpn2.example" });
  expect(JSON.parse(sent("POST", "/api/vpn-servers/s1/peers/remove")!.body)).toEqual({ publicKey: "PPUB" });
  expect(sent("POST", "/api/vpn-servers/s1/down")).toBeTruthy();
});

test("a new peer's config is shown once, then gone from frames and activity", async () => {
  await vpn();
  await m!.press("v");
  await m!.waitFor("phone");
  await m!.press("p");
  await typeText(m!, "laptop");
  await m!.press("return", "n");
  const once = await m!.waitFor("PEERSECRET");
  expect(once).toContain("shown once");
  const after = await m!.press("x");
  expect(after).not.toContain("PEERSECRET");
  expect(m!.app.activity.all().join("\n")).not.toContain("PEERSECRET");
  expect(JSON.parse(sent("POST", "/api/vpn-servers/s1/peers")!.body)).toEqual({ name: "laptop", fullTunnel: false });
});

test("jsonWithSecretString escapes like JSON.stringify", () => {
  const v = 'a"b\\c\nd\te\u0001ü';
  const b = jsonWithSecretString({ id: "x" }, "config", new TextEncoder().encode(v));
  expect(JSON.parse(new TextDecoder().decode(b))).toEqual({ id: "x", config: v });
});
