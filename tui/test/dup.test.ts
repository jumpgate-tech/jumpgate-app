// Each fact appears once: no pane's first line repeats its title, the hints
// bar does not repeat a [key] button the pane already shows, and the main
// pane does not repeat what the status strip says.
import { afterAll, beforeAll, expect, test } from "bun:test";
import { createTestRenderer, type TestRendererSetup } from "@opentui/core/testing";
import type { App } from "../src/shell/app.ts";
import { mountApp } from "../src/shell/start.ts";
import { fakeServer, json, sseResponse, type Fake } from "./fake.ts";
import { baseRoutes, fixture } from "./harness.ts";

const SCREENS = ["connect", "fleet", "alerts", "hosts", "modules", "services", "gateways", "vpn", "jobs", "vault", "keys", "settings", "setup"];

let t: TestRendererSetup;
let app: App;
let fake: Fake;

beforeAll(async () => {
  fake = fakeServer({
    ...baseRoutes(),
    "GET /api/targets": () => json(fixture("targets")),
    "GET /api/controller": () => json(fixture("controllerview")),
    "GET /api/settings": () => json(fixture("settings")),
    "GET /api/modules/catalog": () => json(fixture("catalog")),
    "GET /api/gateways": () => json(fixture("gateways")),
    "GET /api/fleet/alpha/status/stream": () => sseResponse([`data: ${JSON.stringify(fixture("nodestatus"))}\n\n`], true),
    "GET /api/fleet/alpha/disk": () => json(fixture("diskview")),
    "GET /api/targets/alpha/endpoints": () => json(fixture("endpoints")),
    "GET /api/targets/alpha/firewall": () => json(fixture("firewall")),
    "GET /api/targets/alpha/logs/stream": () => sseResponse([`event: reset\ndata: ${JSON.stringify(fixture("loghits"))}\n\n`], true),
  });
  t = await createTestRenderer({ width: 120, height: 40 });
  app = mountApp(t.renderer, {}, "0.0.1", { streamOpts: { delay: () => 5 }, runningEveryMs: 60_000, modalGuardMs: 0 });
  app.attach({ socket: fake.socket, token: "9".repeat(64), system: false });
  for (let i = 0; i < 100 && !(app.state.fleet && app.state.vault && app.state.inbox); i++) await Bun.sleep(10);
});

afterAll(() => {
  app.quit();
  fake.stop();
});

const norm = (s: string) => s.toLowerCase().replace(/[^a-z0-9/]+/g, " ").trim();

/** repeats: line says nothing but the title, or starts with it as a label ("jobs n/a"). */
export function repeats(line: string, title: string): boolean {
  const l = norm(line);
  const ti = norm(title);
  return ti !== "" && (l === ti || l.startsWith(`${ti} `) || l === `${ti}:`);
}

async function draw(id: string, arg?: unknown) {
  app.go(id, arg);
  await Bun.sleep(30);
  app.render();
  await t.renderOnce();
}

const firstLine = (lines: string[]) => lines.find((l) => l.trim() !== "") ?? "";

test("repeats catches a first line that equals the pane's title", () => {
  expect(repeats("Jobs", "Jobs")).toBe(true);
  expect(repeats("jobs n/a", "jobs")).toBe(true);
  expect(repeats("Vault:", "Vault")).toBe(true);
  expect(repeats("No jobs.", "Jobs")).toBe(false);
  expect(repeats("slots  passphrase", "Vault")).toBe(false);
});

for (const id of [...SCREENS, "node"]) {
  test(`${id}: the first line does not repeat the title or a breadcrumb`, async () => {
    await draw(id, id === "node" ? "alpha" : id === "connect" ? { stay: true } : undefined);
    const main = app.shown.main!.lines;
    for (const c of app.crumbs()) expect(repeats(firstLine(main), c)).toBe(false);
  });

  test(`${id}: the hints bar does not repeat a [key] button in the pane`, async () => {
    await draw(id, id === "node" ? "alpha" : id === "connect" ? { stay: true } : undefined);
    const buttons = new Set(app.shown.main!.lines.flatMap((l) => [...l.matchAll(/\[([^\]\s]+)\]/g)].map((m) => m[1]!)));
    const hintKeys = (app.current.hints?.() ?? "").split(/ {2,}/).map((p) => p.split(" ")[0]!).filter(Boolean);
    for (const k of hintKeys) expect(buttons.has(k)).toBe(false);
  });

  test(`${id}: the main pane does not repeat the status strip`, async () => {
    await draw(id, id === "node" ? "alpha" : id === "connect" ? { stay: true } : undefined);
    const main = app.shown.main!.lines.join("\n");
    expect(main).not.toMatch(/state\s+(locked|unlocked)/);
    expect(main).not.toContain("vault is locked");
    expect(main).not.toMatch(/! \d+ decisions/);
    expect(main).not.toContain("0.0.1 |");
  });
}

test("the side panes do not open with their own title", async () => {
  await draw("fleet");
  for (const [pane, title] of [["nav", "jumpgate"], ["activity", "activity"]] as const)
    expect(repeats(firstLine(app.shown[pane]!.lines), title)).toBe(false);
  app.set({ drawer: "open" });
  await draw("fleet");
  expect(repeats(firstLine(app.shown.drawer!.lines), "jobs")).toBe(false);
  app.set({ drawer: "min" });
});

test("the hints row does not repeat the status strip's decisions", async () => {
  await draw("fleet");
  const status = app.shown.status!.lines[0]!;
  const hints = app.shown.hints!.lines[0]!;
  expect(status).toContain("! 1 decisions");
  expect(hints).not.toContain("decision");
});
