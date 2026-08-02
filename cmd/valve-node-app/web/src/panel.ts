// The Easy-Button panel: a single card (list → network → endpoint) that
// replaces the old capability-detected home as the default landing. Task 6
// fills in the list view — the master power button plus one row per chain —
// and wires the full gateway lifecycle (create/start/stop/restart/recreate/
// wipe) so the panel can drive a gateway through every state. Network/
// endpoint detail content lands in later tasks (see panelModel.ts for the
// pure helpers this file builds on).
import "./panel.css";
import * as api from "./api";
import { onAction, escapeHtml, confirmModal, copyToClipboard, openModal, closeModal, modalBody, fmtInt } from "./ui";
import {
  masterState,
  healthClass,
  capabilityCells,
  withoutNetwork,
  withNetwork,
  withUpstream,
  withoutUpstream,
  endpointNameFromUrl,
  networkSlowRate,
  endpointSlowRate,
  type MasterState,
  type CapCell,
} from "./panelModel";
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

// DEVNET_CHAIN_ID is reth's --dev genesis id (catalog.DevnetChainID), mirrors
// rpc.ts's own copy of the same literal. catalog.Networks() (what
// api.getCatalog() returns) deliberately excludes it — it is a private
// scratch chain, not a supported public network — so the add-network picker
// appends it itself, always last.
const DEVNET_CHAIN_ID = 1337;

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
  // endpointErr is networkErr's sibling for the endpoint screen (remove-
  // endpoint failures land here, not on networkErr or actionErr).
  let endpointErr: string | null = null;
  // epHealth is the gateway's analytics scrape, fetched lazily (like
  // netCaps) only once an endpoint's own detail screen needs a per-upstream
  // headLag reading — the list and network screens never need it, so it is
  // never fetched on a poll or on entering the network screen.
  let epHealth: api.GatewayAnalytics | null = null;
  let epHealthGid: string | null = null;
  let epHealthBusy = false;
  // netHealth is the LIVE health scrape driving every dot's animation — the
  // one epHealth deliberately isn't (see epHealth's own comment): fetched on
  // a 5s poll regardless of which screen is open, so a list row or a
  // network's endpoint rows keep twitching (or stop) without the operator
  // having to drill in. netHealthBusy guards the poll against overlap — a
  // slow scrape must not stack a second request behind it.
  let netHealth: api.GatewayAnalytics | null = null;
  let netHealthBusy = false;
  // lastHealthSig is the health-class fingerprint of the last render (see
  // healthSignature) — refreshHealth only calls render() when this changes,
  // so a poll that finds nothing different doesn't tear down and rebuild the
  // DOM (which would restart every dot's CSS animation from frame zero,
  // stutter-stepping a steady twitch every 5s) or blow away in-flight UI
  // state like focus or a copy-flash.
  let lastHealthSig = "";

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
        netHealth,
      });
    }
    if (view.name === "endpoint") {
      return renderEndpoint(gw, view.chainId, view.upstreamId, {
        caps: netCaps,
        capsBusy: netCapsBusy,
        health: epHealth,
        healthBusy: epHealthBusy,
        copyFlash,
        error: endpointErr,
        netHealth,
      });
    }
    return renderList(gw, busy, actionErr, setupLog, netHealth);
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

  // loadHealth reads the gateway's analytics scrape (the source of
  // EndpointHealth.headLag) — see epHealth above for why this is lazy and
  // cached rather than fetched with every gateway poll.
  async function loadHealth(gid: string, refresh: boolean): Promise<void> {
    if (!refresh && epHealthGid === gid && epHealth) return;
    epHealthBusy = true;
    render();
    try {
      epHealth = await api.getGatewayAnalytics(gid);
      epHealthGid = gid;
    } catch {
      epHealth = null;
      epHealthGid = gid;
    }
    epHealthBusy = false;
    render();
  }

  // healthSignature fingerprints every dot the panel currently knows how to
  // draw (every network row, every upstream row, whichever gateway is
  // loaded) as one string of "id:class" pairs. refreshHealth diffs this
  // before and after a scrape so it can skip render() when nothing about the
  // ANIMATION actually changed — see netHealth/lastHealthSig above for why
  // that matters.
  function healthSignature(): string {
    if (!gw) return "";
    const running = gw.status.State === "running";
    const parts: string[] = [];
    for (const nv of gw.networks ?? []) {
      const na = netHealth?.networks?.find((n) => n.chainId === nv.chainId);
      const rate = na ? networkSlowRate(na) : undefined;
      parts.push(`n${nv.chainId}:${healthClass({ running, serviceable: nv.serviceable, slowRate: rate })}`);
      for (const u of nv.upstreams ?? []) {
        const upRate = na ? endpointSlowRate(na, u.id) : undefined;
        parts.push(`u${nv.chainId}/${u.id}:${healthClass({ running, serviceable: !u.problem, slowRate: upRate })}`);
      }
    }
    return parts.join("|");
  }

  // refreshHealth is the panel's live-dot heartbeat: fetch the gateway's
  // analytics scrape, then re-render ONLY if that changed at least one dot's
  // computed class (see healthSignature). netHealthBusy guards against a
  // slow scrape overlapping the next poll tick.
  async function refreshHealth(): Promise<void> {
    if (!gw || netHealthBusy) return;
    netHealthBusy = true;
    try {
      netHealth = await api.getGatewayAnalytics(gw.id);
    } catch {
      // A failed scrape leaves netHealth as whatever it last was — stale-but-
      // real beats every dot flashing to "no data" on one transient error.
    }
    netHealthBusy = false;
    const sig = healthSignature();
    if (sig !== lastHealthSig) {
      lastHealthSig = sig;
      render();
    }
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

  // removeEndpointFlow is removeNetworkFlow's sibling one level down: confirm,
  // write the config without this one upstream, re-provision, then land back
  // on the network the endpoint used to belong to (not the list — the
  // operator was drilled in one level less than removeNetworkFlow's caller).
  async function removeEndpointFlow(g: api.GatewayView, chainId: number, upstreamId: string): Promise<void> {
    const nv = g.networks?.find((n) => n.chainId === chainId);
    const up = findUpstream(g, chainId, upstreamId);
    const ok = await confirmModal({
      title: "Remove endpoint",
      body: `Stop routing to ${up?.label ?? "this endpoint"}? The gateway keeps balancing across whatever else remains on ${nv?.name ?? `chain ${chainId}`}.`,
      confirmLabel: "Remove",
      danger: true,
    });
    if (!ok) return;
    endpointErr = null;
    render();
    try {
      await api.putGatewayConfig(g.id, withoutUpstream(g.config, chainId, upstreamId));
    } catch (e) {
      endpointErr = `Could not remove the endpoint: ${message(e)}`;
      render();
      return;
    }
    view = { name: "network", chainId };
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
      endpointErr = null;
      render();
      return;
    }
    if (action === "add-network") {
      if (!gw || busy) return;
      await openAddNetworkModal(gw);
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
        endpointErr = null;
        render();
        if (gw && netCapsGid !== gw.id) void loadCaps(gw.id, false);
        if (gw && epHealthGid !== gw.id) void loadHealth(gw.id, false);
        return;
      }
      case "add-endpoint": {
        if (!gw || busy || view.name !== "network") return;
        openAddEndpointModal(gw, view.chainId);
        return;
      }
      case "remove-network": {
        if (!gw || busy || view.name !== "network") return;
        await removeNetworkFlow(gw, view.chainId);
        return;
      }
      case "rename-endpoint": {
        if (!gw || busy || view.name !== "endpoint") return;
        const up = findUpstream(gw, view.chainId, view.upstreamId);
        if (!up) return;
        openRenameEndpointModal(gw.id, view.chainId, up.id, up.label);
        return;
      }
      case "edit-address": {
        if (!gw || busy || view.name !== "endpoint") return;
        const up = findUpstream(gw, view.chainId, view.upstreamId);
        // Managed upstreams (a node, a devnet) derive their endpoint on every
        // read — whatever is stored for one is ignored server-side (see
        // GatewayUpstream.Endpoint) — so offering an edit here would be a
        // control that silently does nothing. Only "external" upstreams
        // actually take the value.
        if (!up || up.kind !== "external") return;
        openEditAddressModal(gw.id, view.chainId, up.id, up.endpoint);
        return;
      }
      case "remove-endpoint": {
        if (!gw || busy || view.name !== "endpoint") return;
        await removeEndpointFlow(gw, view.chainId, view.upstreamId);
        return;
      }
      case "recheck": {
        if (!gw) return;
        const tasks: Promise<unknown>[] = [loadCaps(gw.id, true), load(), refreshHealth()];
        if (view.name === "endpoint") tasks.push(loadHealth(gw.id, true));
        await Promise.all(tasks);
        return;
      }
      default:
        return;
    }
  }

  // findUpstream reads the UpstreamView (the resolved, displayable form) for
  // one upstream — used by the endpoint-detail actions to read what to show
  // in a modal (up.label, up.endpoint, up.kind).
  function findUpstream(g: api.GatewayView, chainId: number, upstreamId: string): api.UpstreamView | undefined {
    return g.networks?.find((n) => n.chainId === chainId)?.upstreams?.find((u) => u.id === upstreamId);
  }

  // findConfigUpstream reads the STORED GatewayUpstream for one upstream —
  // the object every write in this section mutates (via withUpstream), never
  // the UpstreamView, whose fields like `endpoint` are server-resolved and
  // would silently discard a managed upstream's TargetID if round-tripped.
  function findConfigUpstream(g: api.GatewayView, chainId: number, upstreamId: string): api.GatewayUpstream | undefined {
    return g.config.Networks?.find((n) => n.ChainID === chainId)?.Upstreams.find((u) => u.ID === upstreamId);
  }

  // appendModalError adds an error line to the currently open modal without
  // closing it — the same pattern rpc.ts's runWipe uses — so a failed save
  // stays next to the input the operator can fix, rather than vanishing into
  // a screen-level error the modal already closed on.
  function appendModalError(msg: string): void {
    const panel = modalBody();
    if (!panel) return;
    const p = document.createElement("p");
    p.className = "error small";
    p.textContent = msg;
    panel.appendChild(p);
  }

  // openRenameEndpointModal prefills the operator-chosen label (up.label —
  // the RESOLVED display name, per the brief, not the raw stored Name, which
  // may be unset). Saving writes Name on the stored upstream and re-
  // provisions; clearing the field back to blank drops Name so the resolver's
  // own default takes back over.
  function openRenameEndpointModal(gid: string, chainId: number, upstreamId: string, currentLabel: string): void {
    openModal(
      `
        <h2>Rename endpoint</h2>
        <label>
          Name
          <input type="text" id="ep-rename-input" autocomplete="off" spellcheck="false" value="${escapeHtml(currentLabel)}" />
        </label>
        <p class="muted small">Clear it to fall back to the automatic name.</p>
        <div class="modal-actions">
          <button class="btn btn-ghost" data-modal-action="cancel">Cancel</button>
          <button class="btn" data-modal-action="save" id="ep-rename-save">Save</button>
        </div>
      `,
      (action) => {
        if (action === "cancel") {
          closeModal();
          return;
        }
        if (action !== "save") return;
        void submitRename(gid, chainId, upstreamId);
      },
    );
    const input = document.getElementById("ep-rename-input") as HTMLInputElement | null;
    input?.focus();
    input?.select();
  }

  async function submitRename(gid: string, chainId: number, upstreamId: string): Promise<void> {
    if (!gw) return;
    const cfgUp = findConfigUpstream(gw, chainId, upstreamId);
    if (!cfgUp) {
      closeModal();
      return;
    }
    const input = document.getElementById("ep-rename-input") as HTMLInputElement | null;
    const btn = document.getElementById("ep-rename-save") as HTMLButtonElement | null;
    const name = input?.value.trim() ?? "";
    if (input) input.disabled = true;
    if (btn) {
      btn.disabled = true;
      btn.textContent = "Saving…";
    }
    const nextUp: api.GatewayUpstream = { ...cfgUp, Name: name || undefined };
    try {
      await api.putGatewayConfig(gid, withUpstream(gw.config, chainId, nextUp));
    } catch (e) {
      appendModalError(`Could not rename the endpoint: ${message(e)}`);
      if (input) input.disabled = false;
      if (btn) {
        btn.disabled = false;
        btn.textContent = "Save";
      }
      return;
    }
    closeModal();
    await provision(gid);
  }

  // openEditAddressModal is rename's sibling for the endpoint's own URL —
  // only ever opened for an "external" upstream (see the edit-address case
  // in handleAction for why managed ones never get here).
  function openEditAddressModal(gid: string, chainId: number, upstreamId: string, currentUrl: string): void {
    openModal(
      `
        <h2>Edit endpoint address</h2>
        <p class="muted small">http://, https://, ws:// or wss://.</p>
        <label>
          URL
          <input type="text" id="ep-addr-input" autocomplete="off" spellcheck="false" value="${escapeHtml(currentUrl)}" placeholder="https://rpc.example.com" />
        </label>
        <div class="modal-actions">
          <button class="btn btn-ghost" data-modal-action="cancel">Cancel</button>
          <button class="btn" data-modal-action="save" id="ep-addr-save">Save</button>
        </div>
      `,
      (action) => {
        if (action === "cancel") {
          closeModal();
          return;
        }
        if (action !== "save") return;
        void submitAddress(gid, chainId, upstreamId);
      },
    );
    const input = document.getElementById("ep-addr-input") as HTMLInputElement | null;
    input?.focus();
    input?.select();
  }

  async function submitAddress(gid: string, chainId: number, upstreamId: string): Promise<void> {
    if (!gw) return;
    const input = document.getElementById("ep-addr-input") as HTMLInputElement | null;
    const btn = document.getElementById("ep-addr-save") as HTMLButtonElement | null;
    const url = input?.value.trim() ?? "";
    if (!/^(https?|wss?):\/\//i.test(url)) {
      appendModalError("It needs a scheme eRPC can dial: http://, https://, ws:// or wss://.");
      return;
    }
    const cfgUp = findConfigUpstream(gw, chainId, upstreamId);
    if (!cfgUp) {
      closeModal();
      return;
    }
    if (input) input.disabled = true;
    if (btn) {
      btn.disabled = true;
      btn.textContent = "Saving…";
    }
    const nextUp: api.GatewayUpstream = { ...cfgUp, Endpoint: url };
    try {
      await api.putGatewayConfig(gid, withUpstream(gw.config, chainId, nextUp));
    } catch (e) {
      appendModalError(`Could not save the address: ${message(e)}`);
      if (input) input.disabled = false;
      if (btn) {
        btn.disabled = false;
        btn.textContent = "Save";
      }
      return;
    }
    closeModal();
    await provision(gid);
  }

  // openAddEndpointModal is the network screen's "Add endpoint" row — a
  // single URL, added as an "external" upstream named from its own domain
  // (endpointNameFromUrl), same default a fresh public endpoint gets
  // anywhere else in this app.
  function openAddEndpointModal(g: api.GatewayView, chainId: number): void {
    openModal(
      `
        <h2>Add an endpoint by URL</h2>
        <p class="muted small">http://, https://, ws:// or wss://.</p>
        <label>
          Endpoint
          <input type="text" id="ep-add-input" autocomplete="off" spellcheck="false" placeholder="https://rpc.example.com" />
        </label>
        <div class="modal-actions">
          <button class="btn btn-ghost" data-modal-action="cancel">Cancel</button>
          <button class="btn" data-modal-action="add" id="ep-add-save">Add endpoint</button>
        </div>
      `,
      (action) => {
        if (action === "cancel") {
          closeModal();
          return;
        }
        if (action !== "add") return;
        void submitAddEndpoint(g.id, chainId);
      },
    );
    document.getElementById("ep-add-input")?.focus();
  }

  async function submitAddEndpoint(gid: string, chainId: number): Promise<void> {
    if (!gw) return;
    const input = document.getElementById("ep-add-input") as HTMLInputElement | null;
    const btn = document.getElementById("ep-add-save") as HTMLButtonElement | null;
    const url = input?.value.trim() ?? "";
    if (!/^(https?|wss?):\/\//i.test(url)) {
      appendModalError("It needs a scheme eRPC can dial: http://, https://, ws:// or wss://.");
      return;
    }
    if (input) input.disabled = true;
    if (btn) {
      btn.disabled = true;
      btn.textContent = "Adding…";
    }
    const up: api.GatewayUpstream = {
      ID: crypto.randomUUID(),
      Kind: "external",
      Endpoint: url,
      Local: false,
      RecentOnly: false,
      Name: endpointNameFromUrl(url),
    };
    try {
      await api.putGatewayConfig(gid, withUpstream(gw.config, chainId, up));
    } catch (e) {
      appendModalError(`Could not add the endpoint: ${message(e)}`);
      if (input) input.disabled = false;
      if (btn) {
        btn.disabled = false;
        btn.textContent = "Add endpoint";
      }
      return;
    }
    closeModal();
    await provision(gid);
  }

  // openAddNetworkModal offers the catalog's networks (Devnet appended,
  // always last — see DEVNET_CHAIN_ID) minus whatever this gateway already
  // fronts. Fetching the catalog is the one part of this flow that cannot
  // start until the operator clicks — the list screen never prefetches it —
  // so `busy` is held for that short round trip too, disabling the rest of
  // the list exactly as it does during a real provision.
  async function openAddNetworkModal(g: api.GatewayView): Promise<void> {
    busy = "add-network";
    actionErr = null;
    render();
    let networks: api.Network[];
    try {
      const catalog = await api.getCatalog();
      networks = catalog.networks ?? [];
    } catch (e) {
      busy = null;
      actionErr = `Could not load the network catalog: ${message(e)}`;
      render();
      return;
    }
    busy = null;
    render();

    const present = new Set((g.networks ?? []).map((n) => n.chainId));
    const picks: { chainId: number; name: string }[] = networks
      .filter((n) => !present.has(n.ChainID))
      .map((n) => ({ chainId: n.ChainID, name: n.Name }));
    // Devnet is not in catalog.Networks() (see DEVNET_CHAIN_ID) but the
    // picker still offers it, last. Picking it goes through the same
    // external-only knownSet path as everything else here — there is no
    // managed-devnet wiring in this flow — so it honestly fails with the
    // server's own "no upstreams" error unless valve has a measured public
    // set for it. That failure surfaces through actionErr like any other.
    if (!present.has(DEVNET_CHAIN_ID)) picks.push({ chainId: DEVNET_CHAIN_ID, name: "Devnet" });
    if (picks.length === 0) {
      actionErr = "Every network valve's catalog knows about is already configured on this gateway.";
      render();
      return;
    }

    openModal(
      `
        <h2>Add a network</h2>
        <ul class="plain-list rpc-picker">
          ${picks
            .map(
              (p) => `
            <li>
              <button class="btn btn-ghost rpc-picker-option" data-modal-action="pick:${p.chainId}">
                <span>${escapeHtml(p.name)}</span>
                <span class="muted small">chain ${p.chainId}</span>
              </button>
            </li>`,
            )
            .join("")}
        </ul>
        <div class="modal-actions">
          <button class="btn btn-ghost" data-modal-action="cancel">Cancel</button>
        </div>
      `,
      (action) => {
        if (action === "cancel") {
          closeModal();
          return;
        }
        if (action.startsWith("pick:")) {
          const chainId = Number.parseInt(action.slice("pick:".length), 10);
          if (!Number.isFinite(chainId)) return;
          closeModal();
          void submitAddNetwork(g.id, chainId);
        }
      },
    );
  }

  // submitAddNetwork reads valve's known set for chainId (the same vetted
  // list "Add valve's set…" offers on the eRPC screen), writes every not-
  // already-added URL from it as one new network, provisions, and — once the
  // stream finishes clean and gw has been reloaded — opens that network's
  // own detail screen, since the operator picked it to go look at it.
  async function submitAddNetwork(gid: string, chainId: number): Promise<void> {
    if (!gw || busy) return;
    busy = "create";
    actionErr = null;
    render();
    let urls: string[];
    try {
      const set = await api.knownSet(gid, chainId);
      urls = (set.endpoints ?? []).filter((e) => !e.alreadyAdded).map((e) => e.url);
    } catch (e) {
      busy = null;
      actionErr = `Could not read valve's known set for chain ${chainId}: ${message(e)}`;
      render();
      return;
    }
    if (urls.length === 0) {
      busy = null;
      actionErr = `valve has no measured endpoints for chain ${chainId} yet, so there was nothing to add.`;
      render();
      return;
    }
    const upstreams: api.GatewayUpstream[] = urls.map((url, i) => ({
      ID: `public-${chainId}-${i + 1}`,
      Kind: "external",
      Endpoint: url,
      Local: false,
      RecentOnly: false,
    }));
    try {
      await api.putGatewayConfig(gid, withNetwork(gw.config, chainId, upstreams));
    } catch (e) {
      busy = null;
      actionErr = `Could not add the network: ${message(e)}`;
      render();
      return;
    }
    // provision() owns `busy` itself (it guards on `if (busy) return` at its
    // own top) — leaving it set to "create" here would make that guard fire
    // and provision() return immediately without recreating the container,
    // so the config write above would silently never go live. Caught live:
    // config.json picked up chain 943 but erpc.yaml (the rendered file the
    // container mounts) stayed stale until this was cleared.
    busy = null;
    await provision(gid, () => {
      view = { name: "network", chainId };
      render();
    });
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
  // same per-machine stream the node wizard and the devnet use. onDone, if
  // given, runs once the stream finishes WITHOUT an error and load() has
  // refreshed gw — used by add-network to land on the new network's own
  // detail screen only once it is actually there to show.
  async function provision(gid: string, onDone?: () => void): Promise<void> {
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
      void load().then(() => {
        if (!ev.err) onDone?.();
      });
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

  // stopped flags that renderPanel's cleanup already ran — load() is async,
  // so a fast mount/unmount (navigate away before the first load resolves)
  // could otherwise have this .then() fire AFTER cleanup and start an
  // interval nothing would ever clear. Checked right before arming poll.
  let stopped = false;
  void load().then(() => {
    if (stopped) return;
    // Baseline the fingerprint against what's ON SCREEN right now (netHealth
    // is still null at this point) so the poll's first tick only forces a
    // render if the live scrape actually changes a dot's class.
    lastHealthSig = healthSignature();
    poll = window.setInterval(() => {
      void refreshHealth();
    }, 5000);
  });
  return () => {
    stopped = true;
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
  netHealth: api.GatewayAnalytics | null,
): string {
  if (gw === null) return renderEmpty(busy, actionErr, setupLog);
  const m = masterState(gw);
  const rows = gw?.networks?.length
    ? gw.networks.map((nv, i) => networkRow(gw, nv, i > 0, netHealth)).join("")
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
function networkRow(gw: api.GatewayView, nv: api.NetworkView, divider: boolean, netHealth: api.GatewayAnalytics | null): string {
  const na = netHealth?.networks?.find((n) => n.chainId === nv.chainId);
  const slowRate = na ? networkSlowRate(na) : undefined;
  const hc = healthClass({ running: gw.status.State === "running", serviceable: nv.serviceable, slowRate });
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
  // netHealth is the panel's live 5s analytics poll (see renderPanel) — the
  // source for this screen's header dot and every endpoint row's dot.
  netHealth: api.GatewayAnalytics | null;
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
  const na = st.netHealth?.networks?.find((n) => n.chainId === chainId);
  const networkRate = na ? networkSlowRate(na) : undefined;
  const hc = healthClass({ running, serviceable: nv.serviceable, slowRate: networkRate });
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
      const upRate = na ? endpointSlowRate(na, u.id) : undefined;
      const uhc = healthClass({ running, serviceable: !u.problem, slowRate: upRate });
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

// EndpointDetailState mirrors NetworkDetailState one level down: the same
// caps cache (already scoped to the whole gateway, just filtered here to one
// upstream) plus epHealth for the one reading this screen adds that the
// network screen doesn't need — headLag.
interface EndpointDetailState {
  caps: api.GatewayCapabilities | null;
  capsBusy: boolean;
  health: api.GatewayAnalytics | null;
  healthBusy: boolean;
  copyFlash: boolean;
  error: string | null;
  // netHealth is the same live 5s poll NetworkDetailState carries — this
  // screen's dot uses it too, so it keeps animating on the same cadence as
  // the list/network screens rather than only on open/recheck like headLag
  // (epHealth/`health` above) does.
  netHealth: api.GatewayAnalytics | null;
}

// singleCapabilities is unionCapabilities narrowed to exactly one upstream —
// no folding, because there is only one endpoint's own verdict to show here.
function singleCapabilities(caps: api.GatewayCapabilities | null, chainId: number, upstreamId: string): Record<string, string> {
  const e = (caps?.endpoints ?? []).find((c) => c.chainId === chainId && c.upstream === upstreamId);
  if (!e) return {};
  const out: Record<string, string> = {};
  for (const key of ["http", "ws", "archive", "trace"]) {
    if (capStatusOf(e, key) === "supported") out[key] = "supported";
  }
  return out;
}

function renderEndpoint(gw: api.GatewayView | null, chainId: number, upstreamId: string, st: EndpointDetailState): string {
  const nv = gw?.networks?.find((n) => n.chainId === chainId);
  const up = nv?.upstreams?.find((u) => u.id === upstreamId);
  if (!gw || !nv || !up) {
    return `
      <div class="p-band p-dhead">
        <span class="p-back" data-action="back-to-network" data-chain-id="${chainId}">${ic("chevL")}</span>
        <span class="p-dtitle"><span class="p-nmtxt">Endpoint</span></span>
      </div>
      <div class="p-band" style="padding:16px;color:var(--dim)">This endpoint is no longer configured.</div>
    `;
  }

  const running = gw.status.State === "running";
  const na = st.netHealth?.networks?.find((n) => n.chainId === chainId);
  const upRate = na ? endpointSlowRate(na, upstreamId) : undefined;
  const hc = healthClass({ running, serviceable: !up.problem, slowRate: upRate });

  // Address band: the endpoint's own dialable URL. Editable only for
  // "external" upstreams — see the edit-address case in handleAction for why
  // managed ones (their Endpoint is derived, never stored) don't get the
  // affordance at all rather than one that would silently do nothing.
  const editable = up.kind === "external";
  const addressBand = `
    <div class="p-band">
      <div class="p-lblrow">
        <span class="p-seclbl">Address</span>
        <span class="p-acts"><span class="p-ic ${st.copyFlash ? "green" : "accent"}" data-action="copy-url" data-url="${escapeHtml(up.endpoint)}" title="Copy the endpoint URL">${ic("copy")}</span></span>
      </div>
      <div class="p-gwurl"${editable ? ' data-action="edit-address" style="cursor:text"' : ""}>${escapeHtml(up.endpoint || "—")}</div>
    </div>
  `;

  // Capabilities band: this ONE upstream's probe result, same cell markup as
  // the network band's folded one.
  const capStatuses = singleCapabilities(st.caps, chainId, upstreamId);
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

  // Status band: Health mirrors the row's own dot, worded from up.problem
  // when the server gave one (a real reason, not a generic label). "behind N
  // blocks" is added only when the gateway's own analytics scored this exact
  // upstream with a positive headLag — never shown, never guessed, when
  // there is nothing real to report.
  const healthWord = !running ? "Stopped" : up.problem ? up.problem : "Healthy";
  const healthEntry = (st.health?.endpoints ?? []).find((e) => e.chainId === chainId && e.upstream === upstreamId);
  const lagLine =
    healthEntry && healthEntry.scored && healthEntry.headLag > 0
      ? `<div class="p-srow"><span class="p-k">Chain head</span><span class="p-v">behind ${fmtInt(healthEntry.headLag)} block${healthEntry.headLag === 1 ? "" : "s"}</span></div>`
      : "";
  const statusBand = `
    <div class="p-band">
      <div class="p-lblrow"><span class="p-seclbl">Status</span><span class="p-acts"><span class="p-ic dim" data-action="recheck" title="Re-check capabilities and reload">${ic("refresh")}</span></span></div>
      <div class="p-srow"><span class="p-k">Health</span><span class="p-v"><span class="p-dot ${hc}"></span> ${escapeHtml(healthWord)}</span></div>
      ${lagLine}
    </div>
  `;

  const removeErrLine = st.error
    ? `<div class="p-band" style="padding:10px 16px;color:var(--red)">${escapeHtml(st.error)}</div>`
    : "";

  return `
    <div class="p-band p-dhead">
      <span class="p-back" data-action="back-to-network" data-chain-id="${chainId}">${ic("chevL")}</span>
      <span class="p-dtitle"><span class="p-dot ${hc}"></span> <span class="p-nmtxt">${escapeHtml(up.label)}</span> <span class="p-pen" data-action="rename-endpoint">${ic("pencil")}</span></span>
    </div>
    ${addressBand}
    ${capsBand}
    ${statusBand}
    ${removeErrLine}
    <div class="p-band p-remove" data-action="remove-endpoint">${ic("trash")} Remove endpoint</div>
  `;
}
