import { afterEach, expect, test } from "bun:test";
import { ConnectScreen } from "../../src/screens/connect.ts";
import { DiscoverError } from "../../src/api/discover.ts";
import { SetupScreen } from "../../src/screens/setup.ts";
import { mount, recorder, type Mounted } from "../harness.ts";

let m: Mounted | undefined;
afterEach(async () => {
  await m?.close();
  m = undefined;
});

test("Linux: the installs run the CLI with fixed argv; o opens the web app", async () => {
  const rec = recorder();
  m = await mount((app) => [new SetupScreen(app)], { app: { run: rec.run, platform: "linux" } });
  m.app.go("setup");
  await m.press("i", "y", "s", "y", "S", "y", "o");
  await Bun.sleep(20);
  expect(rec.calls.map((c) => c.args)).toEqual([["setup", "install-service"], ["setup", "server"], ["setup", "server", "--self-only"], ["open"]]);
});

test("off Linux the installs are hidden", async () => {
  const rec = recorder();
  m = await mount((app) => [new SetupScreen(app)], { app: { run: rec.run, platform: "darwin" } });
  m.app.go("setup");
  const f = await m.press("i", "s");
  expect(f).toContain("Linux only");
  expect(rec.calls).toEqual([]);
});

test("X stops the server after a confirm and returns to Connect", async () => {
  m = await mount(
    (app) => [new ConnectScreen(app, { discover: async () => { throw new DiscoverError("not_running", "No jumpgate server is running."); } }), new SetupScreen(app)],
    { routes: { "POST /api/shutdown": () => new Response(null, { status: 202 }) } },
  );
  m.app.go("setup");
  await m.press("X", "y");
  const f = await m.waitFor("No jumpgate server is running. [s] start it");
  expect(f).toContain("> Connect");
  expect(m.app.state.conn).toBeNull();
});
