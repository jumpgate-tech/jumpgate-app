// #/analytics/:id — diagnosis, to the Control Surface's detection.
//
// The RPC screen answers the two organisational questions: what can this
// endpoint do, and is it carrying the share you intended. It deliberately
// carries no latency and no error history, because those answer "how is it
// doing", which is what you ask AFTER something has told you to look. This is
// that screen, and it is reached from a gateway's bar rather than from the
// nav: you arrive here about a particular gateway, never in general.
//
// The page is two sections and the split is the most important thing on it:
//
//   WHAT YOUR CLIENTS EXPERIENCED   from erpc_network_*, the only family the
//                                   state poller does not touch. Volume,
//                                   failures, latency per method and per
//                                   endpoint.
//   WHAT THE GATEWAY SEES           from erpc_upstream_*/erpc_selection_*,
//                                   every number of which counts the poller
//                                   alongside client traffic because eRPC
//                                   publishes no label that separates them.
//                                   Error classes, lag, and why eRPC is
//                                   choosing what it chooses.
//
// Averaging one into the other produces a number that describes nothing —
// see ClientSection/GatewaySection's own header paragraphs, shown on the
// page itself, not just here.
//
// Nothing is stored. The counters are cumulative since the gateway process
// started; the live rate is computed in this page by diffing consecutive
// polls (useGatewayHealth's 5s refetchInterval), and the sparkline covers
// "since you opened this page" and says so. A reload starts the history
// over, which is honest about what it is rather than pretending to a
// history nobody kept.
//
// Port of analytics.ts. Its `disposed` guard, manual load/poll/render
// bookkeeping and the shared `loadErr` variable are gone: useGateways (one
// fetch, to find the gateway by id) + useGatewayHealth (the 5s poll, same
// api.getGatewayAnalytics call the panel's live dot already polls) own the
// data and its loading/error state. Only the rate history — which no query
// cache can hold, since it's a diff across several polls, not one of them —
// is kept locally, via analyticsModel's pure appendReading.
import { useEffect, useState } from "react";
import { useParams } from "react-router-dom";
import type * as api from "../../api";
import { useGateways, useGatewayHealth } from "../../hooks/gateway";
import { Footer } from "../../components/Footer";
import { ClientSection } from "./ClientSection";
import { GatewaySection } from "./GatewaySection";
import { appendReading, POLL_MS, sinceLabel, type Reading } from "./analyticsModel";

function message(e: unknown): string {
  return e instanceof Error ? e.message : String(e);
}

export function Analytics() {
  const { id } = useParams<{ id: string }>();
  const gid = id ?? "";

  const gatewaysQuery = useGateways();
  const gateway = gatewaysQuery.data?.find((g) => g.id === gid);

  const healthQuery = useGatewayHealth(gid, !!gateway);
  const data = healthQuery.data;

  const [history, setHistory] = useState<Reading[]>([]);
  // Resets the kept rate history when the URL's gateway id changes. Analytics
  // stays mounted across a same-route navigation (React Router doesn't
  // remount for a param change alone), so without this a reading from the
  // PREVIOUS gateway's chains would sit in the same history as the new
  // gateway's — analytics.ts never had this problem because renderAnalytics
  // was called fresh, with a fresh `history` closure, every time the id
  // changed.
  useEffect(() => {
    setHistory([]);
  }, [gid]);
  useEffect(() => {
    if (!data) return;
    setHistory((prev) => appendReading(prev, data, Date.now()));
  }, [data]);

  if (gatewaysQuery.isLoading) {
    return (
      <div>
        <p className="muted">Loading…</p>
        <Footer />
      </div>
    );
  }

  // Mirrors analytics.ts's `loadErr && !data`: a top-level error covers BOTH
  // the gateway lookup failing and the very first analytics poll failing
  // (before any data has ever loaded) — no gateway header in either case,
  // because there's nothing confirmed about the gateway to show yet. Once
  // data has loaded once, a LATER poll failure leaves the last-known data on
  // screen instead — React Query keeps `data` across a failed refetch, so
  // this branch simply stops applying.
  const topError = gatewaysQuery.isError
    ? message(gatewaysQuery.error)
    : !data && healthQuery.isError
      ? message(healthQuery.error)
      : null;

  if (topError) {
    return (
      <div>
        <h1>Analytics</h1>
        <p className="error">{topError}</p>
        <p>
          <a href="#/rpc">← Back to RPC</a>
        </p>
        <Footer />
      </div>
    );
  }

  if (!gateway) {
    return (
      <div>
        <h1>Analytics</h1>
        <p className="error">No gateway called &ldquo;{gid}&rdquo;.</p>
        <p>
          <a href="#/rpc">← Back to RPC</a>
        </p>
        <Footer />
      </div>
    );
  }

  return (
    <div>
      <Head gateway={gateway} data={data} />
      {data ? <Body data={data} history={history} /> : <p className="muted">Reading the gateway&apos;s counters…</p>}
      <Footer />
    </div>
  );
}

function Head({ gateway, data }: { gateway: api.GatewayView; data: api.GatewayAnalytics | undefined }) {
  return (
    <div className="an-head">
      <div>
        <h1>Analytics: {gateway.label}</h1>
        <p className="muted small">
          How this gateway is doing, and why it routes the way it does.{" "}
          <a href="#/rpc">← Back to the Control Surface</a>
        </p>
      </div>
      <div className="an-head-right muted small">
        <WindowNote data={data} />
      </div>
    </div>
  );
}

// windowNote states what the numbers cover. A cumulative counter shown
// without its start reads as a live rate, which is the single easiest way to
// misread this whole page.
function WindowNote({ data }: { data: api.GatewayAnalytics | undefined }) {
  if (!data) return null;
  if (!data.enabled) return <>counters off</>;
  if (data.error) return <>could not be read</>;
  const since = data.since ? sinceLabel(data.since) : null;
  return since ? (
    <>
      totals since the gateway started, {since}
      <br />
      re-read every {POLL_MS / 1000}s
    </>
  ) : (
    <>re-read every {POLL_MS / 1000}s</>
  );
}

function Body({ data, history }: { data: api.GatewayAnalytics; history: Reading[] }) {
  if (!data.enabled) {
    return (
      <div className="card">
        <p className="muted">
          This gateway is not counting its own requests, so there is nothing to show here. Turn
          the counters on in its settings on the <a href="#/rpc">Control Surface</a> — they stay
          on the machine the gateway runs on and nothing is sent anywhere.
        </p>
      </div>
    );
  }
  if (data.error) {
    return (
      <div className="card">
        <p className="error">The gateway&apos;s counters could not be read.</p>
        <p className="muted small">{data.error}</p>
        <p className="muted small">
          The gateway itself may be perfectly healthy — the counters are served on loopback on
          its own machine, so this is a reading problem, not necessarily a serving one.
        </p>
      </div>
    );
  }
  return (
    <>
      <ClientSection data={data} history={history} />
      <GatewaySection data={data} />
    </>
  );
}
