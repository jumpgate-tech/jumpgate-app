// The panel's inline SVG icon set (currentColor stroke) — cross-platform, no
// SF Symbols. Ported verbatim from panel.ts's SPRITE + ic() helper: <Sprite/>
// mounts the hidden <defs> of <symbol>s once per screen, and <Icon name/>
// renders one <svg class="p-i"><use href="#p-<name>"/></svg> referencing it —
// the JSX equivalent of the old ic() string, styled by panel.css's .p-i.
import "../../panel.css";
import type { SVGProps } from "react";

export type IconName =
  | "power"
  | "globe"
  | "ws"
  | "archive"
  | "trace"
  | "lock"
  | "pencil"
  | "trash"
  | "copy"
  | "scale"
  | "refresh"
  | "chevR"
  | "chevL"
  | "plus"
  | "gear";

// Sprite is the hidden <svg> holding every symbol. Render it ONCE near the top
// of a screen (Panel does); every <Icon> below references these by id.
export function Sprite() {
  return (
    <svg width="0" height="0" style={{ position: "absolute" }} aria-hidden="true">
      <defs>
        <symbol id="p-power" viewBox="0 0 24 24">
          <line x1="12" y1="3.5" x2="12" y2="11.5" />
          <path d="M7.5 7a7 7 0 1 0 9 0" />
        </symbol>
        <symbol id="p-globe" viewBox="0 0 24 24">
          <circle cx="12" cy="12" r="8.5" />
          <path d="M3.5 12h17M12 3.5c2.5 2.4 2.5 14.6 0 17M12 3.5c-2.5 2.4-2.5 14.6 0 17" />
        </symbol>
        <symbol id="p-ws" viewBox="0 0 24 24">
          <path d="M4 9h13l-3.5-3.5M20 15H7l3.5 3.5" />
        </symbol>
        <symbol id="p-archive" viewBox="0 0 24 24">
          <path d="M12 3 3 7.5l9 4.5 9-4.5L12 3ZM3 12l9 4.5 9-4.5M3 16.5 12 21l9-4.5" />
        </symbol>
        <symbol id="p-trace" viewBox="0 0 24 24">
          <circle cx="12" cy="12" r="5.5" />
          <path d="M12 1.5v3M12 19.5v3M1.5 12h3M19.5 12h3" />
        </symbol>
        <symbol id="p-lock" viewBox="0 0 24 24">
          <rect x="5" y="11" width="14" height="9.5" rx="2.2" />
          <path d="M8 11V8a4 4 0 0 1 8 0v3" />
        </symbol>
        <symbol id="p-pencil" viewBox="0 0 24 24">
          <path d="M14 5.5l4.5 4.5M4 20l1.2-4.4L16 4.8a2 2 0 0 1 2.8 0l.4.4a2 2 0 0 1 0 2.8L8.4 18.8 4 20Z" />
        </symbol>
        <symbol id="p-trash" viewBox="0 0 24 24">
          <path d="M4 6.5h16M9.5 6.5V5a2 2 0 0 1 2-2h1a2 2 0 0 1 2 2v1.5M6.5 6.5l1 13.5h9l1-13.5M10 10.5v6M14 10.5v6" />
        </symbol>
        <symbol id="p-copy" viewBox="0 0 24 24">
          <rect x="9" y="9" width="11" height="11" rx="2.2" />
          <path d="M5 15V5a2 2 0 0 1 2-2h8" />
        </symbol>
        <symbol id="p-scale" viewBox="0 0 24 24">
          <path d="M12 3v18M7 21h10M12 5 5 8m7-3 7 3M5 8l-3 6a3 3 0 0 0 6 0L5 8Zm14 0-3 6a3 3 0 0 0 6 0l-3-6Z" />
        </symbol>
        <symbol id="p-refresh" viewBox="0 0 24 24">
          <path d="M19.5 12a7.5 7.5 0 1 1-2.2-5.3M19.5 4.5v4h-4" />
        </symbol>
        <symbol id="p-chevR" viewBox="0 0 24 24">
          <path d="M9.5 5.5l6.5 6.5-6.5 6.5" />
        </symbol>
        <symbol id="p-chevL" viewBox="0 0 24 24">
          <path d="M14.5 5.5 8 12l6.5 6.5" />
        </symbol>
        <symbol id="p-plus" viewBox="0 0 24 24">
          <path d="M12 5v14M5 12h14" />
        </symbol>
        <symbol id="p-gear" viewBox="0 0 24 24">
          <circle cx="12" cy="12" r="3.3" />
          <path d="M12 2.2v3.1M12 18.7v3.1M2.2 12h3.1M18.7 12h3.1M5 5l2.2 2.2M16.8 16.8 19 19M19 5l-2.2 2.2M7.2 16.8 5 19" />
        </symbol>
      </defs>
    </svg>
  );
}

// Icon renders one glyph by name. className defaults to "p-i" (the sole class
// the old ic() emitted); the caps meter passes "p-i on" / "p-i hot" to light a
// cell, exactly as capsHtml did.
export function Icon({
  name,
  className = "p-i",
  ...rest
}: { name: IconName; className?: string } & Omit<SVGProps<SVGSVGElement>, "name">) {
  return (
    <svg className={className} {...rest}>
      <use href={`#p-${name}`} />
    </svg>
  );
}
