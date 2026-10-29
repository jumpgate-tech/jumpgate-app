import { afterEach, expect, test } from "bun:test";
import { SettingsScreen } from "../../src/screens/settings.ts";
import { json } from "../fake.ts";
import { fixture, mount, typeText, type Mounted } from "../harness.ts";

let m: Mounted | undefined;
afterEach(async () => {
  await m?.close();
  m = undefined;
});

const KEY = "sk-ant-SECRETAIKEY";

async function open() {
  m = await mount((app) => [new SettingsScreen(app)], {
    routes: {
      "GET /api/settings": () => json(fixture("settings")),
      "PUT /api/settings": (_r, x) => json({ ...fixture("settings"), aiKeySet: true }),
      "GET /api/update": (r) => json({ current: "v0.6.0", latest: new URL(r.url).search ? "v0.7.0" : "v0.6.0" }),
    },
    width: 140,
    height: 40,
  });
  m.app.go("settings");
  return m.waitFor("AI key            set");
}
const puts = () => m!.fake.requests.filter((r) => r.method === "PUT" && r.path === "/api/settings").map((r) => r.body);

test("settings show, the AI key is write-only and never drawn", async () => {
  expect(await open()).toContain("idle lock         30 min");
  await m!.press("k");
  expect(await typeText(m!, KEY)).not.toContain("SECRET");
  await m!.press("return");
  await m!.waitFor("Saved the AI key.");
  expect(JSON.parse(puts()[0]!)).toEqual({ aiKey: KEY });
  expect(await m!.frame()).not.toContain("SECRET");
  expect(m!.app.activity.all().join("\n")).not.toContain("SECRET");
});

test("provider and idle lock save one field; drawer cycles the pref; r refreshes the update check", async () => {
  await open();
  await m!.press("p");
  await m!.press(...Array(20).fill("backspace"));
  await typeText(m!, "openai");
  await m!.press("return");
  await m!.press("i");
  await typeText(m!, "15");
  await m!.press("return");
  await m!.waitFor(() => puts().length === 2);
  expect(puts().map((b) => JSON.parse(b))).toEqual([{ aiProvider: "openai" }, { vaultIdleLockMinutes: 15 }]);
  await m!.press("d");
  await m!.waitFor("drawer            auto");
  await m!.press("r");
  expect(await m!.waitFor("latest: v0.7.0")).toContain("current: v0.6.0");
});
