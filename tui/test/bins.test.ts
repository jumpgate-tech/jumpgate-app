import { expect, test } from "bun:test";
import { chmodSync, mkdirSync, mkdtempSync, writeFileSync } from "node:fs";
import { tmpdir } from "node:os";
import { join } from "node:path";
import { resolveBin } from "../src/bins.ts";

function exe(dir: string, name: string): string {
  mkdirSync(dir, { recursive: true });
  const p = join(dir, name);
  writeFileSync(p, "#!/bin/sh\n");
  chmodSync(p, 0o755);
  return p;
}

const root = mkdtempSync(join(tmpdir(), "jgbins-"));
const onPath = exe(join(root, "path"), "jumpgate");
const sibling = exe(join(root, "app"), "jumpgate");
const noSib = join(root, "nothing", "bun");

test("an absolute JUMPGATE_BIN wins; a relative one is refused", () => {
  expect(resolveBin("jumpgate", "JUMPGATE_BIN", { env: { JUMPGATE_BIN: onPath, PATH: "" }, execPath: noSib })).toBe(onPath);
  expect(resolveBin("jumpgate", "JUMPGATE_BIN", { env: { JUMPGATE_BIN: "bin/jumpgate", PATH: join(root, "path") }, execPath: noSib })).toBeNull();
});

test("then a sibling of the running executable, then absolute PATH entries", () => {
  expect(resolveBin("jumpgate", "JUMPGATE_BIN", { env: { PATH: join(root, "path") }, execPath: join(root, "app", "jumpgate-tui") })).toBe(sibling);
  expect(resolveBin("jumpgate", "JUMPGATE_BIN", { env: { PATH: join(root, "path") }, execPath: noSib })).toBe(onPath);
});

test("relative PATH entries are skipped, so ./jumpgate is never found", () => {
  const cwd = process.cwd();
  process.chdir(join(root, "path"));
  try {
    expect(resolveBin("jumpgate", "JUMPGATE_BIN", { env: { PATH: ".:path" }, execPath: noSib })).toBeNull();
  } finally {
    process.chdir(cwd);
  }
});

import { createTestRenderer } from "@opentui/core/testing";
import { mountApp } from "../src/shell/start.ts";
test("the resolved paths are logged to the activity pane", async () => {
  const t = await createTestRenderer({ width: 100, height: 30 });
  const app = mountApp(t.renderer, {}, "0", { bins: { jumpgate: "/opt/jg/jumpgate", ssh: null } });
  expect(app.activity.all().some((l) => l.endsWith("jumpgate: /opt/jg/jumpgate"))).toBe(true);
  expect(app.activity.all().some((l) => l.includes("ssh: not found"))).toBe(true);
  app.quit();
});
