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

test("an oversized event is dropped and the parser flags overflow", () => {
  const p = parseSSE(100);
  expect(p.push(`data: ${"x".repeat(50)}\n\n`).length).toBe(1);
  expect(p.push(`data: ${"x".repeat(60)}\ndata: ${"y".repeat(60)}\n\n`)).toEqual([]);
  expect(p.overflowed).toBe(true);
  const q = parseSSE(100);
  q.push("data: " + "z".repeat(200)); // no newline: the buffer itself overflows
  expect(q.overflowed).toBe(true);
});

test("many small events in one chunk parse with a moving offset", () => {
  const p = parseSSE();
  const chunk = Array.from({ length: 5000 }, (_, i) => `data: ${i}\n\n`).join("");
  expect(p.push(chunk).length).toBe(5000);
});

test("stream: an oversized event reconnects; the next connection's events arrive", async () => {
  let n = 0;
  f = fakeServer({ "GET /api/jobs/stream": () => sseResponse([n++ ? `data: {"ok":true}\n\n` : `data: "${"x".repeat(500)}"\n\n`]) });
  const ac = new AbortController();
  const seen: SSEMsg<any>[] = [];
  for await (const m of stream<any>({ socket: f.socket, token: null, system: true }, "/api/jobs/stream", ac.signal, { delay: () => 5, max: 100 })) {
    seen.push(m);
    if (m.kind === "event") ac.abort();
  }
  expect(seen.some((m) => m.kind === "state" && m.state === "retrying" && m.error?.code === "too_large")).toBe(true);
  expect(seen.filter((m) => m.kind === "event").map((m) => (m as any).data)).toEqual([{ ok: true }]);
});

import { pair } from "../src/api/client.ts";
test("pair: an oversized event ends the stream with an error, not a crash", async () => {
  f = fakeServer({ "POST /api/targets/b/pair": () => sseResponse([`data: {"line":"${"x".repeat(500)}"}\n\n`, `data: {"done":true}\n\n`]) });
  const evs = [];
  for await (const e of pair({ socket: f.socket, token: null, system: true }, "b", { sudo: false }, undefined, 100)) evs.push(e);
  expect(evs).toEqual([{ error: "the pairing stream sent an event larger than 1 MiB; it was dropped", code: "too_large" }]);
});
