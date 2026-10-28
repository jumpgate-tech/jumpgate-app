import { expect, test } from "bun:test";
import { maskURL } from "../src/maskurl.ts";

test("maskURL hides userinfo, keys and query values", () => {
  expect(maskURL("http://127.0.0.1:8545")).toBe("http://127.0.0.1:8545");
  expect(maskURL("https://user:pw@rpc.example/x")).toBe("https://***/x");
  expect(maskURL("https://rpc.example/v3/0123456789abcdef0123")).toBe("https://rpc.example/v3/***");
  expect(maskURL("https://rpc.example/key/abc")).toBe("https://rpc.example/key/***");
  expect(maskURL("https://rpc.example/x?apikey=s3cret&chain=1")).toBe("https://rpc.example/x?apikey=***&chain=***");
  expect(maskURL("https://rpc.example/x#frag")).toBe("https://rpc.example/x#***");
});

// Ported from internal/tui/maskurl_test.go.
import { MAX_MASK_TEXT, maskText } from "../src/maskurl.ts";
import { sanitizeLine } from "../src/text.ts";

test("maskText fails closed, and leaves plain text alone", () => {
  for (const s of [
    "https:u:SECRET@h", "//u:SECRET@h", "u:SECRET@h", "https://us er:SECRET@h",
    "https://u:SECRET\tx@h/", "see https://u:SECRET x@h/ now", "(https://h/SECRET1234567890abcdef)",
    "host:8545/v3/SECRET1234567890abcdef", "url=https://u:SECRET@h", "x https://h/a;key=SECRET y",
    "u:SECRET @h", "see u:SECRET @h now", "u:SECRET＠h", "u:SECRET%40h", "https:// u:SECRET @h",
  ]) expect(maskText(s)).not.toContain("SECRET");
  for (const s of ["the certificate expires in 6 days", "sudo ufw deny 8545", "8545 is bound to 0.0.0.0"]) expect(maskText(s)).toBe(s);
});

test("maskText: unicode spaces do not split a token", () => {
  for (const sp of [" ", " ", "　"]) expect(maskText("https://h/v3/" + sp + "Zk9SECRETQx7")).not.toContain("SECRET");
});

test("maskText hides a secret behind invisible characters before @", () => {
  const inv = ["​", "‌", "‍", "⁠", "﻿", "­"];
  for (const ch of ["", " ", "＠", "%40", ...inv]) {
    const at = ch === "＠" || ch === "%40" ? "" : "@";
    for (const s of ["u:SECRET " + ch + at + "h", "u:SECRET " + ch + " " + at + "h", "u:SECRET   " + ch + at + "h"]) {
      expect(maskText(s)).not.toContain("SECRET");
      expect(maskText(sanitizeLine(s))).not.toContain("SECRET");
    }
  }
});

test("maskText: a cut walking across a secret leaks nothing", () => {
  for (const secret of ["u:SECRET @h", "u:SECRET  @h", "https://u:SECRET @h/"]) {
    for (let off = 0; off <= secret.length + 2; off++) {
      const pad = MAX_MASK_TEXT - off;
      const s = "a ".repeat(Math.floor(pad / 2)) + "b".repeat(pad % 2) + secret + " tail".repeat(20);
      expect(maskText(s)).not.toContain("SECRET");
    }
  }
  const secret = "https://u:SECRETSECRETSECRET@h/";
  for (let cut = 1; cut < secret.length; cut++) {
    const s = "a ".repeat(Math.floor((MAX_MASK_TEXT - cut) / 2)) + "b".repeat((MAX_MASK_TEXT - cut) % 2) + secret;
    const got = maskText(s);
    expect(got).not.toContain("SEC");
    expect(got.endsWith("…")).toBe(true);
  }
});
