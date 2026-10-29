// Fuzz-style: mutate the fixtures at random (seeded) and feed them through
// the guards and the renderers; nothing may throw, and the whole app keeps
// drawing while its streams send junk.
import { expect, test } from "bun:test";
import { createTestRenderer } from "@opentui/core/testing";
import * as G from "../src/api/guards.ts";
import { fleetRows } from "../src/screens/fleet.ts";
import { detailLines, inboxLines } from "../src/screens/jobs.ts";
import { logLine, statusLines } from "../src/screens/node.ts";
import { alerts } from "../src/shell/alerts.ts";
import { statusLine } from "../src/shell/statusbar.ts";
import { mountApp } from "../src/shell/start.ts";
import { sanitize, sanitizeLine } from "../src/text.ts";
import { fakeServer, json, sseResponse } from "./fake.ts";
import { fixture } from "./harness.ts";

function rng(seed: number) {
  return () => ((seed = (seed * 1103515245 + 12345) & 0x7fffffff) / 0x7fffffff);
}
const JUNK = [null, undefined, 0, -1, 1e308, NaN, "", "\x1b[2J", "x".repeat(300), true, [], {}, [1, "a"], { a: { b: null } }];

function mutate(v: any, r: () => number, rate = r() * 0.06): any {
  if (r() < rate) return JUNK[Math.floor(r() * JUNK.length)];
  if (Array.isArray(v)) return v.map((x) => mutate(x, r, rate));
  if (v && typeof v === "object") {
    const o: any = {};
    for (const [k, x] of Object.entries(v)) if (r() > rate) o[k] = mutate(x, r, rate);
    return o;
  }
  return v;
}

test("sanitize coerces non-strings", () => {
  expect(sanitize(undefined)).toBe("");
  expect(sanitize(null)).toBe("");
  expect(sanitize(42)).toBe("42");
  expect(sanitizeLine({} as unknown)).toBe("[object Object]");
});

test("mutated payloads either fail their guard or render without throwing", () => {
  const r = rng(7);
  const now = new Date("2026-10-28T12:00:00Z");
  let accepted = 0;
  for (let i = 0; i < 3000; i++) {
    const f = mutate(fixture("fleet"), r);
    if (G.isFleet(f)) {
      accepted++;
      fleetRows(f, r() < 0.5);
      alerts(f, now);
      statusLine({ connected: true, myVersion: "1", fleet: f, decisions: 1 });
    }
    const st = mutate(fixture("nodestatus"), r);
    if (G.isNodeStatus(st)) statusLines(st, false);
    const inbox = mutate(fixture("jobsinbox"), r);
    if (G.isJobsInbox(inbox)) {
      inboxLines(inbox.jobs, 0, false);
      for (const j of inbox.jobs) detailLines(j, 0);
    }
    const logs = mutate(fixture("loghits"), r);
    if (G.isLogHits(logs)) logs.map(logLine);
    G.isPairEvent(mutate({ step: "s", line: "l", done: true }, r));
    G.isSetupEvent(mutate(fixture("setupevent"), r));
    G.isJobOutput(mutate(fixture("joboutput"), r));
  }
  expect(accepted).toBeGreaterThan(10); // the guards are not refusing everything
});

test("the app keeps drawing every screen while its streams send junk frames", async () => {
  const r = rng(11);
  const junk = () => Array.from({ length: 40 }, () => `data: ${JSON.stringify(mutate(fixture("fleet"), r)) ?? "null"}\n\n`).concat(["data: not json\n\n", "data: [1,2\n\n", "data: null\n\n"]);
  const fake = fakeServer({
    "GET /api/health": () => json({ ok: true }),
    "GET /api/version": () => json({ version: "0.0.1" }),
    "GET /api/vault": () => json({ state: "locked", slots: [] }),
    "GET /api/ui-prefs": () => json(fixture("uiprefs")),
    "GET /api/jobs/running": () => json(fixture("runningjobs")),
    "GET /api/fleet/stream": () => sseResponse(junk(), true),
    "GET /api/jobs/stream": () => sseResponse(Array.from({ length: 40 }, () => `data: ${JSON.stringify(mutate(fixture("jobsinbox"), r)) ?? "null"}\n\n`), true),
    "GET /api/fleet/alpha/status/stream": () => sseResponse(Array.from({ length: 20 }, () => `data: ${JSON.stringify(mutate(fixture("nodestatus"), r)) ?? "null"}\n\n`), true),
    "GET /api/targets/alpha/logs/stream": () => sseResponse([`event: reset\ndata: ${JSON.stringify(mutate(fixture("loghits"), r))}\n\n`, `data: {"unit":5}\n\n`], true),
  });
  const t = await createTestRenderer({ width: 120, height: 40 });
  const app = mountApp(t.renderer, {}, "0.0.1", { streamOpts: { delay: () => 5 }, runningEveryMs: 60_000 });
  app.attach({ socket: fake.socket, token: null, system: true });
  await Bun.sleep(150);
  for (const id of ["fleet", "alerts", "jobs", "hosts", "modules", "services", "gateways", "vpn", "vault", "keys", "settings", "setup"]) {
    app.go(id);
    app.render();
    await t.renderOnce();
  }
  app.go("node", "alpha");
  for (const k of ["1", "2", "3", "4", "5"]) {
    app.handleKey({ name: k, seq: k, ctrl: false, shift: false, meta: false });
    await Bun.sleep(20);
    app.render();
    await t.renderOnce();
  }
  expect(t.captureCharFrame()).toContain("Fleet › alpha › exec");
  app.quit();
  fake.stop();
}, 20_000);
