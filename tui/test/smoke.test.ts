import { expect, test } from "bun:test";
import pkg from "../package.json";

const run = (...args: string[]) =>
  Bun.spawnSync(["bun", "run", "src/main.ts", ...args], { cwd: import.meta.dir + "/.." });

test("--version prints the name and version", () => {
  const r = run("--version");
  expect(r.exitCode).toBe(0);
  expect(r.stdout.toString().trim()).toBe(`jumpgate-tui ${pkg.version}`);
});

test("--help lists the options", () => {
  const r = run("--help");
  expect(r.exitCode).toBe(0);
  expect(r.stdout.toString()).toContain("--socket");
});

test("an unknown flag is a usage error", () => {
  expect(run("--nope").exitCode).toBe(2);
});
