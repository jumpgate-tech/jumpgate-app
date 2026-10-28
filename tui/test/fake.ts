// A fake jumpgate server on a temp unix socket, for client and screen tests.
import { mkdtempSync, rmSync } from "node:fs";
import { tmpdir } from "node:os";
import { join } from "node:path";

export interface Recorded {
  method: string;
  path: string;
  headers: Headers;
  body: string;
}

export type Handler = (req: Request, rec: Recorded) => Response | Promise<Response>;

export interface Fake {
  socket: string;
  dir: string;
  requests: Recorded[];
  routes: Record<string, Handler>;
  stop(): void;
}

export const json = (v: unknown, status = 200) =>
  new Response(JSON.stringify(v), { status, headers: { "content-type": "application/json" } });

/** fakeServer serves routes keyed by "METHOD /path" (exact path, no query). */
export function fakeServer(routes: Record<string, Handler> = {}): Fake {
  const dir = mkdtempSync(join(tmpdir(), "jgtui-"));
  const socket = join(dir, "s.sock");
  const requests: Recorded[] = [];
  const fake: Fake = { socket, dir, requests, routes, stop: () => {} };
  const server = Bun.serve({
    unix: socket,
    async fetch(req) {
      const u = new URL(req.url);
      const rec: Recorded = { method: req.method, path: u.pathname + u.search, headers: req.headers, body: await req.text() };
      requests.push(rec);
      const h = fake.routes[`${req.method} ${u.pathname}`];
      if (!h) return json({ error: "no such route", code: "not_found" }, 404);
      return h(req, rec);
    },
  });
  fake.stop = () => {
    server.stop(true);
    rmSync(dir, { recursive: true, force: true });
  };
  return fake;
}

/** sse builds a streaming response from frames; it stays open until closed. */
export function sseResponse(frames: string[], keepOpen = false): Response {
  const enc = new TextEncoder();
  const stream = new ReadableStream({
    start(c) {
      for (const f of frames) c.enqueue(enc.encode(f));
      if (!keepOpen) c.close();
    },
  });
  return new Response(stream, { headers: { "content-type": "text/event-stream" } });
}

/** caught resolves to what p rejects with, and fails if p resolves. */
export const caught = (p: Promise<unknown>): Promise<any> =>
  p.then(
    () => {
      throw new Error("expected a rejection");
    },
    (e) => e,
  );
