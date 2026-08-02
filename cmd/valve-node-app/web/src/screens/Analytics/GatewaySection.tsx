// Section 2: "What the gateway sees from your endpoints" — from
// erpc_upstream_*/erpc_selection_*, every number of which counts the state
// poller alongside client traffic because eRPC publishes no label that
// separates them. Port of analytics.ts's gatewaySection/endpointRows/
// selectionCell/errorDetailRow.
import { useState } from "react";
import type { EndpointHealth, ErrorClass, GatewayAnalytics } from "../../api";
import { fmtInt } from "../../ui";
import { Badge } from "../../components/Badge";
import { fmtSeconds } from "./analyticsModel";

export function GatewaySection({ data }: { data: GatewayAnalytics }) {
  const eps = data.endpoints ?? [];
  return (
    <section className="an-section">
      <h2>What the gateway sees from your endpoints</h2>
      <p className="muted small">
        The gateway&apos;s own view, not a client&apos;s. Every count here{" "}
        <strong>includes the gateway&apos;s block-tracking poller</strong>, which calls each
        endpoint on a timer whether or not anyone is using it — on a quiet gateway it is nearly
        all of this. That is why these numbers are much larger than the ones above, and why they
        are not a measure of your traffic.
      </p>
      {eps.length === 0 ? (
        <div className="card">
          <p className="muted">The gateway has not talked to any endpoint yet.</p>
        </div>
      ) : (
        <div className="card">
          <div className="surface-scroll">
            <table className="surface an-endpoints">
              <thead>
                <tr>
                  <th>Endpoint</th>
                  <th className="an-num">Asked</th>
                  <th className="an-num">Errors</th>
                  <th className="an-num">Head lag</th>
                  <th>Selection</th>
                </tr>
              </thead>
              <tbody>
                {eps.map((e) => (
                  <EndpointRow key={e.upstream} e={e} />
                ))}
              </tbody>
            </table>
          </div>
        </div>
      )}
    </section>
  );
}

function EndpointRow({ e }: { e: EndpointHealth }) {
  const [expanded, setExpanded] = useState(false);
  const errors = e.errors ?? [];
  const errorTotal = errors.reduce((sum, c) => sum + c.count, 0);
  const expandable = errors.length > 0;
  return (
    <>
      <tr
        className={`an-endpoint${expandable ? " expandable" : ""}${expanded ? " expanded" : ""}`}
        onClick={expandable ? () => setExpanded((v) => !v) : undefined}
      >
        <td>
          <code>{e.upstream}</code>
          {e.chainId ? <span className="muted small">chain {e.chainId}</span> : null}
          {!e.configured && (
            <Badge
              text="not in config"
              kind="warn"
              title="This gateway's saved configuration no longer lists this endpoint, but eRPC is still reporting on it — the change has not been applied yet."
            />
          )}
        </td>
        <td className="an-num" title="Requests the GATEWAY made to this endpoint, poller included.">
          {fmtInt(e.requests)}
        </td>
        <td className={`an-num${errorTotal > 0 ? " bad" : ""}`}>
          {errorTotal > 0 ? fmtInt(errorTotal) : <span className="muted">0</span>}
        </td>
        <td className="an-num" title="How many blocks behind this endpoint's latest block is.">
          {e.headLag > 0 ? fmtInt(e.headLag) : <span className="muted">0</span>}
        </td>
        <td>
          <SelectionCell e={e} />
        </td>
      </tr>
      {/* Always rendered when expandable, matching the CSS: the row is
          `display: none` until the sibling `tr.an-endpoint.expanded`
          reveals it, exactly like analytics.ts's markup. */}
      {expandable && <ErrorDetailRow e={e} errors={errors} />}
    </>
  );
}

// selectionCell is eRPC's own reasoning: not "is this endpoint carrying the
// wrong share" (the share bar on the Control Surface) but why. Position 0 is
// the one being preferred right now; the score is what put it there.
function SelectionCell({ e }: { e: EndpointHealth }) {
  if (!e.scored) {
    return (
      <span
        className="muted small"
        title="eRPC publishes a score only for endpoints it has evaluated. This one it has not — which is a different thing from scoring it badly."
      >
        not scored
      </span>
    );
  }
  return (
    <span className="an-selection">
      {e.position === 0 ? (
        <Badge text="preferred" kind="ok" title="eRPC is preferring this endpoint right now." />
      ) : (
        <span className="muted small">position {e.position}</span>
      )}
      <span className="muted small" title="eRPC's own score for this endpoint. Higher wins.">
        score {e.score.toFixed(3)}
      </span>
      {e.primarySwitches > 1 && (
        <span
          className="muted small"
          title="How many times eRPC has changed which endpoint it prefers. A number that keeps climbing is a gateway flapping between endpoints, which is a different problem from either of them being slow."
        >
          {fmtInt(e.primarySwitches)} switches
        </span>
      )}
      {e.excludedSeconds > 0 && (
        <Badge
          text={`excluded ${fmtSeconds(e.excludedSeconds)}`}
          kind="warn"
          title="The selection policy has this endpoint excluded."
        />
      )}
    </span>
  );
}

function ErrorDetailRow({ e, errors }: { e: EndpointHealth; errors: ErrorClass[] }) {
  return (
    <tr className="an-error-detail">
      <td colSpan={5}>
        <table className="an-errors">
          <tbody>
            {errors.map((c, i) => (
              <tr key={i}>
                <td className="an-num">{fmtInt(c.count)}</td>
                <td>
                  <code>{c.class}</code>
                </td>
                <td>{c.severity && <Badge text={c.severity} kind={c.severity === "critical" ? "bad" : "warn"} />}</td>
                <td className="muted small">{c.method || ""}</td>
              </tr>
            ))}
          </tbody>
        </table>
        <p className="muted small">
          Errors the gateway saw when it called <code>{e.upstream}</code>. Most of these are
          usually the block-tracking poller rather than a client request — an endpoint failing
          here is worth fixing before a client finds it, not proof that one already has.
        </p>
      </td>
    </tr>
  );
}
