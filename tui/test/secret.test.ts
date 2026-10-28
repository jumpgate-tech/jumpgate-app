import { expect, test } from "bun:test";
import { redact, withSecret } from "../src/secret.ts";

test("withSecret zeroes the buffer when fn resolves", async () => {
  const buf = new TextEncoder().encode("hunter2");
  const out = await withSecret(buf, async (b) => b.length);
  expect(out).toBe(7);
  expect([...buf].every((x) => x === 0)).toBe(true);
});

test("withSecret zeroes the buffer when fn rejects", async () => {
  const buf = new TextEncoder().encode("hunter2");
  await expect(withSecret(buf, async () => { throw new Error("no"); })).rejects.toThrow("no");
  expect([...buf].every((x) => x === 0)).toBe(true);
});

test("redact removes bearer tokens, hex tokens, query secrets and userinfo", () => {
  const hex = "ab".repeat(32);
  expect(redact(`Authorization: Bearer ${hex}`)).not.toContain(hex);
  expect(redact(`Bearer abc.def-ghi`)).toBe("Bearer ***");
  expect(redact(`token ${hex} leaked`)).toBe("token *** leaked");
  expect(redact("GET /api/x?token=s3cret&y=1")).toBe("GET /api/x?token=***&y=1");
  expect(redact("GET /login?code=abc123")).toBe("GET /login?code=***");
  expect(redact("https://user:pw@rpc.example/x")).toBe("https://***@rpc.example/x");
  expect(redact("plain message")).toBe("plain message");
});
