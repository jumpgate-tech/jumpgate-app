import { describe, it, expect, beforeEach } from "vitest";
import type { Hit } from "../../api";
import {
  errorHits,
  explainCandidateLines,
  formatLogTime,
  hasExplainConsent,
  isErrorHit,
  maxExplainLines,
  setExplainConsent,
  severityOf,
} from "./logsModel";

function hit(overrides: Partial<Hit> = {}): Hit {
  return {
    unit: "exec",
    line: "boom",
    at: "2026-08-01T12:00:00Z",
    signature: "sig",
    severity: "info",
    explain: "",
    ...overrides,
  };
}

describe("severityOf", () => {
  it("returns the hit's severity", () => {
    expect(severityOf(hit({ severity: "warn" }))).toBe("warn");
  });

  it("falls back to info for an empty severity", () => {
    expect(severityOf(hit({ severity: "" }))).toBe("info");
  });
});

describe("isErrorHit / errorHits", () => {
  it("treats error and critical as errors, and info/warn as not", () => {
    expect(isErrorHit(hit({ severity: "error" }))).toBe(true);
    expect(isErrorHit(hit({ severity: "critical" }))).toBe(true);
    expect(isErrorHit(hit({ severity: "warn" }))).toBe(false);
    expect(isErrorHit(hit({ severity: "info" }))).toBe(false);
  });

  it("filters a mixed list down to error/critical hits, preserving order", () => {
    const hits = [
      hit({ line: "a", severity: "info" }),
      hit({ line: "b", severity: "error" }),
      hit({ line: "c", severity: "warn" }),
      hit({ line: "d", severity: "critical" }),
    ];
    expect(errorHits(hits).map((h) => h.line)).toEqual(["b", "d"]);
  });
});

describe("explainCandidateLines", () => {
  it("returns only error/critical lines", () => {
    const hits = [hit({ line: "info-line", severity: "info" }), hit({ line: "err-line", severity: "error" })];
    expect(explainCandidateLines(hits)).toEqual(["err-line"]);
  });

  it("caps at maxExplainLines, keeping the most recent", () => {
    const hits = Array.from({ length: maxExplainLines + 10 }, (_, i) => hit({ line: `e${i}`, severity: "error" }));
    const lines = explainCandidateLines(hits);
    expect(lines).toHaveLength(maxExplainLines);
    expect(lines[0]).toBe(`e10`);
    expect(lines[lines.length - 1]).toBe(`e${maxExplainLines + 9}`);
  });

  it("respects a custom limit", () => {
    const hits = Array.from({ length: 5 }, (_, i) => hit({ line: `e${i}`, severity: "error" }));
    expect(explainCandidateLines(hits, 2)).toEqual(["e3", "e4"]);
  });

  it("returns an empty array when there are no error/critical hits", () => {
    expect(explainCandidateLines([hit({ severity: "info" })])).toEqual([]);
  });
});

describe("formatLogTime", () => {
  it("formats an ISO timestamp using toLocaleTimeString", () => {
    const at = "2026-08-01T12:00:00Z";
    expect(formatLogTime(at)).toBe(new Date(at).toLocaleTimeString());
  });
});

describe("hasExplainConsent / setExplainConsent", () => {
  beforeEach(() => {
    localStorage.clear();
  });

  it("is false before consent is ever set", () => {
    expect(hasExplainConsent()).toBe(false);
  });

  it("becomes true once setExplainConsent is called", () => {
    setExplainConsent();
    expect(hasExplainConsent()).toBe(true);
    expect(localStorage.getItem("valve-node-app.explain-consent")).toBe("1");
  });
});
