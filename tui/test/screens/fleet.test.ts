import { afterEach, expect, test } from "bun:test";
import { FleetScreen, fleetRows } from "../../src/screens/fleet.ts";
import { dummy, fixture, mount, type Mounted } from "../harness.ts";
import { sseResponse } from "../fake.ts";

let m: Mounted | undefined;
afterEach(async () => {
  await m?.close();
  m = undefined;
});

test("one row per box with name gate sync peers disk state", () => {
  const f = fixture("fleet");
  f.rows[0].stale.status = false;
  const lines = fleetRows(f, false);
  expect(lines[0]).toMatch(/^name\s+gate\s+sync\s+peers\s+disk\s+state$/);
  expect(lines[1]).toMatch(/^alpha\s+0\.6\.0\s+syncing -3\s+50\/80\s+1314\/2173 GB\s+warn$/);
});

test("missing fields show n/a, never 0", () => {
  const row = { ...fixture("fleet").rows[0], agent: undefined, status: null, disk: null, stale: { status: false, disk: false, firewall: false } };
  const line = fleetRows({ ...fixture("fleet"), rows: [row] }, false)[1]!;
  expect(line).toMatch(/^alpha\s+n\/a\s+n\/a\s+n\/a\s+n\/a\s+warn$/);
});

test("a dropped stream marks the values ~", () => {
  const f = fixture("fleet");
  f.rows[0].stale.status = false;
  expect(fleetRows(f, true)[1]).toContain("~syncing");
});

test("a fleet/stream update replaces the rows, and Enter opens node detail", async () => {
  let n = 0;
  const two = fixture("fleet");
  two.rows.push({ ...two.rows[0], id: "beta" });
  m = await mount((app) => [new FleetScreen(app), { ...dummy("node"), parent: "fleet" }], {
    routes: {
      "GET /api/fleet/stream": () =>
        sseResponse([`data: ${JSON.stringify(n++ ? two : fixture("fleet"))}\n\n`, ...(n > 1 ? [] : [])], n > 1),
    },
  });
  await m.waitFor("alpha");
  const f = await m.waitFor("beta"); // the stream ended once, the reconnect sent two rows
  expect(f).toContain("alpha");
  let opened: unknown;
  m.app.screen("node").enter = (a) => (opened = a);
  await m.press("down", "return");
  expect(opened).toBe("beta");
  expect(await m.frame()).toContain("> Fleet");
});
