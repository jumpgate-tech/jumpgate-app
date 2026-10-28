import { expect, test } from "bun:test";
import { clip, na, pad, sanitize, sanitizeLine } from "../src/text.ts";

test("sanitize strips CSI, OSC, C1 and bidi controls", () => {
  expect(sanitize("a\x1b[31mred\x1b[0m b")).toBe("ared b");
  expect(sanitize("x\x1b]0;title\x07y")).toBe("xy");
  expect(sanitize("x\x1b]8;;http://e\x1b\\link\x1b]8;;\x1b\\y")).toBe("xlinky");
  expect(sanitize("a\u009b2Jb")).toBe("ab");
  expect(sanitize("a‮b​c")).toBe("abc");
  expect(sanitize("bell\x07 bs\x08 del\x7f")).toBe("bell bs del");
  expect(sanitize("keep\ttab\nline")).toBe("keep\ttab\nline");
  expect(sanitizeLine("a\nb\tc")).toBe("a b c");
});

test("clip and pad", () => {
  expect(clip("abcdef", 4)).toBe("abc…");
  expect(clip("abc", 4)).toBe("abc");
  expect(pad("ab", 4)).toBe("ab  ");
});

test("na never shows 0 for unknown", () => {
  expect(na(undefined)).toBe("n/a");
  expect(na(null)).toBe("n/a");
  expect(na(0)).toBe("0");
});

import { wrap } from "../src/text.ts";
test("wrap breaks at spaces and splits long words", () => {
  expect(wrap("aa bb cc", 5)).toEqual(["aa bb", "cc"]);
  expect(wrap("abcdefg", 3)).toEqual(["abc", "def", "g"]);
});
