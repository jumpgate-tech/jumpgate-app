// Pure derivations for the Logs section — kept apart from LogsSection.tsx so
// they're testable without rendering. Mirrors logs.ts's own constants and
// inline helpers: the severity fallback, the error-feed filter, the
// "Explain with AI" candidate-line selection, and its once-per-browser
// localStorage consent gate.
import type { Hit } from "../../api";

// maxRenderedLines caps how many tail rows stay in memory — the server's own
// ring buffer already caps history (internal/logwatch: 1000), this just
// keeps the live view from growing unbounded over a long session.
export const maxRenderedLines = 500;

// maxExplainLines mirrors handleExplain's own default cap
// (maxDefaultExplainHits in internal/server/api.go) so the consent modal
// shows exactly what would be auto-selected server-side, capped the same
// way.
export const maxExplainLines = 40;

const CONSENT_KEY = "valve-node-app.explain-consent";

export type Severity = "info" | "warn" | "error" | "critical";

// severityOf falls back to "info" exactly like logs.ts's `(h.severity as
// Severity) || "info"` — an empty/unknown severity string reads as info
// rather than crashing the badge/class lookup.
export function severityOf(hit: Hit): Severity {
  return (hit.severity as Severity) || "info";
}

export function isErrorHit(hit: Hit): boolean {
  const sev = severityOf(hit);
  return sev === "error" || sev === "critical";
}

// Generic so a caller passing a richer row type (e.g. useLogStream's LogHit,
// which carries a stable `_key`) gets that same type back — the error-feed
// list needs the `_key` to key on, exactly like the tail.
export function errorHits<T extends Hit>(hits: T[]): T[] {
  return hits.filter(isErrorHit);
}

// explainCandidateLines is the excerpt the consent modal shows, and the
// lines sent when the operator already consented: the most recent
// error/critical lines (oldest-to-newest), capped at maxExplainLines —
// mirrors openExplainFlow's `hits.filter(...).map((h) => h.line).slice(-40)`.
export function explainCandidateLines(hits: Hit[], limit = maxExplainLines): string[] {
  return errorHits(hits)
    .map((h) => h.line)
    .slice(-limit);
}

export function formatLogTime(at: string): string {
  return new Date(at).toLocaleTimeString();
}

// hasExplainConsent / setExplainConsent wrap the once-per-browser
// localStorage gate. Both swallow storage errors (private browsing, quota,
// disabled storage): a failed read just means the consent modal shows again,
// which is the safe direction for a consent gate to fail in.
export function hasExplainConsent(): boolean {
  try {
    return localStorage.getItem(CONSENT_KEY) === "1";
  } catch {
    return false;
  }
}

export function setExplainConsent(): void {
  try {
    localStorage.setItem(CONSENT_KEY, "1");
  } catch {
    // Storage unavailable — see hasExplainConsent's comment above.
  }
}
