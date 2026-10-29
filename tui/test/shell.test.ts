import { afterEach, expect, test } from "bun:test";
import { Activity } from "../src/shell/activity.ts";
import { SecretModal } from "../src/shell/confirm.ts";
import { drawerLines, minimizedLine } from "../src/shell/drawer.ts";
import { key } from "../src/shell/keys.ts";
import { statusLine } from "../src/shell/statusbar.ts";
import { json } from "./fake.ts";
import { dummy, fixture, mount, type Mounted } from "./harness.ts";

let m: Mounted | undefined;
afterEach(async () => {
  await m?.close();
  m = undefined;
});

test("the frame shows all five regions", async () => {
  m = await mount(() => [dummy("Fleet"), dummy("Jobs")], { routes: { "GET /api/ui-prefs": () => json({ ...fixture("uiprefs"), drawer: "open" }) } });
  const f = await m.waitFor("boxes 1");
  expect(f).toContain("server 0.0.1 | vault locked | boxes 1 0 ok/1 warn/0 down | ! 1 decisions"); // status
  expect(f).toContain("> Fleet"); // nav
  expect(f).toContain("Fleet body"); // main
  expect(f).toContain("activity"); // activity pane
  expect(f).toContain("! alpha/j1 node.upgrade step 2/2 needs-decision"); // drawer
  expect(f).toContain("q quit"); // hints
});

test("below 80x24 the window-too-small message replaces the layout", async () => {
  m = await mount(() => [dummy("Fleet")], { width: 70, height: 20 });
  const f = await m.frame();
  expect(f).toContain("window too small: 70x20, need 80x24");
  expect(f).not.toContain("Fleet body");
});

test("below 100 columns the activity pane folds away", async () => {
  m = await mount(() => [dummy("Fleet")], { width: 90, height: 30 });
  expect(await m.frame()).not.toContain("activity");
});

test(". toggles the drawer and saves the pref", async () => {
  m = await mount(() => [dummy("Fleet")]);
  await m.waitFor("[1 running]");
  const f = await m.press(".");
  expect(f).toContain("alpha/j1");
  expect(f).not.toContain("[1 running]");
  await Bun.sleep(20);
  const put = m.fake.requests.find((r) => r.method === "PUT" && r.path === "/api/ui-prefs");
  expect(JSON.parse(put!.body).drawer).toBe("open");
  expect(await m.press(".")).toContain("[1 running]");
});

test("a refused drawer save flashes and re-reads the prefs", async () => {
  // 90 columns: no activity pane, so the failure flashes on the hints row.
  m = await mount(() => [dummy("Fleet")], {
    width: 90,
    routes: { "PUT /api/ui-prefs": () => json({ error: "bad prefs", code: "invalid_prefs" }, 400) },
  });
  await m.waitFor("[1 running]");
  await m.press(".");
  const f = await m.waitFor("FAIL save the drawer setting: bad prefs");
  expect(f).toContain("bad prefs");
  await m.waitFor("[1 running]"); // back to the server's "min"
  expect(m.fake.requests.filter((r) => r.path === "/api/ui-prefs" && r.method === "GET").length).toBe(2);
});

test("the drawer never opens itself on a new decision", async () => {
  m = await mount(() => [dummy("Fleet")]);
  await m.waitFor("[1 running]");
  m.app.set({ inbox: { ...fixture("jobsinbox"), jobs: [fixture("jobview"), { ...fixture("jobview"), id: "j2" }] } });
  const f = await m.frame();
  expect(f).toContain("! 2 decisions");
  expect(m.app.state.drawer).toBe("min");
});

test("a . typed into a focused secret input types, not toggles", async () => {
  m = await mount(() => [dummy("Fleet")]);
  await m.waitFor("[1 running]");
  const p = m.app.askSecret("vault passphrase");
  const f = await m.press("a", ".", "b");
  expect(f).toContain("vault passphrase: ***");
  expect(m.app.state.drawer).toBe("min");
  await m.press("return");
  const got = await p;
  expect(new TextDecoder().decode(got!)).toBe("a.b");
});

test("esc or tab focuses the nav, where the arrows move it and enter goes back to the screen", async () => {
  m = await mount(() => [dummy("Fleet"), dummy("Jobs"), dummy("Vault")]);
  expect(m.app.focus).toBe("main");
  expect(await m.press("down")).toContain("> Fleet");
  await m.press("escape");
  expect(m.app.focus).toBe("nav");
  expect(await m.press("down")).toContain("> Jobs");
  expect(await m.press("down")).toContain("> Vault");
  expect(await m.press("up")).toContain("> Jobs");
  await m.press("return");
  expect(m.app.focus).toBe("main");
  await m.press("tab");
  expect(m.app.focus).toBe("activity");
  await m.press("tab");
  expect(m.app.focus).toBe("nav");
  await m.press(key("tab", { shift: true }));
  expect(m.app.focus).toBe("activity");
});

test("? lists the keys", async () => {
  m = await mount(() => [dummy("Fleet")]);
  const f = await m.press("?");
  expect(f).toContain("open or minimize the drawer");
  expect(await m.press("x")).toContain("Fleet body");
});

test("q quits and destroys the renderer", async () => {
  m = await mount(() => [dummy("Fleet")]);
  m.app.handleKey(key("q"));
  await m.app.done;
  expect(m.app.isClosed).toBe(true);
  expect(m.t.renderer.isDestroyed).toBe(true);
});

test("ctrl-c quits from inside a secret prompt and wipes it", async () => {
  m = await mount(() => [dummy("Fleet")]);
  const p = m.app.askSecret("vault passphrase");
  await m.press("s");
  m.app.handleKey(key("c", { ctrl: true, seq: "\x03" }));
  expect(await p).toBeNull();
  expect(m.app.isClosed).toBe(true);
});

test("statusLine is fail-closed and marks skew and staleness", () => {
  expect(statusLine({ connected: true, myVersion: "1" })).toBe("server n/a | vault n/a | boxes n/a | ! n/a decisions");
  expect(statusLine({ connected: true, myVersion: "1", serverVersion: "2", fleet: fixture("fleet"), fleetStale: true, decisions: 0 })).toContain(
    "server 2 [skew!] | vault n/a | boxes 1 0 ok/1 warn/0 down ~ | ! 0 decisions",
  );
});

test("the minimized drawer line counts running jobs; decisions are on the status strip only", () => {
  expect(minimizedLine([])).toBe("");
  expect(minimizedLine(fixture("runningjobs").jobs)).toBe("[1 running]");
  expect(minimizedLine(undefined)).toBe("");
  expect(drawerLines(undefined)).toEqual(["n/a"]);
});

test("activity keeps 200 lines, redacted and sanitized", () => {
  const a = new Activity(() => new Date("2026-10-28T12:00:00Z"));
  for (let i = 0; i < 250; i++) a.add(`line ${i}`);
  expect(a.all().length).toBe(200);
  a.add(`token ${"ab".repeat(32)} \x1b[31mred`);
  expect(a.last(1)[0]).toBe("12:00:00 token *** red");
});

test("SecretModal zeroes its buffer and never shows the text", () => {
  let got: Uint8Array | null = null;
  const s = new SecretModal("p", (v) => (got = v));
  for (const c of "hunter2") s.key(key(c));
  s.key(key("backspace"));
  expect(s.lines().join("\n")).not.toContain("hunter");
  expect(s.value).toBe("");
  s.key(key("return"));
  expect(new TextDecoder().decode(got!)).toBe("hunter");
  expect((s as any).buf.every((b: number) => b === 0)).toBe(true);
});

test("control bytes in a server error reach no pane", async () => {
  const evil = "boom\x1b]0;pwned\x07 \x1b[2J\x1b[31mred \u009b6n \u009d8;;x\u009c end";
  m = await mount(() => [{ id: "Fleet", title: "Fleet", view: () => [evil] }]);
  m.app.fail("x", { message: evil, hint: evil });
  const f = await m.frame();
  expect(m.app.flash).not.toMatch(/[\x00-\x08\x0b-\x1f\x7f-\x9f]/);
  expect(f).not.toMatch(/[\x00-\x09\x0b-\x1f\x7f-\x9f]/);
  expect(f).toContain("FAIL x: boom");
  expect(f).not.toContain("pwned");
});

test("a server error with a credential URL is masked in the flash and the activity", async () => {
  m = await mount(() => [dummy("Fleet")]);
  m.app.fail("x", { message: "upstream https://u:SECRET@rpc.example failed", hint: "check u:SECRET@h" });
  const f = await m.frame();
  expect(f).not.toContain("SECRET");
  expect(m.app.flash).not.toContain("SECRET");
  expect(m.app.activity.all().join("\n")).not.toContain("SECRET");
});

import { TypedConfirm } from "../src/shell/confirm.ts";
test("TypedConfirm: only the exact word; a miss asks again; a blank or unshowable word is refused", () => {
  let got = null as boolean | null;
  const c = new TypedConfirm("Remove?", "alpha", (ok) => (got = ok));
  c.key(key("return"));
  expect(got).toBeNull();
  for (const ch of "alph") c.key(key(ch));
  c.key(key("return"));
  expect(got).toBeNull();
  expect(c.lines(80).join("\n")).toContain('that is not "alpha"');
  for (const ch of "alpha") c.key(key(ch));
  c.key(key("return"));
  expect(got).toBe(true);
  for (const w of ["", "a\x1bb"]) {
    let g = null as boolean | null;
    const r = new TypedConfirm("x", w, (ok) => (g = ok));
    for (const ch of w) r.key(key(ch, { seq: ch }));
    r.key(key("return"));
    expect(g).toBeNull();
    expect(r.refused).toBe(true);
  }
});

test("the y/n confirm no longer takes Enter as yes", async () => {
  m = await mount(() => [dummy("Fleet")]);
  const p = m.app.confirm("go?");
  await m.press("return");
  expect(m.app.modal).not.toBeNull();
  await m.press("n");
  expect(await p).toBe(false);
});
