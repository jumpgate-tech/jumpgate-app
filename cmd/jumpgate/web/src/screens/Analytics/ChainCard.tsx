// One chain's card in "What your clients experienced": the four headline
// stats, the live rate + sparkline computed from this page's own polls, and
// (once the chain has seen a client) the per-method / per-endpoint latency
// tables plus the cache/failure footnote. Port of analytics.ts's chainCard.
import type { NetworkAnalytics } from "../../api";
import { fmtInt } from "../../ui";
import { fmtSeconds, rateFor, rateWindowSeconds, sparklinePoints, type Reading } from "./analyticsModel";
import { LatencyTable } from "./LatencyTable";
import { Sparkline } from "./Sparkline";

export function ChainCard({ n, history }: { n: NetworkAnalytics; history: Reading[] }) {
  const methods = n.methods ?? [];
  const endpoints = n.endpoints ?? [];
  const quiet = n.received === 0;

  return (
    <div className="card an-chain">
      <div className="an-chain-head">
        <span className="band-id">{n.chainId}</span>
        <span className="band-name">{n.name}</span>
        <RateLine history={history} chainId={n.chainId} />
      </div>
      <div className="an-stats">
        <Stat label="Received" value={fmtInt(n.received)} title="what clients asked this chain for" />
        <Stat label="Answered" value={fmtInt(n.answered)} title="returned by one of your endpoints" />
        <Stat
          label="From cache"
          value={fmtInt(n.unattributed)}
          title="answered by the gateway itself, without calling any endpoint"
        />
        <Stat
          label="Failed"
          value={fmtInt(n.failed)}
          title="asked for and never answered"
          kind={n.failed > 0 ? "bad" : undefined}
        />
      </div>
      <Sparkline points={sparklinePoints(history, n.chainId)} />
      {quiet ? (
        <p className="muted small">
          No client has called this chain since the gateway started, so there is no latency to
          report. That is a different thing from a chain that is failing.
        </p>
      ) : (
        <>
          <LatencyTable what="Method" rows={methods.map((m) => ({ label: m.method, l: m }))} />
          <LatencyTable what="Endpoint" rows={endpoints.map((e) => ({ label: e.upstream, l: e }))} />
          <CachedLine n={n} />
        </>
      )}
    </div>
  );
}

function Stat({ label, value, title, kind }: { label: string; value: string; title: string; kind?: "bad" }) {
  return (
    <div className={`an-stat${kind ? ` an-stat-${kind}` : ""}`} title={title}>
      <span className="an-stat-n">{value}</span>
      <span className="an-stat-l">{label}</span>
    </div>
  );
}

// rateLine is the live rate, which needs two readings and says so until it
// has them. A rate of zero and "not measured yet" look identical and are not
// the same claim.
function RateLine({ history, chainId }: { history: Reading[]; chainId: number }) {
  const rate = rateFor(history, chainId);
  if (rate === null) {
    return <span className="an-rate muted small">measuring rate…</span>;
  }
  const span = rateWindowSeconds(history);
  return (
    <span className="an-rate" title={`Measured from this page's own readings, ${span}s apart.`}>
      {rate.toFixed(rate < 10 ? 2 : 0)} req/s <span className="muted">over the last {span}s</span>
    </span>
  );
}

// cachedLine reports the requests the gateway answered by itself — deliberately
// NOT a row in the endpoint table above: they called no endpoint, and giving
// them one would put a server on this screen that does not exist.
function CachedLine({ n }: { n: NetworkAnalytics }) {
  const lines: string[] = [];
  if (n.cached.count > 0) {
    lines.push(
      `${fmtInt(n.cached.count)} of these were answered by the gateway itself from its own cache, without calling any endpoint${
        n.cached.mean === null ? "" : `, in ${fmtSeconds(n.cached.mean)} on average`
      }.`,
    );
  }
  // How long the failures took, which the failure COUNT cannot tell you:
  // failing fast and timing out after thirty seconds are different problems.
  if (n.failedLatency.count > 0 && n.failedLatency.mean !== null) {
    lines.push(
      `The ${fmtInt(n.failedLatency.count)} that failed took ${fmtSeconds(n.failedLatency.mean)} on average to fail.`,
    );
  }
  if (lines.length === 0) return null;
  return <p className="muted small">{lines.join(" ")}</p>;
}
