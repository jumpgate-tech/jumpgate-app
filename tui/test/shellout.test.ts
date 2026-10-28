import { expect, test } from "bun:test";
import { chmodSync, mkdtempSync, readFileSync, statSync, writeFileSync } from "node:fs";
import { tmpdir } from "node:os";
import { join } from "node:path";
import { jumpgateBin, runJumpgate } from "../src/shellout.ts";

function fakeBin(exit = 0): { bin: string; out: string; dir: string } {
  const dir = mkdtempSync(join(tmpdir(), "jgbin-"));
  const out = join(dir, "argv");
  const bin = join(dir, "jumpgate");
  writeFileSync(bin, `#!/bin/sh\nprintf '%s\\n' "$@" > "${out}"\nexit ${exit}\n`);
  chmodSync(bin, 0o755);
  return { bin, out, dir };
}

const spyRenderer = () => {
  const calls: string[] = [];
  return { calls, suspend: async () => void calls.push("suspend"), resume: async () => void calls.push("resume") };
};

test("JUMPGATE_BIN wins, then jumpgate on PATH", () => {
  expect(jumpgateBin({ JUMPGATE_BIN: "/x/jg" })).toBe("/x/jg");
  expect(jumpgateBin({})).toBe("jumpgate");
});

test("a foreground run suspends, runs the fixed argv, resumes and returns the exit code", async () => {
  const { bin, out } = fakeBin(3);
  const r = spyRenderer();
  const code = await runJumpgate(["vault", "backup", "/tmp/f"], { renderer: r, env: { JUMPGATE_BIN: bin } });
  expect(code).toBe(3);
  expect(readFileSync(out, "utf8")).toBe("vault\nbackup\n/tmp/f\n");
  expect(r.calls).toEqual(["suspend", "resume"]);
});

test("the renderer resumes even when the binary is missing", async () => {
  const r = spyRenderer();
  const code = await runJumpgate(["serve"], { renderer: r, env: { JUMPGATE_BIN: "/nonexistent/jumpgate" } });
  expect(code).toBe(127);
  expect(r.calls).toEqual(["suspend", "resume"]);
});

test("a detached run does not suspend and logs to a private file", async () => {
  const { bin, out, dir } = fakeBin(0);
  const log = join(dir, "server.log");
  const r = spyRenderer();
  const code = await runJumpgate(["serve", "--no-open"], { renderer: r, detach: { log, cwd: dir }, env: { JUMPGATE_BIN: bin } });
  expect(code).toBe(0);
  expect(r.calls).toEqual([]);
  for (let i = 0; i < 500; i++) {
    try {
      if (readFileSync(out, "utf8")) break;
    } catch {}
    await Bun.sleep(10);
  }
  expect(readFileSync(out, "utf8")).toBe("serve\n--no-open\n");
  expect(statSync(log).mode & 0o777).toBe(0o600);
});
