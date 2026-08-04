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
  | "gear"
  | "sun"
  | "moon"
  | "server";

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
          {/* A real cog with teeth — the old spoked circle read as a sun, which
              is why the settings control looked like a light/dark toggle. */}
          <circle cx="12" cy="12" r="3.2" />
          <path d="M19.4 15a1.65 1.65 0 0 0 .33 1.82l.06.06a2 2 0 1 1-2.83 2.83l-.06-.06a1.65 1.65 0 0 0-1.82-.33 1.65 1.65 0 0 0-1 1.51V21a2 2 0 0 1-4 0v-.09A1.65 1.65 0 0 0 9 19.4a1.65 1.65 0 0 0-1.82.33l-.06.06a2 2 0 1 1-2.83-2.83l.06-.06a1.65 1.65 0 0 0 .33-1.82 1.65 1.65 0 0 0-1.51-1H3a2 2 0 0 1 0-4h.09A1.65 1.65 0 0 0 4.6 9a1.65 1.65 0 0 0-.33-1.82l-.06-.06a2 2 0 1 1 2.83-2.83l.06.06a1.65 1.65 0 0 0 1.82.33H9a1.65 1.65 0 0 0 1-1.51V3a2 2 0 0 1 4 0v.09a1.65 1.65 0 0 0 1 1.51 1.65 1.65 0 0 0 1.82-.33l.06-.06a2 2 0 1 1 2.83 2.83l-.06.06a1.65 1.65 0 0 0-.33 1.82V9a1.65 1.65 0 0 0 1.51 1H21a2 2 0 0 1 0 4h-.09a1.65 1.65 0 0 0-1.51 1Z" />
        </symbol>
        <symbol id="p-sun" viewBox="0 0 24 24">
          <circle cx="12" cy="12" r="4.2" />
          <path d="M12 1.8v2.6M12 19.6v2.6M4.4 4.4l1.9 1.9M17.7 17.7l1.9 1.9M1.8 12h2.6M19.6 12h2.6M4.4 19.6l1.9-1.9M17.7 6.3l1.9-1.9" />
        </symbol>
        <symbol id="p-moon" viewBox="0 0 24 24">
          <path d="M21 12.8A9 9 0 1 1 11.2 3a7 7 0 0 0 9.8 9.8Z" />
        </symbol>
        {/* server — a managed node/devnet you run, vs the globe for a public
            endpoint. Replaces the word "public"/"yours" on a row. */}
        <symbol id="p-server" viewBox="0 0 24 24">
          <rect x="3.5" y="4.5" width="17" height="6" rx="1.6" />
          <rect x="3.5" y="13.5" width="17" height="6" rx="1.6" />
          <path d="M7 7.5h.01M7 16.5h.01" />
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
