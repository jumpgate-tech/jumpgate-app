import { expect, test } from "bun:test";
import { jsonLines } from "../src/screens/json.ts";

test("jsonLines prints key: value, masks credential strings, keeps times and numbers", () => {
  expect(jsonLines({ a: 1, url: "https://u:SECRET@h/x", at: "2026-10-28T12:00:00Z", list: ["x", { b: true }], k: "\x1b[31mred" })).toEqual([
    "a: 1",
    "url: ***",
    "at: 2026-10-28T12:00:00Z",
    "list:",
    "  - x",
    "  -",
    "    b: true",
    "k: red",
  ]);
});
