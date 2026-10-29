// Where am I: the active nav item, the selected row, the focused pane, the
// breadcrumb, and hints for the focused pane only.
import { afterEach, expect, test } from "bun:test";
import { TextAttributes } from "@opentui/core";
import { FleetScreen } from "../src/screens/fleet.ts";
import { NodeScreen } from "../src/screens/node.ts";
import { styled } from "../src/shell/style.ts";
import { json, sseResponse } from "./fake.ts";
import { dummy, fixture, mount, type Mounted } from "./harness.ts";

let m: Mounted | undefined;
afterEach(async () => {
  await m?.close();
  m = undefined;
});

/** reversedRows lists the screen rows (trimmed text) that have reverse-video cells. */
function reversedRows(mm: Mounted): string[] {
  const f = mm.t.captureSpans();
  return f.lines
    .map((l) => l.spans.filter((s) => s.attributes & TextAttributes.INVERSE).map((s) => s.text).join("").trim())
    .filter(Boolean);
}

/** row returns the frame line that contains s. */
const row = (frame: string, s: string) => frame.split("\n").find((l) => l.includes(s)) ?? "";

const two = () => {
  const f = fixture("fleet");
  f.rows.push({ ...f.rows[0], id: "beta" });
  return f;
};

test("styled reverses only the selected lines, padded to the width", () => {
  const st = styled(["> one", "  two"], 10);
  const chunks = st.chunks.filter((c) => c.text !== "\n");
  expect(chunks[0]!.text).toBe("> one     ");
  expect(chunks[0]!.attributes! & TextAttributes.INVERSE).toBeTruthy();
  expect(chunks[1]!.attributes ?? 0).toBe(0);
});

test("the active nav item and the selected row are marked and drawn in reverse", async () => {
  m = await mount((app) => [new FleetScreen(app), dummy("Jobs")], {
    routes: { "GET /api/fleet/stream": () => sseResponse([`data: ${JSON.stringify(two())}\n\n`], true) },
  });
  let f = await m.waitFor("beta");
  expect(row(f, "> Fleet")).toBeTruthy();
  expect(row(f, "> alpha")).toBeTruthy();
  let rev = reversedRows(m);
  expect(rev.some((r) => r.startsWith("> Fleet"))).toBe(true);
  expect(rev.some((r) => r.startsWith("> alpha"))).toBe(true);
  expect(rev.some((r) => r.includes("beta"))).toBe(false);
  f = await m.press("down");
  rev = reversedRows(m);
  expect(row(f, "> beta")).toBeTruthy();
  expect(rev.some((r) => r.startsWith("> beta"))).toBe(true);
  expect(rev.some((r) => r.includes("alpha"))).toBe(false);
});

test("the focused pane has a double border; tab moves it", async () => {
  m = await mount(() => [dummy("Fleet"), dummy("Jobs")]);
  let f = await m.frame();
  // main is focused: its border is double, the nav's single.
  expect(row(f, "Fleet body")).toMatch(/║Fleet body/);
  expect(row(f, "Jobs")).toMatch(/│  Jobs/);
  f = await m.press("tab");
  expect(m.app.focus).toBe("activity");
  expect(f.split("\n")[1]).toMatch(/╔═activity/);
  f = await m.press("tab");
  expect(m.app.focus).toBe("nav");
  expect(row(f, "Jobs")).toMatch(/║  Jobs/);
  expect(row(f, "Fleet body")).toMatch(/│Fleet body/);
});

test("the hints bar shows the focused pane's keys only", async () => {
  m = await mount((app) => [new FleetScreen(app)]);
  await m.waitFor("alpha");
  let f = await m.frame();
  const last = () => f.trimEnd().split("\n").at(-1)!;
  expect(last()).toContain("up/down select  enter open");
  f = await m.press("tab");
  expect(last()).toContain("up/down scroll");
  expect(last()).not.toContain("enter open");
  f = await m.press("tab");
  expect(last()).toContain("up/down screen");
  expect(last()).not.toContain("scroll");
});

test("the breadcrumb is the main pane's title: Fleet › box › service", async () => {
  const routes = {
    "GET /api/fleet/alpha/status/stream": () => sseResponse([`data: ${JSON.stringify(fixture("nodestatus"))}\n\n`], true),
    "GET /api/fleet/alpha/disk": () => json(fixture("diskview")),
    "GET /api/targets/alpha/endpoints": () => json(fixture("endpoints")),
    "GET /api/targets/alpha/firewall": () => json(fixture("firewall")),
    "GET /api/targets/alpha/logs/stream": () => sseResponse([`event: reset\ndata: ${JSON.stringify(fixture("loghits"))}\n\n`], true),
  };
  m = await mount((app) => [new FleetScreen(app), new NodeScreen(app)], { routes });
  await m.waitFor("alpha");
  expect(await m.frame()).toContain("╔═Fleet═");
  await m.press("return");
  let f = await m.waitFor("overall syncing");
  expect(f).toContain("Fleet › alpha › exec");
  expect(row(f, "> Fleet")).toBeTruthy(); // the nav keeps the parent active
  f = await m.press("b");
  expect(f).toContain("Fleet › alpha › beacon");
  expect(f).not.toContain("service:");
});

test("the activity pane scrolls when focused", async () => {
  m = await mount(() => [dummy("Fleet")], { height: 24 });
  await m.waitFor(() => m!.app.state.fleet !== undefined);
  for (let i = 0; i < 60; i++) m.app.log(`line ${i}`);
  let f = await m.frame();
  expect(f).toContain("line 59");
  await m.press("tab");
  f = await m.press("up", "up", "up");
  expect(f).not.toContain("line 59");
  expect(f).toContain("activity (3 newer below)");
  f = await m.press("down", "down", "down");
  expect(f).toContain("line 59");
});
