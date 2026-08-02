import { describe, it, expect } from "vitest";
import type { GatewayAnalytics, Latency } from "../../api";
import {
  appendReading,
  bandLabel,
  distributionBands,
  fmtSeconds,
  HISTORY,
  rateFor,
  rateWindowSeconds,
  sinceLabel,
  slowestLabel,
  sparklinePoints,
  type Reading,
} from "./analyticsModel";

function reading(t: number, since: string, received: Record<number, number>): Reading {
  return { t, since, received: new Map(Object.entries(received).map(([k, v]) => [Number(k), v])) };
}

function scrape(overrides: Partial<GatewayAnalytics> & { networks?: GatewayAnalytics["networks"] }): GatewayAnalytics {
  return { enabled: true, at: "now", since: "s1", networks: [], endpoints: [], ...overrides };
}

describe("appendReading", () => {
  it("drops the reading when the gateway isn't counting (enabled: false)", () => {
    const next = scrape({ enabled: false, networks: [{ chainId: 1, name: "eth", received: 5, answered: 5, unattributed: 0, failed: 0, methods: [], endpoints: [], cached: { count: 0, mean: null, buckets: [] }, failedLatency: { count: 0, mean: null, buckets: [] } }] });
    expect(appendReading([], next, 1000)).toEqual([]);
  });

  it("drops the reading when the scrape carries an error", () => {
    const next = scrape({ error: "boom" });
    expect(appendReading([], next, 1000)).toEqual([]);
  });

  it("appends a reading built from each network's received count", () => {
    const next = scrape({
      since: "s1",
      networks: [
        { chainId: 1, name: "eth", received: 10, answered: 10, unattributed: 0, failed: 0, methods: [], endpoints: [], cached: { count: 0, mean: null, buckets: [] }, failedLatency: { count: 0, mean: null, buckets: [] } },
        { chainId: 2, name: "pls", received: 3, answered: 3, unattributed: 0, failed: 0, methods: [], endpoints: [], cached: { count: 0, mean: null, buckets: [] }, failedLatency: { count: 0, mean: null, buckets: [] } },
      ],
    });
    const out = appendReading([], next, 5000);
    expect(out).toHaveLength(1);
    expect(out[0]!.t).toBe(5000);
    expect(out[0]!.since).toBe("s1");
    expect(out[0]!.received.get(1)).toBe(10);
    expect(out[0]!.received.get(2)).toBe(3);
  });

  it("throws the history away when `since` changes (a restart)", () => {
    const history = [reading(1000, "s1", { 1: 10 })];
    const next = scrape({ since: "s2", networks: [{ chainId: 1, name: "eth", received: 1, answered: 1, unattributed: 0, failed: 0, methods: [], endpoints: [], cached: { count: 0, mean: null, buckets: [] }, failedLatency: { count: 0, mean: null, buckets: [] } }] });
    const out = appendReading(history, next, 2000);
    expect(out).toHaveLength(1);
    expect(out[0]!.since).toBe("s2");
    expect(out[0]!.received.get(1)).toBe(1);
  });

  it("caps the kept history at HISTORY, dropping the oldest", () => {
    const history: Reading[] = Array.from({ length: HISTORY }, (_, i) => reading(i, "s1", { 1: i }));
    const next = scrape({ since: "s1", networks: [{ chainId: 1, name: "eth", received: 999, answered: 0, unattributed: 0, failed: 0, methods: [], endpoints: [], cached: { count: 0, mean: null, buckets: [] }, failedLatency: { count: 0, mean: null, buckets: [] } }] });
    const out = appendReading(history, next, HISTORY);
    expect(out).toHaveLength(HISTORY);
    expect(out[0]!.t).toBe(1); // the oldest (t=0) was dropped
    expect(out[out.length - 1]!.received.get(1)).toBe(999);
  });

  it("never mutates the input history array", () => {
    const history = [reading(1000, "s1", { 1: 10 })];
    const frozen = Object.freeze([...history]);
    const next = scrape({ since: "s1", networks: [] });
    expect(() => appendReading(frozen, next, 2000)).not.toThrow();
    expect(frozen).toHaveLength(1);
  });
});

describe("rateFor", () => {
  it("is null with fewer than two readings", () => {
    expect(rateFor([], 1)).toBeNull();
    expect(rateFor([reading(1000, "s1", { 1: 5 })], 1)).toBeNull();
  });

  it("is null when the two readings share a timestamp", () => {
    const history = [reading(1000, "s1", { 1: 5 }), reading(1000, "s1", { 1: 10 })];
    expect(rateFor(history, 1)).toBeNull();
  });

  it("is null when the delta is negative", () => {
    const history = [reading(1000, "s1", { 1: 20 }), reading(2000, "s1", { 1: 5 })];
    expect(rateFor(history, 1)).toBeNull();
  });

  it("computes requests/second across the full kept span", () => {
    const history = [reading(0, "s1", { 1: 0 }), reading(10_000, "s1", { 1: 100 })];
    expect(rateFor(history, 1)).toBe(10);
  });

  it("treats a chain missing from a reading as 0 received", () => {
    const history = [reading(0, "s1", {}), reading(1000, "s1", { 1: 5 })];
    expect(rateFor(history, 1)).toBe(5);
  });
});

describe("rateWindowSeconds", () => {
  it("is 0 with fewer than two readings", () => {
    expect(rateWindowSeconds([])).toBe(0);
  });

  it("rounds the span between the first and last reading", () => {
    const history = [reading(0, "s1", {}), reading(4_600, "s1", {})];
    expect(rateWindowSeconds(history)).toBe(5);
  });
});

describe("sparklinePoints", () => {
  it("is empty with fewer than three readings", () => {
    expect(sparklinePoints([reading(0, "s1", {}), reading(1000, "s1", {})], 1)).toEqual([]);
  });

  it("computes one per-interval rate per gap", () => {
    const history = [
      reading(0, "s1", { 1: 0 }),
      reading(1000, "s1", { 1: 5 }),
      reading(2000, "s1", { 1: 15 }),
    ];
    expect(sparklinePoints(history, 1)).toEqual([5, 10]);
  });

  it("treats a negative interval delta as 0 rather than a negative rate", () => {
    const history = [
      reading(0, "s1", { 1: 10 }),
      reading(1000, "s1", { 1: 2 }),
      reading(2000, "s1", { 1: 12 }),
    ];
    expect(sparklinePoints(history, 1)).toEqual([0, 10]);
  });
});

describe("fmtSeconds", () => {
  it.each([
    [-1, "—"],
    [NaN, "—"],
    [0.0001, "<1ms"],
    [0.076, "76ms"],
    [0.5, "500ms"],
    [1, "1.0s"],
    [9.99, "10.0s"],
    [10, "10s"],
    [59, "59s"],
    [65, "1m"],
    [125, "2m"],
  ])("fmtSeconds(%p) -> %p", (input, expected) => {
    expect(fmtSeconds(input)).toBe(expected);
  });
});

describe("bandLabel", () => {
  it("names +Inf as the top band", () => {
    expect(bandLabel("+Inf")).toBe("30s or more");
  });

  it("names a numeric bound via fmtSeconds", () => {
    expect(bandLabel("0.05")).toBe("under 50ms");
    expect(bandLabel("30")).toBe("under 30s");
  });

  it("falls back to the raw string when it doesn't parse as a number", () => {
    expect(bandLabel("nope")).toBe("under nope");
  });
});

describe("distributionBands", () => {
  const BUCKETS = [
    { le: "0.05", count: 2 },
    { le: "0.5", count: 5 },
    { le: "5", count: 5 },
    { le: "30", count: 5 },
    { le: "+Inf", count: 6 },
  ];

  it("is empty when there are no buckets", () => {
    const l: Latency = { count: 0, mean: null, buckets: [] };
    expect(distributionBands(l)).toEqual([]);
  });

  it("is empty when count is 0 even if buckets are present", () => {
    const l: Latency = { count: 0, mean: null, buckets: BUCKETS };
    expect(distributionBands(l)).toEqual([]);
  });

  it("turns cumulative buckets into per-band counts", () => {
    const l: Latency = { count: 6, mean: 0.02, buckets: BUCKETS };
    expect(distributionBands(l)).toEqual([
      { label: "under 50ms", n: 2 },
      { label: "under 500ms", n: 3 },
      { label: "under 5.0s", n: 0 },
      { label: "under 30s", n: 0 },
      { label: "30s or more", n: 1 },
    ]);
  });
});

describe("slowestLabel", () => {
  it("is empty for no bands", () => {
    expect(slowestLabel([])).toBe("");
  });

  it("names the last non-empty band", () => {
    expect(
      slowestLabel([
        { label: "under 50ms", n: 2 },
        { label: "under 500ms", n: 0 },
        { label: "30s or more", n: 0 },
      ]),
    ).toBe("slowest under 50ms");
  });

  it("prefers a later non-empty band over an earlier one", () => {
    expect(
      slowestLabel([
        { label: "under 50ms", n: 2 },
        { label: "30s or more", n: 1 },
      ]),
    ).toBe("slowest 30s or more");
  });
});

describe("sinceLabel", () => {
  it("is null for an empty string", () => {
    expect(sinceLabel("")).toBeNull();
  });

  it("is null for an unparseable date", () => {
    expect(sinceLabel("not-a-date")).toBeNull();
  });

  it("formats a valid ISO timestamp", () => {
    const label = sinceLabel("2026-08-01T12:00:00Z");
    expect(typeof label).toBe("string");
    expect(label).not.toBe("");
  });
});
