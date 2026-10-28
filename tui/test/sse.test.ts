import { afterEach, expect, test } from "bun:test";
import { backoffMs, parseSSE, stream, type SSEMsg } from "../src/api/sse.ts";
import { fakeServer, json, sseResponse, type Fake } from "./fake.ts";

let f: Fake | undefined;
afterEach(() => f?.stop());

test("parseSSE handles frames split across chunks, comments and named events", () => {
  const p = parseSSE();
  const out = [
    ...p.push(": ping\n\nevent: res"),
    ...p.push('et\ndata: [1,\ndata: 2]\n\nda'),
    ...p.push('ta: {"a":1}\r\n\r\n'),
    ...p.push("data: cut"),
  ];
  expect(out).toEqual([
    { event: "reset", data: "[1,\n2]" },
    { event: "message", data: '{"a":1}' },
  ]);
});

test("backoff is 1, 2, 4, 8 s, capped at 15 s", () => {
  expect([0, 1, 2, 3, 4, 9].map(backoffMs)).toEqual([1000, 2000, 4000, 8000, 15000, 15000]);
});

test("stream decodes events, reconnects after a drop, and stops on abort", async () => {
  let n = 0;
  f = fakeServer({ "GET /api/fleet/stream": () => sseResponse([`data: {"n":${++n}}\n\n`]) });
  const ac = new AbortController();
  const seen: SSEMsg<{ n: number }>[] = [];
  for await (const m of stream<{ n: number }>({ socket: f.socket, token: "t", system: false }, "/api/fleet/stream", ac.signal, { delay: () => 5 })) {
    seen.push(m);
    if (m.kind === "event" && m.data.n === 2) ac.abort();
  }
  const events = seen.filter((m) => m.kind === "event").map((m) => (m.kind === "event" ? m.data.n : 0));
  expect(events).toEqual([1, 2]);
  expect(seen.some((m) => m.kind === "state" && m.state === "retrying")).toBe(true);
  expect(f.requests[0]!.headers.get("authorization")).toBe("Bearer t");
});

test("a 4xx ends the stream as failed, with no retry", async () => {
  f = fakeServer({ "GET /api/fleet/x/status/stream": () => json({ error: "no such box", code: "target_not_found" }, 404) });
  const seen: SSEMsg<unknown>[] = [];
  for await (const m of stream(f && { socket: f.socket, token: null, system: true }, "/api/fleet/x/status/stream", new AbortController().signal, { delay: () => 1 })) seen.push(m);
  const last = seen.at(-1);
  expect(last?.kind === "state" && last.state).toBe("failed");
  expect(f.requests.length).toBe(1);
});
