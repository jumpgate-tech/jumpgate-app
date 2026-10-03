// The per-method / per-endpoint latency table shared by both an "under WHAT
// clients experienced" cards: count, mean and the distribution of how long
// requests took. Port of analytics.ts's latencyTable/latencyRow/distribution.
import type { Latency } from "../../api";
import { fmtInt } from "../../ui";
import { distributionBands, fmtSeconds, slowestLabel } from "./analyticsModel";

export function LatencyTable({ what, rows }: { what: string; rows: { label: string; l: Latency }[] }) {
  if (rows.length === 0) return null;
  return (
    <div className="surface-scroll">
      <table className="surface an-latency">
        <thead>
          <tr>
            <th>{what}</th>
            <th className="an-num">Requests</th>
            <th className="an-num">Mean</th>
            <th>How long they took</th>
          </tr>
        </thead>
        <tbody>
          {rows.map((r) => (
            <LatencyRow key={r.label} label={r.label} l={r.l} />
          ))}
        </tbody>
      </table>
    </div>
  );
}

function LatencyRow({ label, l }: { label: string; l: Latency }) {
  return (
    <tr>
      <td>
        <code>{label}</code>
      </td>
      <td className="an-num">{fmtInt(l.count)}</td>
      <td className="an-num">{l.mean === null ? <span className="muted">—</span> : fmtSeconds(l.mean)}</td>
      <td>
        <Distribution l={l} />
      </td>
    </tr>
  );
}

// Distribution turns the cumulative buckets into the bands they describe and
// draws them as a stacked bar, plus the slowest non-empty band as text. No
// percentile is shown — see distributionBands's own comment for why.
function Distribution({ l }: { l: Latency }) {
  const bands = distributionBands(l);
  if (bands.length === 0) return <span className="muted small">—</span>;
  const title = bands
    .filter((b) => b.n > 0)
    .map((b) => `${b.n} ${b.label}`)
    .join(" · ");
  return (
    <>
      <span className="an-dist" title={title}>
        {bands.map(
          (b, i) =>
            b.n > 0 && <span key={i} className={`an-band an-band-${Math.min(i, 4)}`} style={{ flex: b.n }} />,
        )}
      </span>
      <span className="muted small">{slowestLabel(bands)}</span>
    </>
  );
}
