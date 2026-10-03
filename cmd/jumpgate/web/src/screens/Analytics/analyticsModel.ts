// Pure derivations for the Analytics screen (#/analytics/:id) — kept apart
// from the components so the rate/sparkline/distribution math is testable
// without rendering. Ported straight from analytics.ts's remember/rateFor/
// sparkline/distribution/bandLabel/slowestLabel/fmtSeconds, which is the
// behavioral spec this file must match exactly.
import type { Bucket, GatewayAnalytics, Latency } from "../../api";

// POLL_MS is the read cadence passed to useGatewayHealth's refetchInterval.
// Each poll is one curl on the gateway's own machine — cheap, but it's a
// command on a possibly-remote box, so this is not a second-by-second screen.
export const POLL_MS = 5000;

// HISTORY is how many readings the sparkline/rate keep: 60 x 5s ~= five
// minutes — long enough to watch a deploy or an incident develop, and short
// enough that the memory is trivially bounded.
export const HISTORY = 60;

// A reading kept for the rate calculation: when it was taken, and the
// per-chain received counter at that moment.
export interface Reading {
  t: number;
  since: string;
  received: Map<number, number>;
}

// appendReading is remember() from analytics.ts, made pure: given the
// current kept history and a freshly-polled scrape (plus the clock reading
// for it), returns the NEXT history — the input is never mutated.
//
// A reading is dropped entirely when the gateway isn't counting or its
// counters couldn't be read — those don't belong in a rate. A restart (the
// process's `since` changing) throws the whole history away rather than
// diffing across it: every counter resets to zero on restart, so a diff
// across that instant isn't a rate — it's either the whole previous
// process's traffic read as a spike, or the new process's first requests
// read as a plummet from a number that no longer exists.
export function appendReading(history: readonly Reading[], next: GatewayAnalytics, now: number): Reading[] {
  if (!next.enabled || next.error) return [...history];
  const last = history[history.length - 1];
  const base = last && last.since !== next.since ? [] : history;
  const received = new Map<number, number>();
  for (const n of next.networks ?? []) received.set(n.chainId, n.received);
  const updated = [...base, { t: now, since: next.since, received }];
  return updated.length > HISTORY ? updated.slice(updated.length - HISTORY) : updated;
}

// rateFor is requests/second for one chain across the WHOLE kept history —
// null until there are two readings to diff, or if the delta somehow went
// negative (defensive: appendReading already resets on a `since` change, so
// this mainly guards against a counter that dropped without one).
export function rateFor(history: readonly Reading[], chainId: number): number | null {
  if (history.length < 2) return null;
  const first = history[0]!;
  const last = history[history.length - 1]!;
  const secs = (last.t - first.t) / 1000;
  if (secs <= 0) return null;
  const delta = (last.received.get(chainId) ?? 0) - (first.received.get(chainId) ?? 0);
  if (delta < 0) return null;
  return delta / secs;
}

// rateWindowSeconds is the span rateFor's number covers — the "over the
// last Ns" qualifier shown beside it. 0 before there are two readings.
export function rateWindowSeconds(history: readonly Reading[]): number {
  if (history.length < 2) return 0;
  return Math.round((history[history.length - 1]!.t - history[0]!.t) / 1000);
}

// sparklinePoints is the per-interval rate across the kept history — one
// point per gap between consecutive readings. Empty before there are three
// readings (two gaps), the same threshold analytics.ts's sparkline() used to
// decide there's nothing worth drawing yet.
export function sparklinePoints(history: readonly Reading[], chainId: number): number[] {
  if (history.length < 3) return [];
  const points: number[] = [];
  for (let i = 1; i < history.length; i++) {
    const a = history[i - 1]!;
    const b = history[i]!;
    const secs = (b.t - a.t) / 1000;
    const delta = (b.received.get(chainId) ?? 0) - (a.received.get(chainId) ?? 0);
    points.push(secs > 0 && delta >= 0 ? delta / secs : 0);
  }
  return points;
}

// fmtSeconds renders a duration given in seconds the way a person reading a
// latency would say it: milliseconds under a second, seconds above.
//
// Anything under half a millisecond is "<1ms", not "0ms". FOUND BY RUNNING
// IT: a chain's cached eth_chainId averaged 76µs and rendered as "0ms" — the
// same lie this function refuses elsewhere by sending a null mean, arrived at
// by rounding instead. Zero milliseconds is not a duration anything takes.
export function fmtSeconds(s: number): string {
  if (!Number.isFinite(s) || s < 0) return "—";
  if (s > 0 && s < 0.0005) return "<1ms";
  if (s < 1) return `${Math.round(s * 1000)}ms`;
  if (s < 60) return `${s < 10 ? s.toFixed(1) : Math.round(s)}s`;
  return `${Math.round(s / 60)}m`;
}

// bandLabel names a cumulative bucket's upper bound in words. eRPC's bucket
// bounds are fixed: 0.05, 0.5, 5 and 30 seconds, then "+Inf".
export function bandLabel(le: string): string {
  if (le === "+Inf") return "30s or more";
  const secs = Number(le);
  if (!Number.isFinite(secs)) return `under ${le}`;
  return `under ${fmtSeconds(secs)}`;
}

export interface Band {
  label: string;
  n: number;
}

// distributionBands turns a latency's cumulative buckets into the discrete
// bands they describe — the count that finished in EACH range, not up to it.
//
// The bands are what gets drawn, and no percentile is: eRPC's bucket bounds
// are neighbours an order of magnitude apart, so a p95 interpolated between
// two of them would be a number this page invented. "5 of 5 answered within
// 500ms" is the same fact, and it's what the buckets literally say.
//
// Returns [] both when there's nothing to band (no buckets, or the latency
// never counted anything) and when the buckets exist but every band comes
// out empty — callers show the same "—" for either.
export function distributionBands(l: Latency): Band[] {
  const buckets: Bucket[] = l.buckets ?? [];
  if (buckets.length === 0 || l.count === 0) return [];
  let prev = 0;
  const bands: Band[] = [];
  for (const b of buckets) {
    const n = b.count - prev;
    prev = b.count;
    bands.push({ label: bandLabel(b.le), n: Math.max(0, n) });
  }
  return bands.some((b) => b.n > 0) ? bands : [];
}

// slowestLabel names the slowest band that actually holds anything, because
// that's the fact worth reading at a glance: a row whose worst band is
// "under 50ms" needs no further attention, and one whose worst is "30s or
// more" needs nothing else said about it.
export function slowestLabel(bands: readonly Band[]): string {
  for (let i = bands.length - 1; i >= 0; i--) {
    if (bands[i]!.n > 0) return `slowest ${bands[i]!.label}`;
  }
  return "";
}

// sinceLabel formats the gateway's process-start time for the "totals
// since…" note, or null when `since` is empty or doesn't parse to a valid
// date — the caller falls back to a note with no timestamp in it rather than
// print "Invalid Date".
export function sinceLabel(since: string): string | null {
  if (!since) return null;
  const d = new Date(since);
  return Number.isNaN(d.getTime()) ? null : d.toLocaleString();
}
