// The Easy-Button panel: a single card (list → network → endpoint) that
// replaces the old capability-detected home as the default landing. Task 6
// fills in the list view — the master power button plus one row per chain —
// and wires the full gateway lifecycle (create/start/stop/restart/recreate/
// wipe) so the panel can drive a gateway through every state. Network/
// endpoint detail content lands in later tasks (see panelModel.ts for the
// pure helpers this file builds on).
import "./panel.css";
import * as api from "./api";
import { onAction, escapeHtml, confirmModal } from "./ui";
import { masterState, healthClass, capabilityCells, type MasterState, type CapCell } from "./panelModel";

// Inline SVG sprite (currentColor stroke) — cross-platform, no SF Symbols.
const SPRITE = `<svg width="0" height="0" style="position:absolute" aria-hidden="true"><defs>
  <symbol id="p-power" viewBox="0 0 24 24"><line x1="12" y1="3.5" x2="12" y2="11.5"/><path d="M7.5 7a7 7 0 1 0 9 0"/></symbol>
  <symbol id="p-globe" viewBox="0 0 24 24"><circle cx="12" cy="12" r="8.5"/><path d="M3.5 12h17M12 3.5c2.5 2.4 2.5 14.6 0 17M12 3.5c-2.5 2.4-2.5 14.6 0 17"/></symbol>
  <symbol id="p-ws" viewBox="0 0 24 24"><path d="M4 9h13l-3.5-3.5M20 15H7l3.5 3.5"/></symbol>
  <symbol id="p-archive" viewBox="0 0 24 24"><path d="M12 3 3 7.5l9 4.5 9-4.5L12 3ZM3 12l9 4.5 9-4.5M3 16.5 12 21l9-4.5"/></symbol>
  <symbol id="p-trace" viewBox="0 0 24 24"><circle cx="12" cy="12" r="5.5"/><path d="M12 1.5v3M12 19.5v3M1.5 12h3M19.5 12h3"/></symbol>
  <symbol id="p-lock" viewBox="0 0 24 24"><rect x="5" y="11" width="14" height="9.5" rx="2.2"/><path d="M8 11V8a4 4 0 0 1 8 0v3"/></symbol>
  <symbol id="p-pencil" viewBox="0 0 24 24"><path d="M14 5.5l4.5 4.5M4 20l1.2-4.4L16 4.8a2 2 0 0 1 2.8 0l.4.4a2 2 0 0 1 0 2.8L8.4 18.8 4 20Z"/></symbol>
  <symbol id="p-trash" viewBox="0 0 24 24"><path d="M4 6.5h16M9.5 6.5V5a2 2 0 0 1 2-2h1a2 2 0 0 1 2 2v1.5M6.5 6.5l1 13.5h9l1-13.5M10 10.5v6M14 10.5v6"/></symbol>
  <symbol id="p-copy" viewBox="0 0 24 24"><rect x="9" y="9" width="11" height="11" rx="2.2"/><path d="M5 15V5a2 2 0 0 1 2-2h8"/></symbol>
  <symbol id="p-scale" viewBox="0 0 24 24"><path d="M12 3v18M7 21h10M12 5 5 8m7-3 7 3M5 8l-3 6a3 3 0 0 0 6 0L5 8Zm14 0-3 6a3 3 0 0 0 6 0l-3-6Z"/></symbol>
  <symbol id="p-refresh" viewBox="0 0 24 24"><path d="M19.5 12a7.5 7.5 0 1 1-2.2-5.3M19.5 4.5v4h-4"/></symbol>
  <symbol id="p-chevR" viewBox="0 0 24 24"><path d="M9.5 5.5l6.5 6.5-6.5 6.5"/></symbol>
  <symbol id="p-chevL" viewBox="0 0 24 24"><path d="M14.5 5.5 8 12l6.5 6.5"/></symbol>
  <symbol id="p-plus" viewBox="0 0 24 24"><path d="M12 5v14M5 12h14"/></symbol>
</defs></svg>`;
const ic = (id: string) => `<svg class="p-i"><use href="#p-${id}"/></svg>`;

type View = { name: "list" } | { name: "network"; chainId: number } | { name: "endpoint"; chainId: number; upstreamId: string };

// FINAL_STEP is the id every gateway setup plan ends on (mirrors rpc.ts).
const FINAL_STEP = "run";

export function renderPanel(root: HTMLElement): () => void {
  let gw: api.GatewayView | null = null;
  let view: View = { name: "list" };
  let err: string | null = null;
  let poll: number | null = null;
  // busy is the lifecycle action currently in flight (e.g. "stop", "create",
  // "wipe") — set while the panel manages ONE gateway, so one flag suffices.
  let busy: string | null = null;
  let actionErr: string | null = null;
  let streamStop: (() => void) | null = null;

  root.innerHTML = SPRITE + `<div class="p-wrap"><div class="p-panel" id="p-card"></div></div>`;
  const card = root.querySelector<HTMLElement>("#p-card")!;

  async function load(): Promise<void> {
    try {
      const res = await api.getGateways();
      gw = primaryGateway(res.gateways);
      err = null;
    } catch (e) {
      err = message(e);
    }
    render();
  }
  function render(): void {
    card.innerHTML = renderView();
  }
  function renderView(): string {
    if (err) return bandError(err);
    if (view.name === "network") return renderNetwork(gw, view.chainId);
    if (view.name === "endpoint") return renderEndpoint(gw, view.chainId, view.upstreamId);
    return renderList(gw, busy, actionErr);
  }

  onAction(card, (action, el) => {
    void handleAction(action, el);
  });
  async function handleAction(action: string, el: HTMLElement): Promise<void> {
    if (action === "power") {
      if (!gw || busy) return;
      const m = masterState(gw);
      if (m.tone === "blocked") return;
      if (gw.status.State === "running" && m.actions.includes("stop")) { await runAction(gw.id, "stop"); return; }
      if (m.actions.includes("start")) { await runAction(gw.id, "start"); return; }
      if (m.actions.includes("create")) { await provision(gw.id); return; }
      return;
    }
    if (action === "open-network") {
      view = { name: "network", chainId: Number(el.dataset.chainId) };
      render();
      return;
    }
    if (action === "back-to-list") {
      view = { name: "list" };
      render();
      return;
    }
    if (action === "add-network") {
      // No-op stub this task — the real add-a-chain flow lands in Task 10.
      return;
    }
    switch (action) {
      case "gw-start":
      case "gw-stop":
      case "gw-restart":
        if (gw && !busy) await runAction(gw.id, action.slice("gw-".length) as api.ContainerActionKind);
        return;
      case "gw-create":
      case "gw-recreate":
        if (gw && !busy) await provision(gw.id);
        return;
      case "gw-wipe":
        if (gw && !busy) await runWipe(gw);
        return;
      default:
        return;
    }
  }

  // runAction mirrors rpc.ts's runAction: start/stop/restart go straight to
  // the container action endpoint — no config write, no stream to follow.
  async function runAction(gid: string, kind: api.ContainerActionKind): Promise<void> {
    if (busy) return;
    busy = kind;
    actionErr = null;
    render();
    try {
      await api.gatewayAction(gid, kind);
    } catch (e) {
      actionErr = `${kind} failed: ${message(e)}`;
    }
    busy = null;
    await load();
  }

  // provision mirrors rpc.ts's provision: create/recreate run the gateway's
  // setup plan and follow the PLACEMENT machine's setup event stream — the
  // same per-machine stream the node wizard and the devnet use.
  async function provision(gid: string): Promise<void> {
    if (busy) return;
    busy = "create";
    actionErr = null;
    render();

    let started: { targetId: string };
    try {
      started = await api.provisionGateway(gid);
    } catch (e) {
      actionErr = message(e);
      busy = null;
      render();
      return;
    }

    streamStop?.();
    streamStop = api.streamSetup(started.targetId, (ev) => {
      const finished = !!ev.err || (ev.stepId === FINAL_STEP && !!ev.done);
      if (!finished) return;
      streamStop?.();
      streamStop = null;
      busy = null;
      if (ev.err) actionErr = "Provisioning failed — see the machine's setup log.";
      void load();
    });
  }

  // runWipe confirms (wipeDiscards names exactly what is destroyed) before
  // calling the destructive endpoint, then reloads to show the wiped state.
  async function runWipe(g: api.GatewayView): Promise<void> {
    const ok = await confirmModal({
      title: `Wipe ${g.label}`,
      body: `This destroys ${g.wipeDiscards}. Every chain it fronts stops being served until it comes back. Nothing behind it — no node, no devnet, no public endpoint — is touched.`,
      confirmLabel: "Wipe",
      danger: true,
    });
    if (!ok) return;
    busy = "wipe";
    actionErr = null;
    render();
    try {
      // wipeGateway resolves normally even on a PARTIAL failure (the wipe
      // itself happened; a cascade in front of it — e.g. re-creating this
      // gateway or restarting something that fronts it — did not), so its
      // report.error has to be surfaced rather than dropped.
      const result = await api.wipeGateway(g.id);
      if (result.error) actionErr = result.error;
    } catch (e) {
      actionErr = `wipe failed: ${message(e)}`;
    }
    busy = null;
    await load();
  }

  void load();
  return () => {
    if (poll) window.clearInterval(poll);
    streamStop?.();
  };
}

// primaryGateway: the panel manages one gateway — the local one, else the first.
function primaryGateway(gws: api.GatewayView[] | null): api.GatewayView | null {
  if (!gws || gws.length === 0) return null;
  return gws.find((g) => g.placement.targetId === "local") ?? gws[0];
}
function message(e: unknown): string {
  return e instanceof Error ? e.message : String(e);
}
function bandError(m: string): string {
  return `<div class="p-band" style="padding:16px;color:var(--red)">${escapeHtml(m)}</div>`;
}

// --- list view -------------------------------------------------------------

function renderList(gw: api.GatewayView | null, busy: string | null, actionErr: string | null): string {
  const m = masterState(gw);
  const rows = gw?.networks?.length
    ? gw.networks.map((nv, i) => networkRow(gw, nv, i > 0)).join("")
    : "";
  return `
    <div class="p-band p-phead">
      <span class="p-brand"><span class="p-bd"></span> Valve</span>
      <span class="p-sum">${escapeHtml(m.sub)}</span>
    </div>
    <div class="p-band">
      ${powerBand(gw, m, busy, actionErr)}
    </div>
    <div class="p-band">
      <div class="p-lblrow"><span class="p-seclbl">Networks</span></div>
      ${rows}
      <div class="p-row p-rowdiv addr" data-action="add-network">
        <span class="p-lead">${ic("plus")}</span>
        <span class="p-nm">Add a network</span>
      </div>
    </div>
  `;
}

// primaryAction picks the transition the big round button performs — the
// same rule masterState's callers apply: stop when running, else start, else
// create. It only ever returns an action the server actually listed.
function primaryAction(gw: api.GatewayView, m: MasterState): string | null {
  if (m.tone === "blocked") return null;
  if (gw.status.State === "running" && m.actions.includes("stop")) return "stop";
  if (m.actions.includes("start")) return "start";
  if (m.actions.includes("create")) return "create";
  return null;
}

const ACTION_LABEL: Record<string, string> = {
  start: "Start", stop: "Stop", restart: "Restart", create: "Create", recreate: "Recreate", wipe: "Wipe",
};
const ACTION_ICON: Record<string, string> = { restart: "refresh", recreate: "refresh", wipe: "trash" };

// powerBand renders the round power button (the primary transition) plus a
// row of small secondary chips for every OTHER action the server listed —
// this is what lets the panel cycle through every state (wipe, stop, start,
// restart, recreate) rather than offering only one button.
function powerBand(gw: api.GatewayView | null, m: MasterState, busy: string | null, actionErr: string | null): string {
  const subText = m.tone === "blocked" ? (m.blocked ?? "") : m.sub;
  const busyClass = busy ? " busy" : "";
  const errLine = actionErr ? `<div class="p-ps" style="color:var(--red)">${escapeHtml(actionErr)}</div>` : "";
  const power = `
    <div class="p-power${busyClass}" data-action="power">
      <div class="p-pbtn ${m.tone}">${ic("power")}</div>
      <div class="p-pmeta">
        <div class="p-pl">${escapeHtml(m.label)}</div>
        <div class="p-ps"${m.tone === "blocked" ? ' style="color:var(--red)"' : ""}>${escapeHtml(subText)}</div>
        ${errLine}
      </div>
    </div>
  `;
  const chips = gw ? chipsHtml(gw, m, busy) : "";
  return power + chips;
}

// chipsHtml renders every action gw.actions carries MINUS the one the power
// button already performs, so the operator can always reach the remaining
// transitions (e.g. recreate/wipe while running, or restart while stopped)
// without the panel ever offering an action the server didn't list.
function chipsHtml(gw: api.GatewayView, m: MasterState, busy: string | null): string {
  const primary = primaryAction(gw, m);
  const remaining = (gw.actions ?? []).filter((a) => a !== primary);
  if (remaining.length === 0) return "";
  const items = remaining
    .map((a) => {
      const label = ACTION_LABEL[a] ?? a;
      const icon = ACTION_ICON[a] ? ic(ACTION_ICON[a]) : "";
      const danger = a === "wipe" ? " danger" : "";
      return `<button type="button" class="p-chip${danger}" data-action="gw-${a}" data-gid="${escapeHtml(gw.id)}"${busy ? " disabled" : ""}>${icon}${escapeHtml(label)}</button>`;
    })
    .join("");
  return `<div class="p-chips">${items}</div>`;
}

const CAP_ICON: Record<string, string> = { http: "globe", ws: "ws", archive: "archive", trace: "trace" };

function capsHtml(cells: CapCell[]): string {
  return cells
    .map((c) => `<svg class="p-i${c.hot ? " hot" : c.lit ? " on" : ""}"><use href="#p-${CAP_ICON[c.key]}"/></svg>`)
    .join("");
}

// networkRow: 18px lead health dot, name, capability meter (all-unlit until
// Task 9 wires real probed capabilities), chevron. Clicking drills in.
function networkRow(gw: api.GatewayView, nv: api.NetworkView, divider: boolean): string {
  const hc = healthClass({ running: gw.status.State === "running", serviceable: nv.serviceable });
  const cells = capabilityCells({});
  return `
    <div class="p-row${divider ? " p-rowdiv" : ""}" data-action="open-network" data-chain-id="${nv.chainId}">
      <span class="p-lead"><span class="p-dot ${hc}"></span></span>
      <span class="p-nm">${escapeHtml(nv.name)}</span>
      <span class="p-caps">${capsHtml(cells)}</span>
      <span class="p-chev">${ic("chevR")}</span>
    </div>
  `;
}

// --- network / endpoint detail ---------------------------------------------
// Minimal placeholders — the real content (endpoints list, gateway URL,
// capabilities, status, remove) arrives in Task 9.

function renderNetwork(gw: api.GatewayView | null, chainId: number): string {
  const nv = gw?.networks?.find((n) => n.chainId === chainId);
  return `
    <div class="p-band p-dhead">
      <span class="p-back" data-action="back-to-list">${ic("chevL")}</span>
      <span class="p-dtitle"><span class="p-nmtxt">${escapeHtml(nv?.name ?? `Chain ${chainId}`)}</span></span>
    </div>
    <div class="p-band" style="padding:16px;color:var(--dim)">Network detail is coming soon.</div>
  `;
}
function renderEndpoint(_gw: api.GatewayView | null, _c: number, _u: string): string {
  return "";
}
