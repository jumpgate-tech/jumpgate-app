// The small colored status pill — the React equivalent of ui.ts's badge()
// string helper. JSX escapes `text` itself, so callers don't need escapeHtml.
export type BadgeKind = "ok" | "bad" | "warn" | "neutral";

export function Badge({ text, kind }: { text: string; kind: BadgeKind }) {
  return <span className={`badge badge-${kind}`}>{text}</span>;
}
