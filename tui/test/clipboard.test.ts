import { expect, test } from "bun:test";
import { chmodSync, mkdtempSync, writeFileSync } from "node:fs";
import { tmpdir } from "node:os";
import { join } from "node:path";
import { clipboardArgv, copyText, osc52, runWithInput } from "../src/clipboard.ts";

const bin = (dir: string, name: string, body = "#!/bin/sh\ncat > \"$OUT\"\n") => {
  const p = join(dir, name);
  writeFileSync(p, body);
  chmodSync(p, 0o755);
  return p;
};

test("osc52 base64-encodes the text for the terminal clipboard", () => {
  expect(osc52("curl x | sh")).toBe(`\x1b]52;c;${Buffer.from("curl x | sh").toString("base64")}\x07`);
});

test("clipboardArgv picks the platform's program by absolute path only", () => {
  const d = mkdtempSync(join(tmpdir(), "clip-"));
  const pb = bin(d, "pbcopy");
  const wl = bin(d, "wl-copy");
  const xc = bin(d, "xclip");
  expect(clipboardArgv({ PATH: d }, "darwin")).toEqual([pb]);
  expect(clipboardArgv({ PATH: d, WAYLAND_DISPLAY: "w", DISPLAY: ":0" }, "linux")).toEqual([wl]);
  expect(clipboardArgv({ PATH: d, DISPLAY: ":0" }, "linux")).toEqual([xc, "-selection", "clipboard"]);
  expect(clipboardArgv({ PATH: d }, "linux")).toBeNull(); // no display: OSC 52 only
  expect(clipboardArgv({ PATH: `.:relative:${tmpdir()}/none` }, "darwin")).toBeNull();
});

test("copyText sends OSC 52 and feeds the program; a failing program is reported", async () => {
  const d = mkdtempSync(join(tmpdir(), "clip-"));
  const out = join(d, "out");
  const p = bin(d, "pbcopy", `#!/bin/sh\ncat > '${out}'\n`);
  const oscs: string[] = [];
  expect(await copyText("hello", { osc: (s) => oscs.push(s), argv: [p], run: runWithInput })).toBe(`copied (OSC 52 and ${p})`);
  expect(await Bun.file(out).text()).toBe("hello");
  expect(oscs).toEqual([osc52("hello")]);
  const bad = bin(d, "bad", "#!/bin/sh\nexit 3\n");
  expect(await copyText("x", { osc: () => {}, argv: [bad], run: runWithInput })).toBe(`copied (OSC 52); ${bad} exited 3`);
  expect(await copyText("x", { osc: () => {}, argv: null, run: runWithInput })).toBe("copied (terminal clipboard, OSC 52)");
});
