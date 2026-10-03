// useLogStream — the Logs section's data layer: the initial recent-hits
// fetch (api.getLogs) followed by the live tail (api.streamLogs), merged
// into one capped-length array. This is a dedicated hook file — not
// hooks/gateway.ts or hooks/target.ts — both to keep this ring-buffer merge
// logic (nothing else in this app does it) out of those files, and so this
// section can land standalone while those files are mid-migration by other
// work.
//
// It mirrors logs.ts's own init(): fetch up to 200 recent hits first, THEN
// open the stream, so no live hit can arrive before the seed is in place.
// An error from the initial fetch stops there (mirrors init()'s own early
// return on a failed getLogs) — the stream is never opened. Every hit after
// that, seed or live, is capped at maxRenderedLines the same way logs.ts's
// `hits.splice(0, hits.length - maxRenderedLines)` was.
import { useEffect, useRef, useState } from "react";
import * as api from "../api";
import { maxRenderedLines } from "../screens/Machine/logsModel";

// LogHit is a wire Hit plus a client-assigned, monotonically increasing
// `_key`. api.Hit.signature is NOT unique per line — it is a classification
// pattern NAME (empty "" for unclassified lines; see internal/logwatch), so it
// cannot key a React list. The array index can't either: the tail appends
// every stream frame and front-splices at the 500-line cap, so an index churns
// on every new line and React remounts the whole tail. `_key` is stable per
// surviving row across that splice and unique across the list.
export type LogHit = api.Hit & { _key: number };

export interface UseLogStreamResult {
  hits: LogHit[];
  loading: boolean;
  error: string | null;
}

// useLogStream fetches+streams targetId's logs whenever enabled is true.
// The Logs section only enables this once the target has completed setup —
// mirroring logs.ts's own `if (!target.wire)` guard, which stays the
// CALLER's responsibility, not this hook's. Disabling (or changing
// targetId) tears down any open stream and resets to the empty state.
export function useLogStream(targetId: string, enabled: boolean): UseLogStreamResult {
  const [hits, setHits] = useState<LogHit[]>([]);
  const [loading, setLoading] = useState(enabled);
  const [error, setError] = useState<string | null>(null);
  // Monotonic id source for `_key`. Never reset — ids only need to be unique
  // within the current list, and letting it grow across targetId/enabled
  // resets keeps every assigned key globally distinct with no bookkeeping.
  const nextKey = useRef(0);

  useEffect(() => {
    if (!enabled || !targetId) {
      setHits([]);
      setLoading(false);
      setError(null);
      return;
    }

    let disposed = false;
    let stop: (() => void) | null = null;
    setHits([]);
    setError(null);
    setLoading(true);

    void (async () => {
      let recent: api.Hit[];
      try {
        recent = await api.getLogs(targetId, 200);
      } catch (err) {
        if (disposed) return;
        setError(err instanceof Error ? err.message : String(err));
        setLoading(false);
        return;
      }
      if (disposed) return;
      setHits(recent.slice(-maxRenderedLines).map((h) => ({ ...h, _key: nextKey.current++ })));
      setLoading(false);

      stop = api.streamLogs(targetId, (hit) => {
        if (disposed) return;
        setHits((prev) => {
          const next = [...prev, { ...hit, _key: nextKey.current++ }];
          return next.length > maxRenderedLines ? next.slice(next.length - maxRenderedLines) : next;
        });
      });
    })();

    return () => {
      disposed = true;
      stop?.();
    };
  }, [targetId, enabled]);

  return { hits, loading, error };
}
