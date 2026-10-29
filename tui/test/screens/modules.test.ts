import { afterEach, expect, test } from "bun:test";
import { ModulesScreen } from "../../src/screens/modules.ts";
import { json, sseResponse } from "../fake.ts";
import { fixture, mount, typeText, type Mounted } from "../harness.ts";

let m: Mounted | undefined;
afterEach(async () => {
  await m?.close();
  m = undefined;
});

async function modules(put: (body: any) => Response = () => json({ status: "saved" })) {
  m = await mount((app) => [new ModulesScreen(app)], {
    routes: {
      "GET /api/catalog": () => json(fixture("catalog")),
      "GET /api/targets/alpha/modules": () => json({ ...fixture("modulesview"), modules: [{ ...fixture("modulesview").modules[0], note: "posting to https://u:SECRET@h" }] }),
      "PUT /api/targets/alpha/modules/watch": (_r, rec) => put(JSON.parse(rec.body)),
      "GET /api/targets/alpha/setup/stream": () => sseResponse([`data: {"stepId":"pull","line":"pulling the image"}\n\n`, `data: {"stepId":"","done":true}\n\n`], true),
    },
    width: 120,
    height: 34,
  });
  m.app.go("modules");
  await m.waitFor("Watch");
  return m;
}

test("shows the catalog and the selected box's modules, masked", async () => {
  await modules();
  await m!.press("return");
  const f = await m!.waitFor("watch   active");
  expect(f).not.toContain("SECRET");
});

test("space toggles a module off; c sends raw JSON that round-trips the SetModule fixture", async () => {
  const bodies: any[] = [];
  await modules((b) => (bodies.push(b), json({ status: "saved" })));
  await m!.press("return");
  await m!.waitFor("watch   active");
  await m!.press("space", "y");
  await m!.waitFor(() => bodies.length === 1);
  expect(bodies[0]).toEqual({ enabled: false });
  await m!.press("c");
  await typeText(m!, JSON.stringify(fixture("setmodule")));
  await m!.press("return");
  await m!.waitFor(() => bodies.length === 2);
  expect(bodies[1]).toEqual(fixture("setmodule"));
});

test("bad JSON is refused before any request", async () => {
  await modules();
  await m!.press("return");
  await m!.waitFor("watch   active");
  await m!.press("c");
  await typeText(m!, "{nope");
  await m!.press("return");
  expect(await m!.frame()).toContain("FAIL not JSON");
  expect(m!.fake.requests.some((r) => r.method === "PUT")).toBe(false);
});

test("a started module follows the setup stream into the activity", async () => {
  await modules(() => json({ status: "started" }));
  await m!.press("return");
  await m!.waitFor("watch   active");
  await m!.press("space", "y");
  await m!.waitFor(() => m!.app.activity.all().some((l) => l.includes("setup alpha: pull pulling the image")));
  await m!.waitFor(() => m!.app.activity.all().some((l) => l.includes("setup alpha: done")));
});
