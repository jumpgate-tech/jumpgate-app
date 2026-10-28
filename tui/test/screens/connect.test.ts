import { afterEach, expect, test } from "bun:test";
import { DiscoverError, NOT_ADMIN } from "../../src/api/discover.ts";
import type { Conn } from "../../src/api/http.ts";
import { ConnectScreen } from "../../src/screens/connect.ts";
import { json } from "../fake.ts";
import { dummy, mount, type Mounted } from "../harness.ts";

let m: Mounted | undefined;
afterEach(async () => {
  await m?.close();
  m = undefined;
});

test("not running: offers to start, shells out jumpgate serve detached, polls, then goes to Fleet", async () => {
  let up = false;
  const runs: { args: string[]; detach: boolean }[] = [];
  let conn: Conn | undefined;
  m = await mount(
    (app) => [
      new ConnectScreen(app, { next: "Fleet",
        discover: async () => {
          if (!up) throw new DiscoverError("not_running", "No jumpgate server is running.");
          return conn!;
        },
        run: async (args, o) => {
          runs.push({ args, detach: !!o.detach });
          up = true;
          return 0;
        },
        pollMs: 5,
        timeoutMs: 1000,
        home: "/tmp/jg-home",
      }),
      dummy("Fleet"),
    ],
    { attach: false },
  );
  conn = m.conn;
  m.app.go("connect");
  expect(await m.waitFor("No jumpgate server is running. [s] start it")).toBeTruthy();
  await m.press("s");
  const f = await m.waitFor("> Fleet");
  expect(f).toContain("Fleet body");
  expect(runs).toEqual([{ args: ["serve", "--no-open"], detach: true }]);
  expect(m.app.state.conn).toBe(conn);
});

test("not an admin: names the group, and offers no start", async () => {
  m = await mount(
    (app) => [new ConnectScreen(app, { next: "Fleet", discover: async () => { throw new DiscoverError("not_admin", NOT_ADMIN); } }), dummy("Fleet")],
    { attach: false },
  );
  m.app.go("connect");
  const f = await m.waitFor("jumpgate-admin");
  expect(f).not.toContain("[s] start it");
});

test("skew: the strip shows [skew!] and the screen shows both versions", async () => {
  m = await mount((app) => [new ConnectScreen(app, { next: "Fleet", discover: async () => m!.conn }), dummy("Fleet")], {
    attach: false,
    routes: { "GET /api/version": () => json({ version: "0.6.0" }) },
  });
  m.app.go("connect");
  await m.waitFor("[skew!]");
  m.app.go("connect", { stay: true });
  const f = await m.waitFor("server 0.6.0, this dashboard 0.0.1");
  expect(f).toContain("[skew!]");
});
