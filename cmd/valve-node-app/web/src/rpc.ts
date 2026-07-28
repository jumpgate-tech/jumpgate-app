// #/rpc — eRPC as a LAYER over the whole fleet. The Control Surface.
//
// The geometry IS the argument, and it is one table:
//
//   THE BAR     a full-width header per gateway. It spans the width because it
//               fronts everything below it: state, the URL callers dial, the
//               lifecycle actions.
//   THE BANDS   one row per chain, spanning every column, wrapping the
//               endpoints that serve it. The routing hierarchy becomes
//               STRUCTURAL rather than described — you can see that these
//               three endpoints are what /main/evm/369 resolves to, because
//               they are physically underneath it.
//   THE ROWS    one per endpoint: what it is, whether it can be used, what it
//               can DO, and what share of the traffic it is actually carrying.
//
// This replaced a chip row plus a panel showing ONE selected chain. Selection
// was the problem: a gateway's whole job is fronting several chains, and a
// screen that shows one at a time cannot answer "which of my chains is
// misrouting" without clicking through every one of them. Every chain is now
// visible at once and there is no selection state to get out of step.
//
// Two columns are deliberately not here. Latency and request-rate history
// answer "how is it doing", which is diagnosis; this screen answers the two
// organisational questions — what can this endpoint do, and is it carrying the
// share you intended — which is detection. Diagnosis belongs on an analytics
// page you open once this screen has told you something is off.
//
// The rules this file inherits, none of them negotiable:
//   - no native confirm()/alert(). openModal/confirmModal from ui.ts.
//   - never offer an action that can only fail: the server decides what is
//     permitted (gatewayActions, upstreamActions) and this file renders that
//     list verbatim.
//   - state, and the REASON something is unavailable, live ON the thing you
//     interact with, visible before you click.
import * as api from "./api";
import {
  badge,
  closeModal,
  confirmModal,
  copyToClipboard,
  dot,
  escapeHtml,
  footer,
  modalBody,
  onAction,
  openModal,
  wireDropdowns,
} from "./ui";

// CAP_ORDER is the capability set every endpoint row renders, in this order,
// whether or not the endpoint has an opinion about each one.
//
// Rendering the FULL set is the point. A struck-through tag is an absence you
// can see without reading, which is how a chain whose every endpoint lacks
// WebSocket announces itself at a glance instead of after a support ticket. A
// short list would make "we asked and it cannot" indistinguishable from "we
// never asked".
const CAP_ORDER = ["http", "ws", "archive", "trace"] as const;

// CAP_TAGS are the short forms shown in the table. The API sends a full label
// per capability, but a table column needs four characters, not "WebSocket".
const CAP_TAGS: Record<string, string> = {
  http: "HTTP",
  ws: "WS",
  archive: "ARCHIVE",
  trace: "TRACE",
};

// FINAL_STEP is the id every gateway plan ends on (preflight, config, run).
// The setup event stream has no terminal frame, so this is what tells the
// progress log it is finished and the bar can be re-read.
const FINAL_STEP = "run";

interface ActionButton {
  label: string;
  title: string;
  className: string;
}

const ACTION_BUTTONS: Record<string, ActionButton> = {
  start: { label: "Start", title: "Start the existing gateway container", className: "btn" },
  stop: { label: "Stop", title: "Stop the gateway. Its configuration is kept.", className: "btn btn-ghost" },
  restart: {
    label: "Restart",
    title: "Restart the gateway. This also clears its cached per-chain head, which is what a chain reset needs.",
    className: "btn btn-ghost",
  },
  create: { label: "Create gateway", title: "Create the container from the configuration below", className: "btn" },
  recreate: {
    label: "Re-create (apply config)",
    title:
      "Replace the container so the saved configuration takes effect. A container's port and mounts are fixed when it is created, so this is the only way to apply a change.",
    className: "btn btn-ghost",
  },
  wipe: { label: "Wipe…", title: "Destroy the gateway container and rebuild it", className: "btn btn-danger" },
};

export function renderRPC(root: HTMLElement): () => void {
  let disposed = false;
  let data: api.GatewaysResponse | null = null;
  let loadErr: string | null = null;

  // Measured traffic and probed capabilities per gateway, both fetched
  // separately from the gateway list and both allowed to be missing.
  //
  // They are separate requests because they have different costs and different
  // failure modes: traffic is one loopback curl on the gateway's machine,
  // capabilities opens real sockets to real endpoints. Folding either into the
  // list would mean a gateway whose counters cannot be read loses its whole
  // card rather than one column of it — on the exact screen you opened because
  // something was wrong.
  const traffic: Record<string, api.GatewayTraffic | null> = {};
  const caps: Record<string, api.GatewayCapabilities | null> = {};
  const capsBusy: Record<string, boolean> = {};
  const busy: Record<string, string | null> = {};
  const actionErr: Record<string, string | null> = {};
  const activity: Record<string, string[]> = {};
  // The bar's own settings (port, bind) are edited in place, opened per
  // gateway. Kept out of the DOM for the same reason as focus.
  const settingsOpen: Record<string, boolean> = {};
  // The live HTTPS verification per gateway: the last result this screen ran
  // (the server also returns its own last one on the view, which is what shows
  // after a reload), whether one is in flight, and why one failed to run.
  const verifyResult: Record<string, api.TlsVerification | null> = {};
  const verifyBusy: Record<string, boolean> = {};
  const verifyErr: Record<string, string | null> = {};
  let streamStop: (() => void) | null = null;

  root.innerHTML = `
    <div class="page-head">
      <h1>RPC</h1>
      <button class="btn btn-ghost" data-action="refresh">Refresh</button>
    </div>
    <p class="muted">
      eRPC sits above everything else here. One gateway fronts as many chains as you
      list, and each chain can be served by a devnet on this machine, a node on any
      machine you manage, or a public endpoint — a gateway names the machine it runs
      on, it does not belong to it.
    </p>
    <div id="rpc-body"><p class="muted">Loading…</p></div>
    ${footer()}
  `;
  const body = root.querySelector<HTMLElement>("#rpc-body")!;

  onAction(root, (action, el) => {
    void handleAction(action, el);
  });
  wireDropdowns(root, () => {});

  void load();

  // --- data ---------------------------------------------------------------

  async function load(): Promise<void> {
    try {
      const next = await api.getGateways();
      if (disposed) return;
      data = next;
      loadErr = null;
    } catch (err) {
      if (disposed) return;
      data = null;
      loadErr = message(err);
    }
    render();
    // After the render, never before it: the table stands up immediately with
    // its share and capability columns pending, rather than the whole screen
    // waiting on the slowest endpoint probe in the fleet.
    for (const gw of data?.gateways ?? []) {
      void loadTraffic(gw.id);
      void loadCapabilities(gw.id, false);
    }
  }

  // loadTraffic reads one gateway's counters. A failure is stored as null and
  // rendered as an explained blank column — never as an error banner, because
  // unreadable counters say nothing about whether the gateway is serving.
  async function loadTraffic(gid: string): Promise<void> {
    try {
      const t = await api.getGatewayTraffic(gid);
      if (disposed) return;
      traffic[gid] = t;
    } catch {
      if (disposed) return;
      traffic[gid] = null;
    }
    render();
  }

  async function loadCapabilities(gid: string, refresh: boolean): Promise<void> {
    capsBusy[gid] = refresh;
    if (refresh) render();
    try {
      const c = await api.getGatewayCapabilities(gid, refresh);
      if (disposed) return;
      caps[gid] = c;
    } catch {
      if (disposed) return;
      caps[gid] = null;
    }
    capsBusy[gid] = false;
    render();
  }

  function gatewayOf(gid: string): api.GatewayView | undefined {
    return (data?.gateways ?? []).find((g) => g.id === gid);
  }

  // networkOf survives the loss of the selection model because the modals
  // still act on one NAMED chain — "add an endpoint to 369" — which is a
  // different thing from the screen having a focused chain.
  function networkOf(gw: api.GatewayView, chainId: number): api.NetworkView | undefined {
    return (gw.networks ?? []).find((n) => n.chainId === chainId);
  }

  // shareOf and capsOf join the two side-channel reads back to a row by
  // upstream id. Both the traffic route and the capabilities route derive that
  // id from the same configuration by the same rule, which is what makes the
  // join safe — see catalog.GeneratedUpstreamID.
  function shareOf(gid: string, chainId: number, upstreamId: string): api.UpstreamShare | undefined {
    const net = (traffic[gid]?.networks ?? []).find((n) => n.chainId === chainId);
    return (net?.upstreams ?? []).find((u) => u.upstream === upstreamId);
  }

  function capsOf(gid: string, chainId: number, upstreamId: string): api.EndpointCapabilities | undefined {
    return (caps[gid]?.endpoints ?? []).find((e) => e.chainId === chainId && e.upstream === upstreamId);
  }

  // --- render -------------------------------------------------------------

  function render(): void {
    if (disposed) return;
    if (loadErr) {
      body.innerHTML = `<p class="error">Could not read the gateways: ${escapeHtml(loadErr)}</p>`;
      return;
    }
    if (!data) {
      body.innerHTML = `<p class="muted">Loading…</p>`;
      return;
    }
    const gateways = data.gateways ?? [];
    body.innerHTML = `
      ${gateways.map(gatewayBlock).join("")}
      ${gateways.length === 0 ? emptyState() : ""}
      <div class="card-actions rpc-add-gateway">
        <button class="btn${gateways.length ? " btn-ghost" : ""}" data-action="add-gateway">Add a gateway</button>
      </div>
    `;
  }

  function emptyState(): string {
    const targets = data?.targets ?? [];
    if (targets.length === 0) {
      return `
        <div class="card empty-state">
          <p class="muted">
            No machines yet. A gateway is a container, so it has to run somewhere —
            add a machine on <a href="#/targets">Machines</a> first.
          </p>
        </div>
      `;
    }
    return `
      <div class="card empty-state">
        <p class="muted">
          No gateway yet. A gateway is one eRPC instance fronting however many chains you
          list; it addresses a chain by URL path, so a single port serves all of them —
          and the same path serves WebSocket.
        </p>
      </div>
    `;
  }

  // gatewayBlock is one gateway: the full-width bar, and under it the table of
  // every chain it fronts with the endpoints that serve each one.
  function gatewayBlock(gw: api.GatewayView): string {
    return `
      <section class="rpc-gateway">
        ${gatewayBar(gw)}
        ${gw.error ? errorBlock(gw) : ""}
        ${gw.blocked ? `<div class="banner banner-warn">${escapeHtml(gw.blocked)}</div>` : ""}
        ${(gw.warnings ?? []).map((wmsg) => `<div class="banner banner-warn">${escapeHtml(wmsg)}</div>`).join("")}
        ${tlsBanner(gw)}
        ${actionErr[gw.id] ? `<p class="error small">${escapeHtml(actionErr[gw.id]!)}</p>` : ""}
        ${activityBlock(gw)}
        ${settingsOpen[gw.id] ? settingsBlock(gw) : ""}
        ${networksTable(gw)}
      </section>
    `;
  }

  // ---- TIER 1: the bar ---------------------------------------------------

  function gatewayBar(gw: api.GatewayView): string {
    const running = gw.status.State === "running";
    return `
      <div class="rpc-bar${running ? "" : " rpc-bar-down"}">
        <div class="rpc-bar-head">
          <div class="rpc-bar-id">
            ${stateDot(gw)}
            <strong>${escapeHtml(gw.label)}</strong>
            ${stateBadge(gw)}
            <span class="muted small">on ${escapeHtml(gw.placement.targetId)} · ${escapeHtml(gw.placement.backend)}</span>
          </div>
          <div class="rpc-bar-actions">
            ${(gw.actions ?? []).map((a) => actionButton(gw, a)).join("")}
            <button class="btn btn-ghost" data-action="toggle-settings" data-gid="${escapeHtml(gw.id)}">
              ${settingsOpen[gw.id] ? "Close" : "Settings"}
            </button>
            <button class="btn btn-ghost" data-action="forget-gateway" data-gid="${escapeHtml(gw.id)}"
                    title="Remove this gateway from valve-node-app. Its container is left alone.">Forget…</button>
          </div>
        </div>
        <div class="rpc-bar-url">
          ${
            running
              ? `<code class="endpoint-url">${escapeHtml(gw.baseUrl)}</code>
                 <button class="btn btn-ghost" data-action="copy" data-copy="${escapeHtml(gw.baseUrl)}">Copy</button>
                 <span class="muted small">a chain is addressed by path, e.g. <code>${escapeHtml((gw.networks ?? [])[0]?.path ?? "/main/evm/&lt;chainId&gt;")}</code></span>`
              : `<span class="muted small">Not serving — it will answer on <code>${escapeHtml(gw.baseUrl)}</code> once it is running.</span>`
          }
        </div>
      </div>
    `;
  }

  function stateBadge(gw: api.GatewayView): string {
    switch (gw.status.State) {
      case "running":
        return badge("running", "ok");
      case "created-but-stopped":
        return badge("stopped", "warn");
      case "not-created":
        return badge("not created", "neutral");
      default:
        return badge("unknown", "bad");
    }
  }

  function stateDot(gw: api.GatewayView): string {
    if (gw.status.State === "running") return dot("ok");
    if (gw.status.State === "unknown") return dot("bad");
    return dot("neutral");
  }

  function errorBlock(gw: api.GatewayView): string {
    return `
      <div class="banner banner-bad">
        <strong>This gateway could not be read.</strong>
        <div class="small">${escapeHtml(gw.error ?? "")}</div>
        ${gw.hint ? `<div class="small">${escapeHtml(gw.hint)}</div>` : ""}
      </div>
    `;
  }

  function actionButton(gw: api.GatewayView, action: string): string {
    const def = ACTION_BUTTONS[action];
    if (!def) return "";
    const running = busy[gw.id];
    return `
      <button class="${def.className}" data-action="gw-${action}" data-gid="${escapeHtml(gw.id)}"
              title="${escapeHtml(def.title)}" ${running ? "disabled" : ""}>
        ${running === action ? `<span class="spinner" aria-label="working"></span>` : escapeHtml(def.label)}
      </button>
    `;
  }

  function activityBlock(gw: api.GatewayView): string {
    const lines = activity[gw.id] ?? [];
    if (lines.length === 0) return "";
    return `
      <div class="config-block">
        <p class="muted small">Provisioning on ${escapeHtml(gw.placement.targetId)}</p>
        <pre class="step-log">${escapeHtml(lines.join("\n"))}</pre>
      </div>
    `;
  }

  // ---- TIER 2: the chains, inside the bar --------------------------------

  // ---- the table: bands wrapping their endpoints -------------------------

  // networksTable renders every chain this gateway fronts, each as a band row
  // spanning the full width with its endpoints beneath it.
  //
  // There is no selection and no focused chain. A gateway's job is fronting
  // several chains, so "which of my chains is misrouting" has to be answerable
  // by looking, not by clicking through each one in turn.
  function networksTable(gw: api.GatewayView): string {
    const nets = gw.networks ?? [];
    if (nets.length === 0) {
      return `
        <div class="card rpc-surface">
          <p class="muted small">
            No networks yet. eRPC refuses a configuration with none, so add one before
            creating the gateway.
          </p>
          <div class="card-actions">
            <button class="btn" data-action="add-chain" data-gid="${escapeHtml(gw.id)}">Add a network</button>
          </div>
        </div>
      `;
    }
    return `
      <div class="card rpc-surface">
        ${surfaceHead(gw)}
        <div class="surface-scroll">
          <table class="surface">
            <thead>
              <tr>
                <th class="col-endpoint">Endpoint</th>
                <th>Role</th>
                <th>State</th>
                <th>Capabilities</th>
                <th class="col-share">Share of traffic</th>
                <th class="col-act"></th>
              </tr>
            </thead>
            <tbody>
              ${nets.map((n) => networkBand(gw, n) + endpointRows(gw, n)).join("")}
            </tbody>
          </table>
        </div>
        ${trafficFootnote(gw)}
      </div>
    `;
  }

  // surfaceHead carries the one control that acts on the whole table — a
  // capability re-probe — and says when the probes last ran. A capability
  // verdict with no timestamp invites being read as live, and it is not: it is
  // cached precisely because probing opens real sockets.
  function surfaceHead(gw: api.GatewayView): string {
    const c = caps[gw.id];
    const when = c?.at ? `probed ${escapeHtml(shortTime(c.at))}` : "not probed yet";
    return `
      <div class="surface-head">
        <span class="muted small">${when}</span>
        <button class="btn btn-ghost" data-action="reprobe" data-gid="${escapeHtml(gw.id)}"
                title="Ask every endpoint what it can do, again. This opens real connections to them."
                ${capsBusy[gw.id] ? "disabled" : ""}>
          ${capsBusy[gw.id] ? `<span class="spinner" aria-label="probing"></span>` : "Re-probe"}
        </button>
        <button class="btn btn-ghost" data-action="add-chain" data-gid="${escapeHtml(gw.id)}">+ Network</button>
      </div>
    `;
  }

  // networkBand is the row that WRAPS a chain's endpoints. Its pill states the
  // one thing worth knowing about the chain as a whole, in priority order:
  // unserviceable beats under-used beats healthy, because a chain that cannot
  // answer at all makes the traffic split irrelevant.
  function networkBand(gw: api.GatewayView, n: api.NetworkView): string {
    const broken = !n.serviceable;
    return `
      <tr class="band${broken ? " band-bad" : ""}">
        <td colspan="6">
          <div class="band-inner">
            <span class="band-id">${n.chainId}</span>
            <span class="band-name">${escapeHtml(n.name)}</span>
            <code class="band-path">${escapeHtml(n.path)}</code>
            ${
              n.url
                ? `<button class="btn btn-ghost btn-tiny" data-action="copy" data-copy="${escapeHtml(n.url)}"
                           title="Copy ${escapeHtml(n.url)}">Copy URL</button>`
                : ""
            }
            <span class="band-right">
              ${bandPill(gw, n)}
              <button class="btn btn-ghost btn-tiny" data-action="add-endpoint"
                      data-gid="${escapeHtml(gw.id)}" data-chain="${n.chainId}">+ Endpoint</button>
              <button class="btn btn-ghost btn-tiny" data-action="remove-chain"
                      data-gid="${escapeHtml(gw.id)}" data-chain="${n.chainId}">Remove</button>
            </span>
          </div>
          ${(n.warnings ?? []).map((wmsg) => `<div class="band-warn">${escapeHtml(wmsg)}</div>`).join("")}
        </td>
      </tr>
    `;
  }

  function bandPill(gw: api.GatewayView, n: api.NetworkView): string {
    if (!n.serviceable) return badge("no usable endpoint", "bad");

    // "Subscriptions unavailable" is a chain-level fact even though it is
    // measured per endpoint: eth_subscribe fails on this path when NOTHING
    // behind it speaks WebSocket, and that is invisible on any single row.
    const ups = n.upstreams ?? [];
    const probed = ups
      .map((u) => capsOf(gw.id, n.chainId, u.id))
      .filter((e): e is api.EndpointCapabilities => !!e && !e.unprobeable);
    if (probed.length > 0 && probed.every((e) => statusOf(e, "ws") === "unsupported")) {
      return badge("subscriptions unavailable", "bad");
    }

    // Under-used: a preferred endpoint carrying materially less than intended.
    // Stated on the band because the cause is usually one row down, and the
    // band is where the eye lands first.
    const shares = ups.map((u) => shareOf(gw.id, n.chainId, u.id));
    if (shares.some((s, i) => s && s.diverged && (ups[i]?.local ?? false))) {
      return badge("your endpoint is under-used", "warn");
    }
    return badge(`${ups.length} endpoint${ups.length === 1 ? "" : "s"}`, "ok");
  }

  // endpointRows renders a chain's servers. It states what each endpoint IS
  // before what it is called, because "the devnet on this machine" is the fact
  // an operator reasons about and the URL is the consequence.
  function endpointRows(gw: api.GatewayView, n: api.NetworkView): string {
    const ups = n.upstreams ?? [];
    if (ups.length === 0) {
      return `
        <tr class="ep"><td colspan="6" class="muted small">
          No endpoint yet, so there is nowhere for calls on this path to go.
        </td></tr>
      `;
    }
    return ups.map((u) => endpointRow(gw, n, u)).join("");
  }

  function endpointRow(gw: api.GatewayView, n: api.NetworkView, u: api.UpstreamView): string {
    const key = `${gw.id}|${n.chainId}|${u.id}`;
    const actions = u.actions ?? [];
    return `
      <tr class="ep${u.problem ? " ep-bad" : ""}">
        <td class="col-endpoint">
          <div class="ep-what">
            ${u.problem ? dot("bad") : dot("ok")}
            <span class="ep-label">${escapeHtml(u.label)}</span>
          </div>
          <code class="ep-url">${escapeHtml(u.endpoint || "—")}</code>
          ${u.problem ? `<div class="error small">${escapeHtml(u.problem)}</div>` : ""}
        </td>
        <td>${u.local ? "Yours" : "Public"}</td>
        <td>${epStateBadge(u)}</td>
        <td>${capCell(gw, n, u)}</td>
        <td class="col-share">${shareCell(gw, n, u)}</td>
        <td class="col-act">
          ${
            actions.includes("reset")
              ? `<button class="btn btn-ghost btn-tiny" data-action="reset-devnet" data-key="${escapeHtml(key)}"
                         data-target="${escapeHtml(u.targetId ?? "")}"
                         title="Throw this devnet's chain away and start again from genesis. It is a scratch chain — this is routine."
                         ${busy[gw.id] ? "disabled" : ""}>
                   ${busy[gw.id] === "reset" ? `<span class="spinner" aria-label="working"></span>` : "Reset"}
                 </button>`
              : ""
          }
          <button class="btn btn-ghost btn-tiny" data-action="remove-endpoint" data-key="${escapeHtml(key)}">Remove</button>
        </td>
      </tr>
    `;
  }

  function epStateBadge(u: api.UpstreamView): string {
    if (u.problem) return badge("unusable", "bad");
    if (u.recentOnly) return badge("recent blocks", "warn");
    return u.local ? badge("serving", "ok") : badge("fallback", "neutral");
  }

  // ---- capabilities ------------------------------------------------------

  // statusOf resolves one capability, including the one that is not in the
  // capability list at all.
  //
  // There is no "http" probe and there should not be: answering JSON-RPC over
  // HTTP is what REACHABILITY means, and the prober records it as
  // Endpoint.Reachable rather than as an eleventh method call. Synthesising the
  // tag from that here is what keeps the column honest — reading the absent key
  // literally would render HTTP as permanently unknown on every endpoint,
  // including ones we had just successfully talked to.
  function statusOf(e: api.EndpointCapabilities | undefined, key: string): api.CapabilityStatus | undefined {
    if (!e) return undefined;
    if (key === "http") {
      if (e.unprobeable) return "inconclusive";
      return e.reachable ? "supported" : "unsupported";
    }
    return (e.capabilities ?? []).find((c) => c.key === key)?.status;
  }

  // capCell renders the full capability set, always, so an absence is a
  // visible gap rather than a missing element nobody notices.
  //
  // The three off-states are deliberately distinct. "unsupported" is grey and
  // struck through — the endpoint simply does not offer it. "missing" is red —
  // the same absence, but on a chain where nothing else offers it either, so it
  // is actually breaking something. "inconsistent" is its own mark, because a
  // load-balanced endpoint whose members disagree is a fact worth seeing rather
  // than a number to average away.
  function capCell(gw: api.GatewayView, n: api.NetworkView, u: api.UpstreamView): string {
    const e = capsOf(gw.id, n.chainId, u.id);
    if (!e) {
      return `<span class="muted small">${caps[gw.id] === undefined ? "probing…" : "—"}</span>`;
    }
    if (e.unprobeable) {
      // A stated reason, not a blank. "We could not ask from here" is a
      // different claim from "it cannot do this", and conflating them would
      // mark a perfectly capable endpoint as lacking everything.
      return `<span class="caps-none" title="${escapeHtml(e.unprobeable)}">not probeable from here</span>`;
    }
    return `<span class="caps">${CAP_ORDER.map((k) => capTag(gw, n, e, k)).join("")}</span>`;
  }

  function capTag(
    gw: api.GatewayView,
    n: api.NetworkView,
    e: api.EndpointCapabilities,
    key: string,
  ): string {
    const cap = (e.capabilities ?? []).find((c) => c.key === key);
    const status = statusOf(e, key) ?? "inconclusive";
    const tag = CAP_TAGS[key] ?? key.toUpperCase();

    let cls = "cap";
    if (status === "unsupported") {
      cls = chainLacksEntirely(gw, n, key) ? "cap missing" : "cap off";
    } else if (status === "inconclusive") {
      cls = "cap unknown";
    } else if (status === "inconsistent") {
      cls = "cap mixed";
    }

    // The evidence, not just the verdict. HTTP borrows the reachability
    // detail, which is the sentence that actually explains it.
    const why = cap?.detail
      ? `${cap.label}: ${cap.detail}`
      : key === "http" && e.reachDetail
        ? `Answers JSON-RPC over HTTP: ${e.reachDetail}`
        : `${tag}: no verdict`;
    return `<span class="${cls}" title="${escapeHtml(why)}">${escapeHtml(tag)}</span>`;
  }

  // chainLacksEntirely is what turns a grey absence red: this capability is
  // missing from EVERY probed endpoint on the chain, so the gap is not a
  // property of one endpoint, it is a hole in the path.
  function chainLacksEntirely(gw: api.GatewayView, n: api.NetworkView, key: string): boolean {
    const probed = (n.upstreams ?? [])
      .map((u) => capsOf(gw.id, n.chainId, u.id))
      .filter((e): e is api.EndpointCapabilities => !!e && !e.unprobeable);
    return probed.length > 0 && probed.every((e) => statusOf(e, key) === "unsupported");
  }

  // ---- traffic share -----------------------------------------------------

  // shareCell is the bar: actual as the fill, intended as a tick, the number
  // beside it. When the two diverge the number goes amber, because that gap is
  // the symptom worth chasing — a latency chart never tells you that your own
  // node is being bypassed.
  function shareCell(gw: api.GatewayView, n: api.NetworkView, u: api.UpstreamView): string {
    const t = traffic[gw.id];
    if (t === undefined) return `<span class="muted small">reading…</span>`;
    if (t === null) return `<span class="muted small" title="The counters could not be read.">—</span>`;
    if (!t.enabled) {
      return `<span class="muted small" title="This gateway's request counters are turned off in its settings.">counters off</span>`;
    }

    const s = shareOf(gw.id, n.chainId, u.id);
    const net = (t.networks ?? []).find((x) => x.chainId === n.chainId);
    if (!s || !net || net.attributed === 0) {
      // No traffic is not a share of zero — there is no denominator. Saying
      // "no traffic yet" beats drawing an empty bar that reads as "this
      // endpoint is being starved".
      return `<span class="muted small">no traffic yet</span>`;
    }

    const pct = Math.round(s.actual * 100);
    const tickPct = Math.round(s.intended * 100);
    const fill = s.diverged ? (u.local ? "warn" : "") : "ok";
    const title =
      `${s.succeeded.toLocaleString()} of ${net.attributed.toLocaleString()} answered requests` +
      ` · routing intends ${tickPct}%` +
      (s.unconfigured ? " · this endpoint is no longer in the saved configuration" : "");

    return `
      <span class="share" title="${escapeHtml(title)}">
        <span class="bar">
          <span class="fill${fill ? " " + fill : ""}" style="width:${pct}%"></span>
          <span class="tick" style="left:${tickPct}%"></span>
        </span>
        <span class="share-n${s.diverged ? " warn" : ""}">${pct}%</span>
        ${s.unconfigured ? badge("not in config", "warn") : ""}
      </span>
    `;
  }

  // trafficFootnote says what window the numbers cover and what they cost. A
  // cumulative counter shown without its start looks like a live rate.
  function trafficFootnote(gw: api.GatewayView): string {
    const t = traffic[gw.id];
    if (!t) return "";
    if (!t.enabled) {
      return `<p class="muted small">
        This gateway is not counting its requests, so there is no traffic share to show.
        Turn the counters on in Settings — they stay on the machine the gateway runs on
        and nothing is sent anywhere.
      </p>`;
    }
    if (t.error) {
      return `<p class="muted small">The request counters could not be read: ${escapeHtml(t.error)}</p>`;
    }
    return `<p class="muted small">
      Share is measured from the gateway's own counters since it started${
        t.since ? ` (${escapeHtml(shortTime(t.since))})` : ""
      }. The tick is the share routing intends: your own endpoints carry a chain, public
      ones are there for when they cannot.
    </p>`;
  }

  // shortTime renders an ISO timestamp as a local time, falling back to the
  // raw string rather than showing "Invalid Date" if the server ever sends
  // something unexpected.
  function shortTime(iso: string): string {
    const d = new Date(iso);
    return Number.isNaN(d.getTime()) ? iso : d.toLocaleString();
  }

  // ---- the bar's settings ------------------------------------------------

  function settingsBlock(gw: api.GatewayView): string {
    const c = gw.config;
    return `
      <div class="card config-block">
        <p class="muted small">Gateway settings — saved here, applied by “Re-create”.</p>
        <label>
          Listen port
          <input type="text" inputmode="numeric" id="gw-${escapeHtml(gw.id)}-port" value="${c.Port}" autocomplete="off" />
        </label>
        <label>
          Bind address <span class="muted">— 127.0.0.1 keeps it on that machine; 0.0.0.0 exposes it to your network</span>
          <input type="text" id="gw-${escapeHtml(gw.id)}-bind" value="${escapeHtml(c.BindAddr)}" autocomplete="off" spellcheck="false" />
        </label>
        <p class="muted small">
          Requests are addressed by path: <code>/${escapeHtml(c.ProjectID)}/evm/&lt;chainId&gt;</code>. One port serves every
          network in the bar above, and the same path serves WebSocket with a <code>ws://</code> scheme.
        </p>
        ${metricsField(gw)}
        ${tlsFields(gw)}
        <div class="card-actions">
          <button class="btn" data-action="save-settings" data-gid="${escapeHtml(gw.id)}">Save settings</button>
        </div>
      </div>
    `;
  }

  // metricsField is the counters' off switch.
  //
  // It says what the thing IS rather than naming a technology, because
  // "metrics" is a word that makes people think of telemetry leaving the
  // machine, and this is the opposite of that: eRPC counting its own requests,
  // in its own process, on a port bound to loopback. Saying so plainly is what
  // makes leaving it on an informed choice rather than an unread default.
  function metricsField(gw: api.GatewayView): string {
    const on = !gw.config.MetricsOff;
    return `
      <label class="check">
        <input type="checkbox" id="gw-${escapeHtml(gw.id)}-metrics" ${on ? "checked" : ""} />
        Count this gateway's own requests
      </label>
      <p class="muted small">
        The gateway counts which endpoints answer its requests, so this screen can show
        where your traffic is actually going. The counters stay on the machine the gateway
        runs on — they are served on loopback and nothing is sent anywhere. Turn this off
        and the share column goes blank.
      </p>
    `;
  }

  // tlsFields is the HTTPS front, edited in the same block as the port and
  // bind because it IS the port and bind once it is on: a fronted gateway
  // publishes no plaintext port at all, and the https URL replaces the http
  // one everywhere.
  //
  // Two sources, no ACME. "internal" needs no domain, no network and no
  // service to run, at the cost of one trust-store install per machine;
  // "files" is a certificate already on disk (`tailscale cert`, a bundled one,
  // localhost.direct). ACME is deliberately absent — for a name resolving to
  // loopback the only usable challenge is DNS-01, which needs zone-write
  // credentials, and that is worse than the trust-store click it saves.
  function tlsFields(gw: api.GatewayView): string {
    const id = escapeHtml(gw.id);
    const t = gw.config.TLS ?? null;
    const on = t?.Enabled ?? false;
    const source = t?.CertSource || "internal";
    const suggested = gw.tls?.suggestedHostname ?? "";
    return `
      <hr />
      <label class="check">
        <input type="checkbox" id="gw-${id}-tls" ${on ? "checked" : ""} />
        Serve HTTPS (a Caddy container in front of eRPC)
      </label>
      <p class="muted small">
        A page served over <code>https://</code> cannot call an <code>http://</code> endpoint. Chrome and Firefox make an
        exception for <code>http://localhost</code>; Safari does not, and every browser blocks it for any other address —
        so a gateway on a LAN or Tailscale address is unusable from a browser dApp without this.
      </p>
      <label>
        Hostname <span class="muted">— must resolve to this machine</span>
        <input type="text" id="gw-${id}-tls-host" value="${escapeHtml(t?.Hostname ?? suggested)}"
               placeholder="${escapeHtml(suggested || "gateway.example.com")}" autocomplete="off" spellcheck="false" />
      </label>
      ${
        suggested
          ? `<p class="muted small">
               The default is <code>${escapeHtml(suggested)}</code>. That whole domain's wildcard resolves to
               <code>127.0.0.1</code> from any network, so the name works on this machine with nothing to install and
               no hosts file to edit — and it is unique to this install, so two machines never serve different
               certificates for the same name.
             </p>`
          : ""
      }
      <label>
        HTTPS port
        <input type="text" inputmode="numeric" id="gw-${id}-tls-port" value="${t?.HTTPSPort || 443}" autocomplete="off" />
      </label>
      <label>
        Certificate
        <select id="gw-${id}-tls-source">
          <option value="internal" ${source === "internal" ? "selected" : ""}>Caddy's own authority — works offline, one trust-store install</option>
          <option value="files" ${source === "files" ? "selected" : ""}>A certificate file on this machine</option>
        </select>
      </label>
      <label>
        Certificate file <span class="muted">— path on that machine, used only for “a certificate file”</span>
        <input type="text" id="gw-${id}-tls-cert" value="${escapeHtml(t?.CertFile ?? "")}"
               placeholder="/var/lib/valve-node-app/tls/cert.pem" autocomplete="off" spellcheck="false" />
      </label>
      <label>
        Private key file
        <input type="text" id="gw-${id}-tls-key" value="${escapeHtml(t?.KeyFile ?? "")}"
               placeholder="/var/lib/valve-node-app/tls/key.pem" autocomplete="off" spellcheck="false" />
      </label>
      <p class="muted small">
        If that certificate is missing, unreadable, expired or does not cover the hostname, HTTPS stays on and falls
        back to Caddy's own authority — with the reason shown above. A dead endpoint is worse than a one-time browser
        warning, and certificate lifetimes are shrinking every year.
      </p>
      ${verifyPanel(gw)}
    `;
  }

  // verifyPanel is the live "is HTTPS actually serving?" check, sitting with
  // the TLS settings because that is where the question is asked.
  //
  // It is a BUTTON, not something that runs on every render: the check opens
  // real connections, subscribes, and waits for a block. What it shows is
  // per-assertion, because "HTTPS works" is five different claims and the
  // interesting failures are the ones where four of them hold — a certificate
  // for the wrong name, or a front that serves every call except the
  // subscriptions a dApp is waiting on.
  function verifyPanel(gw: api.GatewayView): string {
    const gid = escapeHtml(gw.id);
    const on = gw.config.TLS?.Enabled ?? false;
    const res = verifyResult[gw.id] ?? gw.tls?.verification ?? null;
    const busy = verifyBusy[gw.id] ?? false;
    const err = verifyErr[gw.id] ?? null;

    return `
      <hr />
      <div class="card-actions">
        <button class="btn btn-ghost" data-action="verify-tls" data-gid="${gid}" ${on && !busy ? "" : "disabled"}
                title="Open a real connection to this front: handshake, certificate name, chain, an RPC call and a subscription.">
          ${busy ? `<span class="spinner" aria-label="verifying"></span> Verifying…` : "Verify HTTPS now"}
        </button>
        ${on ? "" : `<span class="muted small">Turn HTTPS on and re-create the gateway — there is nothing to verify yet.</span>`}
      </div>
      ${err ? `<p class="error small">${escapeHtml(err)}</p>` : ""}
      ${res ? verifyReport(res) : ""}
    `;
  }

  function verifyReport(v: api.TlsVerification): string {
    const rows = (v.assertions ?? [])
      .map(
        (a) => `
          <li class="small">
            ${verifyBadge(a.status)}
            <strong>${escapeHtml(a.title)}</strong>
            <div class="muted">${escapeHtml(a.detail)}</div>
          </li>`,
      )
      .join("");
    return `
      <div class="banner ${v.ok ? (v.subscriptionsOk ? "banner-ok" : "banner-warn") : "banner-bad"}">
        ${escapeHtml(v.summary)}
      </div>
      <ul class="verify-list">${rows}</ul>
      <p class="muted small">
        Checked ${escapeHtml(new Date(v.at).toLocaleString())} against <code>${escapeHtml(v.address)}</code>
        ${v.notAfter ? `· certificate valid until <code>${escapeHtml(new Date(v.notAfter).toLocaleString())}</code> (${escapeHtml(v.expiresIn ?? "")})` : ""}
      </p>
      ${v.expiryWarning ? `<div class="banner banner-warn">${escapeHtml(v.expiryWarning)}</div>` : ""}
    `;
  }

  function verifyBadge(status: string): string {
    switch (status) {
      case "pass":
        return badge("pass", "ok");
      case "fail":
        return badge("fail", "bad");
      case "unavailable":
        return badge("unavailable", "warn");
      default:
        return badge("skipped", "neutral");
    }
  }

  async function verifyTLS(gid: string): Promise<void> {
    verifyBusy[gid] = true;
    verifyErr[gid] = null;
    render();
    try {
      verifyResult[gid] = await api.verifyGatewayTls(gid);
    } catch (err) {
      verifyErr[gid] = `${message(err)}${hintOf(err)}`;
    } finally {
      verifyBusy[gid] = false;
      render();
    }
  }

  // tlsBanner is what the front is ACTUALLY doing, as opposed to what was
  // configured. It is separate from the settings block because it must be
  // visible without opening anything: a silent fallback would be the failure
  // the fallback exists to prevent, just relocated.
  function tlsBanner(gw: api.GatewayView): string {
    const t = gw.tls;
    if (!t?.enabled) return "";
    const parts: string[] = [];
    if (t.fallback) {
      parts.push(`<div class="banner banner-warn">${escapeHtml(t.fallback)}</div>`);
    }
    if (t.error) {
      parts.push(`<div class="banner banner-warn">HTTPS front: ${escapeHtml(t.error)}</div>`);
    } else if (t.status?.State !== "running") {
      parts.push(
        `<div class="banner banner-warn">The HTTPS front (<code>${escapeHtml(t.containerName ?? "")}</code>) is
         ${escapeHtml(t.status?.State ?? "unknown")}, so nothing is answering on
         <code>${escapeHtml(t.url ?? "")}</code> even if the gateway itself is up.</div>`,
      );
    }
    // The last live verification, surfaced WITHOUT opening settings when it
    // found something. A front that terminates TLS perfectly and refuses every
    // subscription looks identical to a healthy one from out here, so the one
    // place that knows says so where the state is shown.
    const v = verifyResult[gw.id] ?? t.verification ?? null;
    if (v && (!v.ok || !v.subscriptionsOk)) {
      parts.push(
        `<div class="banner ${v.ok ? "banner-warn" : "banner-bad"}">${escapeHtml(v.summary)}
         <div class="small">Checked ${escapeHtml(new Date(v.at).toLocaleString())} — open Settings for the full check.</div></div>`,
      );
    }
    if (v?.expiryWarning) {
      parts.push(`<div class="banner banner-warn">${escapeHtml(v.expiryWarning)}</div>`);
    }
    if (t.rootCaPath && t.effectiveCertSource === "internal") {
      parts.push(
        `<p class="muted small">This gateway is served by Caddy's own certificate authority. Install
         <code>${escapeHtml(t.rootCaPath)}</code> (on ${escapeHtml(gw.placement.targetId)}) into the trust store of every
         device that will call it, and the browser warning goes away.</p>`,
      );
    }
    return parts.join("");
  }

  // --- config editing -----------------------------------------------------

  // storedConfig is a deep copy of what the SERVER holds, which is the thing
  // an edit must be applied to. Copying matters: every edit below mutates it
  // before PUTting, and mutating the rendered view in place would leave the
  // screen showing a change that was never saved.
  function storedConfig(gw: api.GatewayView): api.GatewayConfig {
    return {
      ...gw.config,
      Networks: (gw.config.Networks ?? []).map((n) => ({
        ChainID: n.ChainID,
        Upstreams: n.Upstreams.map((u) => ({ ...u })),
      })),
    };
  }

  async function saveConfig(gid: string, cfg: api.GatewayConfig, note?: string): Promise<boolean> {
    actionErr[gid] = null;
    try {
      await api.putGatewayConfig(gid, cfg);
    } catch (err) {
      actionErr[gid] = `${note ? note + ": " : ""}${message(err)}`;
      render();
      return false;
    }
    await load();
    return true;
  }

  // --- actions ------------------------------------------------------------

  async function handleAction(action: string, el: HTMLElement): Promise<void> {
    const gid = el.dataset.gid ?? "";
    switch (action) {
      case "refresh":
        await load();
        return;
      case "copy":
        if (el.dataset.copy) await copyButton(el, el.dataset.copy);
        return;
      case "reprobe":
        await loadCapabilities(gid, true);
        return;
      case "toggle-settings":
        settingsOpen[gid] = !settingsOpen[gid];
        render();
        return;
      case "save-settings":
        await saveSettings(gid);
        return;
      case "verify-tls":
        await verifyTLS(gid);
        return;
      case "gw-start":
      case "gw-stop":
      case "gw-restart":
        await runAction(gid, action.slice("gw-".length) as api.ContainerActionKind);
        return;
      case "gw-create":
      case "gw-recreate":
        await provision(gid);
        return;
      case "gw-wipe":
        openWipeModal(gid);
        return;
      case "add-gateway":
        openAddGatewayModal();
        return;
      case "forget-gateway":
        await forgetGateway(gid);
        return;
      case "add-chain":
        openAddChainModal(gid);
        return;
      case "remove-chain":
        await removeChain(gid, Number.parseInt(el.dataset.chain ?? "", 10));
        return;
      case "add-endpoint":
        openAddEndpointModal(gid, Number.parseInt(el.dataset.chain ?? "", 10));
        return;
      case "remove-endpoint":
        await removeEndpoint(el.dataset.key ?? "");
        return;
      case "reset-devnet":
        await resetDevnet(el.dataset.key ?? "", el.dataset.target ?? "");
        return;
      default:
        return;
    }
  }

  async function saveSettings(gid: string): Promise<void> {
    const gw = gatewayOf(gid);
    if (!gw) return;
    const cfg = storedConfig(gw);
    const portEl = root.querySelector<HTMLInputElement>(`#gw-${CSS.escape(gid)}-port`);
    const bindEl = root.querySelector<HTMLInputElement>(`#gw-${CSS.escape(gid)}-bind`);
    if (portEl) {
      const n = Number.parseInt(portEl.value.trim(), 10);
      if (Number.isFinite(n)) cfg.Port = n;
    }
    if (bindEl) cfg.BindAddr = bindEl.value.trim();
    const metricsEl = root.querySelector<HTMLInputElement>(`#gw-${CSS.escape(gid)}-metrics`);
    // Stored as the negative, matching the server: the zero value has to mean
    // ON, so that a config written before this setting existed keeps counting
    // rather than silently going dark on its next save.
    if (metricsEl) cfg.MetricsOff = !metricsEl.checked;
    cfg.TLS = readTLS(gid, gw);

    const wasRunning = gw.status.State === "running";
    if (await saveConfig(gid, cfg, "Saving settings")) {
      settingsOpen[gid] = false;
      if (wasRunning) {
        actionErr[gid] = null;
        note(gid, "Saved. The running container still has the old port and bind — press “Re-create (apply config)” to put them into effect.");
      }
      render();
    }
  }

  // readTLS builds the stored TLS block from the form. A DISABLED front is
  // kept rather than dropped, so turning HTTPS off and on again does not lose
  // the hostname and cert paths that were typed.
  function readTLS(gid: string, gw: api.GatewayView): api.GatewayTLS | null {
    const pick = <T extends HTMLElement>(suffix: string) =>
      root.querySelector<T>(`#gw-${CSS.escape(gid)}-${suffix}`);
    const enabledEl = pick<HTMLInputElement>("tls");
    if (!enabledEl) return gw.config.TLS ?? null;

    const port = Number.parseInt(pick<HTMLInputElement>("tls-port")?.value.trim() ?? "", 10);
    return {
      Enabled: enabledEl.checked,
      Hostname: pick<HTMLInputElement>("tls-host")?.value.trim() ?? "",
      CertSource: pick<HTMLSelectElement>("tls-source")?.value ?? "internal",
      CertFile: pick<HTMLInputElement>("tls-cert")?.value.trim() ?? "",
      KeyFile: pick<HTMLInputElement>("tls-key")?.value.trim() ?? "",
      HTTPSPort: Number.isFinite(port) ? port : 443,
      BindAddr: gw.config.TLS?.BindAddr ?? "",
      ImageRef: gw.config.TLS?.ImageRef ?? "",
    };
  }

  function note(gid: string, text: string): void {
    activity[gid] = [text];
  }

  async function runAction(gid: string, kind: api.ContainerActionKind): Promise<void> {
    if (busy[gid]) return;
    busy[gid] = kind;
    actionErr[gid] = null;
    render();
    try {
      await api.gatewayAction(gid, kind);
    } catch (err) {
      actionErr[gid] = `${kind} failed: ${message(err)}${hintOf(err)}`;
    }
    busy[gid] = null;
    await load();
  }

  // provision runs the gateway's setup plan and follows the PLACEMENT
  // machine's setup event stream — the same per-machine stream the node
  // wizard and the devnet use. The server tells us which machine that is
  // rather than this file re-deriving the placement rule.
  async function provision(gid: string): Promise<void> {
    if (busy[gid]) return;
    busy[gid] = "create";
    actionErr[gid] = null;
    activity[gid] = ["starting…"];
    render();

    let started: { targetId: string };
    try {
      started = await api.provisionGateway(gid);
    } catch (err) {
      actionErr[gid] = `${message(err)}${hintOf(err)}`;
      activity[gid] = [];
      busy[gid] = null;
      render();
      return;
    }

    streamStop?.();
    streamStop = api.streamSetup(started.targetId, (ev) => {
      if (disposed) return;
      const line = ev.err ? `${ev.stepId}: ${ev.err}` : ev.line ? `${ev.stepId}: ${ev.line}` : `${ev.stepId}: done`;
      activity[gid] = [...(activity[gid] ?? []).filter((l) => l !== "starting…"), line];
      const finished = !!ev.err || (ev.stepId === FINAL_STEP && !!ev.done);
      if (finished) {
        streamStop?.();
        streamStop = null;
        busy[gid] = null;
        if (ev.err) actionErr[gid] = "Provisioning failed — see the log below.";
        void load();
        return;
      }
      render();
    });
  }

  async function forgetGateway(gid: string): Promise<void> {
    const gw = gatewayOf(gid);
    if (!gw) return;
    const ok = await confirmModal({
      title: `Forget ${gw.label}`,
      body: `valve-node-app will forget this gateway's configuration. Its container "${gw.containerName}" on ${gw.placement.targetId} is NOT touched — if it is running it keeps running and keeps serving. Stop or wipe it first if you wanted it gone.`,
      confirmLabel: "Forget it",
      danger: true,
    });
    if (!ok) return;
    try {
      await api.deleteGateway(gid);
    } catch (err) {
      actionErr[gid] = message(err);
      render();
      return;
    }
    await load();
  }

  // --- chains -------------------------------------------------------------

  // openAddChainModal offers the catalog's networks, then the devnet, then a
  // custom chain id — and never offers one that is already there, because an
  // option that can only fail is worse than no option.
  function openAddChainModal(gid: string): void {
    const gw = gatewayOf(gid);
    if (!gw) return;
    const present = new Set((gw.networks ?? []).map((n) => n.chainId));
    const presets = data?.presets ?? [];
    const available = presets.filter((p) => !present.has(p.chainId));
    const already = presets.filter((p) => present.has(p.chainId));

    // The devnet preset is the one that can provision itself. Whether it can
    // is a fact about the PLACEMENT machine, so it is resolved before the
    // option is drawn rather than discovered on click.
    const placementHasDevnet = (data?.targets ?? []).some(
      (t) => t.id === gw.placement.targetId && t.hasDevnet,
    );

    openModal(
      `
        <h2>Add a network</h2>
        <p class="muted small">
          eRPC addresses a chain by URL path, so adding one costs no port and no second process —
          it is another path on <code>${escapeHtml(gw.baseUrl)}</code>.
        </p>
        <ul class="plain-list rpc-picker">
          ${available
            .map(
              (p) => `
            <li>
              <button class="btn btn-ghost rpc-picker-option" data-modal-action="preset:${p.chainId}">
                <span>${escapeHtml(p.name)}</span>
                <span class="muted small">chain ${p.chainId}${
                  p.devnet
                    ? placementHasDevnet
                      ? " · uses the devnet on " + escapeHtml(gw.placement.targetId)
                      : " · will create a devnet on " + escapeHtml(gw.placement.targetId)
                    : ""
                }</span>
              </button>
            </li>`,
            )
            .join("")}
          <li>
            <button class="btn btn-ghost rpc-picker-option" data-modal-action="custom">
              <span>Add custom…</span>
              <span class="muted small">any chain id — a gateway can front a chain this app cannot run a node for</span>
            </button>
          </li>
        </ul>
        ${
          already.length
            ? `<p class="muted small">Already fronted: ${escapeHtml(already.map((p) => p.name).join(", "))}.</p>`
            : ""
        }
        <div class="modal-actions">
          <button class="btn btn-ghost" data-modal-action="cancel">Cancel</button>
        </div>
      `,
      (action) => {
        if (action === "cancel") {
          closeModal();
          return;
        }
        if (action === "custom") {
          openCustomChainModal(gid);
          return;
        }
        if (action.startsWith("preset:")) {
          const chainId = Number.parseInt(action.slice("preset:".length), 10);
          const preset = presets.find((p) => p.chainId === chainId);
          closeModal();
          if (preset?.devnet) {
            void addDevnetChain(gid, chainId, placementHasDevnet);
          } else {
            void addChain(gid, chainId);
          }
        }
      },
    );
  }

  function openCustomChainModal(gid: string): void {
    openModal(
      `
        <h2>Add a custom network</h2>
        <p class="muted small">
          Any EVM chain id. Nothing here restricts it to the chains this app can run a node for —
          fronting somebody else's chain is a perfectly good use of a gateway.
        </p>
        <label>
          Chain id
          <input type="text" inputmode="numeric" id="custom-chain-id" autocomplete="off" placeholder="8453" />
        </label>
        <p class="muted small" id="custom-chain-err"></p>
        <div class="modal-actions">
          <button class="btn btn-ghost" data-modal-action="cancel">Cancel</button>
          <button class="btn" data-modal-action="add">Add network</button>
        </div>
      `,
      (action) => {
        if (action === "cancel") {
          closeModal();
          return;
        }
        if (action !== "add") return;
        const input = document.getElementById("custom-chain-id") as HTMLInputElement | null;
        const errEl = document.getElementById("custom-chain-err");
        const chainId = Number.parseInt(input?.value.trim() ?? "", 10);
        if (!Number.isFinite(chainId) || chainId <= 0) {
          if (errEl) errEl.className = "error small";
          if (errEl) errEl.textContent = "A chain id is a positive whole number.";
          return;
        }
        closeModal();
        void addChain(gid, chainId);
      },
    );
    document.getElementById("custom-chain-id")?.focus();
  }

  // addChain adds an empty network. It is deliberately allowed to be empty:
  // "which chain" and "served by what" are two decisions, and forcing them
  // into one modal is what makes an add-a-chain flow feel like a form.
  async function addChain(gid: string, chainId: number): Promise<void> {
    const gw = gatewayOf(gid);
    if (!gw) return;
    const cfg = storedConfig(gw);
    const nets = cfg.Networks ?? [];
    if (nets.some((n) => n.ChainID === chainId)) return;
    nets.push({ ChainID: chainId, Upstreams: [] });
    cfg.Networks = nets;
    // A chain with no upstream cannot be rendered, so it is saved by adding
    // it and letting the server report the network as unserviceable — which
    // it does, on the band, before anything is provisioned.
    if (await saveConfigTolerantly(gid, cfg)) {
      render();
      // Straight into picking an endpoint. A network with none is not saved
      // on the gateway (eRPC refuses it), so leaving the operator on an empty
      // chain would leave them holding something that disappears on the next
      // refresh — the two steps are one intent.
      openAddEndpointModal(gid, chainId);
    }
  }

  // saveConfigTolerantly is saveConfig for the one edit the server's
  // validation legitimately rejects: a network with no upstream yet. The
  // gateway's own config route refuses it (eRPC would), so an empty network
  // is carried with a placeholder-free upstream list only in the UI until an
  // endpoint is added — and the save is attempted so any OTHER error is
  // surfaced honestly.
  async function saveConfigTolerantly(gid: string, cfg: api.GatewayConfig): Promise<boolean> {
    const pruned: api.GatewayConfig = {
      ...cfg,
      Networks: (cfg.Networks ?? []).filter((n) => n.Upstreams.length > 0),
    };
    const ok = await saveConfig(gid, pruned);
    if (!ok) return false;
    // The empty network exists only client-side until it has an endpoint;
    // hold it in the local view so the operator can act on it.
    const gw = gatewayOf(gid);
    if (gw) {
      for (const n of cfg.Networks ?? []) {
        if (n.Upstreams.length === 0 && !(gw.networks ?? []).some((x) => x.chainId === n.ChainID)) {
          gw.config.Networks = [...(gw.config.Networks ?? []), { ChainID: n.ChainID, Upstreams: [] }];
          gw.networks = [
            ...(gw.networks ?? []),
            {
              chainId: n.ChainID,
              // The catalog's own name, so a chain that has just been added
              // reads the same as one the server has already echoed back.
              name: (data?.presets ?? []).find((p) => p.chainId === n.ChainID)?.name ?? `Chain ${n.ChainID}`,
              path: `/${gw.config.ProjectID}/evm/${n.ChainID}`,
              upstreams: [],
              serviceable: false,
              warnings: ["This network has no endpoint yet, so it is not saved on the gateway until you add one."],
            },
          ];
        }
      }
    }
    return true;
  }

  // addDevnetChain is the devnet preset's flow. If the placement machine has
  // no devnet, adding the chain first would leave a network with nothing
  // behind it — so it sends the operator to the machine that can create one
  // instead of quietly producing a broken chain.
  async function addDevnetChain(gid: string, chainId: number, hasDevnet: boolean): Promise<void> {
    const gw = gatewayOf(gid);
    if (!gw) return;
    if (!hasDevnet) {
      openModal(
        `
          <h2>Create a devnet first</h2>
          <p>
            There is no devnet on <code>${escapeHtml(gw.placement.targetId)}</code>, so adding chain ${chainId} here
            would create a network with nothing behind it.
          </p>
          <p class="muted small">
            A devnet belongs to a machine — it is reth in --dev mode in a container on that box —
            so it is created on that machine's own screen. Come back here afterwards and this option
            will point the gateway straight at it.
          </p>
          <div class="modal-actions">
            <button class="btn btn-ghost" data-modal-action="cancel">Cancel</button>
            <a class="btn" href="#/services/${encodeURIComponent(gw.placement.targetId)}" data-modal-action="go">Create a devnet on ${escapeHtml(gw.placement.targetId)}</a>
          </div>
        `,
        () => closeModal(),
      );
      return;
    }

    const cfg = storedConfig(gw);
    const nets = cfg.Networks ?? [];
    // Mirrors catalog.GatewayForDevnet — the devnet as the preferred upstream
    // for its own chain, and no fallback: nothing public serves this chain, so
    // a fallback could only answer for a different one.
    const upstream: api.GatewayUpstream = {
      ID: "devnet",
      Kind: "managed-devnet",
      TargetID: gw.placement.targetId,
      Endpoint: "",
      Local: true,
      RecentOnly: false,
    };
    const existing = nets.find((n) => n.ChainID === chainId);
    if (existing) existing.Upstreams.push(upstream);
    else nets.push({ ChainID: chainId, Upstreams: [upstream] });
    cfg.Networks = nets;
    await saveConfig(gid, cfg, "Adding the devnet");
  }

  async function removeChain(gid: string, chainId: number): Promise<void> {
    const gw = gatewayOf(gid);
    if (!gw || !Number.isFinite(chainId)) return;
    const n = networkOf(gw, chainId);
    const ok = await confirmModal({
      title: `Remove ${n?.name ?? `chain ${chainId}`}`,
      body: `This gateway will stop serving ${n?.path ?? `chain ${chainId}`}. Nothing on the other end is touched — the nodes and endpoints behind it keep running.`,
      confirmLabel: "Remove network",
      danger: true,
    });
    if (!ok) return;
    const cfg = storedConfig(gw);
    cfg.Networks = (cfg.Networks ?? []).filter((x) => x.ChainID !== chainId);
    await saveConfig(gid, cfg, "Removing the network");
  }

  // --- endpoints ----------------------------------------------------------

  // parseKey splits an upstream row's "gid|chain|upstreamId" handle.
  function parseKey(key: string): { gid: string; chainId: number; upstreamId: string } | null {
    const parts = key.split("|");
    if (parts.length !== 3) return null;
    return { gid: parts[0], chainId: Number.parseInt(parts[1], 10), upstreamId: parts[2] };
  }

  async function removeEndpoint(key: string): Promise<void> {
    const parsed = parseKey(key);
    if (!parsed) return;
    const gw = gatewayOf(parsed.gid);
    if (!gw) return;
    const cfg = storedConfig(gw);
    const net = (cfg.Networks ?? []).find((n) => n.ChainID === parsed.chainId);
    if (!net) return;
    const idx = net.Upstreams.findIndex((u, i) => (u.ID || `${parsed.chainId}-${i}`) === parsed.upstreamId);
    if (idx < 0) return;

    const ok = await confirmModal({
      title: "Remove this endpoint",
      body: "The gateway stops routing to it. Whatever is on the other end — a node, a devnet, a public endpoint — is left completely alone.",
      confirmLabel: "Remove",
      danger: true,
    });
    if (!ok) return;
    net.Upstreams.splice(idx, 1);
    await saveConfig(parsed.gid, cfg, "Removing the endpoint");
  }

  // openAddEndpointModal is the three ways an endpoint can come to exist, in
  // the order they are usually wanted: something in the fleet, something
  // public that is actually answering, or a URL typed by hand.
  function openAddEndpointModal(gid: string, chainId: number): void {
    const gw = gatewayOf(gid);
    if (!gw || !Number.isFinite(chainId)) return;
    const sources = (data?.sources ?? []).filter((s) => s.chainId === chainId);
    const net = networkOf(gw, chainId);
    const taken = new Set(
      (net?.upstreams ?? [])
        .filter((u) => u.kind !== "external")
        .map((u) => `${u.kind}|${u.targetId ?? ""}`),
    );
    const offer = sources.filter((s) => !taken.has(`${s.kind}|${s.targetId}`));

    openModal(
      `
        <h2>Add an endpoint for ${escapeHtml(net?.name ?? `chain ${chainId}`)}</h2>
        ${
          offer.length
            ? `<p class="muted small">Machines you manage that serve this chain. These are stored as a reference, not a URL —
                 move the node's port and the gateway follows it.</p>
               <ul class="plain-list rpc-picker">
                 ${offer
                   .map(
                     (s) => `
                   <li>
                     <button class="btn btn-ghost rpc-picker-option" data-modal-action="source:${escapeHtml(s.kind)}:${escapeHtml(s.targetId)}">
                       <span>${escapeHtml(s.label)}</span>
                       <span class="muted small">${escapeHtml(s.endpoint)}</span>
                     </button>
                   </li>`,
                   )
                   .join("")}
               </ul>`
            : `<p class="muted small">No machine you manage serves chain ${chainId}.</p>`
        }
        <div class="modal-actions modal-actions-stack">
          <button class="btn btn-ghost" data-modal-action="discover">Find public endpoints…</button>
          <button class="btn btn-ghost" data-modal-action="manual">Enter a URL by hand…</button>
        </div>
        <div class="modal-actions">
          <button class="btn btn-ghost" data-modal-action="cancel">Cancel</button>
        </div>
      `,
      (action) => {
        if (action === "cancel") {
          closeModal();
          return;
        }
        if (action === "discover") {
          void openDiscoverModal(gid, chainId);
          return;
        }
        if (action === "manual") {
          openManualEndpointModal(gid, chainId);
          return;
        }
        if (action.startsWith("source:")) {
          const [, kind, targetId] = action.split(":");
          closeModal();
          void addManagedUpstream(gid, chainId, kind as api.UpstreamKind, targetId);
        }
      },
    );
  }

  async function addManagedUpstream(
    gid: string,
    chainId: number,
    kind: api.UpstreamKind,
    targetId: string,
  ): Promise<void> {
    const gw = gatewayOf(gid);
    if (!gw) return;
    const cfg = storedConfig(gw);
    const nets = cfg.Networks ?? [];
    const upstream: api.GatewayUpstream = {
      ID: `${kind === "managed-devnet" ? "devnet" : "node"}-${targetId}`,
      Kind: kind,
      TargetID: targetId,
      Endpoint: "",
      Local: true,
      RecentOnly: false,
    };
    const existing = nets.find((n) => n.ChainID === chainId);
    if (existing) existing.Upstreams.push(upstream);
    else nets.push({ ChainID: chainId, Upstreams: [upstream] });
    cfg.Networks = nets;
    await saveConfig(gid, cfg, "Adding the endpoint");
  }

  // openDiscoverModal is internal/chainlist put in front of the operator: the
  // canonical chain feed, minus the ${API_KEY} provider slots, with every
  // remaining URL probed for eth_chainId. What is offered is what answered.
  async function openDiscoverModal(gid: string, chainId: number): Promise<void> {
    openModal(
      `
        <h2>Public endpoints for chain ${chainId}</h2>
        <p class="muted small">
          Reading chainid.network and asking each endpoint for its chain id. Only the ones that
          answer — with the right chain — are offered, because a feed lists plenty that no longer work.
        </p>
        <p><span class="spinner" aria-label="working"></span> probing…</p>
        <div class="modal-actions"><button class="btn btn-ghost" data-modal-action="cancel">Cancel</button></div>
      `,
      (action) => {
        if (action === "cancel") closeModal();
      },
    );

    let result: api.ChainlistResult;
    try {
      result = await api.discoverEndpoints(chainId);
    } catch (err) {
      const panel = modalBody();
      if (panel) {
        const p = document.createElement("p");
        p.className = "error small";
        p.textContent = `Could not discover endpoints: ${message(err)}`;
        panel.appendChild(p);
      }
      return;
    }
    if (disposed) return;

    const live = (result.endpoints ?? []).filter((e) => e.status === "live" || e.status === "unprobed");
    const rejected = (result.endpoints ?? []).filter((e) => e.status === "rejected");

    openModal(
      `
        <h2>Public endpoints for chain ${chainId}</h2>
        ${
          result.source === "vendored"
            ? `<div class="banner banner-warn">chainid.network was unreachable, so this is the list valve-node-app ships with.
                 ${result.fetchError ? `<div class="small">${escapeHtml(result.fetchError)}</div>` : ""}</div>`
            : ""
        }
        ${
          live.length
            ? `<p class="muted small">${live.length} answered for this chain. Pick one to add it as a fallback upstream.</p>
               <ul class="plain-list rpc-picker">
                 ${live
                   .map(
                     (e) => `
                   <li>
                     <button class="btn btn-ghost rpc-picker-option" data-modal-action="add:${encodeURIComponent(e.url)}">
                       <span><code>${escapeHtml(e.url)}</code></span>
                       <span class="muted small">${e.status === "live" ? `answered in ${e.latencyMs ?? 0} ms` : "not probed (WebSocket)"}</span>
                     </button>
                   </li>`,
                   )
                   .join("")}
               </ul>`
            : `<p class="error small">Nothing in the feed answered for chain ${chainId} right now.</p>`
        }
        ${
          rejected.length
            ? `<details class="rpc-rejected">
                 <summary class="muted small">${rejected.length} were not offered — why</summary>
                 <ul class="plain-list">
                   ${rejected
                     .map((e) => `<li class="muted small"><code>${escapeHtml(e.url)}</code> — ${escapeHtml(e.reason ?? "rejected")}</li>`)
                     .join("")}
                 </ul>
               </details>`
            : ""
        }
        <div class="modal-actions"><button class="btn btn-ghost" data-modal-action="cancel">Close</button></div>
      `,
      (action) => {
        if (action === "cancel") {
          closeModal();
          return;
        }
        if (action.startsWith("add:")) {
          closeModal();
          void addExternalUpstream(gid, chainId, decodeURIComponent(action.slice("add:".length)));
        }
      },
    );
  }

  function openManualEndpointModal(gid: string, chainId: number): void {
    openModal(
      `
        <h2>Add an endpoint by URL</h2>
        <p class="muted small">
          http://, https://, ws:// or wss://. eRPC infers WebSocket from the scheme — there is no
          separate setting — and a ws upstream also serves ordinary calls.
        </p>
        <label>
          Endpoint
          <input type="text" id="manual-endpoint" autocomplete="off" spellcheck="false" placeholder="https://rpc.example.com" />
        </label>
        <label class="radio">
          <input type="checkbox" id="manual-recent" />
          Recent blocks only <span class="muted">— tick for a pruned node that cannot answer historical state</span>
        </label>
        <p class="muted small" id="manual-err"></p>
        <div class="modal-actions">
          <button class="btn btn-ghost" data-modal-action="cancel">Cancel</button>
          <button class="btn" data-modal-action="add">Add endpoint</button>
        </div>
      `,
      (action) => {
        if (action === "cancel") {
          closeModal();
          return;
        }
        if (action !== "add") return;
        const input = document.getElementById("manual-endpoint") as HTMLInputElement | null;
        const recent = document.getElementById("manual-recent") as HTMLInputElement | null;
        const errEl = document.getElementById("manual-err");
        const url = input?.value.trim() ?? "";
        if (!/^(https?|wss?):\/\//i.test(url)) {
          if (errEl) {
            errEl.className = "error small";
            errEl.textContent = "It needs a scheme eRPC can dial: http://, https://, ws:// or wss://.";
          }
          return;
        }
        closeModal();
        void addExternalUpstream(gid, chainId, url, recent?.checked ?? false);
      },
    );
    document.getElementById("manual-endpoint")?.focus();
  }

  async function addExternalUpstream(
    gid: string,
    chainId: number,
    url: string,
    recentOnly = false,
  ): Promise<void> {
    const gw = gatewayOf(gid);
    if (!gw) return;
    const cfg = storedConfig(gw);
    const nets = cfg.Networks ?? [];
    const existing = nets.find((n) => n.ChainID === chainId);
    const count = (existing?.Upstreams.length ?? 0) + 1;
    const upstream: api.GatewayUpstream = {
      ID: `public-${chainId}-${count}`,
      Kind: "external",
      Endpoint: url,
      Local: false,
      RecentOnly: recentOnly,
    };
    if (existing) existing.Upstreams.push(upstream);
    else nets.push({ ChainID: chainId, Upstreams: [upstream] });
    cfg.Networks = nets;
    await saveConfig(gid, cfg, "Adding the endpoint");
  }

  // --- devnet reset -------------------------------------------------------

  // resetDevnet is the one destructive action here that is deliberately NOT
  // gated behind a typed confirmation. A devnet is a scratch chain and
  // throwing it away is routine; friction would be a bug, not safety. It goes
  // through the wipe machinery, not a restart, because that is what restarts
  // the gateways in front of it — eRPC's per-chain head only moves forward,
  // so a reset chain is invisible to it and it keeps advertising a head the
  // chain no longer has.
  async function resetDevnet(key: string, targetId: string): Promise<void> {
    const parsed = parseKey(key);
    if (!parsed || !targetId) return;
    const ok = await confirmModal({
      title: "Reset this devnet",
      body: `The chain on ${targetId} starts again from genesis at block 0 — every block, transaction and account it has produced is discarded. The gateways in front of it are restarted so they stop advertising the old head.`,
      confirmLabel: "Reset the chain",
    });
    if (!ok) return;

    busy[parsed.gid] = "reset";
    actionErr[parsed.gid] = null;
    render();

    let result: api.WipeResult;
    try {
      result = await api.resetDevnet(targetId);
    } catch (err) {
      actionErr[parsed.gid] = `Reset failed: ${message(err)}${hintOf(err)}`;
      busy[parsed.gid] = null;
      render();
      return;
    }
    busy[parsed.gid] = null;
    showResetResult(targetId, result);
    await load();
  }

  // showResetResult surfaces the cascade. A restart nobody can see is
  // indistinguishable from one that did not happen, and the cascade exists
  // precisely because its absence is silent rather than loud.
  function showResetResult(targetId: string, r: api.WipeResult): void {
    const lines: string[] = [];
    lines.push(r.report.ContainerRemoved ? "The old chain was removed." : "There was no devnet container to remove.");
    if (r.report.Recreated) lines.push("A fresh chain was started from genesis.");
    const cascaded = r.report.Cascaded ?? [];
    const skipped = r.report.CascadeSkipped ?? [];

    openModal(
      `
        <h2>Devnet on ${escapeHtml(targetId)} reset</h2>
        <ul class="plain-list">${lines.map((l) => `<li>${escapeHtml(l)}</li>`).join("")}</ul>
        ${
          cascaded.length
            ? `<p class="ok">Restarted in front of it: ${escapeHtml(cascaded.join(", "))} — the cached head was cleared, so each now reports this chain's real height rather than the one from before the reset.</p>`
            : `<p class="muted small">Nothing needed restarting in front of it.</p>`
        }
        ${
          skipped.length
            ? `<p class="muted small">Not restarted (they were not running, so they held no stale head): ${escapeHtml(skipped.join(", "))}.</p>`
            : ""
        }
        ${
          r.error
            ? `<p class="error">The reset itself succeeded, but something in front of it could not be restarted — it is now serving a head this chain no longer has. Restart it by hand.</p>
               <p class="error small">${escapeHtml(r.error)}</p>`
            : ""
        }
        <div class="modal-actions"><button class="btn" data-modal-action="close">Close</button></div>
      `,
      () => closeModal(),
    );
  }

  // --- wipe ---------------------------------------------------------------

  function openWipeModal(gid: string): void {
    const gw = gatewayOf(gid);
    if (!gw) return;
    openModal(
      `
        <h2>Wipe ${escapeHtml(gw.label)}</h2>
        <p class="error">This destroys ${escapeHtml(gw.wipeDiscards)}</p>
        <p>Every chain it fronts stops being served until it comes back. Nothing behind it — no node, no devnet, no public endpoint — is touched.</p>
        <p>Type <code>${escapeHtml(gid)}</code> to confirm.</p>
        <input type="text" id="wipe-confirm-input" autocomplete="off" spellcheck="false" />
        <div class="modal-actions">
          <button class="btn btn-ghost" data-modal-action="cancel">Cancel</button>
          <button class="btn btn-danger" data-modal-action="confirm" id="wipe-confirm-btn" disabled>Wipe ${escapeHtml(gid)}</button>
        </div>
      `,
      (action) => {
        if (action === "cancel" || action === "close") {
          closeModal();
          void load();
          return;
        }
        if (action === "confirm") void runWipe(gid);
      },
    );
    const input = document.getElementById("wipe-confirm-input") as HTMLInputElement | null;
    const btn = document.getElementById("wipe-confirm-btn") as HTMLButtonElement | null;
    input?.addEventListener("input", () => {
      if (btn) btn.disabled = input.value.trim() !== gid;
    });
    input?.focus();
  }

  async function runWipe(gid: string): Promise<void> {
    const btn = document.getElementById("wipe-confirm-btn") as HTMLButtonElement | null;
    if (btn) {
      btn.disabled = true;
      btn.textContent = "Wiping…";
    }
    let result: api.WipeResult;
    try {
      result = await api.wipeGateway(gid);
    } catch (err) {
      const panel = modalBody();
      if (panel) {
        const p = document.createElement("p");
        p.className = "error small";
        p.textContent = `Wipe failed: ${message(err)}${hintOf(err)}`;
        panel.appendChild(p);
      }
      if (btn) {
        btn.disabled = false;
        btn.textContent = `Wipe ${gid}`;
      }
      return;
    }
    openModal(
      `
        <h2>${escapeHtml(gid)} wiped</h2>
        <ul class="plain-list">
          <li>${result.report.ContainerRemoved ? "Container removed." : "There was no container to remove."}</li>
          ${result.report.Recreated ? "<li>Container re-created from your saved configuration.</li>" : ""}
        </ul>
        ${result.error ? `<p class="error small">${escapeHtml(result.error)}</p>` : ""}
        <div class="modal-actions"><button class="btn" data-modal-action="close">Close</button></div>
      `,
      () => {
        closeModal();
        void load();
      },
    );
  }

  // --- add a gateway ------------------------------------------------------

  function openAddGatewayModal(): void {
    const targets = data?.targets ?? [];
    const existing = new Set((data?.gateways ?? []).map((g) => g.id));
    if (targets.length === 0) {
      openModal(
        `
          <h2>No machines yet</h2>
          <p>A gateway is a container, so it has to run somewhere. Add a machine first.</p>
          <div class="modal-actions">
            <button class="btn btn-ghost" data-modal-action="cancel">Cancel</button>
            <a class="btn" href="#/targets" data-modal-action="cancel">Go to Machines</a>
          </div>
        `,
        () => closeModal(),
      );
      return;
    }
    const suggested = existing.has("default") ? "" : "default";

    openModal(
      `
        <h2>Add a gateway</h2>
        <p class="muted small">
          A gateway NAMES the machine it runs on; it does not belong to it. Its endpoints can be
          anywhere — this machine's devnet, a node on another box, a public endpoint.
        </p>
        <label>
          Name <span class="muted">— becomes its container name, so lower-case letters, digits, dot, dash or underscore</span>
          <input type="text" id="new-gw-id" autocomplete="off" spellcheck="false" value="${escapeHtml(suggested)}" placeholder="edge" />
        </label>
        <label>
          Runs on
          <select id="new-gw-target">
            ${targets.map((t) => `<option value="${escapeHtml(t.id)}">${escapeHtml(t.id)} (${escapeHtml(t.mode)})</option>`).join("")}
          </select>
        </label>
        <label>
          Listen port
          <input type="text" inputmode="numeric" id="new-gw-port" value="4000" autocomplete="off" />
        </label>
        <p class="muted small" id="new-gw-err"></p>
        <div class="modal-actions">
          <button class="btn btn-ghost" data-modal-action="cancel">Cancel</button>
          <button class="btn" data-modal-action="create">Create gateway</button>
        </div>
      `,
      (action) => {
        if (action === "cancel") {
          closeModal();
          return;
        }
        if (action !== "create") return;
        void createGateway();
      },
    );
    document.getElementById("new-gw-id")?.focus();
  }

  async function createGateway(): Promise<void> {
    const idEl = document.getElementById("new-gw-id") as HTMLInputElement | null;
    const targetEl = document.getElementById("new-gw-target") as HTMLSelectElement | null;
    const portEl = document.getElementById("new-gw-port") as HTMLInputElement | null;
    const errEl = document.getElementById("new-gw-err");
    const id = idEl?.value.trim() ?? "";
    const targetId = targetEl?.value ?? "";
    const port = Number.parseInt(portEl?.value.trim() ?? "", 10);
    const showErr = (msg: string): void => {
      if (errEl) {
        errEl.className = "error small";
        errEl.textContent = msg;
      }
    };
    if (!id) {
      showErr("Give it a name — it becomes the container's name, which is how it is found again.");
      return;
    }
    if (!targetId) {
      showErr("Pick the machine it runs on.");
      return;
    }
    try {
      await api.createGateway({
        id,
        placement: { targetId, backend: "docker" },
        config: {
          ProjectID: "main",
          BindAddr: "127.0.0.1",
          Port: Number.isFinite(port) ? port : 4000,
          Networks: [],
        },
      });
    } catch (err) {
      showErr(message(err));
      return;
    }
    closeModal();
    await load();
  }

  // --- misc ---------------------------------------------------------------

  async function copyButton(el: HTMLElement, value: string): Promise<void> {
    const ok = await copyToClipboard(value);
    const original = el.textContent;
    el.textContent = ok ? "Copied!" : "Copy failed";
    setTimeout(() => {
      if (!disposed) el.textContent = original;
    }, 1500);
  }

  function message(err: unknown): string {
    return err instanceof Error ? err.message : String(err);
  }

  // hintOf surfaces the server's operator-facing hint verbatim — it is
  // written for exactly this moment ("start Docker Desktop / OrbStack /
  // colima"), and paraphrasing it would only make it vaguer.
  function hintOf(err: unknown): string {
    return err instanceof api.ApiError && err.hint ? ` — ${err.hint}` : "";
  }

  return () => {
    disposed = true;
    streamStop?.();
    closeModal();
  };
}
