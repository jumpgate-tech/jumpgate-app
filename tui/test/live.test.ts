// Opt-in end-to-end run against a real `jumpgate serve`: set
// JUMPGATE_LIVE_HOME to its ~/.jumpgate and JUMPGATE_LIVE_PASSFILE to a file
// holding the vault passphrase. Frames go to JUMPGATE_LIVE_FRAMES if set.
//
// With JUMPGATE_LIVE_BOXES (two boxes' ssh host:port, comma separated, that
// root reaches with the key file JUMPGATE_LIVE_KEY) and JUMPGATE_LIVE_HOOK it also drives the boxes from
// scripts/e2e-tui.sh, which sets all of this up: pair both from Hosts, the
// fleet and node detail, restart a service, rename legacy names and answer the
// job's decisions, a module, invite, upgrade a gate, lock and unlock.
import { expect, test } from "bun:test";
import { appendFileSync, readFileSync } from "node:fs";
import { createTestRenderer } from "@opentui/core/testing";
import { mountApp } from "../src/shell/start.ts";
import { key } from "../src/shell/keys.ts";
import type { App } from "../src/shell/app.ts";

const home = process.env.JUMPGATE_LIVE_HOME;
const passFile = process.env.JUMPGATE_LIVE_PASSFILE;
const boxes = (process.env.JUMPGATE_LIVE_BOXES ?? "").split(",").filter(Boolean);
const hook = process.env.JUMPGATE_LIVE_HOOK;

type Frame = () => Promise<string>;

function driver(app: App, frame: Frame) {
  const until = async (ok: (f: string) => boolean, what = "screen", ms = 30_000) => {
    const end = Date.now() + ms;
    for (;;) {
      const f = await frame();
      if (ok(f)) return f;
      if (Date.now() > end) throw new Error(`timeout waiting for ${what}:\n${f}\nactivity:\n${app.activity.all().slice(-15).join("\n")}`);
      await Bun.sleep(50);
    }
  };
  // logged waits for an activity line (its time stamp cut off) newer than the
  // last one it returned, and fails on a FAIL line in between.
  let seen = 0;
  const logged = async (re: RegExp, ms = 60_000) => {
    const end = Date.now() + ms;
    for (;;) {
      const all = app.activity.all().slice(seen).map((l) => l.slice(9));
      const at = all.findIndex((l) => re.test(l));
      if (at >= 0) {
        seen += at + 1;
        return all[at]!;
      }
      const bad = all.find((l) => l.startsWith("FAIL"));
      if (bad) throw new Error(`activity failed while waiting for ${re}: ${bad}`);
      if (Date.now() > end) throw new Error(`timeout waiting for ${re}; activity:\n${all.slice(-15).join("\n")}`);
      await Bun.sleep(50);
    }
  };
  // Keys typed within 150 ms of a modal opening are flushed, so wait for it.
  const modal = async () => {
    for (let i = 0; i < 400 && !app.modal; i++) await Bun.sleep(25);
    if (!app.modal) throw new Error("no modal opened:\n" + (await frame()));
    await Bun.sleep(200);
  };
  const type = (s: string) => {
    for (const c of s) app.handleKey(key(c === " " ? "space" : c, { seq: c }));
  };
  const press = (name: string, seq?: string) => app.handleKey(key(name, seq === undefined ? {} : { seq }));
  const answer = async (s: string) => {
    await modal();
    type(s);
    press("return");
    await frame();
  };
  const mark = () => void (seen = app.activity.all().length);
  return { until, logged, mark, modal, type, press, answer };
}

test.skipIf(!home || !passFile)("live: connect, unlock, fleet, jobs", async () => {
  process.env.JUMPGATE_HOME = home;
  process.env.JUMPGATE_SYSTEM_DIR = "/nonexistent-jumpgate-system";
  const t = await createTestRenderer({ width: 120, height: 34 });
  const app = mountApp(t.renderer, {}, "live");
  const frame = async () => (app.render(), await t.renderOnce(), t.captureCharFrame());
  const d = driver(app, frame);
  const save = (name: string, f: string) => process.env.JUMPGATE_LIVE_FRAMES && appendFileSync(process.env.JUMPGATE_LIVE_FRAMES, `--- ${name}\n${f}\n`);
  const pass = readFileSync(passFile!, "utf8").trim();
  const noSecret = (f: string) => expect(f).not.toContain(pass);

  app.go("connect");
  const fleet = await d.until((f) => f.includes("> Fleet") && f.includes("vault locked"), "fleet, locked");
  save("fleet (vault locked)", fleet);

  const unlock = async () => {
    app.go("vault");
    d.press("u");
    await d.modal();
    d.type(pass);
    const prompt = await frame();
    save("unlock prompt", prompt);
    noSecret(prompt);
    d.press("return");
    const vault = await d.until((f) => f.includes("state  unlocked"), "unlocked");
    noSecret(vault);
    return vault;
  };
  save("vault unlocked", await unlock());

  if (boxes.length >= 2 && hook) {
    d.mark();
    // --- pair both boxes from Hosts: probe, typed host-key confirm, add, pair over ssh.
    for (const [i, addr] of boxes.entries()) {
      const name = `box${i + 1}`;
      app.go("hosts");
      await d.until((f) => f.includes("[a] add over SSH"), "hosts");
      d.press("a");
      await d.answer(name);
      await d.answer(`root@${addr}`);
      await d.answer("");
      await d.answer(process.env.JUMPGATE_LIVE_KEY ?? "");
      await d.modal();
      const hk = await frame();
      save(`hosts add ${name}: host key`, hk);
      expect(hk).toContain("Type yes to confirm");
      d.type("yes");
      d.press("return");
      await d.modal(); // the installed-address question
      d.press("return");
      await d.logged(new RegExp(`^paired ${name}: gate 0x`), 180_000);
    }
    app.go("hosts");
    save("hosts", await d.until((f) => f.includes("box1") && f.includes("box2") && f.includes("paired  root@"), "hosts list"));

    // --- fleet and node detail.
    app.go("fleet");
    save("fleet", await d.until((f) => /box1\s+0x/.test(f) && /box2\s+0x/.test(f), "fleet rows", 90_000));
    d.press("return");
    save("node box1: status", await d.until((f) => f.includes("Node box1") && f.includes("overall "), "node status", 60_000));
    d.press("2");
    save("node box1: disk", await d.until((f) => f.includes("fit "), "disk", 60_000));
    d.press("5");
    save("node box1: logs", await frame());
    d.press("1");

    // --- restart a service.
    d.press("r");
    await d.modal();
    save("restart confirm", await frame());
    d.press("y");
    await d.logged(/^restart exec on box1: /, 120_000);
    save("node box1: after restart", await frame());
    d.press("escape");

    // --- a job with decisions: rename the legacy names on box1, answer with the typed confirm.
    app.go("hosts");
    await d.until((f) => f.includes("> box1"), "box1 selected");
    d.press("M", "M");
    await d.modal();
    d.press("y");
    const started = await d.logged(/^started box1\//);
    const jobId = started.split("/")[1]!.split(":")[0]!;
    app.go("jobs");
    const answers = ["stop-processes", "kill-processes"];
    for (const want of answers) {
      await d.until(() => !!app.state.inbox?.jobs.find((j) => j.id === jobId && j.state === "needs-decision" && j.question?.options.some((o) => o.id === want)), `decision ${want}`, 240_000);
      if (!(await frame()).includes("output:")) {
        save("jobs inbox (decision)", await frame());
        d.press("return");
      }
      const job = app.state.inbox!.jobs.find((j) => j.id === jobId)!;
      const idx = job.question!.options.findIndex((o) => o.id === want);
      save(`job decision (${want})`, await d.until((f) => f.includes("? "), "question"));
      d.press(String(idx + 1));
      d.press("a");
      await d.modal();
      save(`typed confirm (${want})`, await frame());
      d.type("no");
      d.press("return");
      const miss = await d.until((f) => f.includes("that is not"), "typed-confirm miss");
      save("typed confirm: a miss", miss);
      d.type(want);
      d.press("return");
      await d.logged(new RegExp(`^answered box1/${jobId}: `));
    }
    await d.until(() => app.state.inbox?.jobs.find((j) => j.id === jobId)?.state === "succeeded", "job succeeded", 300_000);
    save("job succeeded", await frame());
    d.press("escape");

    // --- a module: box2 has no node; turn the health watch on for box1's.
    app.go("modules");
    await d.until((f) => f.includes("boxes ([enter] open)"), "modules");
    d.press("down");
    d.press("return");
    save("modules on box2", await d.until((f) => f.includes("modules on box2") && !f.includes("loading..."), "box2 modules", 60_000));
    d.press("escape");
    d.press("up");
    d.press("return");
    await d.until((f) => f.includes("modules on box1") && !f.includes("loading..."), "box1 modules", 60_000);
    for (let i = 0; i < 12 && !(await frame()).includes("> watch"); i++) d.press("down");
    save("modules on box1", await frame());
    d.press("c");
    await d.answer('{"enabled":true}');
    await d.logged(/^watch on box1: /);
    await d.logged(/^setup box1: done$/, 300_000);
    save("modules on box1: watch on", await d.until((f) => /> watch\s+active/.test(f), "watch active", 60_000));

    // --- invite.
    app.go("hosts");
    d.press("i");
    await d.answer("box3");
    save("hosts invite", await d.until((f) => f.includes("on the box, as root:"), "invite"));

    // --- upgrade box2's gate: the hook opts it in and gives the controller a newer gate.
    const h = Bun.spawnSync([hook, "hook", "pre-upgrade", "2"], { stdout: "inherit", stderr: "inherit" });
    expect(h.exitCode).toBe(0);
    d.press("down");
    await d.until((f) => f.includes("> box2"), "box2 selected");
    d.press("u");
    await d.logged(/^gate on box2: v0\.0\.90 -> v0\.0\.91$/, 120_000);
    save("hosts upgrade", await frame());
  }

  // --- lock and unlock.
  app.go("vault");
  d.press("l");
  await d.modal();
  d.press("y");
  save("vault locked", await d.until((f) => f.includes("state  locked"), "locked"));
  save("vault unlocked again", await unlock());

  app.go("jobs");
  const jobs = await d.until((f) => f.includes("No jobs.") || f.includes("step "), "jobs");
  save("jobs", jobs);
  expect(app.activity.all().join("\n")).not.toContain(pass);
  app.quit();
}, 1_200_000);
