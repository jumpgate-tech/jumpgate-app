import { afterAll, beforeAll, expect, test } from "bun:test";
import * as api from "../src/api/client.ts";
import type { Conn } from "../src/api/http.ts";
import { ROUTES } from "../src/api/routes.ts";
import { fakeServer, json, sseResponse, type Fake } from "./fake.ts";

let f: Fake;
let c: Conn;
beforeAll(() => {
  f = fakeServer();
  c = { socket: f.socket, token: "t".repeat(64), system: false };
});
afterAll(() => f.stop());

const routeRe = (r: string) => {
  const [m, p] = r.split(" ");
  return { m, re: new RegExp("^" + p!.replace(/\{[^}]+\}/g, "[^/]+") + "$") };
};
const matchesRoute = (method: string, path: string) =>
  ROUTES.some((r) => {
    const { m, re } = routeRe(r);
    return m === method && re.test(path.split("?")[0]!);
  });

type Case = [name: string, method: string, path: string, call: () => Promise<unknown>, body?: unknown];
const cases: Case[] = [
  ["health", "GET", "/api/health", () => api.health(c)],
  ["version", "GET", "/api/version", () => api.version(c)],
  ["vault", "GET", "/api/vault", () => api.vault(c)],
  ["lock", "POST", "/api/vault/lock", () => api.lock(c)],
  ["fleet", "GET", "/api/fleet", () => api.fleet(c)],
  ["fleetOne", "GET", "/api/fleet/a%20b", () => api.fleetOne(c, "a b")],
  ["targets", "GET", "/api/targets", () => api.targets(c)],
  ["disk", "GET", "/api/fleet/alpha/disk", () => api.disk(c, "alpha")],
  ["measureDisk", "POST", "/api/fleet/alpha/disk/measure", () => api.measureDisk(c, "alpha")],
  ["endpoints", "GET", "/api/targets/alpha/endpoints", () => api.endpoints(c, "alpha")],
  ["firewall", "GET", "/api/targets/alpha/firewall", () => api.firewall(c, "alpha")],
  ["logs", "GET", "/api/targets/alpha/logs?n=50", () => api.logs(c, "alpha", 50)],
  ["serviceAction", "POST", "/api/targets/alpha/services/geth/restart", () => api.serviceAction(c, "alpha", "geth", "restart")],
  ["jobs", "GET", "/api/jobs", () => api.jobs(c)],
  ["runningJobs", "GET", "/api/jobs/running", () => api.runningJobs(c)],
  ["jobDetail", "GET", "/api/jobs/alpha/j1", () => api.jobDetail(c, "alpha", "j1")],
  ["decide", "POST", "/api/jobs/alpha/j1/decide", () => api.decide(c, "alpha", "j1", { decision: "approve", option: "go" }), { decision: "approve", option: "go" }],
  ["adopt", "POST", "/api/jobs/alpha/j1/adopt", () => api.adopt(c, "alpha", "j1")],
  ["prefs", "GET", "/api/ui-prefs", () => api.prefs(c)],
  ["savePrefs", "PUT", "/api/ui-prefs", () => api.savePrefs(c, { drawer: "min" } as never), { drawer: "min" }],
];

for (const [name, method, path, call, body] of cases) {
  test(`${name} sends ${method} ${path}`, async () => {
    const bare = path.split("?")[0]!;
    f.routes[`${method} ${bare}`] = () => json({});
    const before = f.requests.length;
    await call();
    const rec = f.requests[before]!;
    expect(rec.method).toBe(method);
    expect(rec.path).toBe(path);
    if (body !== undefined) expect(JSON.parse(rec.body)).toEqual(body);
    expect(matchesRoute(method, path)).toBe(true);
  });
}

const streams: [string, string, (s: AbortSignal) => AsyncGenerator<unknown>][] = [
  ["watchFleet", "/api/fleet/stream", (s) => api.watchFleet(c, s)],
  ["watchStatus", "/api/fleet/alpha/status/stream", (s) => api.watchStatus(c, "alpha", s)],
  ["watchLogs", "/api/targets/alpha/logs/stream?backlog=10", (s) => api.watchLogs(c, "alpha", s, 10)],
  ["watchJobs", "/api/jobs/stream", (s) => api.watchJobs(c, s)],
  ["watchJobOutput", "/api/jobs/alpha/j1/output/stream", (s) => api.watchJobOutput(c, "alpha", "j1", s)],
];
for (const [name, path, open] of streams) {
  test(`${name} opens GET ${path}`, async () => {
    f.routes[`GET ${path.split("?")[0]}`] = () => sseResponse(["data: {}\n\n"], true);
    const ac = new AbortController();
    for await (const m of open(ac.signal)) if ((m as { kind: string }).kind === "event") ac.abort();
    const rec = f.requests.findLast((r) => r.path === path);
    expect(rec?.method).toBe("GET");
    expect(matchesRoute("GET", path)).toBe(true);
  });
}

test("unlock sends the passphrase as base64 and zeroes it", async () => {
  f.routes["POST /api/vault/unlock"] = () => json({ state: "unlocked", slots: [] });
  const pass = new TextEncoder().encode("correct horse");
  const v = await api.unlock(c, pass);
  expect(v.state).toBe("unlocked");
  const rec = f.requests.findLast((r) => r.path === "/api/vault/unlock")!;
  expect(Buffer.from(JSON.parse(rec.body).passphrase, "base64").toString()).toBe("correct horse");
  expect([...pass].every((b) => b === 0)).toBe(true);
});

test("unlock zeroes the passphrase on a refusal too", async () => {
  f.routes["POST /api/vault/unlock"] = () => json({ error: "wrong passphrase", code: "bad_request" }, 403);
  const pass = new TextEncoder().encode("nope");
  await expect(api.unlock(c, pass)).rejects.toThrow("wrong passphrase");
  expect([...pass].every((b) => b === 0)).toBe(true);
});

test("secretBody matches standard base64 for every padding length", () => {
  for (const s of ["", "a", "ab", "abc", "abcd", "ünïcødé"]) {
    const body = new TextDecoder().decode(api.secretBody("passphrase", new TextEncoder().encode(s)));
    expect(body).toBe(JSON.stringify({ passphrase: Buffer.from(s).toString("base64") }));
  }
});
