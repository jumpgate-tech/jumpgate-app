// The per-chain sparkline — an unlabelled SVG polyline showing the SHAPE of
// the request rate since the page was opened (a step, a spike, a stall).
// Deliberately no y-axis scale: a precise value read off a five-minute
// window that resets on reload would invite reading precision that isn't
// there. Renders nothing until there are at least two intervals to draw and
// at least one of them is non-zero — mirrors analytics.ts's sparkline().
const WIDTH = 240;
const HEIGHT = 28;

export function Sparkline({ points }: { points: number[] }) {
  if (points.length === 0) return null;
  const max = Math.max(...points);
  if (max <= 0) return null;

  const step = points.length > 1 ? WIDTH / (points.length - 1) : WIDTH;
  const path = points
    .map((p, i) => `${(i * step).toFixed(1)},${(HEIGHT - (p / max) * HEIGHT).toFixed(1)}`)
    .join(" ");

  return (
    <div className="an-spark" title={`Request rate since you opened this page. Peak ${max.toFixed(2)} req/s.`}>
      <svg viewBox={`0 0 ${WIDTH} ${HEIGHT}`} preserveAspectRatio="none" role="img" aria-label="request rate">
        <polyline points={path} />
      </svg>
      <span className="muted small">rate since you opened this page · peak {max.toFixed(2)} req/s</span>
    </div>
  );
}
