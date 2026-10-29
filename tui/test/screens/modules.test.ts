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
  // Live: the list stayed as it was before the run; it is read again once the run ends.
  const reads = () => m!.fake.requests.filter((r) => r.path === "/api/targets/alpha/modules").length;
  await m!.waitFor(() => reads() === 3);
});

// Live: turning a module on moved it up the list on the reload, and the
// cursor stayed on the row, so it then sat on another module.
test("the cursor follows its module across a reload", async () => {
  const cat = fixture("catalog");
  const watch = cat.modules[0];
  let have: any[] = [];
  m = await mount((app) => [new ModulesScreen(app)], {
    routes: {
      "GET /api/catalog": () => json({ ...cat, modules: [{ ...watch, id: "checkpointz", name: "Checkpointz" }, watch] }),
      "GET /api/targets/alpha/modules": () => json({ modules: have, gateStale: false, message: "" }),
      "PUT /api/targets/alpha/modules/watch": () => ((have = [{ ...fixture("modulesview").modules[0], note: "", error: "" }]), json({ status: "saved" })),
    },
    width: 120,
    height: 34,
  });
  m.app.go("modules");
  await m.waitFor("Checkpointz");
  await m.press("return");
  await m.waitFor("> checkpointz");
  await m.press("down", "space", "y");
  expect(await m.waitFor("watch   active")).toContain("> watch   active");
});

// Live: a box that reports no modules showed "no modules", and nothing in
// the catalog could be turned on for it.
test("a catalog module the box does not report can be turned on", async () => {
  const bodies: any[] = [];
  m = await mount((app) => [new ModulesScreen(app)], {
    routes: {
      "GET /api/catalog": () => json(fixture("catalog")),
      "GET /api/targets/alpha/modules": () => json({ modules: [], gateStale: false, message: "" }),
      "PUT /api/targets/alpha/modules/watch": (_r, rec) => (bodies.push(JSON.parse(rec.body)), json({ status: "saved" })),
    },
    width: 120,
    height: 34,
  });
  m.app.go("modules");
  await m.waitFor("Watch");
  await m.press("return");
  expect(await m.waitFor("not on this box")).toContain("> watch   off");
  await m.press("space", "y");
  await m.waitFor(() => bodies.length === 1);
  expect(bodies[0]).toEqual({ enabled: true });
});
