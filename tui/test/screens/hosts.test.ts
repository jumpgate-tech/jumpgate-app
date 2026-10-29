import { afterEach, expect, test } from "bun:test";
import { HostsScreen, inviteCommand, inviteLines } from "../../src/screens/hosts.ts";
import { json, sseResponse } from "../fake.ts";
import { fixture, mount, recorder, say, typeText, type Mounted } from "../harness.ts";

let m: Mounted | undefined;
afterEach(async () => {
  await m?.close();
  m = undefined;
});

const unknown = { hops: [{ hostPort: "10.0.0.9:22", fingerprint: "SHA256:fullprint", keyType: "ssh-ed25519", state: "unknown", probeId: "p1" }], allConfirmed: false };

function routes(extra: Record<string, any> = {}) {
  return {
    "GET /api/targets": () => json(fixture("targets")),
    "POST /api/hostkeys/probe": () => json(unknown),
    "POST /api/hostkeys/confirm": () => new Response(null, { status: 204 }),
    "POST /api/targets": () => new Response(null, { status: 201 }),
    "POST /api/targets/beta/pair": () =>
      sseResponse([`data: {"step":"install","line":"installing the gate"}\n\n`, `data: {"done":true,"agent":"0xfeed"}\n\n`]),
    "GET /api/pairing-info": () => json(fixture("pairinginfo")),
    "POST /api/targets/alpha/agent/upgrade": () => json(fixture("agentupgrade")),
    "DELETE /api/targets/alpha": () => new Response(null, { status: 204 }),
    "GET /api/hostkeys": () => json(fixture("recordedhostkeys")),
    "POST /api/hostkeys/forget": () => new Response(null, { status: 204 }),
    "GET /api/fleet/alpha/ssh": () => json(fixture("sshcommand")),
    "POST /api/jobs/alpha": () => json(fixture("jobview")),
    ...extra,
  };
}

const clip = { oscs: [] as string[], runs: [] as { argv: string[]; input: string }[] };

async function hosts(o: { version?: string } = {}) {
  clip.oscs = [];
  clip.runs = [];
  const rec = recorder();
  const argvs: string[][] = [];
  m = await mount((app) => [new HostsScreen(app)], {
    routes: routes(o.version ? { "GET /api/version": () => json({ version: o.version }), "GET /api/pairing-info": () => json({ ...fixture("pairinginfo"), scope: "" }) } : {}),
    app: {
      run: rec.run,
      runArgv: async (a) => (argvs.push(a), 0),
      clipboard: { osc: (q) => clip.oscs.push(q), argv: ["/usr/bin/pbcopy"], run: async (argv, input) => (clip.runs.push({ argv, input }), 0) },
    },
    width: 120,
    height: 34,
  });
  m.app.go("hosts");
  const f = await m.waitFor("alpha");
  expect(f).toContain("alpha          gate");
  return { rec, argvs };
}

test("add a box: probe, show the full fingerprint, confirm, add, pair into the activity", async () => {
  await hosts();
  await m!.press("a");
  await typeText(m!, "beta");
  await m!.press("return");
  await typeText(m!, "ops@10.0.0.9:22");
  await m!.press("return");
  await m!.press("return"); // no jump host
  await m!.press("return"); // the ssh-agent's key
  const fp = await m!.waitFor("SHA256:fullprint");
  expect(fp).toContain("10.0.0.9:22");
  expect(fp).toContain("Type yes to confirm");
  await say(m!, "yes");
  await m!.waitFor("installed address");
  await m!.press("return"); // install over ssh
  await m!.waitFor(() => m!.app.activity.all().some((l) => l.includes("paired beta: gate 0xfeed")));
  const req = (p: string) => JSON.parse(m!.fake.requests.find((r) => r.path === p && r.method === "POST")!.body);
  expect(req("/api/hostkeys/probe")).toEqual({ ssh: { Host: "10.0.0.9", User: "ops", Port: 22 } });
  expect(req("/api/hostkeys/confirm")).toEqual({ probeId: "p1", fingerprint: "SHA256:fullprint" });
  expect(req("/api/targets")).toEqual({ id: "beta", mode: "ssh", ssh: { Host: "10.0.0.9", User: "ops", Port: 22 } });
  expect(req("/api/targets/beta/pair")).toEqual({ sudo: true });
  expect(m!.app.activity.all().join("\n")).toContain("installing the gate");
});

// Live: a box whose key is a file, with no ssh-agent, could not be added
// from the TUI (the server wants a key path or an agent).
test("add a box with a key file: the path goes with the probe and the target", async () => {
  await hosts();
  await m!.press("a");
  await say(m!, "beta");
  await say(m!, "root@10.0.0.9:2222");
  await m!.press("return"); // no jump host
  await m!.waitFor("ssh key file");
  await say(m!, "/home/me/.ssh/box_ed25519");
  await m!.waitFor("SHA256:fullprint");
  await say(m!, "yes");
  await m!.waitFor("installed address");
  await m!.press("return");
  await m!.waitFor(() => m!.app.activity.all().some((l) => l.includes("paired beta")));
  const req = (p: string) => JSON.parse(m!.fake.requests.find((r) => r.path === p && r.method === "POST")!.body);
  const ssh = { Host: "10.0.0.9", User: "root", Port: 2222, KeyPath: "/home/me/.ssh/box_ed25519" };
  expect(req("/api/hostkeys/probe")).toEqual({ ssh });
  expect(req("/api/targets")).toEqual({ id: "beta", mode: "ssh", ssh });
  expect(req("/api/targets/beta/pair")).toEqual({ sudo: false });
});

test("a refused fingerprint adds nothing", async () => {
  await hosts();
  await m!.press("a");
  await typeText(m!, "beta");
  await m!.press("return");
  await typeText(m!, "root@10.0.0.9");
  await m!.press("return", "return", "return");
  await m!.waitFor("SHA256:fullprint");
  await m!.press("n");
  await Bun.sleep(20);
  expect(m!.fake.requests.some((r) => r.path === "/api/targets" && r.method === "POST")).toBe(false);
});

test("add this machine, upgrade, remove, forget a key, ssh, rename legacy names", async () => {
  const { rec, argvs } = await hosts();
  await m!.press("L");
  await typeText(m!, "here");
  await m!.press("return");
  await m!.press("u");
  await m!.waitFor(() => m!.app.activity.all().some((l) => l.includes("gate on alpha: v0.5.0 -> v0.6.0")));
  await m!.press("f");
  await m!.waitFor("SHA256:abc");
  await m!.press("y");
  await m!.waitFor(() => m!.fake.requests.some((r) => r.path === "/api/hostkeys/forget"));
  await m!.press("x");
  await m!.waitFor(() => argvs.length > 0);
  await m!.press("M", "y");
  await m!.waitFor(() => m!.fake.requests.some((r) => r.path === "/api/jobs/alpha"));
  await m!.press("D");
  await say(m!, "alpha");
  await m!.waitFor(() => m!.fake.requests.some((r) => r.method === "DELETE"));
  expect(rec.calls).toEqual([{ args: ["hosts", "add", "here", "--local"] }]);
  expect(argvs).toEqual([["/usr/bin/ssh", ...fixture("sshcommand").argv.slice(1)]]);
  expect(JSON.parse(m!.fake.requests.find((r) => r.path === "/api/hostkeys/forget")!.body)).toEqual({ hostPort: "10.0.0.2:22", fingerprint: "SHA256:abc" });
  expect(JSON.parse(m!.fake.requests.find((r) => r.path === "/api/jobs/alpha")!.body)).toEqual({ kind: "node.migrate-names" });
});

test("invite prints the CLI's lines", () => {
  const A = "0x" + "ab".repeat(20);
  const lines = inviteLines("v0.6.0", { controller: A, transportKey: "ssh-ed25519 AAAA it's", system: false }, "beta");
  expect(lines).toEqual([
    "on the box, as root:",
    "curl -fsSL https://jumpgate.app/install.sh | sh -s -- --agent \\",
    `  --version v0.6.0 --controller ${A} --transport-key 'ssh-ed25519 AAAA it'"'"'s'`,
    "then: jumpgate hosts add beta --ssh jumpgate@HOST --installed <address printed on the box>",
  ]);
  expect(inviteLines("v0.6.0", { controller: "0xab; id", transportKey: "k", system: false }, "b")[0]).toContain("no valid controller");
  expect(inviteLines("v0.6.0 $(id)", { controller: A, transportKey: "k", system: false }, "b")[0]).toContain("development build");
  expect(inviteLines("dev", { controller: "0xabc", transportKey: "k", system: false }, "b")[0]).toContain("development build");
  expect(inviteLines("v0.6.0", { controller: "0xabc", transportKey: "k", system: false, scope: "self" }, "b")[0]).toContain("self-only");
});

const CMD =
  "curl -fsSL https://jumpgate.app/install.sh | sh -s -- --agent --version v0.6.0 " +
  "--controller 0x00000000000000000000000000000000000000ab --transport-key 'ssh-ed25519 AAAA jumpgate'";

test("inviteCommand is the CLI's command on one line, or the FAIL line", () => {
  const A = "0x" + "ab".repeat(20);
  expect(inviteCommand("v0.6.0", { controller: A, transportKey: "ssh-ed25519 AAAA it's", system: false }, "beta")).toEqual({
    cmd: `curl -fsSL https://jumpgate.app/install.sh | sh -s -- --agent --version v0.6.0 --controller ${A} --transport-key 'ssh-ed25519 AAAA it'"'"'s'`,
  });
  expect(inviteCommand("dev", { controller: A, transportKey: "k", system: false }, "b")).toEqual({
    fail: "FAIL this controller is a development build (dev); an invite needs a released version",
  });
});

test("i shows the invite command outside every border, as one unwrapped line", async () => {
  await hosts({ version: "v0.6.0" });
  await m!.press("i");
  await typeText(m!, "gamma");
  await m!.press("return");
  const f = await m!.waitFor("--controller 0x0000");
  expect(f).toContain("on the box, as root, run the command below the panes. [c] copy");
  expect(f).toContain("then: jumpgate hosts add gamma --ssh jumpgate@HOST --installed <address");
  expect(f).toContain("printed on the box>");
  // The strip is full width with no border character: its rows, joined, are exactly the command.
  const rows = f.split("\n");
  const start = rows.findIndex((r) => r.startsWith("curl -fsSL"));
  expect(start).toBeGreaterThan(0);
  const n = Math.ceil(CMD.length / 120);
  const strip = rows.slice(start, start + n);
  expect(strip.join("").trimEnd()).toBe(CMD);
  for (const r of strip) expect(r).not.toMatch(/[│║┃|]$|^[│║┃]/);
  // It sits below the main pane's bottom border.
  expect(rows[start - 1]).toMatch(/[└╚]/);
  expect(f).toContain("c copy the invite");
});

test("c copies the invite command by OSC 52 and the local copy program", async () => {
  await hosts({ version: "v0.6.0" });
  await m!.press("c");
  expect(clip.oscs).toEqual([]); // nothing to copy before an invite
  await m!.press("i");
  await typeText(m!, "gamma");
  await m!.press("return");
  await m!.waitFor("--controller 0x0000");
  await m!.press("c");
  await m!.waitFor(() => m!.app.activity.all().some((l) => l.includes("invite command: copied")));
  expect(clip.oscs).toEqual([`\x1b]52;c;${Buffer.from(CMD).toString("base64")}\x07`]);
  expect(clip.runs).toEqual([{ argv: ["/usr/bin/pbcopy"], input: CMD }]);
  expect(m!.app.activity.all().some((l) => l.includes("invite command: copied (OSC 52 and /usr/bin/pbcopy)"))).toBe(true);
});

test("a failed invite shows no command and copies nothing", async () => {
  await hosts(); // a self-only controller
  await m!.press("i");
  await typeText(m!, "gamma");
  await m!.press("return");
  const f = await m!.waitFor("FAIL");
  expect(f).not.toContain("curl -fsSL");
  await m!.press("c");
  expect(clip.oscs).toEqual([]);
  expect(clip.runs).toEqual([]);
});

test("a server ssh argv of another shape is refused and never run", async () => {
  const { argvs } = await hosts();
  const evil = { argv: [...fixture("sshcommand").argv.slice(0, -2), "-oLocalCommand=id", "--", "10.0.0.2"], display: "x" };
  m!.fake.routes["GET /api/fleet/alpha/ssh"] = () => json(evil);
  await m!.press("x");
  await m!.waitFor(() => m!.app.activity.all().some((l) => l.includes("refusing the server's ssh")));
  m!.fake.routes["GET /api/fleet/alpha/ssh"] = () => json({ ...fixture("sshcommand"), argv: ["/tmp/evil", ...fixture("sshcommand").argv.slice(1)] });
  await m!.press("x");
  await Bun.sleep(30);
  expect(argvs).toEqual([]);
});

test("a buffered Enter lands on the trust modal and does not trust", async () => {
  await hosts();
  await m!.press("a");
  await typeText(m!, "beta");
  await m!.press("return");
  await typeText(m!, "ops@10.0.0.9");
  await m!.press("return");
  await m!.press("return"); // no jump host
  m!.app.opts.modalGuardMs = 150; // the trust modal opens with the real guard
  await m!.press("return"); // the ssh-agent's key
  await m!.waitFor("SHA256:fullprint");
  m!.app.handleKey({ name: "return", seq: "\r", ctrl: false, shift: false, meta: false }); // typed ahead: flushed
  await Bun.sleep(200);
  m!.app.handleKey({ name: "return", seq: "\r", ctrl: false, shift: false, meta: false }); // Enter alone: not yes
  await Bun.sleep(20);
  expect(await m!.frame()).toContain("Type yes to confirm");
  expect(m!.fake.requests.some((r) => r.path === "/api/hostkeys/confirm")).toBe(false);
  expect(m!.fake.requests.some((r) => r.path === "/api/targets" && r.method === "POST")).toBe(false);
});
