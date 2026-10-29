import { afterEach, expect, test } from "bun:test";
import { chmodSync, mkdirSync, mkdtempSync, writeFileSync } from "node:fs";
import { tmpdir } from "node:os";
import { join } from "node:path";
import { discover, DiscoverError } from "../src/api/discover.ts";
import { fakeServer, json, type Fake } from "./fake.ts";

const TOKEN = "f".repeat(64);
let fakes: Fake[] = [];
afterEach(() => {
  fakes.forEach((f) => f.stop());
  fakes = [];
});

function fake(): Fake {
  const f = fakeServer({ "GET /api/health": () => json({ ok: true }) });
  fakes.push(f);
  return f;
}

function home(serverJson?: object, mode = 0o600): string {
  const h = mkdtempSync(join(tmpdir(), "jghome-"));
  mkdirSync(join(h, "run"), { recursive: true });
  if (serverJson) {
    const p = join(h, "run", "server.json");
    writeFileSync(p, JSON.stringify(serverJson));
    chmodSync(p, mode);
  }
  return h;
}

const noSystem = () => mkdtempSync(join(tmpdir(), "jgsys-"));

test("system.json wins and carries no token", async () => {
  const f = fake();
  const sys = noSystem();
  writeFileSync(join(sys, "system.json"), JSON.stringify({ socket: f.socket, version: "1.0", uid: process.getuid!() }));
  const h = home({ socket: "/nope", token: TOKEN });
  const c = await discover({ JUMPGATE_SYSTEM_DIR: sys, JUMPGATE_HOME: h });
  expect(c.system).toBe(true);
  expect(c.token).toBeNull();
  expect(c.socket).toBe(f.socket);
  expect(f.requests[0]!.headers.get("authorization")).toBeNull();
});

test("a user server.json gives socket and token", async () => {
  const f = fake();
  const h = home({ socket: f.socket, token: TOKEN, version: "0.6.0", httpAddr: "127.0.0.1:1" });
  const c = await discover({ JUMPGATE_SYSTEM_DIR: noSystem(), JUMPGATE_HOME: h });
  expect(c.system).toBe(false);
  expect(c.token).toBe(TOKEN);
  expect(c.version).toBe("0.6.0");
  expect(f.requests[0]!.headers.get("authorization")).toBe(`Bearer ${TOKEN}`);
});

test("a server.json readable by others is refused", async () => {
  const f = fake();
  const h = home({ socket: f.socket, token: TOKEN }, 0o644);
  const err = await discover({ JUMPGATE_SYSTEM_DIR: noSystem(), JUMPGATE_HOME: h }).catch((e: any) => e);
  expect(err).toBeInstanceOf(DiscoverError);
  expect(err.kind).toBe("insecure");
  expect(String(err)).not.toContain(TOKEN);
});

test("a missing token on a user server is an error", async () => {
  const f = fake();
  const h = home({ socket: f.socket });
  const err = await discover({ JUMPGATE_SYSTEM_DIR: noSystem(), JUMPGATE_HOME: h }).catch((e: any) => e);
  expect(err.kind).toBe("bad_file");
});

test("no server.json means not running", async () => {
  const err = await discover({ JUMPGATE_SYSTEM_DIR: noSystem(), JUMPGATE_HOME: home() }).catch((e: any) => e);
  expect(err.kind).toBe("not_running");
});

test("a dead socket means not running, and the token stays out of the message", async () => {
  const h = home({ socket: join(tmpdir(), "gone.sock"), token: TOKEN });
  const err = await discover({ JUMPGATE_SYSTEM_DIR: noSystem(), JUMPGATE_HOME: h }).catch((e: any) => e);
  expect(err.kind).toBe("not_running");
  expect(String(err) + err.stack).not.toContain(TOKEN);
});

// root reads a mode-000 dir anyway (the CI image runs as root), so the check cannot fail there.
test.skipIf(process.getuid?.() === 0)("an unreadable system dir means not an admin", async () => {
  const sys = noSystem();
  writeFileSync(join(sys, "system.json"), "{}");
  chmodSync(sys, 0o000);
  try {
    const err = await discover({ JUMPGATE_SYSTEM_DIR: sys, JUMPGATE_HOME: home() }).catch((e: any) => e);
    expect(err.kind).toBe("not_admin");
    expect(err.message).toContain("jumpgate-admin");
  } finally {
    chmodSync(sys, 0o700);
  }
});
