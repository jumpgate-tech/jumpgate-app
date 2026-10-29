// Mount the App on an OpenTUI test renderer against a fake server.
import { createTestRenderer, type TestRendererSetup } from "@opentui/core/testing";
import { readFileSync } from "node:fs";
import { join } from "node:path";
import type { Conn } from "../src/api/http.ts";
import { App, type AppOptions, type Screen } from "../src/shell/app.ts";
import { key, type Key } from "../src/shell/keys.ts";
import { fakeServer, json, sseResponse, type Fake, type Handler } from "./fake.ts";

export const fixture = (name: string): any =>
  JSON.parse(readFileSync(join(import.meta.dir, "fixtures", `${name}.json`), "utf8"));

export const TOKEN = "9".repeat(64);

/** baseRoutes serve the fixtures for every feed the App opens on attach. */
export function baseRoutes(): Record<string, Handler> {
  return {
    "GET /api/health": () => json({ ok: true }),
    "GET /api/version": () => json({ version: "0.0.1" }),
    "GET /api/vault": () => json(fixture("vaultview")),
    "GET /api/controller": () => json(fixture("controllerview")),
    "GET /api/ui-prefs": () => json({ ...fixture("uiprefs"), drawer: "min" }),
    "PUT /api/ui-prefs": (_r, rec) => json(JSON.parse(rec.body)),
    "GET /api/jobs/running": () => json(fixture("runningjobs")),
    "GET /api/fleet/stream": () => sseResponse([`data: ${JSON.stringify(fixture("fleet"))}\n\n`], true),
    "GET /api/jobs/stream": () => sseResponse([`data: ${JSON.stringify(fixture("jobsinbox"))}\n\n`], true),
  };
}

export interface Mounted {
  t: TestRendererSetup;
  app: App;
  fake: Fake;
  conn: Conn;
  frame(): Promise<string>;
  press(...keys: (string | Key)[]): Promise<string>;
  waitFor(text: string | ((f: string) => boolean)): Promise<string>;
  close(): Promise<void>;
}

export async function mount(
  screens: (app: App) => Screen[],
  o: { width?: number; height?: number; routes?: Record<string, Handler>; attach?: boolean; app?: Partial<AppOptions> } = {},
): Promise<Mounted> {
  const fake = fakeServer({ ...baseRoutes(), ...o.routes });
  const conn: Conn = { socket: fake.socket, token: TOKEN, system: false };
  const t = await createTestRenderer({ width: o.width ?? 100, height: o.height ?? 30 });
  const app = new App(t.renderer, { myVersion: "0.0.1", streamOpts: { delay: () => 5 }, runningEveryMs: 60_000, modalGuardMs: 0, bins: { jumpgate: "/opt/jg/jumpgate", ssh: "/usr/bin/ssh" }, ...o.app });
  for (const s of screens(app)) app.add(s);
  if (o.attach !== false) app.attach(conn);
  const frame = async () => {
    app.render();
    await t.renderOnce();
    return t.captureCharFrame();
  };
  const m: Mounted = {
    t,
    app,
    fake,
    conn,
    frame,
    async press(...keys) {
      for (const k of keys) {
        app.handleKey(typeof k === "string" ? key(k) : k);
        await Bun.sleep(1);
      }
      await Bun.sleep(1);
      return frame();
    },
    async waitFor(text) {
      const ok = typeof text === "string" ? (f: string) => f.includes(text) : text;
      for (let i = 0; i < 200; i++) {
        const f = await frame();
        if (ok(f)) return f;
        await Bun.sleep(10);
      }
      throw new Error(`timed out waiting for ${String(text)}; last frame:\n${await frame()}`);
    },
    async close() {
      app.quit();
      fake.stop();
    },
  };
  return m;
}

export const dummy = (id: string, body: string[] = [`${id} body`]): Screen => ({ id, title: id, view: () => body });

/** recorder is a fake runJumpgate that records each argv and its options. */
export function recorder(code = 0) {
  const calls: { args: string[]; pause?: boolean; detach?: boolean }[] = [];
  const run = async (args: string[], o: { pause?: boolean; detach?: unknown }) => {
    calls.push({ args, ...(o.pause ? { pause: true } : {}), ...(o.detach ? { detach: true } : {}) });
    return code;
  };
  return { calls, run };
}

export const typeText = (m: Mounted, s: string) => {
  for (const c of s) m.app.handleKey(key(c === " " ? "space" : c, { seq: c }));
  return m.frame();
};

/** say types word into a typed confirm and presses Enter. */
export const say = async (m: Mounted, word: string) => {
  await typeText(m, word);
  return m.press("return");
};
