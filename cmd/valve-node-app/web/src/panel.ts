// The Easy-Button panel: a single card (list → network → endpoint) that
// replaces the old capability-detected home as the default landing. Task 6
// fills in the list view — the master power button plus one row per chain —
// and wires the full gateway lifecycle (create/start/stop/restart/recreate/
// wipe) so the panel can drive a gateway through every state. Network/
// endpoint detail content lands in later tasks (see panelModel.ts for the
// pure helpers this file builds on).
import "./panel.css";
import * as api from "./api";
import { onAction, escapeHtml, confirmModal, copyToClipboard } from "./ui";
import { masterState, healthClass, capabilityCells, withoutNetwork, type MasterState, type CapCell } from "./panelModel";
import { SETUP_CHAINS, internalTLSConfig } from "./home";

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
  // setupLog accumulates the one-click setup's progress lines (registering
  // the machine, the Docker gate, creating the gateway, each setup-stream
  // event) — cleared at the start of every attempt so a retry doesn't show
  // a previous failed run's tail above the new one.
  let setupLog: string[] = [];
  // Network-detail-only state: capabilities are probed lazily (opening the
  // gateway's real sockets), cached per gateway id, and reused across
  // re-renders until "recheck" or a different gateway asks for them.
  let netCaps: api.GatewayCapabilities | null = null;
  let netCapsGid: string | null = null;
  let netCapsBusy = false;
  // tlsVerify is the LAST live tls/verify result run from this screen — it
  // starts from gw.tls.verification (the server's own last check) and is
  // replaced once the operator clicks the lock.
  let tlsVerify: api.TlsVerification | null = null;
  let tlsBusy = false;
  let tlsErr: string | null = null;
  let copyFlash = false;
  // networkErr surfaces a remove-network failure on the network screen itself
  // — actionErr is the list view's lifecycle-action error and stays reserved
  // for that.
  let networkErr: string | null = null;

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
    if (view.name === "network") {
      return renderNetwork(gw, view.chainId, {
        caps: netCaps,
        capsBusy: netCapsBusy,
        tls: tlsVerify,
        tlsBusy,
        tlsErr,
        copyFlash,
        error: networkErr,
      });
    }
    if (view.name === "endpoint") return renderEndpoint(gw, view.chainId, view.upstreamId);
    return renderList(gw, busy, actionErr, setupLog);
  }

  // loadCaps probes (or reads the cached probe of) a gateway's capabilities —
  // real sockets against real endpoints, so it is never fired on a poll, only
  // on entering the network screen and on an explicit "recheck".
  async function loadCaps(gid: string, refresh: boolean): Promise<void> {
    netCapsBusy = true;
    render();
    try {
      netCaps = await api.getGatewayCapabilities(gid, refresh);
      netCapsGid = gid;
    } catch {
      netCaps = null;
      netCapsGid = gid;
    }
    netCapsBusy = false;
    render();
  }

  // removeNetworkFlow confirms (a network's URL and every endpoint under it
  // stop being served) before writing the config without this chain and
  // re-provisioning — the same write-then-provision cycle every other config
  // change on this panel follows, so a removed chain actually disappears from
  // the running container rather than just the stored config.
  async function removeNetworkFlow(g: api.GatewayView, chainId: number): Promise<void> {
    const nv = g.networks?.find((n) => n.chainId === chainId);
    const ok = await confirmModal({
      title: "Remove network",
      body: `Stop serving ${nv?.name ?? `chain ${chainId}`}?`,
      confirmLabel: "Remove",
      danger: true,
    });
    if (!ok) return;
    networkErr = null;
    render();
    try {
      await api.putGatewayConfig(g.id, withoutNetwork(g.config, chainId));
    } catch (e) {
      networkErr = `Could not remove the network: ${message(e)}`;
      render();
      return;
    }
    // The config write succeeded — the chain is gone from what's stored, so
    // there is nothing left to show on this screen even before the
    // re-provision finishes. Hand off to provision(), which owns busy/render
    // and reloads once the setup stream completes (mirrors gw-recreate).
    view = { name: "list" };
    render();
    await provision(g.id);
  }

  onAction(card, (action, el) => {
    void handleAction(action, el);
  });
  async function handleAction(action: string, el: HTMLElement): Promise<void> {
    if (action === "setup") {
      if (busy) return;
      await runSetup();
      return;
    }
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
      const chainId = Number(el.dataset.chainId);
      view = { name: "network", chainId };
      networkErr = null;
      tlsVerify = null;
      tlsErr = null;
      render();
      if (gw && netCapsGid !== gw.id) void loadCaps(gw.id, false);
      return;
    }
    if (action === "back") {
      view = { name: "list" };
      render();
      return;
    }
    if (action === "back-to-network") {
      const chainId = Number(el.dataset.chainId);
      view = Number.isFinite(chainId) ? { name: "network", chainId } : { name: "list" };
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
      case "copy-url": {
        const url = el.dataset.url ?? "";
        if (!url) return;
        const ok = await copyToClipboard(url);
        if (ok) {
          copyFlash = true;
          render();
          window.setTimeout(() => {
            copyFlash = false;
            render();
          }, 1200);
        }
        return;
      }
      case "verify-tls": {
        if (!gw || tlsBusy) return;
        tlsBusy = true;
        tlsErr = null;
        render();
        try {
          tlsVerify = await api.verifyGatewayTls(gw.id);
        } catch (e) {
          tlsErr = message(e);
        }
        tlsBusy = false;
        render();
        return;
      }
      case "open-endpoint": {
        const chainId = Number(el.dataset.chainId);
        const upstreamId = el.dataset.upstreamId ?? "";
        if (!Number.isFinite(chainId) || !upstreamId) return;
        view = { name: "endpoint", chainId, upstreamId };
        render();
        return;
      }
      case "add-endpoint":
        // No-op stub this task — the real add-an-endpoint flow lands in Task 10.
        return;
      case "remove-network": {
        if (!gw || busy || view.name !== "network") return;
        await removeNetworkFlow(gw, view.chainId);
        return;
      }
      case "recheck": {
        if (!gw) return;
        await Promise.all([loadCaps(gw.id, true), load()]);
        return;
      }
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
      if (ev.err) actionErr = `Provisioning failed: ${ev.err}`;
      void load();
    });
  }

  // runSetup is the empty state's one-click: it stands up a whole gateway
  // from nothing, mirroring home.ts's setupEndpoint step for step (register
  // this machine, gate on Docker, create the gateway, add valve's known set
  // for Ethereum + PulseChain, provision, follow the setup stream) so the
  // panel's own path never drifts from the eRPC screen's. It shares
  // SETUP_CHAINS and internalTLSConfig with home.ts rather than a second copy.
  async function runSetup(): Promise<void> {
    if (busy) return;
    busy = "setup";
    actionErr = null;
    setupLog = [];
    render();

    const say = (line: string): void => {
      setupLog = [...setupLog, line];
      render();
    };
    const fail = (msg: string, h?: string): void => {
      busy = null;
      actionErr = h ? `${msg} — ${h}` : msg;
      render();
    };

    say("Preparing your endpoint…");

    // 1. Register this machine as a local target if it is not already one.
    try {
      const targets = await api.listTargets();
      if (!targets.some((t) => t.id === "local")) {
        await api.addTarget({ id: "local", mode: "local" });
      }
    } catch (e) {
      fail(`Could not register this machine: ${message(e)}`, hint(e));
      return;
    }

    // 2. A gateway is a container, so this needs a reachable Docker engine —
    //    checked up front, clearly, with the engine's own hint, rather than
    //    failing later mid-provision.
    try {
      const c = await api.getContainers("local");
      if (!c.docker.reachable) {
        fail(
          c.docker.detail || "A gateway runs as a container, and no Docker engine answered on this machine.",
          c.docker.hint || "Start Docker Desktop, OrbStack or colima, then try again.",
        );
        return;
      }
    } catch (e) {
      fail(`Could not check Docker on this machine: ${message(e)}`, hint(e));
      return;
    }

    // 3. Create the gateway, fronted by the internal-CA HTTPS front.
    say("Creating the gateway…");
    let gid = "default";
    try {
      const created = await api.createGateway({
        id: gid,
        placement: { targetId: "local", backend: "docker" },
        config: internalTLSConfig([]),
      });
      gid = created.id;
    } catch (e) {
      fail(`Could not create the gateway: ${message(e)}`, hint(e));
      return;
    }

    // 4. Add valve's known set for Ethereum + PulseChain — the same vetted,
    //    measured set the RPC screen's "Add valve's set…" adds. Devnet is
    //    deliberately excluded: this builds a real public endpoint, not a
    //    scratch chain.
    say("Adding Ethereum and PulseChain endpoints…");
    const networks: api.GatewayNetwork[] = [];
    for (const { chainId } of SETUP_CHAINS) {
      try {
        const set = await api.knownSet(gid, chainId);
        const urls = (set.endpoints ?? []).filter((e) => !e.alreadyAdded).map((e) => e.url);
        if (urls.length === 0) continue;
        networks.push({
          ChainID: chainId,
          Upstreams: urls.map((url, i) => ({
            ID: `public-${chainId}-${i + 1}`,
            Kind: "external",
            Endpoint: url,
            Local: false,
            RecentOnly: false,
          })),
        });
      } catch (e) {
        fail(`Could not read valve's set for chain ${chainId}: ${message(e)}`, hint(e));
        return;
      }
    }
    if (networks.length === 0) {
      fail("valve has no measured endpoints for Ethereum or PulseChain right now, so there was nothing to add.");
      return;
    }
    try {
      await api.putGatewayConfig(gid, internalTLSConfig(networks));
    } catch (e) {
      fail(`Could not save the endpoints: ${message(e)}`, hint(e));
      return;
    }

    // 5. Provision the container and follow the placement machine's setup
    //    stream to completion, exactly as provision() above does for an
    //    existing gateway's create/recreate.
    say("Starting the gateway… the first run pulls the eRPC and Caddy images.");
    let started: { targetId: string };
    try {
      started = await api.provisionGateway(gid);
    } catch (e) {
      fail(`Could not start the gateway: ${message(e)}`, hint(e));
      return;
    }

    streamStop?.();
    streamStop = api.streamSetup(started.targetId, (ev) => {
      const line = ev.err ? `${ev.stepId}: ${ev.err}` : ev.line ? `${ev.stepId}: ${ev.line}` : `${ev.stepId}: done`;
      say(line);
      const finished = !!ev.err || (ev.stepId === FINAL_STEP && !!ev.done);
      if (!finished) return;
      streamStop?.();
      streamStop = null;
      busy = null;
      // Unconditional, mirroring provision() below: by the time this stream
      // fires, createGateway + putGatewayConfig have already succeeded, so a
      // "default" gateway record exists server-side even on an error here —
      // returning early without reloading would leave gw null and the panel
      // stuck showing the empty "Set up my endpoint" hero over a gateway that
      // now actually exists, so a retry would 400 with "already exists"
      // instead of showing the real failure. actionErr survives the reload
      // (load() never touches it), so the reason still shows once the
      // now-non-null gateway state renders.
      if (ev.err) actionErr = `Provisioning failed: ${ev.err}`;
      setupLog = [];
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
// hint surfaces the server's operator-facing hint verbatim (e.g. "start
// Docker Desktop / OrbStack / colima") — the same helper home.ts's
// setupEndpoint uses, kept local here since it's two lines and both files
// exporting it from a shared spot would be more indirection than it's worth.
function hint(e: unknown): string | undefined {
  return e instanceof api.ApiError ? e.hint : undefined;
}
function bandError(m: string): string {
  return `<div class="p-band" style="padding:16px;color:var(--red)">${escapeHtml(m)}</div>`;
}

// --- list view -------------------------------------------------------------

function renderList(
  gw: api.GatewayView | null,
  busy: string | null,
  actionErr: string | null,
  setupLog: string[],
): string {
  if (gw === null) return renderEmpty(busy, actionErr, setupLog);
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

// renderEmpty is the zero-gateway state: no gateway exists yet on this
// fleet, so instead of a networks list the card shows one centered hero —
// the same power button, dimmed, as the one-click "set up my endpoint"
// action. setupLog/actionErr surface the one-click flow's progress and any
// failure right on this card; nothing here navigates away.
function renderEmpty(busy: string | null, actionErr: string | null, setupLog: string[]): string {
  const running = busy === "setup";
  const errLine = actionErr ? `<div class="p-emptyerr">${escapeHtml(actionErr)}</div>` : "";
  const log = setupLog.length
    ? `<div class="p-setup-log" aria-live="polite">${setupLog.map((l) => `<div>${escapeHtml(l)}</div>`).join("")}</div>`
    : "";
  return `
    <div class="p-band p-phead">
      <span class="p-brand"><span class="p-bd"></span> Valve</span>
    </div>
    <div class="p-band p-empty">
      <button type="button" class="p-emptybtn" data-action="setup"${running ? " disabled" : ""}>
        <div class="p-pbtn off big${running ? " busy" : ""}">${ic("power")}</div>
      </button>
      <div class="p-emptytitle">Set up my endpoint</div>
      <div class="p-emptysub">
        One click gets you a managed RPC endpoint for Ethereum and PulseChain — no node required.
      </div>
      ${errLine}
      ${log}
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
  // hintLine surfaces the server's own operator-facing reason (e.g. "start
  // Docker Desktop / OrbStack / colima") under the blocked sub-line — the
  // detail that turns "blocked" into something actionable.
  const hintLine = m.tone === "blocked" && gw?.hint ? `<div class="p-ps">${escapeHtml(gw.hint)}</div>` : "";
  const power = `
    <div class="p-power${busyClass}" data-action="power">
      <div class="p-pbtn ${m.tone}">${ic("power")}</div>
      <div class="p-pmeta">
        <div class="p-pl">${escapeHtml(m.label)}</div>
        <div class="p-ps"${m.tone === "blocked" ? ' style="color:var(--red)"' : ""}>${escapeHtml(subText)}</div>
        ${hintLine}
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

// networkRow: 18px lead health dot, name, capability meter, chevron.
// Clicking drills in. The meter here stays unlit — the real probe (real
// sockets against real endpoints) is deliberately never run for a whole
// gateway's worth of chains just to paint the list; it fires lazily, once,
// when a network's own detail screen opens (see renderNetwork/loadCaps).
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

// --- network detail ----------------------------------------------------------

// NetworkDetailState carries everything renderNetwork needs that isn't
// pure — probed capabilities, the last live TLS check, in-flight/error
// flags — all owned by renderPanel; renderNetwork stays a plain
// string-returning function, same shape as renderList.
interface NetworkDetailState {
  caps: api.GatewayCapabilities | null;
  capsBusy: boolean;
  tls: api.TlsVerification | null;
  tlsBusy: boolean;
  tlsErr: string | null;
  copyFlash: boolean;
  error: string | null;
}

// capStatusOf mirrors rpc.ts's statusOf (~L1195-1202): there is no "http"
// probe result because answering JSON-RPC over HTTP IS reachability,
// recorded as EndpointCapabilities.reachable rather than an eleventh method
// call.
function capStatusOf(e: api.EndpointCapabilities, key: string): api.CapabilityStatus | undefined {
  if (key === "http") {
    if (e.unprobeable) return "inconclusive";
    return e.reachable ? "supported" : "unsupported";
  }
  return (e.capabilities ?? []).find((c) => c.key === key)?.status;
}

// unionCapabilities folds every probed upstream on this chain into one
// verdict per capability: supported if ANY upstream on the chain supports
// it. The network-level question is "can I get this AT ALL through this
// gateway's one URL for this chain", which a balanced/failover front answers
// with the best of its upstreams, not the worst.
function unionCapabilities(
  caps: api.GatewayCapabilities | null,
  chainId: number,
  upstreamIds: string[],
): Record<string, string> {
  const endpoints = (caps?.endpoints ?? []).filter((e) => e.chainId === chainId && upstreamIds.includes(e.upstream));
  const out: Record<string, string> = {};
  for (const key of ["http", "ws", "archive", "trace"]) {
    if (endpoints.some((e) => capStatusOf(e, key) === "supported")) out[key] = "supported";
  }
  return out;
}

function renderNetwork(gw: api.GatewayView | null, chainId: number, st: NetworkDetailState): string {
  const nv = gw?.networks?.find((n) => n.chainId === chainId);
  if (!gw || !nv) {
    return `
      <div class="p-band p-dhead">
        <span class="p-back" data-action="back">${ic("chevL")}</span>
        <span class="p-dtitle"><span class="p-nmtxt">Chain ${chainId}</span></span>
      </div>
      <div class="p-band" style="padding:16px;color:var(--dim)">This network is no longer configured.</div>
    `;
  }

  const running = gw.status.State === "running";
  const hc = healthClass({ running, serviceable: nv.serviceable });
  const ups = nv.upstreams ?? [];

  // Gateway band: the one dialable URL, a lock reflecting the last real
  // tls/verify (gw.tls.verification is the server's own last check; a click
  // here replaces it with a fresh one), and a copy icon that flashes green
  // once the clipboard write lands.
  const tlsResult = st.tls ?? gw.tls.verification ?? null;
  const tlsOk = tlsResult?.ok === true;
  const lockTitle = st.tlsBusy
    ? "Verifying…"
    : tlsOk
      ? `Verified ${tlsResult ? new Date(tlsResult.at).toLocaleString() : ""}`
      : "Verify HTTPS now";
  const tlsErrLine = st.tlsErr
    ? `<div class="p-ps" style="color:var(--red);padding:0 var(--gut) 10px">${escapeHtml(st.tlsErr)}</div>`
    : "";
  const gatewayBand = `
    <div class="p-band">
      <div class="p-lblrow">
        <span class="p-seclbl">Gateway <span style="color:var(--dim3);letter-spacing:0"> · balanced across all</span></span>
        <span class="p-acts">
          <span class="p-ic ${tlsOk ? "green" : "dim"}" data-action="verify-tls" title="${escapeHtml(lockTitle)}">${ic("lock")}</span>
          <span class="p-ic ${st.copyFlash ? "green" : "accent"}" data-action="copy-url" data-url="${escapeHtml(nv.url ?? "")}" title="Copy the gateway URL">${ic("copy")}</span>
        </span>
      </div>
      <div class="p-gwurl">${escapeHtml(nv.url || "—")}</div>
      ${tlsErrLine}
    </div>
  `;

  // Endpoints band: one row per configured upstream, health dot first, plus
  // the (stub) "Add endpoint" row Task 10 wires up for real.
  const upRows = ups
    .map((u, i) => {
      const uhc = healthClass({ running, serviceable: !u.problem });
      return `
        <div class="p-row${i > 0 ? " p-rowdiv" : ""}" data-action="open-endpoint" data-chain-id="${nv.chainId}" data-upstream-id="${escapeHtml(u.id)}">
          <span class="p-lead"><span class="p-dot ${uhc}"></span></span>
          <span class="p-nm">${escapeHtml(u.label)}</span>
          <span class="p-chev">${ic("chevR")}</span>
        </div>
      `;
    })
    .join("");
  const endpointsBand = `
    <div class="p-band">
      <div class="p-lblrow"><span class="p-seclbl">Endpoints · ${ups.length}</span></div>
      ${upRows}
      <div class="p-row${ups.length > 0 ? " p-rowdiv" : ""} addr" data-action="add-endpoint">
        <span class="p-lead">${ic("plus")}</span>
        <span class="p-nm">Add endpoint</span>
      </div>
    </div>
  `;

  // Capabilities band: folded from the probe (union across this chain's
  // upstreams — see unionCapabilities). "probing…" shows only while the
  // first lazy fetch for this gateway is in flight; on "recheck" the cells
  // keep showing the previous verdict until the refresh lands.
  const capStatuses = unionCapabilities(st.caps, chainId, ups.map((u) => u.id));
  const capCells = capabilityCells(capStatuses);
  const capsBand = `
    <div class="p-band">
      <div class="p-lblrow"><span class="p-seclbl">Capabilities</span></div>
      ${
        st.capsBusy && !st.caps
          ? `<div class="p-caprow" style="color:var(--dim2)">probing…</div>`
          : `<div class="p-caprow">${capCells
              .map((c) => `<span class="p-capitem${c.lit ? " lit" : ""}">${ic(CAP_ICON[c.key])}${escapeHtml(c.label)}</span>`)
              .join("")}</div>`
      }
    </div>
  `;

  // Status band: Health mirrors the list row's own dot+word. Chain head is
  // omitted rather than faked — there is no per-network head reading yet.
  const healthWord = !running ? "Stopped" : nv.serviceable ? "Healthy" : "Unserviceable";
  const statusBand = `
    <div class="p-band">
      <div class="p-lblrow"><span class="p-seclbl">Status</span><span class="p-acts"><span class="p-ic dim" data-action="recheck" title="Re-check capabilities and reload">${ic("refresh")}</span></span></div>
      <div class="p-srow"><span class="p-k">Health</span><span class="p-v"><span class="p-dot ${hc}"></span> ${escapeHtml(healthWord)}</span></div>
    </div>
  `;

  const removeErrLine = st.error
    ? `<div class="p-band" style="padding:10px 16px;color:var(--red)">${escapeHtml(st.error)}</div>`
    : "";

  return `
    <div class="p-band p-dhead">
      <span class="p-back" data-action="back">${ic("chevL")}</span>
      <span class="p-dtitle"><span class="p-dot ${hc}"></span> <span class="p-nmtxt">${escapeHtml(nv.name)}</span></span>
    </div>
    ${gatewayBand}
    ${endpointsBand}
    ${capsBand}
    ${statusBand}
    ${removeErrLine}
    <div class="p-band p-remove" data-action="remove-network">${ic("trash")} Remove network</div>
  `;
}

// --- endpoint detail -----------------------------------------------------
// Minimal placeholder — the real content (address editor, capabilities,
// status, remove) arrives in Task 10. It still gives the operator a working
// way back rather than a dead end.
function renderEndpoint(gw: api.GatewayView | null, chainId: number, upstreamId: string): string {
  const nv = gw?.networks?.find((n) => n.chainId === chainId);
  const up = nv?.upstreams?.find((u) => u.id === upstreamId);
  return `
    <div class="p-band p-dhead">
      <span class="p-back" data-action="back-to-network" data-chain-id="${chainId}">${ic("chevL")}</span>
      <span class="p-dtitle"><span class="p-nmtxt">${escapeHtml(up?.label ?? "Endpoint")}</span></span>
    </div>
    <div class="p-band" style="padding:16px;color:var(--dim)">Endpoint detail is coming soon.</div>
  `;
}
