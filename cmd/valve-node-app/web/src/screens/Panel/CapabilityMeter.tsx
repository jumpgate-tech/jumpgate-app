// The capability meter, in its two forms — both fed by panelModel's
// capabilityCells so the fixed order (HTTP · WS · Archive · Trace) and the
// lit/hot rules live in one place:
//   • CapabilityDots — the tiny icon row on a list network row. Always drawn
//     from an empty status set there (the list never runs the real probe), so
//     it sits dim; kept as a component so the markup matches capsHtml exactly.
//   • CapabilityMeter — the labelled p-caprow band on the network/endpoint
//     detail screens, folded from a real probe's statuses.
import { capabilityCells, type CapCell } from "../../panelModel";
import { Icon, type IconName } from "./icons";

// CAP_ICON maps a capability key to its glyph (ported from panel.ts).
const CAP_ICON: Record<string, IconName> = {
  http: "globe",
  ws: "ws",
  archive: "archive",
  trace: "trace",
};

// CapabilityDots is the small unlabelled meter used on list rows (capsHtml).
// Each glyph is icon + colour only, so it carries its meaning as an accessible
// name — role="img" + aria-label "HTTP: supported" / "Trace: unavailable".
export function CapabilityDots({ cells }: { cells: CapCell[] }) {
  return (
    <span className="p-caps">
      {cells.map((c) => (
        <Icon
          key={c.key}
          name={CAP_ICON[c.key]}
          className={`p-i${c.hot ? " hot" : c.lit ? " on" : ""}`}
          role="img"
          aria-label={`${c.label}: ${c.lit ? "supported" : "unavailable"}`}
        />
      ))}
    </span>
  );
}

// CapabilityMeter is the labelled band on the detail screens. statuses is the
// folded verdict per capability key ("supported" lights the cell).
export function CapabilityMeter({ statuses }: { statuses: Record<string, string> }) {
  const cells = capabilityCells(statuses);
  return (
    <div className="p-caprow">
      {cells.map((c) => (
        <span key={c.key} className={`p-capitem${c.lit ? " lit" : ""}`}>
          <Icon name={CAP_ICON[c.key]} />
          {c.label}
          {/* lit/unlit is otherwise colour-only; a visually-hidden word makes
              the state audible without changing the meter's look. */}
          <span className="p-vh">: {c.lit ? "supported" : "unavailable"}</span>
        </span>
      ))}
    </div>
  );
}

// CapsBand wraps the meter with the three states the detail screens show while
// a probe is in flight: "probing…" only on the FIRST lazy fetch (busy with no
// data yet), the probe's own failure (err with no data), otherwise the meter —
// which on a "recheck" keeps the previous verdict until the refresh lands.
export function CapsBand({
  statuses,
  busy,
  err,
  hasData,
  running,
}: {
  statuses: Record<string, string>;
  busy: boolean;
  err: string | null;
  hasData: boolean;
  running: boolean;
}) {
  // A stopped gateway can't answer a capability probe — say so instead of
  // running it and showing a raw connection error.
  if (!running) return <div className="p-caprow" style={{ color: "var(--dim2)" }}>Start the gateway to check capabilities.</div>;
  if (busy && !hasData) return <div className="p-caprow" style={{ color: "var(--dim2)" }}>probing…</div>;
  if (err && !hasData) return <div className="p-caprow p-caperr">Couldn't check capabilities — {err}</div>;
  return <CapabilityMeter statuses={statuses} />;
}
