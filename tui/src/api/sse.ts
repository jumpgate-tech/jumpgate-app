// stream(): an SSE stream over the unix socket that reconnects with backoff
// (1, 2, 4, 8 s, capped at 15 s) until its signal aborts. A 4xx other than
// 408/429 ends it as "failed": retrying cannot fix a removed target.
import { ApiError, open, type Conn } from "./http.ts";

export interface RawEvent {
  event: string;
  data: string;
}

export type StreamState = "connecting" | "live" | "retrying" | "failed";

export type SSEMsg<T> =
  | { kind: "event"; event: string; data: T }
  | { kind: "state"; state: StreamState; error?: ApiError; attempt: number };

/** MAX_EVENT bounds the unparsed buffer and one event's data, in characters (1 MiB). */
export const MAX_EVENT = 1 << 20;

export interface SSEParser {
  push(chunk: string): RawEvent[];
  /** overflowed is set once a line or an event outgrew MAX_EVENT; the caller must reconnect. */
  readonly overflowed: boolean;
}

/** parseSSE returns a push parser; an unterminated event is held back. */
export function parseSSE(max = MAX_EVENT): SSEParser {
  let buf = "";
  let pos = 0;
  let name = "";
  let data: string[] = [];
  let size = 0;
  let overflowed = false;
  return {
    get overflowed() {
      return overflowed;
    },
    push(chunk: string): RawEvent[] {
      const out: RawEvent[] = [];
      if (overflowed) return out;
      buf += chunk;
      for (;;) {
        const i = buf.indexOf("\n", pos);
        if (i < 0) break;
        const line = buf.slice(pos, buf[i - 1] === "\r" && i > pos ? i - 1 : i);
        pos = i + 1;
        if (line === "") {
          if (data.length) out.push({ event: name || "message", data: data.join("\n") });
          name = "";
          data = [];
          size = 0;
        } else if (line.startsWith(":")) {
          /* comment */
        } else if (line.startsWith("event:")) {
          name = line.slice(6).trim();
        } else if (line.startsWith("data:")) {
          const d = line.slice(5).replace(/^ /, "");
          size += d.length + 1;
          if (size > max) {
            overflowed = true;
            buf = "";
            pos = 0;
            data = [];
            return out;
          }
          data.push(d);
        }
      }
      if (pos > 0) {
        buf = buf.slice(pos);
        pos = 0;
      }
      if (buf.length > max) {
        overflowed = true;
        buf = "";
        data = [];
      }
      return out;
    },
  };
}

export const backoffMs = (attempt: number) => Math.min(1000 * 2 ** Math.min(attempt, 4), 15000);

const final = (e: unknown) =>
  e instanceof ApiError && e.status >= 400 && e.status < 500 && e.status !== 408 && e.status !== 429;

function sleep(ms: number, signal: AbortSignal): Promise<void> {
  return new Promise((resolve) => {
    if (signal.aborted) return resolve();
    const t = setTimeout(done, ms);
    function done() {
      clearTimeout(t);
      signal.removeEventListener("abort", done);
      resolve();
    }
    signal.addEventListener("abort", done);
  });
}

export interface StreamOpts {
  /** delay before reconnect attempt n, in ms (tests shrink it). */
  delay?: (attempt: number) => number;
  /** max overrides MAX_EVENT (tests). */
  max?: number;
}

export async function* stream<T = unknown>(
  conn: Conn,
  path: string,
  signal: AbortSignal,
  opts: StreamOpts = {},
): AsyncGenerator<SSEMsg<T>> {
  const delay = opts.delay ?? backoffMs;
  let attempt = 0;
  yield { kind: "state", state: "connecting", attempt };
  while (!signal.aborted) {
    let error: ApiError | undefined;
    try {
      const res = await open(conn, "GET", path, undefined, signal, { accept: "text/event-stream" });
      yield { kind: "state", state: "live", attempt };
      const parser = parseSSE(opts.max);
      const dec = new TextDecoder();
      const reader = res.body!.getReader();
      try {
        for (;;) {
          const { done, value } = await reader.read();
          if (done) break;
          const evs = parser.push(dec.decode(value, { stream: true }));
          if (parser.overflowed) {
            await reader.cancel().catch(() => {});
            throw new ApiError("a stream event was larger than 1 MiB", 0, "too_large");
          }
          for (const ev of evs) {
            let data: T;
            try {
              data = JSON.parse(ev.data) as T;
            } catch {
              continue; // a frame we cannot read is skipped
            }
            attempt = 0;
            yield { kind: "event", event: ev.event, data };
          }
        }
      } finally {
        reader.releaseLock();
      }
    } catch (e) {
      if (signal.aborted) return;
      if (final(e)) {
        yield { kind: "state", state: "failed", error: e as ApiError, attempt };
        return;
      }
      error = e instanceof ApiError ? e : new ApiError("the stream dropped", 0, "unreachable");
    }
    if (signal.aborted) return;
    yield { kind: "state", state: "retrying", error, attempt };
    await sleep(delay(attempt++), signal);
  }
}
