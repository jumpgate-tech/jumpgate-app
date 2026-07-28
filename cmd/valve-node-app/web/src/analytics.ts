// #/analytics/<gid> — diagnosis, to the Control Surface's detection.
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
// Averaging one into the other produces a number that describes nothing: on a
// quiet gateway the second section is almost entirely the poller. Section
// headers say which question each number answers, because a footnote nobody
// reads is not a disclosure.
//
// Nothing is stored. The counters are cumulative since the gateway process
// started, so totals are shown against that start; the live rate is computed
// in this page by diffing consecutive polls, and the sparkline covers "since
// you opened this page" and says so. A reload starts the history over, which
// is honest about what it is rather than pretending to a history nobody kept.
import * as api from "./api";
import { escapeHtml, fmtInt, footer, onAction } from "./ui";

// POLL_MS is the read cadence. Each poll is one curl on the gateway's own
// machine, which is cheap — but it is a command on a possibly-remote box, so
// this is not a second-by-second screen.
const POLL_MS = 5000;

// HISTORY is how many readings the sparkline keeps: 60 × 5s ≈ five minutes,
// which is long enough to watch a deploy or an incident develop and short
// enough that the memory is trivially bounded.
const HISTORY = 60;

// A reading kept for the rate calculation: when it was taken, and the
// per-chain received counter at that moment.
interface Reading {
  t: number;
  since: string;
  received: Map<number, number>;
}

export function renderAnalytics(root: HTMLElement, gid: string): () => void {
  let disposed = false;
  let gateway: api.GatewayView | null = null;
  let data: api.GatewayAnalytics | null = null;
  let loadErr: string | null = null;
  let timer: number | null = null;
  const history: Reading[] = [];

  root.innerHTML = `<div id="an-body"><p class="muted">Loading…</p></div><div id="an-footer">${footer()}</div>`;
  const body = root.querySelector<HTMLElement>("#an-body")!;

  onAction(root, (action, el) => {
    if (action === "toggle-endpoint") {
      el.closest<HTMLElement>(".an-endpoint")?.classList.toggle("expanded");
    }
  });

  void init();

  async function init(): Promise<void> {
    try {
      const gws = await api.getGateways();
      gateway = (gws.gateways ?? []).find((g) => g.id === gid) ?? null;
    } catch (err) {
      if (disposed) return;
      loadErr = String(err instanceof Error ? err.message : err);
      render();
      return;
    }
    if (disposed) return;
    if (!gateway) {
      render();
      return;
    }
    await poll();
    timer = window.setInterval(() => void poll(), POLL_MS);
  }

  async function poll(): Promise<void> {
    try {
      const next = await api.getGatewayAnalytics(gid);
      if (disposed) return;
      remember(next);
      data = next;
      loadErr = null;
    } catch (err) {
      if (disposed) return;
      loadErr = String(err instanceof Error ? err.message : err);
    }
    render();
  }

  // remember appends one reading to the rate history, and throws the history
  // away when the gateway has restarted.
  //
  // A restart resets every counter to zero, so a diff across it is not a rate
  // — it is the whole history of the previous process appearing as a negative
  // spike, or the new process's first requests appearing as a plummet from a
  // number that no longer exists. `since` is the process start time, so a
  // change in it is exactly that event.
  function remember(next: api.GatewayAnalytics): void {
    if (!next.enabled || next.error) return;
    const last = history[history.length - 1];
    if (last && last.since !== next.since) history.length = 0;

    const received = new Map<number, number>();
    for (const n of next.networks ?? []) received.set(n.chainId, n.received);
    history.push({ t: Date.now(), since: next.since, received });
    if (history.length > HISTORY) history.shift();
  }

  function render(): void {
    if (disposed) return;
    body.innerHTML = view();
  }

  function view(): string {
    if (loadErr && !data) {
      return `<h1>Analytics</h1><p class="error">${escapeHtml(loadErr)}</p><p><a href="#/rpc">← Back to RPC</a></p>`;
    }
    if (!gateway) {
      return `
        <h1>Analytics</h1>
        <p class="error">No gateway called “${escapeHtml(gid)}”.</p>
        <p><a href="#/rpc">← Back to RPC</a></p>
      `;
    }
    return `
      ${head(gateway)}
      ${data ? bodyFor(data) : `<p class="muted">Reading the gateway's counters…</p>`}
    `;
  }

  function head(gw: api.GatewayView): string {
    return `
      <div class="an-head">
        <div>
          <h1>Analytics: ${escapeHtml(gw.label)}</h1>
          <p class="muted small">
            How this gateway is doing, and why it routes the way it does.
            <a href="#/rpc">← Back to the Control Surface</a>
          </p>
        </div>
        <div class="an-head-right muted small">${windowNote()}</div>
      </div>
    `;
  }

  // windowNote states what the numbers cover. A cumulative counter shown
  // without its start reads as a live rate, which is the single easiest way to
  // misread this whole page.
  function windowNote(): string {
    if (!data) return "";
    if (!data.enabled) return "counters off";
    if (data.error) return "could not be read";
    const since = data.since ? new Date(data.since) : null;
    const ok = since && !Number.isNaN(since.getTime());
    return ok
      ? `totals since the gateway started, ${escapeHtml(since.toLocaleString())}<br />re-read every ${POLL_MS / 1000}s`
      : `re-read every ${POLL_MS / 1000}s`;
  }

  function bodyFor(a: api.GatewayAnalytics): string {
    if (!a.enabled) {
      return `
        <div class="card">
          <p class="muted">
            This gateway is not counting its own requests, so there is nothing to show here.
            Turn the counters on in its settings on the <a href="#/rpc">Control Surface</a> —
            they stay on the machine the gateway runs on and nothing is sent anywhere.
          </p>
        </div>
      `;
    }
    if (a.error) {
      return `
        <div class="card">
          <p class="error">The gateway's counters could not be read.</p>
          <p class="muted small">${escapeHtml(a.error)}</p>
          <p class="muted small">
            The gateway itself may be perfectly healthy — the counters are served on
            loopback on its own machine, so this is a reading problem, not necessarily a
            serving one.
          </p>
        </div>
      `;
    }
    return clientSection(a) + gatewaySection(a);
  }

  // ---- section 1: what your clients experienced ---------------------------

  function clientSection(a: api.GatewayAnalytics): string {
    const nets = a.networks ?? [];
    return `
      <section class="an-section">
        <h2>What your clients experienced</h2>
        <p class="muted small">
          Counted on the path a client's request actually takes. The gateway's own
          block-tracking poller does not appear here at all, which is what makes these
          numbers about your users rather than about the gateway.
        </p>
        ${
          nets.length === 0
            ? `<div class="card"><p class="muted">This gateway fronts no chains yet.</p></div>`
            : nets.map((n) => chainCard(n)).join("")
        }
      </section>
    `;
  }

  function chainCard(n: api.NetworkAnalytics): string {
    const methods = n.methods ?? [];
    const endpoints = n.endpoints ?? [];
    const quiet = n.received === 0;
    return `
      <div class="card an-chain">
        <div class="an-chain-head">
          <span class="band-id">${n.chainId}</span>
          <span class="band-name">${escapeHtml(n.name)}</span>
          ${rateLine(n)}
        </div>
        <div class="an-stats">
          ${stat("Received", fmtInt(n.received), "what clients asked this chain for")}
          ${stat("Answered", fmtInt(n.answered), "returned by one of your endpoints")}
          ${stat("From cache", fmtInt(n.unattributed), "answered by the gateway itself, without calling any endpoint")}
          ${stat("Failed", fmtInt(n.failed), "asked for and never answered", n.failed > 0 ? "bad" : "")}
        </div>
        ${sparkline(n.chainId)}
        ${
          quiet
            ? `<p class="muted small">No client has called this chain since the gateway started, so there is no latency to report. That is a different thing from a chain that is failing.</p>`
            : latencyTable("Method", methods.map((m) => ({ label: m.method, l: m }))) +
              latencyTable("Endpoint", endpoints.map((e) => ({ label: e.upstream, l: e }))) +
              cachedLine(n)
        }
      </div>
    `;
  }

  function stat(label: string, value: string, title: string, kind = ""): string {
    return `
      <div class="an-stat${kind ? " an-stat-" + kind : ""}" title="${escapeHtml(title)}">
        <span class="an-stat-n">${escapeHtml(value)}</span>
        <span class="an-stat-l">${escapeHtml(label)}</span>
      </div>
    `;
  }

  // rateLine is the live rate, which needs two readings and says so until it
  // has them. A rate of zero and "not measured yet" look identical and are not
  // the same claim.
  function rateLine(n: api.NetworkAnalytics): string {
    const r = rateFor(n.chainId);
    if (r === null) {
      return `<span class="an-rate muted small">measuring rate…</span>`;
    }
    const span = Math.round((history[history.length - 1]!.t - history[0]!.t) / 1000);
    return `<span class="an-rate" title="Measured from this page's own readings, ${span}s apart.">
      ${escapeHtml(r.toFixed(r < 10 ? 2 : 0))} req/s <span class="muted">over the last ${span}s</span>
    </span>`;
  }

  // rateFor is requests per second across the whole kept history, or null
  // until there are two readings to diff.
  function rateFor(chainId: number): number | null {
    if (history.length < 2) return null;
    const first = history[0]!;
    const last = history[history.length - 1]!;
    const secs = (last.t - first.t) / 1000;
    if (secs <= 0) return null;
    const delta = (last.received.get(chainId) ?? 0) - (first.received.get(chainId) ?? 0);
    if (delta < 0) return null;
    return delta / secs;
  }

  // sparkline draws the per-interval rate across the kept history. It is
  // deliberately unlabelled on the y axis: it is there to show SHAPE — a step,
  // a spike, a stall — and a scale would invite reading precise values off a
  // five-minute window that resets on reload.
  function sparkline(chainId: number): string {
    if (history.length < 3) return "";
    const points: number[] = [];
    for (let i = 1; i < history.length; i++) {
      const a = history[i - 1]!;
      const b = history[i]!;
      const secs = (b.t - a.t) / 1000;
      const delta = (b.received.get(chainId) ?? 0) - (a.received.get(chainId) ?? 0);
      points.push(secs > 0 && delta >= 0 ? delta / secs : 0);
    }
    const max = Math.max(...points);
    if (max <= 0) return "";

    const w = 240;
    const h = 28;
    const step = points.length > 1 ? w / (points.length - 1) : w;
    const path = points
      .map((p, i) => `${(i * step).toFixed(1)},${(h - (p / max) * h).toFixed(1)}`)
      .join(" ");
    return `
      <div class="an-spark" title="Request rate since you opened this page. Peak ${max.toFixed(2)} req/s.">
        <svg viewBox="0 0 ${w} ${h}" preserveAspectRatio="none" role="img" aria-label="request rate">
          <polyline points="${path}" />
        </svg>
        <span class="muted small">rate since you opened this page · peak ${escapeHtml(max.toFixed(2))} req/s</span>
      </div>
    `;
  }

  // cachedLine reports the requests the gateway answered by itself, which are
  // deliberately NOT a row in the endpoint table above: they called no
  // endpoint, and giving them one would put a server on this screen that does
  // not exist. They are usually the fastest requests the gateway serves, so
  // leaving them out entirely would make the endpoint numbers look like the
  // whole story.
  function cachedLine(n: api.NetworkAnalytics): string {
    const lines: string[] = [];
    if (n.cached.count > 0) {
      lines.push(
        `${escapeHtml(fmtInt(n.cached.count))} of these were answered by the gateway itself from its own
         cache, without calling any endpoint${
           n.cached.mean === null ? "" : `, in ${escapeHtml(fmtSeconds(n.cached.mean))} on average`
         }.`,
      );
    }
    // How long the failures took, which the failure COUNT cannot tell you:
    // failing fast and timing out after thirty seconds are different problems
    // with different causes.
    if (n.failedLatency.count > 0 && n.failedLatency.mean !== null) {
      lines.push(
        `The ${escapeHtml(fmtInt(n.failedLatency.count))} that failed took
         ${escapeHtml(fmtSeconds(n.failedLatency.mean))} on average to fail.`,
      );
    }
    return lines.length === 0 ? "" : `<p class="muted small">${lines.join(" ")}</p>`;
  }

  function latencyTable(what: string, rows: { label: string; l: api.Latency }[]): string {
    if (rows.length === 0) return "";
    return `
      <div class="surface-scroll">
        <table class="surface an-latency">
          <thead>
            <tr>
              <th>${escapeHtml(what)}</th>
              <th class="an-num">Requests</th>
              <th class="an-num">Mean</th>
              <th>How long they took</th>
            </tr>
          </thead>
          <tbody>
            ${rows.map((r) => latencyRow(r.label, r.l)).join("")}
          </tbody>
        </table>
      </div>
    `;
  }

  function latencyRow(label: string, l: api.Latency): string {
    return `
      <tr>
        <td><code>${escapeHtml(label)}</code></td>
        <td class="an-num">${fmtInt(l.count)}</td>
        <td class="an-num">${l.mean === null ? `<span class="muted">—</span>` : escapeHtml(fmtSeconds(l.mean))}</td>
        <td>${distribution(l)}</td>
      </tr>
    `;
  }

  // distribution turns the cumulative buckets into the bands they describe.
  //
  // The bands are drawn, and no percentile is. eRPC's bucket bounds are 0.05,
  // 0.5, 5 and 30 seconds — neighbours an order of magnitude apart — so a p95
  // interpolated between two of them would be a number this page invented.
  // "5 of 5 answered within 500ms" is the same fact, and it is what the
  // buckets literally say.
  function distribution(l: api.Latency): string {
    const buckets = l.buckets ?? [];
    if (buckets.length === 0 || l.count === 0) return `<span class="muted small">—</span>`;

    let prev = 0;
    const bands: { label: string; n: number }[] = [];
    for (const b of buckets) {
      const n = b.count - prev;
      prev = b.count;
      bands.push({ label: bandLabel(b.le), n: Math.max(0, n) });
    }
    const total = bands.reduce((sum, b) => sum + b.n, 0);
    if (total === 0) return `<span class="muted small">—</span>`;

    return `
      <span class="an-dist" title="${escapeHtml(bands.filter((b) => b.n > 0).map((b) => `${b.n} ${b.label}`).join(" · "))}">
        ${bands
          .map((b, i) =>
            b.n === 0
              ? ""
              : `<span class="an-band an-band-${Math.min(i, 4)}" style="flex:${b.n}"></span>`,
          )
          .join("")}
      </span>
      <span class="muted small">${escapeHtml(slowestLabel(bands))}</span>
    `;
  }

  // slowestLabel names the slowest band that actually holds anything, because
  // that is the fact worth reading at a glance: a row whose worst band is
  // "under 50ms" needs no further attention, and one whose worst is "30s or
  // more" needs nothing else said about it.
  function slowestLabel(bands: { label: string; n: number }[]): string {
    for (let i = bands.length - 1; i >= 0; i--) {
      if (bands[i]!.n > 0) return `slowest ${bands[i]!.label}`;
    }
    return "";
  }

  function bandLabel(le: string): string {
    if (le === "+Inf") return "30s or more";
    const secs = Number(le);
    if (!Number.isFinite(secs)) return `under ${le}`;
    return `under ${fmtSeconds(secs)}`;
  }

  // ---- section 2: what the gateway sees -----------------------------------

  function gatewaySection(a: api.GatewayAnalytics): string {
    const eps = a.endpoints ?? [];
    return `
      <section class="an-section">
        <h2>What the gateway sees from your endpoints</h2>
        <p class="muted small">
          The gateway's own view, not a client's. Every count here <strong>includes the
          gateway's block-tracking poller</strong>, which calls each endpoint on a timer
          whether or not anyone is using it — on a quiet gateway it is nearly all of this.
          That is why these numbers are much larger than the ones above, and why they are
          not a measure of your traffic.
        </p>
        ${
          eps.length === 0
            ? `<div class="card"><p class="muted">The gateway has not talked to any endpoint yet.</p></div>`
            : `<div class="card">
                 <div class="surface-scroll">
                   <table class="surface an-endpoints">
                     <thead>
                       <tr>
                         <th>Endpoint</th>
                         <th class="an-num">Asked</th>
                         <th class="an-num">Errors</th>
                         <th class="an-num">Head lag</th>
                         <th>Selection</th>
                       </tr>
                     </thead>
                     <tbody>${eps.map((e) => endpointRows(e)).join("")}</tbody>
                   </table>
                 </div>
               </div>`
        }
      </section>
    `;
  }

  function endpointRows(e: api.EndpointHealth): string {
    const errors = e.errors ?? [];
    const errorTotal = errors.reduce((sum, c) => sum + c.count, 0);
    const expandable = errors.length > 0;
    return `
      <tr class="an-endpoint${expandable ? " expandable" : ""}" ${expandable ? `data-action="toggle-endpoint"` : ""}>
        <td>
          <code>${escapeHtml(e.upstream)}</code>
          ${e.chainId ? `<span class="muted small">chain ${e.chainId}</span>` : ""}
          ${
            e.configured
              ? ""
              : `<span class="badge badge-warn" title="This gateway's saved configuration no longer lists this endpoint, but eRPC is still reporting on it — the change has not been applied yet.">not in config</span>`
          }
        </td>
        <td class="an-num" title="Requests the GATEWAY made to this endpoint, poller included.">${fmtInt(e.requests)}</td>
        <td class="an-num${errorTotal > 0 ? " bad" : ""}">${errorTotal > 0 ? fmtInt(errorTotal) : `<span class="muted">0</span>`}</td>
        <td class="an-num" title="How many blocks behind this endpoint's latest block is.">${
          e.headLag > 0 ? fmtInt(e.headLag) : `<span class="muted">0</span>`
        }</td>
        <td>${selectionCell(e)}</td>
      </tr>
      ${expandable ? errorDetailRow(e, errors) : ""}
    `;
  }

  // selectionCell is eRPC's own reasoning, which is the answer to the question
  // an amber share bar on the Control Surface raises and cannot answer: not
  // "is this endpoint carrying the wrong share" but why. Position 0 is the one
  // being preferred right now; the score is what put it there.
  function selectionCell(e: api.EndpointHealth): string {
    const bits: string[] = [];
    // An endpoint eRPC has never scored is not the preferred one. FOUND BY
    // RUNNING IT: a devnet that could not be reached at all — 61 transport
    // failures and nothing else — read as "preferred", because position 0
    // means preferred and an unset number is also 0.
    if (!e.scored) {
      return `<span class="muted small" title="eRPC publishes a score only for endpoints it has evaluated. This one it has not — which is a different thing from scoring it badly.">not scored</span>`;
    }
    bits.push(
      e.position === 0
        ? `<span class="badge badge-ok" title="eRPC is preferring this endpoint right now.">preferred</span>`
        : `<span class="muted small">position ${escapeHtml(String(e.position))}</span>`,
    );
    bits.push(`<span class="muted small" title="eRPC's own score for this endpoint. Higher wins.">score ${escapeHtml(e.score.toFixed(3))}</span>`);
    if (e.primarySwitches > 1) {
      bits.push(
        `<span class="muted small" title="How many times eRPC has changed which endpoint it prefers. A number that keeps climbing is a gateway flapping between endpoints, which is a different problem from either of them being slow.">${fmtInt(
          e.primarySwitches,
        )} switches</span>`,
      );
    }
    if (e.excludedSeconds > 0) {
      bits.push(
        `<span class="badge badge-warn" title="The selection policy has this endpoint excluded.">excluded ${escapeHtml(
          fmtSeconds(e.excludedSeconds),
        )}</span>`,
      );
    }
    return `<span class="an-selection">${bits.join(" ")}</span>`;
  }

  function errorDetailRow(e: api.EndpointHealth, errors: api.ErrorClass[]): string {
    return `
      <tr class="an-error-detail">
        <td colspan="5">
          <table class="an-errors">
            <tbody>
              ${errors
                .map(
                  (c) => `
                    <tr>
                      <td class="an-num">${fmtInt(c.count)}</td>
                      <td><code>${escapeHtml(c.class)}</code></td>
                      <td>${
                        c.severity
                          ? `<span class="badge badge-${c.severity === "critical" ? "bad" : "warn"}">${escapeHtml(c.severity)}</span>`
                          : ""
                      }</td>
                      <td class="muted small">${escapeHtml(c.method || "")}</td>
                    </tr>`,
                )
                .join("")}
            </tbody>
          </table>
          <p class="muted small">
            Errors the gateway saw when it called <code>${escapeHtml(e.upstream)}</code>. Most of
            these are usually the block-tracking poller rather than a client request — an
            endpoint failing here is worth fixing before a client finds it, not proof that
            one already has.
          </p>
        </td>
      </tr>
    `;
  }

  return () => {
    disposed = true;
    if (timer !== null) window.clearInterval(timer);
  };
}

// fmtSeconds renders a duration given in seconds the way a person reading a
// latency would say it: milliseconds under a second, seconds above.
//
// Anything under half a millisecond is "<1ms", not "0ms". FOUND BY RUNNING IT:
// a chain's cached eth_chainId averaged 76µs and rendered as "0ms" — the same
// lie this page refuses elsewhere by sending a null mean, arrived at by
// rounding instead. Zero milliseconds is not a duration anything takes.
export function fmtSeconds(s: number): string {
  if (!Number.isFinite(s) || s < 0) return "—";
  if (s > 0 && s < 0.0005) return "<1ms";
  if (s < 1) return `${Math.round(s * 1000)}ms`;
  if (s < 60) return `${s < 10 ? s.toFixed(1) : Math.round(s)}s`;
  return `${Math.round(s / 60)}m`;
}
