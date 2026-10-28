// Opt-in end-to-end run against a real `jumpgate serve`: set
// JUMPGATE_LIVE_HOME to its ~/.jumpgate and JUMPGATE_LIVE_PASSFILE to a file
// holding the vault passphrase. Frames go to JUMPGATE_LIVE_FRAMES if set.
import { expect, test } from "bun:test";
import { appendFileSync, readFileSync } from "node:fs";
import { createTestRenderer } from "@opentui/core/testing";
import { mountApp } from "../src/shell/start.ts";
import { key } from "../src/shell/keys.ts";

const home = process.env.JUMPGATE_LIVE_HOME;
const passFile = process.env.JUMPGATE_LIVE_PASSFILE;

test.skipIf(!home || !passFile)("live: connect, unlock, fleet, jobs", async () => {
  process.env.JUMPGATE_HOME = home;
  process.env.JUMPGATE_SYSTEM_DIR = "/nonexistent-jumpgate-system";
  const t = await createTestRenderer({ width: 110, height: 30 });
  const app = mountApp(t.renderer, {}, "live");
  const frame = async () => (app.render(), await t.renderOnce(), t.captureCharFrame());
  const until = async (ok: (f: string) => boolean) => {
    for (let i = 0; i < 300; i++) {
      const f = await frame();
      if (ok(f)) return f;
      await Bun.sleep(20);
    }
    throw new Error("timeout:\n" + (await frame()));
  };
  const save = (name: string, f: string) => process.env.JUMPGATE_LIVE_FRAMES && appendFileSync(process.env.JUMPGATE_LIVE_FRAMES, `--- ${name}\n${f}\n`);
  const pass = readFileSync(passFile!, "utf8").trim();

  app.go("connect");
  const fleet = await until((f) => f.includes("> Fleet") && f.includes("vault locked"));
  save("fleet (vault locked)", fleet);

  app.go("vault");
  app.handleKey(key("u"));
  for (const c of pass) app.handleKey(key(c, { seq: c }));
  const prompt = await frame();
  save("unlock prompt", prompt);
  expect(prompt).not.toContain(pass);
  app.handleKey(key("return"));
  const vault = await until((f) => f.includes("state  unlocked"));
  save("vault unlocked", vault);
  expect(vault).not.toContain(pass);

  app.go("fleet");
  save("fleet", await until((f) => f.includes("vault unlocked") && (f.includes("No boxes") || f.includes("name "))));
  app.go("jobs");
  const jobs = await until((f) => f.includes("No jobs.") || f.includes("step "));
  save("jobs", jobs);
  expect(app.activity.all().join("\n")).not.toContain(pass);
  app.quit();
}, 30_000);
