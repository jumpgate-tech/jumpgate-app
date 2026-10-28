import { afterEach, expect, test } from "bun:test";
import { diskLines, endpointLines, firewallLines, NodeScreen, statusLines } from "../../src/screens/node.ts";
import { FleetScreen } from "../../src/screens/fleet.ts";
import { json, sseResponse } from "../fake.ts";
import { fixture, mount, type Mounted } from "../harness.ts";

let m: Mounted | undefined;
afterEach(async () => {
  await m?.close();
  m = undefined;
});

const evil = { unit: "geth", line: "boom \x1b]0;pwn\x07\x1b[2Jdone", at: "2026-10-28T12:00:00Z", severity: "warn" };

const routes = {
  "GET /api/fleet/alpha/status/stream": () => sseResponse([`data: ${JSON.stringify(fixture("nodestatus"))}\n\n`], true),
  "GET /api/fleet/alpha/disk": () => json(fixture("diskview")),
  "POST /api/fleet/alpha/disk/measure": () => json({ ...fixture("diskview"), freeBytes: 5e9 }),
  "GET /api/targets/alpha/endpoints": () => json({ ...fixture("endpoints"), ExecHTTP: "https://user:pw@rpc.example/key/abc" }),
  "GET /api/targets/alpha/firewall": () => json(fixture("firewall")),
  "GET /api/targets/alpha/logs/stream": () =>
    sseResponse([`event: reset\ndata: ${JSON.stringify(fixture("loghits"))}\n\n`, `data: ${JSON.stringify(evil)}\n\n`], true),
  "POST /api/targets/alpha/services/beacon/restart": () => json({ Active: true }),
};

test("sections render from fixtures", () => {
  expect(statusLines(fixture("nodestatus"), false)[0]).toBe("overall syncing");
  expect(statusLines(undefined, false)).toEqual(["status n/a"]);
  expect(statusLines(fixture("nodestatus"), true)[1]).toContain("~synced");
  expect(diskLines(fixture("diskview"))).toContain("fit       ok");
  expect(endpointLines(fixture("endpoints"))[1]).toContain("FAIL unreachable");
  expect(firewallLines(fixture("firewall"))[0]).toBe("[pass   ] SSH is key-only - PasswordAuthentication no");
});

test("node detail: tabs, masked endpoints, sanitized logs, a confirmed restart", async () => {
  m = await mount((app) => [new FleetScreen(app), new NodeScreen(app)], { routes, width: 120, height: 34 });
  await m.waitFor("alpha");
  await m.press("return");
  expect(await m.waitFor("overall syncing")).toContain("Node alpha");
  await m.press("3");
  const ep = await m.waitFor("exec    https://");
  expect(ep).toContain("https://***/key/***");
  expect(ep).not.toContain("pw@");
  await m.press("5");
  const logs = await m.waitFor("boom");
  expect(logs).toContain("peer dropped");
  expect(logs).not.toContain("pwn");
  expect(logs).not.toMatch(/\x1b/);
  await m.press("2", "d");
  await m.waitFor("free      5.0 GB");
  await m.press("b", "r");
  expect(await m.frame()).toContain("restart the beacon service on alpha?");
  await m.press("y");
  await m.waitFor(() => m!.app.activity.all().some((l) => l.endsWith("restart beacon on alpha: active")));
  expect(await m.press("escape")).toContain("> Fleet");
});

test("check Detail and Fix, the tunnel hint and errors mask credentials", () => {
  const items = [{ ID: "rpc", Title: "RPC", Why: "", Status: "fail", Detail: "found https://u:SECRET@h/x", Fix: "set url=https://h/key/SECRET" }];
  expect(firewallLines(items).join("\n")).not.toContain("SECRET");
  const e = { ...fixture("endpoints"), TunnelHint: "ssh -L 1:h:2 u:SECRET@h" };
  expect(endpointLines(e).join("\n")).not.toContain("SECRET");
});
