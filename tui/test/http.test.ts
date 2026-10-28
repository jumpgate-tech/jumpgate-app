import { afterEach, expect, test } from "bun:test";
import { ApiError, request } from "../src/api/http.ts";
import { caught, fakeServer, json, type Fake } from "./fake.ts";

const TOKEN = "a".repeat(64);
let f: Fake;
afterEach(() => f?.stop());

test("Bearer is sent only when there is a token", async () => {
  f = fakeServer({ "GET /api/vault": () => json({ state: "locked", slots: [] }) });
  await request({ socket: f.socket, token: TOKEN, system: false }, "GET", "/api/vault");
  await request({ socket: f.socket, token: null, system: true }, "GET", "/api/vault");
  expect(f.requests[0]!.headers.get("authorization")).toBe(`Bearer ${TOKEN}`);
  expect(f.requests[1]!.headers.get("authorization")).toBeNull();
});

test("a JSON body is sent with its content type", async () => {
  f = fakeServer({ "PUT /api/ui-prefs": (_r, rec) => json(JSON.parse(rec.body)) });
  const out = await request<{ drawer: string }>({ socket: f.socket, token: TOKEN, system: false }, "PUT", "/api/ui-prefs", { drawer: "min" });
  expect(out.drawer).toBe("min");
  expect(f.requests[0]!.headers.get("content-type")).toBe("application/json");
});

test("an error carries the server's code, message and hint, never the token", async () => {
  f = fakeServer({ "POST /api/vault/unlock": () => json({ error: "wrong passphrase", code: "bad_passphrase", hint: "try again" }, 403) });
  const err = await caught(request({ socket: f.socket, token: TOKEN, system: false }, "POST", "/api/vault/unlock", {}));
  expect(err).toBeInstanceOf(ApiError);
  expect(err.status).toBe(403);
  expect(err.code).toBe("bad_passphrase");
  expect(err.message).toBe("wrong passphrase");
  expect(err.hint).toBe("try again");
  expect(String(err) + err.stack).not.toContain(TOKEN);
});

test("an unreachable socket is code unreachable, status 0", async () => {
  const err = await caught(request({ socket: "/tmp/does-not-exist.sock", token: TOKEN, system: false }, "GET", "/api/health"));
  expect(err).toBeInstanceOf(ApiError);
  expect(err.code).toBe("unreachable");
  expect(err.status).toBe(0);
  expect(String(err)).not.toContain(TOKEN);
});

test("an empty 2xx body resolves to undefined", async () => {
  f = fakeServer({ "DELETE /api/targets/x": () => new Response(null, { status: 204 }) });
  const out = await request({ socket: f.socket, token: TOKEN, system: false }, "DELETE", "/api/targets/x");
  expect(out).toBeUndefined();
});
