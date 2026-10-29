// Ported from internal/tui/alerts_test.go.
import { expect, test } from "bun:test";
import type { Fleet, FleetRow } from "../src/api/types.ts";
import { alertLine, alerts } from "../src/shell/alerts.ts";
import { AlertsScreen } from "../src/screens/alerts.ts";
import { fixture, mount } from "./harness.ts";

const now = new Date("2026-10-28T12:00:00Z");
const seen = new Date(now.getTime() - 10_000).toISOString();
const up = (mod: (r: FleetRow) => void): Fleet => {
  const r: FleetRow = {
    id: "b", link: "agent", thisMachine: false, reachable: true, lastSeen: seen, status: null, disk: null, firewall: null, jobs: null,
    error: null, statusError: null, diskError: null, stale: { status: false, disk: false, firewall: false }, alertCount: 0,
  };
  mod(r);
  return { at: now.toISOString(), intervalSeconds: 15, rows: [r] };
};
const pe = (m: string) => ({ error: m, code: "", at: seen });

const cases: [string, Fleet, any[] | undefined, string, string][] = [
  ["down", up((r) => (r.reachable = false)), undefined, "FAIL", "down 10s"],
  ["error", up((r) => (r.error = { code: "boom", error: "it broke" })), undefined, "FAIL", "boom: it broke"],
  ["status error", up((r) => (r.statusError = pe("rpc timeout"))), undefined, "FAIL", "status: rpc timeout"],
  ["disk error", up((r) => (r.diskError = pe("du failed"))), undefined, "FAIL", "disk: du failed"],
  ["disk stale", up((r) => (r.stale.disk = true)), undefined, "warn", "disk stale"],
  ["firewall fail", up((r) => (r.firewall = { pass: 0, warn: 0, fail: 1, unknown: 0, grade: "fail" })), undefined, "FAIL", "firewall fail"],
  ["firewall warn", up((r) => (r.firewall = { pass: 0, warn: 1, fail: 0, unknown: 0, grade: "warn" })), undefined, "warn", "firewall warn"],
  ["disk short", up((r) => (r.disk = { ...fixture("diskview"), fit: "short" })), undefined, "FAIL", "disk short"],
  ["disk tight", up((r) => (r.disk = { ...fixture("diskview"), fit: "tight" })), undefined, "warn", "disk tight"],
  ["decision", up((r) => (r.jobs = { running: 1, needsDecision: 2 })), undefined, "warn", "2 needs decision"],
  ["gateway warning", up(() => {}), [{ id: "rpc-main", label: "rpc-main", status: { State: "running" }, warnings: ["upstream slow"] }], "warn", "upstream slow"],
  ["gateway state", up(() => {}), [{ id: "rpc-main", label: "rpc-main", status: { State: "stopped" } }], "FAIL", "gateway stopped"],
];
for (const [name, fl, gws, sev, text] of cases)
  test(`alert: ${name}`, () => {
    const got = alerts(fl, now, gws);
    expect(got.length).toBe(1);
    expect([got[0]!.sev, got[0]!.text]).toEqual([sev, text]);
  });

test("status stale after two intervals", () => {
  for (const [age, want] of [[31_000, 1], [29_000, 0]] as const) {
    const f = up((r) => (r.status = { ...fixture("nodestatus"), at: new Date(now.getTime() - age).toISOString(), headLag: 0, diskUsedPct: 10 }));
    expect(alerts(f, now).map((a) => a.text)).toEqual(want ? ["status stale"] : []);
  }
});

test("a gateway warning is masked and sanitized", () => {
  const got = alerts({ at: "", intervalSeconds: 15, rows: [] }, now, [{ id: "g", status: { State: "running" }, warnings: ["upstream https://u:p@h.example/key123 failed\x1b[31m"] }]);
  expect(got[0]!.text).not.toContain("u:p");
  expect(got[0]!.text).not.toContain("key123");
  expect(got[0]!.text).not.toContain("\x1b");
});

test("alerts are deterministic and FAIL comes first", () => {
  const f = fixture("fleet");
  const two = { ...f, rows: [{ ...f.rows[0], id: "zz" }, { ...f.rows[0], id: "aa", reachable: false }] };
  const a = alerts(two, now).map(alertLine);
  const b = alerts({ ...two, rows: [...two.rows].reverse() }, now).map(alertLine);
  expect(a).toEqual(b);
  expect(a[0]).toMatch(/^FAIL aa: down/);
});

test("the Alerts screen lists them; the activity counts the first ones and logs new ones once", async () => {
  const m = await mount((app) => [new AlertsScreen(app)]);
  m.app.go("alerts");
  const f = await m.waitFor("FAIL alpha:");
  expect(f).toContain("warn alpha: 1 needs decision");
  const count = (s: string) => m.app.activity.all().filter((l) => l.includes(s)).length;
  expect(count("alerts open: see Alerts")).toBe(1);
  expect(count("alert warn alpha: 1 needs decision")).toBe(0);
  const more = fixture("fleet");
  more.rows[0].jobs = { running: 0, needsDecision: 2 };
  m.app.set({ fleet: more });
  m.app.set({ fleet: more });
  expect(count("alert warn alpha: 2 needs decision")).toBe(1);
  await m.close();
});
