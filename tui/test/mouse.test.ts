// Mouse: clicks and the wheel, sent through OpenTUI's mock mouse on the test
// renderer, the way a terminal reports them.
import { afterEach, expect, test } from "bun:test";
import { FleetScreen } from "../src/screens/fleet.ts";
import { HostsScreen } from "../src/screens/hosts.ts";
import { NodeScreen } from "../src/screens/node.ts";
import { VaultScreen } from "../src/screens/vault.ts";
import { buttonAt, buttonKey, inList, statusTarget } from "../src/shell/mouse.ts";
import { json, sseResponse } from "./fake.ts";
import { dummy, fixture, mount, recorder, type Mounted } from "./harness.ts";

let m: Mounted | undefined;
afterEach(async () => {
  await m?.close();
  m = undefined;
});

/** at finds text on the frame: the cell of its first character. */
function at(frame: string, text: string, nth = 0): { x: number; y: number } {
  const rows = frame.split("\n");
  let n = 0;
  for (let y = 0; y < rows.length; y++) {
    let from = 0;
    for (;;) {
      const x = rows[y]!.indexOf(text, from);
      if (x < 0) break;
      if (n++ === nth) return { x: [...rows[y]!.slice(0, x)].length, y };
      from = x + 1;
    }
  }
  throw new Error(`${text} is not on the frame:\n${frame}`);
}

async function click(text: string, o: { double?: boolean; dx?: number } = {}) {
  const p = at(await m!.frame(), text);
  if (o.double) await m!.t.mockMouse.doubleClick(p.x + (o.dx ?? 0), p.y);
  else await m!.t.mockMouse.click(p.x + (o.dx ?? 0), p.y);
  await Bun.sleep(5);
  return m!.frame();
}

const twoBoxes = () => {
  const f = fixture("fleet");
  f.rows.push({ ...f.rows[0], id: "beta" });
  return f;
};

const nodeRoutes: Record<string, any> = {
  "GET /api/fleet/stream": () => sseResponse([`data: ${JSON.stringify(twoBoxes())}\n\n`], true),
};
for (const id of ["alpha", "beta"])
  Object.assign(nodeRoutes, {
    [`GET /api/fleet/${id}/status/stream`]: () => sseResponse([`data: ${JSON.stringify(fixture("nodestatus"))}\n\n`], true),
    [`GET /api/fleet/${id}/disk`]: () => json(fixture("diskview")),
    [`GET /api/targets/${id}/endpoints`]: () => json(fixture("endpoints")),
    [`GET /api/targets/${id}/firewall`]: () => json(fixture("firewall")),
    [`GET /api/targets/${id}/logs/stream`]: () =>
      sseResponse([`event: reset\ndata: ${JSON.stringify(Array.from({ length: 80 }, (_, i) => ({ ...fixture("loghits")[0], line: `log ${i}` })))}\n\n`], true),
  });

test("buttonAt finds the [key] under a column, label included", () => {
  const l = "[u] unlock   [l] lock   [esc] back";
  expect(buttonAt(l, 0)?.name).toBe("u");
  expect(buttonAt(l, 6)?.name).toBe("u");
  expect(buttonAt(l, 11)).toBeNull(); // the gap between buttons
  expect(buttonAt(l, 14)?.name).toBe("l");
  expect(buttonAt(l, 26)?.name).toBe("escape");
  const big = buttonAt("[R] restore", 1)!;
  expect([big.name, big.seq, big.shift]).toEqual(["r", "R", true]);
  expect(buttonKey("1-3")).toBeNull();
  expect(buttonKey("pass   ")).toBeNull();
  expect(buttonKey("any key")?.name).toBe("escape");
});

test("inList joins only the run of list lines around the selection", () => {
  const lines = ["boxes:", "> a", "  b", "", "  c"];
  expect(inList(lines, 1, 2)).toBe(true);
  expect(inList(lines, 1, 0)).toBe(false);
  expect(inList(lines, 1, 4)).toBe(false);
});

test("statusTarget maps the strip's parts to screens", () => {
  const s = "server 1 | vault locked | boxes 1 | ! 1 decisions | setup 3/6";
  expect(statusTarget(s, 12)).toBe("vault");
  expect(statusTarget(s, 27)).toBe("fleet");
  expect(statusTarget(s, 40)).toBe("jobs");
  expect(statusTarget(s, s.length - 2)).toBe("setup");
  expect(statusTarget(s, 2)).toBeNull();
});

test("clicking a nav item opens it and focuses the nav", async () => {
  m = await mount(() => [dummy("Fleet"), dummy("Jobs"), dummy("Vault")]);
  const f = await click("  Vault");
  expect(f).toContain("> Vault");
  expect(f).toContain("Vault body");
  expect(m.app.focus).toBe("nav");
});

test("the wheel over the nav moves it", async () => {
  m = await mount(() => [dummy("Fleet"), dummy("Jobs")]);
  const p = at(await m.frame(), "> Fleet");
  await m.t.mockMouse.scroll(p.x, p.y, "down");
  expect(await m.frame()).toContain("> Jobs");
});

test("a click selects a row, a double click opens it, and the main pane takes focus", async () => {
  m = await mount((app) => [new FleetScreen(app), new NodeScreen(app)], { routes: nodeRoutes });
  await m.waitFor("beta");
  m.app.setFocus("nav");
  let f = await click("beta");
  expect(f).toMatch(/> beta/);
  expect(m.app.current.id).toBe("fleet");
  expect(m.app.focus).toBe("main");
  f = await click("alpha", { double: true });
  expect(m.app.current.id).toBe("node");
  expect(await m.waitFor("overall syncing")).toContain("Fleet › alpha › exec");
});

test("enter opens the clicked row too", async () => {
  m = await mount((app) => [new FleetScreen(app), new NodeScreen(app)], { routes: nodeRoutes });
  await m.waitFor("beta");
  await click("beta");
  await m.press("return");
  expect(await m.waitFor("overall syncing")).toContain("Fleet › beta › exec");
});

test("the wheel moves a list's selection and scrolls node logs", async () => {
  m = await mount((app) => [new FleetScreen(app), new NodeScreen(app)], { routes: nodeRoutes, height: 24, width: 130 });
  await m.waitFor("beta");
  const p = at(await m.frame(), "alpha");
  await m.t.mockMouse.scroll(p.x, p.y, "down");
  expect(await m.frame()).toMatch(/> beta/);
  await m.press("return");
  await m.waitFor("overall syncing");
  await click("5 logs");
  let f = await m.waitFor("log 79");
  const q = at(f, "log 79");
  for (let i = 0; i < 5; i++) await m.t.mockMouse.scroll(q.x, q.y, "up");
  f = await m.frame();
  expect(f).not.toContain("log 79");
  expect(f).toContain("log 74");
});

test("clicking a node tab opens that section", async () => {
  m = await mount((app) => [new FleetScreen(app), new NodeScreen(app)], { routes: nodeRoutes });
  await m.waitFor("beta");
  await m.press("return");
  await m.waitFor("overall syncing");
  const f = await click("3 endpoints");
  expect(f).toContain("[3 endpoints]");
  expect(f).toContain("exec    ");
});

test("the wheel over the activity pane scrolls it", async () => {
  m = await mount(() => [dummy("Fleet")], { height: 24 });
  await m.waitFor(() => m!.app.state.fleet !== undefined);
  for (let i = 0; i < 60; i++) m.app.log(`line ${i}`);
  const p = at(await m.frame(), "line 59");
  await m.t.mockMouse.scroll(p.x, p.y, "up");
  const f = await m.frame();
  expect(f).not.toContain("line 59");
  expect(f).toContain("newer below");
});

test("clicking a [key] button runs its action; modal buttons answer it", async () => {
  m = await mount((app) => [new VaultScreen(app)], {
    routes: { "GET /api/settings": () => json(fixture("settings")), "POST /api/vault/lock": () => json({ ...fixture("vaultview"), state: "locked" }) },
  });
  m.app.go("vault");
  await m.waitFor("slots");
  let f = await click("[l] lock");
  expect(f).toContain("Lock the vault?");
  f = await click("[n] no");
  expect(f).not.toContain("Lock the vault?");
  expect(m.fake.requests.some((r) => r.path === "/api/vault/lock")).toBe(false);
  await click("[l] lock");
  await click("[y] yes");
  await m.waitFor(() => m!.fake.requests.some((r) => r.path === "/api/vault/lock"));
});

test("a click never confirms a typed confirmation; it only cancels", async () => {
  m = await mount((app) => [new HostsScreen(app)], { routes: { "GET /api/targets": () => json(fixture("targets")) } });
  m.app.go("hosts");
  await m.waitFor("alpha");
  const p = m.app.confirmTyped("Remove alpha?", "alpha");
  let done = false;
  void p.then(() => (done = true));
  let f = await m.frame();
  // Clicking the prompt, the typed line and the word itself does nothing.
  for (const s of ["Type alpha", "> _", "Remove alpha?"]) f = await click(s);
  expect(done).toBe(false);
  expect(f).toContain("Type alpha to confirm");
  await click("[esc] cancel");
  expect(await p).toBe(false);
});

test("a click on the drawer or the folded drawer toggles it", async () => {
  m = await mount(() => [dummy("Fleet")]);
  await m.waitFor("[1 running]");
  let f = await click("[1 running]");
  expect(f).toContain("alpha/j1");
  expect(m.app.state.drawer).toBe("open");
  f = await click("alpha/j1");
  expect(m.app.state.drawer).toBe("min");
  expect(f).toContain("[1 running]");
});

test("a click on the status strip opens what it names", async () => {
  m = await mount(() => [dummy("fleet"), dummy("vault"), dummy("jobs")]);
  await m.waitFor("vault locked");
  await click("vault locked");
  expect(m.app.current.id).toBe("vault");
  await click("! 1 decisions");
  expect(m.app.current.id).toBe("jobs");
});

test("clicks are dropped while a modal is still too new to answer", async () => {
  m = await mount(() => [dummy("Fleet")], { app: { modalGuardMs: 10_000 } });
  const p = m.app.confirm("Stop?");
  await click("[y] yes");
  expect(m.app.modal).not.toBeNull();
  m.app.modal!.key({ name: "n", seq: "n", ctrl: false, shift: false, meta: false });
  expect(await p).toBe(false);
});

test("the runner records a click-started CLI action like a key", async () => {
  const r = recorder();
  m = await mount((app) => [new VaultScreen(app)], { app: { run: r.run }, routes: { "GET /api/settings": () => json(fixture("settings")) } });
  m.app.go("vault");
  await m.waitFor("slots");
  await click("[i] init");
  await m.waitFor(() => r.calls.length > 0);
  expect(r.calls[0]!.args).toEqual(["vault", "init"]);
});
