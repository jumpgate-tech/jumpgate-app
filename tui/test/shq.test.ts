import { expect, test } from "bun:test";
import { shq } from "../src/shq.ts";

test("shq leaves plain words bare and single-quotes the rest", () => {
  expect(shq("v0.6.0")).toBe("v0.6.0");
  expect(shq("0xab")).toBe("0xab");
  expect(shq("ssh-ed25519 AAAA it's")).toBe(`'ssh-ed25519 AAAA it'"'"'s'`);
  expect(shq("$(id)")).toBe("'$(id)'");
  expect(shq("a;b")).toBe("'a;b'");
  expect(shq("")).toBe("''");
  expect(shq("`x`")).toBe("'`x`'");
});

test("a quoted value round-trips through sh", () => {
  for (const v of ["it's", "$(id)", "a b\tc", "'\"'", "`x`"]) {
    const r = Bun.spawnSync(["sh", "-c", `printf %s ${shq(v)}`]);
    expect(r.stdout.toString()).toBe(v);
  }
});
