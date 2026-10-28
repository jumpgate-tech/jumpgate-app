// jsonLines shows a server reply as indented "key: value" lines. Keys are
// sanitized; string values are sanitized and masked with maskText, the way
// the Go TUI masks server text, except RFC 3339 times, which carry no secret.
import { maskText } from "../maskurl.ts";
import { sanitizeLine } from "../text.ts";

const TIME = /^\d{4}-\d\d-\d\dT\d\d:\d\d:\d\d(\.\d+)?(Z|[+-]\d\d:\d\d)$/;

export const maskValue = (s: string) => (TIME.test(s) ? s : maskText(sanitizeLine(s)));

function scalar(v: unknown): string {
  if (typeof v === "string") return maskValue(v);
  if (v === null) return "null";
  return String(v);
}

export function jsonLines(v: unknown, indent = ""): string[] {
  if (v === undefined) return [`${indent}(no body)`];
  if (Array.isArray(v)) {
    if (!v.length) return [`${indent}[]`];
    return v.flatMap((x) => (x !== null && typeof x === "object" ? [`${indent}-`, ...jsonLines(x, indent + "  ")] : [`${indent}- ${scalar(x)}`]));
  }
  if (v !== null && typeof v === "object") {
    const entries = Object.entries(v as Record<string, unknown>);
    if (!entries.length) return [`${indent}{}`];
    return entries.flatMap(([k, x]) => {
      const key = sanitizeLine(k);
      return x !== null && typeof x === "object" ? [`${indent}${key}:`, ...jsonLines(x, indent + "  ")] : [`${indent}${key}: ${scalar(x)}`];
    });
  }
  return [indent + scalar(v)];
}
