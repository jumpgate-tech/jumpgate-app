// #/rpc — eRPC as a LAYER over the whole fleet.
//
// The screen is three tiers, and the geometry is the argument:
//
//   TIER 1  a FULL-WIDTH BAR per gateway. It spans the width because it
//           fronts everything below it: state, the URL callers dial, the
//           lifecycle actions — and, inside the bar, the networks it serves
//           as selectable chips. "eRPC fronts these N networks" is one line.
//   TIER 2  those chips. Clicking one focuses the area below on it; `+` adds
//           a network. A chain with no working endpoint is styled apart,
//           because it is a network eRPC will accept and then fail to serve,
//           and that must not look identical to a healthy one.
//   TIER 3  UNDER the bar, the endpoints of the ONE selected chain: what each
//           one is (this machine's devnet, a node on box-a, a public
//           endpoint), whether it can be used, and what may be done to it.
//
// Chips rather than a dropdown, up to CHIP_LIMIT: a gateway typically fronts
// two to five chains, and a row of chips shows the whole answer at a glance
// where a dropdown would hide all but one. Past that threshold the row stops
// being legible and it collapses into the existing dropdown control — same
// selection model, one visible at a time.
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
  dropdown,
  escapeHtml,
  footer,
  modalBody,
  onAction,
  openModal,
  wireDropdowns,
} from "./ui";

// CHIP_LIMIT is where a chip row stops being readable and becomes a
// dropdown. Both forms drive the same single-selection state.
const CHIP_LIMIT = 6;

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

  // The focused chain per gateway — tier 2's selection, which is what tier 3
  // renders. Held here rather than in the DOM so any re-render keeps it.
  const focus: Record<string, number | null> = {};
  const busy: Record<string, string | null> = {};
  const actionErr: Record<string, string | null> = {};
  const activity: Record<string, string[]> = {};
  // The bar's own settings (port, bind) are edited in place, opened per
  // gateway. Kept out of the DOM for the same reason as focus.
  const settingsOpen: Record<string, boolean> = {};
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
  wireDropdowns(root, (id, value) => {
    if (id.startsWith("chain-")) {
      const gid = id.slice("chain-".length);
      focus[gid] = Number.parseInt(value, 10);
      render();
    }
  });

  void load();

  // --- data ---------------------------------------------------------------

  async function load(): Promise<void> {
    try {
      const next = await api.getGateways();
      if (disposed) return;
      data = next;
      loadErr = null;
      // Focus the first chain of each gateway that has none yet, and drop a
      // focus whose chain has since been removed.
      for (const gw of next.gateways ?? []) {
        const nets = gw.networks ?? [];
        const current = focus[gw.id];
        if (current == null || !nets.some((n) => n.chainId === current)) {
          focus[gw.id] = nets.length ? nets[0].chainId : null;
        }
      }
    } catch (err) {
      if (disposed) return;
      data = null;
      loadErr = message(err);
    }
    render();
  }

  function gatewayOf(gid: string): api.GatewayView | undefined {
    return (data?.gateways ?? []).find((g) => g.id === gid);
  }

  function networkOf(gw: api.GatewayView, chainId: number | null): api.NetworkView | undefined {
    if (chainId == null) return undefined;
    return (gw.networks ?? []).find((n) => n.chainId === chainId);
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

  // gatewayBlock is one gateway: the full-width bar (tier 1 + 2) and, under
  // it, the focused chain's endpoints (tier 3).
  function gatewayBlock(gw: api.GatewayView): string {
    const focused = networkOf(gw, focus[gw.id] ?? null);
    return `
      <section class="rpc-gateway">
        ${gatewayBar(gw)}
        ${gw.error ? errorBlock(gw) : ""}
        ${gw.blocked ? `<div class="banner banner-warn">${escapeHtml(gw.blocked)}</div>` : ""}
        ${(gw.warnings ?? []).map((wmsg) => `<div class="banner banner-warn">${escapeHtml(wmsg)}</div>`).join("")}
        ${actionErr[gw.id] ? `<p class="error small">${escapeHtml(actionErr[gw.id]!)}</p>` : ""}
        ${activityBlock(gw)}
        ${settingsOpen[gw.id] ? settingsBlock(gw) : ""}
        ${upstreamsPanel(gw, focused)}
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
        ${chainRow(gw)}
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

  // chainRow renders the networks as selectable chips, or — once there are
  // more than CHIP_LIMIT — as a dropdown, because a chip row past that width
  // stops being scannable and starts being a wall.
  function chainRow(gw: api.GatewayView): string {
    const nets = gw.networks ?? [];
    const selected = focus[gw.id] ?? null;

    const add = `
      <button class="chip chip-add" data-action="add-chain" data-gid="${escapeHtml(gw.id)}"
              title="Add a network for this gateway to front">+ Network</button>
    `;

    if (nets.length === 0) {
      return `
        <div class="rpc-chiprow">
          <span class="muted small">No networks yet — eRPC refuses a configuration with none, so add one before creating the gateway.</span>
          ${add}
        </div>
      `;
    }

    if (nets.length > CHIP_LIMIT) {
      const options = nets.map((n) => ({
        value: String(n.chainId),
        // The dropdown carries the same "this one is broken" signal the chips
        // do; losing it on the collapse would make the warning depend on how
        // many chains you happen to have.
        label: `${n.name} (${n.chainId})${n.serviceable ? "" : " — no working endpoint"}`,
      }));
      return `
        <div class="rpc-chiprow">
          <span class="muted small">Fronting ${nets.length} networks</span>
          ${dropdown(`chain-${gw.id}`, options, selected == null ? null : String(selected))}
          ${add}
        </div>
      `;
    }

    return `
      <div class="rpc-chiprow">
        ${nets.map((n) => chip(gw, n, n.chainId === selected)).join("")}
        ${add}
      </div>
    `;
  }

  function chip(gw: api.GatewayView, n: api.NetworkView, selected: boolean): string {
    // A chain with no working endpoint is a network eRPC will accept and then
    // fail every call on. It gets its own state on the chip rather than
    // looking exactly like a healthy one.
    const broken = !n.serviceable;
    return `
      <button class="chip card-selectable${selected ? " selected" : ""}${broken ? " chip-bad" : ""}"
              data-action="select-chain" data-gid="${escapeHtml(gw.id)}" data-chain="${n.chainId}"
              title="${escapeHtml(broken ? `${n.name}: no endpoint on this chain can be used right now` : `${n.name} · ${n.path}`)}">
        <span class="chip-dot">${broken ? dot("bad") : dot("ok")}</span>
        <span class="chip-name">${escapeHtml(n.name)}</span>
        <span class="chip-id">${n.chainId}</span>
      </button>
    `;
  }

  // ---- TIER 3: the focused chain's endpoints -----------------------------

  function upstreamsPanel(gw: api.GatewayView, n: api.NetworkView | undefined): string {
    if (!n) {
      return `<div class="card rpc-upstreams"><p class="muted small">Pick a network above to see the servers behind it.</p></div>`;
    }
    const ups = n.upstreams ?? [];
    return `
      <div class="card rpc-upstreams">
        <div class="service-head">
          <h2>${escapeHtml(n.name)} <span class="muted">· chain ${n.chainId}</span></h2>
          <div class="card-actions">
            <button class="btn" data-action="add-endpoint" data-gid="${escapeHtml(gw.id)}" data-chain="${n.chainId}">Add an endpoint</button>
            <button class="btn btn-ghost" data-action="remove-chain" data-gid="${escapeHtml(gw.id)}" data-chain="${n.chainId}">Remove network</button>
          </div>
        </div>
        ${
          n.url
            ? `<div class="endpoint-row">${dot("ok")}<span class="muted small">callers dial</span>
                 <code class="endpoint-url">${escapeHtml(n.url)}</code>
                 <button class="btn btn-ghost" data-action="copy" data-copy="${escapeHtml(n.url)}">Copy</button></div>`
            : `<p class="muted small">Path <code>${escapeHtml(n.path)}</code> — the full URL appears once the gateway is running.</p>`
        }
        ${(n.warnings ?? []).map((wmsg) => `<div class="banner banner-warn">${escapeHtml(wmsg)}</div>`).join("")}
        ${ups.map((u) => upstreamRow(gw, n, u)).join("")}
        ${ups.length === 0 ? `<p class="muted small">No endpoint yet, so there is nowhere for calls on this path to go.</p>` : ""}
      </div>
    `;
  }

  // upstreamRow states what the endpoint IS before what it is called, because
  // "the devnet on this machine" is the fact an operator reasons about and
  // the URL is the consequence.
  function upstreamRow(gw: api.GatewayView, n: api.NetworkView, u: api.UpstreamView): string {
    const key = `${gw.id}|${n.chainId}|${u.id}`;
    const actions = u.actions ?? [];
    return `
      <div class="upstream-row${u.problem ? " upstream-row-bad" : ""}">
        <span class="upstream-state">${u.problem ? dot("bad") : dot("ok")}</span>
        <div class="upstream-what">
          <div class="upstream-label">
            ${escapeHtml(u.label)}
            ${u.local ? badge("preferred", "ok") : badge("fallback", "neutral")}
            ${u.recentOnly ? badge("recent blocks only", "warn") : ""}
          </div>
          <code class="endpoint-url">${escapeHtml(u.endpoint || "—")}</code>
          ${u.problem ? `<div class="error small">${escapeHtml(u.problem)}</div>` : ""}
        </div>
        <div class="card-actions">
          ${
            actions.includes("reset")
              ? `<button class="btn" data-action="reset-devnet" data-key="${escapeHtml(key)}" data-target="${escapeHtml(u.targetId ?? "")}"
                         title="Throw this devnet's chain away and start again from genesis. It is a scratch chain — this is routine."
                         ${busy[gw.id] ? "disabled" : ""}>
                   ${busy[gw.id] === "reset" ? `<span class="spinner" aria-label="working"></span>` : "Reset"}
                 </button>`
              : ""
          }
          <button class="btn btn-ghost" data-action="remove-endpoint" data-key="${escapeHtml(key)}">Remove</button>
        </div>
      </div>
    `;
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
        <div class="card-actions">
          <button class="btn" data-action="save-settings" data-gid="${escapeHtml(gw.id)}">Save settings</button>
        </div>
      </div>
    `;
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
      case "select-chain":
        focus[gid] = Number.parseInt(el.dataset.chain ?? "", 10);
        render();
        return;
      case "toggle-settings":
        settingsOpen[gid] = !settingsOpen[gid];
        render();
        return;
      case "save-settings":
        await saveSettings(gid);
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
    focus[gid] = chainId;
    // A chain with no upstream cannot be rendered, so it is saved by adding
    // it and letting the server report the network as unserviceable — which
    // it does, on the chip, before anything is provisioned.
    if (await saveConfigTolerantly(gid, cfg)) {
      focus[gid] = chainId;
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
    focus[gid] = chainId;
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
    focus[gid] = null;
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
