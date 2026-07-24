// #/setup/<id> — the guided setup wizard: pick a network, pick a valid
// exec/beacon client pair, pick full vs. archive + a data dir, review, then
// run with live SSE step progress.
//
// DEVIATION FROM THE BRIEF: the brief's review screen was specified to
// "show the exact units to be written", but there is no API that renders
// units ahead of time (catalog.RenderUnits is a server-internal helper
// setup.Plan calls; it's never exposed over HTTP, and setup.Plan itself
// only exists once a run actually starts). The review screen below shows
// the WireConfig summary that will be POSTed, plus a step list — the step
// list is a small client-side mirror of setup.Plan's fixed step sequence
// (preflight, toolchain, install-exec, install-beacon, wire, start,
// handshake; see internal/setup/steps.go), not something the API returns.
import * as api from "./api";
import { badge, dropdown, DropdownOption, escapeHtml, fmtBytes, footer, onAction, wireDropdowns } from "./ui";

type WizardStep = "network" | "clients" | "mode" | "review" | "run";

// STEP_PLAN mirrors internal/setup/steps.go's Plan() — a fixed sequence
// regardless of which clients are chosen (only the titles vary slightly,
// which we don't try to reproduce; the real titles come from the SSE
// stream's stepId once the run starts).
const STEP_PLAN: { id: string; title: string }[] = [
  { id: "preflight", title: "Preflight checks" },
  { id: "toolchain", title: "Ensure git + build toolchains" },
  { id: "install-exec", title: "Install execution client" },
  { id: "install-beacon", title: "Install beacon client" },
  { id: "wire", title: "Write JWT secret and systemd units" },
  { id: "start", title: "Start execution and beacon services" },
  { id: "handshake", title: "Verify execution/beacon handshake" },
];

// Port defaults mirror internal/catalog/units.go's defaultExecHTTPPort /
// defaultBeaconHTTPPort / defaultExecP2PPort — the wizard only sends a port
// field to the server when it differs from its default (see startSetup).
const DEFAULT_EXEC_HTTP_PORT = 8545;
const DEFAULT_BEACON_HTTP_PORT = 5052;
const DEFAULT_EXEC_P2P_PORT = 30303;

const NETWORK_ORDER = [369, 943, 1];
const NETWORK_BADGE: Record<number, string> = {
  369: "default",
  943: "practise here first",
};

interface State {
  targetId: string;
  step: WizardStep;
  catalog: api.Catalog | null;
  loadError: string | null;
  chainId: number | null;
  execId: string | null;
  beaconId: string | null;
  archive: boolean;
  dataDir: string;
  jwtPath: string;
  execHTTPPort: string;
  beaconHTTPPort: string;
  execP2PPort: string;
  execHTTPPortError: string | null;
  beaconHTTPPortError: string | null;
  execP2PPortError: string | null;
  rpcBindAddr: string;
  rpcBindAddrError: string | null;
  // Free-disk probe at the chosen data location.
  freeBytes: number | null;
  probedPath: string | null;
  diskProbing: boolean;
  diskError: string | null;
  downgradeNote: string | null;
  // Consensus checkpoint sync.
  checkpoint: boolean;
  checkpointUrl: string;
  checkpointUrlError: string | null;
  starting: boolean;
  startError: string | null;
  events: api.SetupEvent[];
  streamStop: (() => void) | null;
}

export function renderWizard(root: HTMLElement, targetId: string): () => void {
  let disposed = false;
  const state: State = {
    targetId,
    step: "network",
    catalog: null,
    loadError: null,
    chainId: 369,
    execId: null,
    beaconId: null,
    archive: true,
    dataDir: "",
    jwtPath: "",
    execHTTPPort: "",
    beaconHTTPPort: "",
    execP2PPort: "",
    execHTTPPortError: null,
    beaconHTTPPortError: null,
    execP2PPortError: null,
    rpcBindAddr: "",
    rpcBindAddrError: null,
    freeBytes: null,
    probedPath: null,
    diskProbing: false,
    diskError: null,
    downgradeNote: null,
    checkpoint: true,
    checkpointUrl: "",
    checkpointUrlError: null,
    starting: false,
    startError: null,
    events: [],
    streamStop: null,
  };

  root.innerHTML = `<h1>Setup: ${escapeHtml(targetId)}</h1><div id="wizard-body"><p class="muted">Loading catalog…</p></div><div id="wizard-footer">${footer()}</div>`;
  const body = root.querySelector<HTMLElement>("#wizard-body")!;
  const footerEl = root.querySelector<HTMLElement>("#wizard-footer")!;

  onAction(root, (action, el) => {
    handleAction(action, el);
  });

  wireDropdowns(root, (id, value) => {
    if (id === "exec-select") state.execId = value;
    else if (id === "beacon-select") state.beaconId = value;
    render();
  });

  // Delegated change handler for the mode-step inputs that need to react
  // immediately (rather than only on navigation): a new data location
  // re-probes free disk and re-evaluates the archive/full fit; toggling
  // checkpoint sync re-renders to show/hide the URL field.
  root.addEventListener("change", (ev) => {
    const t = ev.target;
    if (!(t instanceof HTMLInputElement)) return;
    if (t.id === "data-dir-input") {
      readModeInputs();
      void probeDisk();
    } else if (t.id === "checkpoint-toggle") {
      state.checkpoint = t.checked;
      render();
    }
  });

  load();

  async function load(): Promise<void> {
    try {
      const [catalog, targets] = await Promise.all([api.getCatalog(), api.listTargets()]);
      if (disposed) return;
      state.catalog = catalog;
      const existing = targets.find((t) => t.id === targetId);
      if (existing?.wire) {
        state.chainId = existing.wire.ChainID;
        state.execId = existing.wire.ExecID;
        state.beaconId = existing.wire.BeaconID;
        state.archive = existing.wire.Archive;
        if (existing.wire.ExecHTTPPort) state.execHTTPPort = String(existing.wire.ExecHTTPPort);
        if (existing.wire.BeaconHTTPPort) state.beaconHTTPPort = String(existing.wire.BeaconHTTPPort);
        if (existing.wire.ExecP2PPort) state.execP2PPort = String(existing.wire.ExecP2PPort);
        if (existing.wire.RPCBindAddr) state.rpcBindAddr = existing.wire.RPCBindAddr;
      }
      render();
    } catch (err) {
      if (disposed) return;
      state.loadError = String(err instanceof Error ? err.message : err);
      render();
    }
  }

  function render(): void {
    if (state.loadError) {
      body.innerHTML = `<p class="error">Failed to load: ${escapeHtml(state.loadError)}</p>`;
      return;
    }
    if (!state.catalog) return;

    body.innerHTML = `
      ${wizardProgress(state.step)}
      ${renderStep()}
    `;
    updateFooter();
  }

  // updateFooter refreshes the "learn more" link's per-context deep link to
  // the currently selected network's LearnURL, once a network is chosen.
  function updateFooter(): void {
    const net = state.catalog?.networks.find((n) => n.ChainID === state.chainId);
    footerEl.innerHTML = net ? footer(net.Name, net.LearnURL) : footer();
  }

  function renderStep(): string {
    switch (state.step) {
      case "network":
        return renderNetworkStep();
      case "clients":
        return renderClientsStep();
      case "mode":
        return renderModeStep();
      case "review":
        return renderReviewStep();
      case "run":
        return renderRunStep();
    }
  }

  function renderNetworkStep(): string {
    const catalog = state.catalog!;
    const cards = NETWORK_ORDER.map((chainId) => {
      const net = catalog.networks.find((n) => n.ChainID === chainId);
      if (!net) return "";
      const selected = state.chainId === chainId;
      const tag = NETWORK_BADGE[chainId] ? badge(NETWORK_BADGE[chainId]!, chainId === 369 ? "ok" : "warn") : "";
      return `
        <button class="card card-selectable ${selected ? "selected" : ""}" data-action="pick-network" data-chain-id="${chainId}" type="button">
          <h3>${escapeHtml(net.Name)} <span class="muted">(chain ${chainId})</span></h3>
          ${tag}
        </button>
      `;
    }).join("");

    return `
      <section>
        <h2>1. Choose a network</h2>
        <div class="card-grid">${cards}</div>
        <div class="wizard-actions">
          <button class="btn" data-action="goto-clients" ${state.chainId === null ? "disabled" : ""}>Next: clients</button>
        </div>
      </section>
    `;
  }

  function renderClientsStep(): string {
    const catalog = state.catalog!;
    const net = catalog.networks.find((n) => n.ChainID === state.chainId);
    if (!net) return `<p class="error">Unknown network.</p>`;

    if (state.execId === null || !net.ExecClients.includes(state.execId)) {
      state.execId = net.ExecClients[0] ?? null;
    }
    if (state.beaconId === null || !net.BeaconClients.includes(state.beaconId)) {
      state.beaconId = net.BeaconClients[0] ?? null;
    }

    const execOptions = net.ExecClients.map((id) => clientOption(id, catalog));
    const beaconOptions = net.BeaconClients.map((id) => clientOption(id, catalog));

    return `
      <section>
        <h2>2. Choose your client pair</h2>
        <p class="muted">Only combinations known to work on ${escapeHtml(net.Name)} are offered.</p>
        <p class="muted small">
          The <strong>provider</strong> shown for each client is the org that publishes it —
          some are the original upstream team, others are forks. Check the source if you only
          want to run a client from a particular team.
        </p>
        <label>
          Execution client
          ${dropdown("exec-select", execOptions, state.execId)}
        </label>
        ${clientSourceLine(state.execId, catalog)}
        <label>
          Beacon client
          ${dropdown("beacon-select", beaconOptions, state.beaconId)}
        </label>
        ${clientSourceLine(state.beaconId, catalog)}
        <div class="wizard-actions">
          <button class="btn btn-ghost" data-action="goto-network">Back</button>
          <button class="btn" data-action="goto-mode">Next: mode</button>
        </div>
      </section>
    `;
  }

  // approxSize formats a decimal-TB dataset estimate for display, dropping to
  // GB below a terabyte. Kept in decimal (not binary) so the figures match
  // learn.valve.city's snapshot sizes and the Go catalog's ExpectedBytes.
  function approxSize(sizeTB: number): string {
    if (sizeTB <= 0) return "—";
    if (sizeTB >= 1) return `~${sizeTB.toFixed(1)} TB`;
    return `~${Math.round(sizeTB * 1000)} GB`;
  }

  // FIT_MARGIN matches the server preflight's 10% headroom over the raw
  // dataset estimate.
  const FIT_MARGIN = 1.1;

  function tierNeeds(net: api.Network): { archive: number; full: number } {
    const archive = net.ArchiveSizeTB * 1e12 * FIT_MARGIN;
    return { archive, full: archive / 2 };
  }

  // storageStatusHtml renders the free-disk readout for the chosen data
  // location: how much is free and whether each tier fits, plus any
  // auto-downgrade note or a hard "neither fits" warning.
  function storageStatusHtml(net: api.Network | undefined, path: string): string {
    if (!net) return "";
    if (state.diskProbing) {
      return `<p class="muted small">Checking free space at <code>${escapeHtml(path)}</code>…</p>`;
    }
    if (state.diskError) {
      return `<p class="error small">Couldn't read free space at <code>${escapeHtml(path)}</code>: ${escapeHtml(state.diskError)}</p>`;
    }
    if (state.freeBytes === null || state.probedPath !== path) return "";
    const needs = tierNeeds(net);
    const archiveFits = state.freeBytes >= needs.archive;
    const fullFits = state.freeBytes >= needs.full;
    const line = `<p class="muted small">Free at <code>${escapeHtml(path)}</code>: <strong>${fmtBytes(state.freeBytes)}</strong> — archive ${archiveFits ? "fits" : "won't fit"} (~${approxSize(net.ArchiveSizeTB)}), full ${fullFits ? "fits" : "won't fit"} (~${approxSize(net.ArchiveSizeTB / 2)}).</p>`;
    let note = "";
    if (state.downgradeNote) {
      note = `<p class="banner banner-warn">${escapeHtml(state.downgradeNote)}</p>`;
    } else if (!fullFits) {
      note = `<p class="banner banner-warn">Neither mode fits at this location (full needs ~${approxSize(net.ArchiveSizeTB / 2)}). Choose a location with more space.</p>`;
    }
    return line + note;
  }

  // evaluateFit runs after a disk probe: if archive is selected but the
  // location can't hold it (while full would fit), downgrade to full and
  // record a note. Only ever triggered by a probe (location change / step
  // entry), never by a manual mode pick — so it never fights the operator
  // who deliberately re-selects archive.
  function evaluateFit(net: api.Network | undefined, path: string): void {
    state.downgradeNote = null;
    if (!net || state.freeBytes === null) return;
    const needs = tierNeeds(net);
    if (state.archive && state.freeBytes < needs.archive && state.freeBytes >= needs.full) {
      state.archive = false;
      state.downgradeNote = `Not enough space at ${path} for archive (~${approxSize(net.ArchiveSizeTB)}) — switched to Full (~${approxSize(net.ArchiveSizeTB / 2)}). Pick a location with more room to run archive.`;
    }
  }

  async function probeDisk(): Promise<void> {
    if (state.chainId === null) return;
    const net = state.catalog?.networks.find((n) => n.ChainID === state.chainId);
    const path = (state.dataDir || `/var/lib/valve-node/${state.chainId}`).trim();
    state.diskProbing = true;
    state.diskError = null;
    render();
    try {
      const { freeBytes } = await api.getDiskFree(state.targetId, path);
      if (disposed) return;
      state.freeBytes = freeBytes;
      state.probedPath = path;
      evaluateFit(net, path);
    } catch (err) {
      if (disposed) return;
      state.freeBytes = null;
      state.probedPath = path;
      state.diskError = String(err instanceof Error ? err.message : err);
    }
    state.diskProbing = false;
    render();
  }

  // validateCheckpointUrl mirrors the server's check: empty is fine (network
  // default), otherwise it must be an http(s) URL.
  function validateCheckpointUrl(raw: string): string | null {
    if (!raw) return null;
    if (!/^https?:\/\/.+/i.test(raw)) {
      return "Enter an http(s) URL, or leave blank for the network default.";
    }
    return null;
  }

  function clientOption(id: string, catalog: api.Catalog): DropdownOption {
    const client = catalog.clients.find((c) => c.id === id);
    return { value: id, label: client ? `${client.id} — ${clientProvider(client.repo)}` : id };
  }

  // clientProvider extracts the publishing org from a client's source repo
  // (e.g. https://github.com/valve-tech/reth → "valve-tech"), so the picker
  // shows who actually provides each client — upstream team vs a fork.
  function clientProvider(repo: string): string {
    const parts = repo.split("/");
    return parts.length >= 4 ? parts[3] : repo;
  }

  // clientSourceLine renders the selected client's source repository as a
  // link, so the provenance is verifiable, not just a label.
  function clientSourceLine(id: string | null, catalog: api.Catalog): string {
    const client = id ? catalog.clients.find((c) => c.id === id) : undefined;
    if (!client) return "";
    const shown = client.repo.replace(/^https?:\/\//, "");
    return `<p class="muted small">Source: <a href="${escapeHtml(client.repo)}" target="_blank" rel="noopener noreferrer">${escapeHtml(shown)}</a></p>`;
  }

  function renderModeStep(): string {
    const defaultDataDir = state.chainId !== null ? `/var/lib/valve-node/${state.chainId}` : "";
    const net = state.catalog?.networks.find((n) => n.ChainID === state.chainId);
    const archiveTB = net?.ArchiveSizeTB ?? 0;
    const fullSizeCell = net ? approxSize(archiveTB / 2) : "Smaller";
    const archiveSizeCell = net ? approxSize(archiveTB) : "Much larger";
    const netLabel = net ? ` on ${escapeHtml(net.Name)}` : "";
    return `
      <section>
        <h2>3. Choose sync mode</h2>
        <p class="muted">
          Both modes run a fully-validating node — same security, same current-state RPC.
          The difference is how much <strong>historical</strong> state is kept.
        </p>
        <table class="compare-table">
          <thead>
            <tr><th>What you get</th><th>Full</th><th>Archive</th></tr>
          </thead>
          <tbody>
            <tr><th>Current state &amp; recent blocks</th><td class="yes">Yes</td><td class="yes">Yes</td></tr>
            <tr><th>Send transactions, normal RPC</th><td class="yes">Yes</td><td class="yes">Yes</td></tr>
            <tr><th>Historical state (balances, <code>eth_call</code>) at any past block</th><td class="limited">Recent only (~128 blocks)</td><td class="yes">Full history</td></tr>
            <tr><th>Tracing / <code>debug_trace</code> on old blocks</th><td class="limited">Recent only</td><td class="yes">Full history</td></tr>
            <tr><th>Approx. disk footprint${netLabel}</th><td class="yes">${fullSizeCell}</td><td class="limited">${archiveSizeCell}</td></tr>
            <tr><th>Initial sync time${netLabel}</th><td class="yes">${net ? escapeHtml(net.SyncLabel) : "Faster"}</td><td class="limited">${net ? escapeHtml(net.GenesisSyncLabel) : "Much slower"}</td></tr>
            <tr><th>Best for</th><td>Validators, wallets, everyday RPC</td><td>Explorers, analytics, historical queries</td></tr>
          </tbody>
        </table>
        <p class="muted small">
          Disk sizes and sync times are rough baselines — both vary by client and scale with the
          target's CPU and disk speed.
        </p>
        <label class="radio">
          <input type="radio" name="mode" value="archive" data-action="pick-mode" ${state.archive ? "checked" : ""} />
          <span><strong>Archive</strong> — full historical state · ${archiveSizeCell}${net ? "" : " disk"} <span class="muted">(recommended — keep more archive nodes on the network)</span></span>
        </label>
        <label class="radio">
          <input type="radio" name="mode" value="full" data-action="pick-mode" ${!state.archive ? "checked" : ""} />
          <span><strong>Full</strong> — pruned, everyday RPC · ${fullSizeCell}${net ? "" : " disk"}</span>
        </label>

        <div class="config-block">
          <label>
            Data location <span class="muted">(default: ${escapeHtml(defaultDataDir)})</span>
            <input id="data-dir-input" type="text" placeholder="${escapeHtml(defaultDataDir)}" value="${escapeHtml(state.dataDir)}" />
          </label>
          ${storageStatusHtml(net, state.dataDir || defaultDataDir)}
        </div>

        <div class="config-block">
          <label class="radio">
            <input type="checkbox" id="checkpoint-toggle" ${state.checkpoint ? "checked" : ""} />
            <span><strong>Checkpoint sync</strong> — start near the chain head in minutes (recommended). Uncheck to sync the beacon chain from genesis: fully trustless, but much slower.</span>
          </label>
          ${
            state.checkpoint
              ? `<label>
                   Checkpoint URL <span class="muted">(default: ${escapeHtml(net?.CheckpointURL ?? "")})</span>
                   <input id="checkpoint-url-input" type="text" placeholder="${escapeHtml(net?.CheckpointURL ?? "")}" value="${escapeHtml(state.checkpointUrl)}" />
                 </label>
                 ${state.checkpointUrlError ? `<p class="error small">${escapeHtml(state.checkpointUrlError)}</p>` : ""}
                 <p class="muted small">The beacon client trusts this endpoint for its starting checkpoint. Leave blank for the network default.</p>`
              : `<p class="muted small">The beacon client will validate every block from genesis — no trusted checkpoint, but this can take days.</p>`
          }
        </div>

        <details class="advanced">
          <summary>Advanced</summary>
          <label>
            JWT secret path <span class="muted">(default: &lt;data dir&gt;/jwt.hex)</span>
            <input id="jwt-path-input" type="text" placeholder="${escapeHtml(defaultDataDir)}/jwt.hex" value="${escapeHtml(state.jwtPath)}" />
          </label>
          <label>
            Execution HTTP port <span class="muted">(default: ${DEFAULT_EXEC_HTTP_PORT})</span>
            <input id="exec-http-port-input" type="text" inputmode="numeric" placeholder="${DEFAULT_EXEC_HTTP_PORT}" value="${escapeHtml(state.execHTTPPort)}" />
          </label>
          ${state.execHTTPPortError ? `<p class="error small">${escapeHtml(state.execHTTPPortError)}</p>` : ""}
          <label>
            Beacon HTTP port <span class="muted">(default: ${DEFAULT_BEACON_HTTP_PORT})</span>
            <input id="beacon-http-port-input" type="text" inputmode="numeric" placeholder="${DEFAULT_BEACON_HTTP_PORT}" value="${escapeHtml(state.beaconHTTPPort)}" />
          </label>
          ${state.beaconHTTPPortError ? `<p class="error small">${escapeHtml(state.beaconHTTPPortError)}</p>` : ""}
          <label>
            Execution p2p port <span class="muted">(default: ${DEFAULT_EXEC_P2P_PORT})</span>
            <input id="exec-p2p-port-input" type="text" inputmode="numeric" placeholder="${DEFAULT_EXEC_P2P_PORT}" value="${escapeHtml(state.execP2PPort)}" />
          </label>
          ${state.execP2PPortError ? `<p class="error small">${escapeHtml(state.execP2PPortError)}</p>` : ""}
          <label>
            RPC bind address <span class="muted">(default: 127.0.0.1, loopback-only)</span>
            <input id="rpc-bind-addr-input" type="text" inputmode="text" placeholder="127.0.0.1" value="${escapeHtml(state.rpcBindAddr)}" />
          </label>
          ${state.rpcBindAddrError ? `<p class="error small">${escapeHtml(state.rpcBindAddrError)}</p>` : ""}
          <p class="muted small">
            Leave any of these blank to use the default. The engine API port (8551) is fixed and
            loopback-only — it isn't configurable. Set the RPC bind address to this box's
            <strong>Tailscale IP</strong> (or another trusted overlay address) to reach the node's
            exec/beacon RPC from your own machine without an SSH tunnel. Note: the RPC is
            <strong>unauthenticated</strong>, so anyone on that network can drive the node — only
            bind to a trusted, private overlay, never a public address.
          </p>
        </details>
        <div class="wizard-actions">
          <button class="btn btn-ghost" data-action="goto-clients">Back</button>
          <button class="btn" data-action="goto-review">Next: review</button>
        </div>
      </section>
    `;
  }

  function renderReviewStep(): string {
    const catalog = state.catalog!;
    const net = catalog.networks.find((n) => n.ChainID === state.chainId);
    const dataDir = state.dataDir || `/var/lib/valve-node/${state.chainId}`;
    const jwtPath = state.jwtPath || `${dataDir}/jwt.hex`;

    const stepRows = STEP_PLAN.map((s) => `<li>${escapeHtml(s.title)}</li>`).join("");

    const execHTTPPort = portOverride(state.execHTTPPort, DEFAULT_EXEC_HTTP_PORT);
    const beaconHTTPPort = portOverride(state.beaconHTTPPort, DEFAULT_BEACON_HTTP_PORT);
    const execP2PPort = portOverride(state.execP2PPort, DEFAULT_EXEC_P2P_PORT);
    const portsRow =
      execHTTPPort || beaconHTTPPort || execP2PPort
        ? `<tr><th>Non-default ports</th><td>${[
            execHTTPPort ? `exec HTTP ${execHTTPPort}` : null,
            beaconHTTPPort ? `beacon HTTP ${beaconHTTPPort}` : null,
            execP2PPort ? `exec p2p ${execP2PPort}` : null,
          ]
            .filter((s): s is string => s !== null)
            .map(escapeHtml)
            .join(", ")}</td></tr>`
        : "";

    const { addr: rpcBindAddr } = parseBindAddr(state.rpcBindAddr);
    const rpcBindRow = rpcBindAddr
      ? `<tr><th>RPC bind address</th><td><code>${escapeHtml(rpcBindAddr)}</code> <span class="muted">(reachable off-box — unauthenticated, keep it on a trusted overlay)</span></td></tr>`
      : "";

    return `
      <section>
        <h2>4. Review</h2>
        <table class="review-table">
          <tbody>
            <tr><th>Target</th><td>${escapeHtml(state.targetId)}</td></tr>
            <tr><th>Network</th><td>${escapeHtml(net?.Name ?? String(state.chainId))} (chain ${state.chainId})</td></tr>
            <tr><th>Execution client</th><td>${escapeHtml(state.execId ?? "")}</td></tr>
            <tr><th>Beacon client</th><td>${escapeHtml(state.beaconId ?? "")}</td></tr>
            <tr><th>Mode</th><td>${state.archive ? "Archive" : "Full"}</td></tr>
            <tr><th>Data directory</th><td><code>${escapeHtml(dataDir)}</code></td></tr>
            <tr><th>JWT secret path</th><td><code>${escapeHtml(jwtPath)}</code></td></tr>
            <tr><th>Checkpoint sync</th><td>${
              state.checkpoint
                ? `<code>${escapeHtml(state.checkpointUrl || net?.CheckpointURL || "")}</code>`
                : "off — syncing from genesis"
            }</td></tr>
            ${portsRow}
            ${rpcBindRow}
          </tbody>
        </table>
        <p class="muted small">
          There is no preview API for the exact files/units that will be
          written — the list below is the fixed step sequence setup always
          runs; the actual commands and file contents stream live once you
          start.
        </p>
        <ol class="step-preview">${stepRows}</ol>
        ${state.startError ? `<p class="error">${escapeHtml(state.startError)}</p>` : ""}
        <div class="wizard-actions">
          <button class="btn btn-ghost" data-action="goto-mode">Back</button>
          <button class="btn btn-primary" data-action="start-setup" ${state.starting ? "disabled" : ""}>
            ${state.starting ? "Starting…" : "Start setup"}
          </button>
        </div>
      </section>
    `;
  }

  function renderRunStep(): string {
    const catalog = state.catalog!;
    const net = catalog.networks.find((n) => n.ChainID === state.chainId);
    const learnUrl = net?.LearnURL;

    const doneIds = new Set(state.events.filter((e) => e.done).map((e) => e.stepId));
    const erroredIds = new Set(state.events.filter((e) => e.err).map((e) => e.stepId));
    const linesByStep = new Map<string, string[]>();
    for (const ev of state.events) {
      if (!ev.line) continue;
      const list = linesByStep.get(ev.stepId) ?? [];
      list.push(ev.line);
      linesByStep.set(ev.stepId, list);
    }

    const items = STEP_PLAN.map((s) => {
      const isDone = doneIds.has(s.id);
      const isError = erroredIds.has(s.id);
      const mark = isError ? badge("failed", "bad") : isDone ? badge("done", "ok") : badge("pending", "neutral");
      const lines = (linesByStep.get(s.id) ?? []).slice(-5);
      const errLine = state.events.find((e) => e.stepId === s.id && e.err)?.err;
      const handshakeNote =
        s.id === "handshake"
          ? `<p class="muted small">"Talking" means the beacon client can reach the execution client's Engine API over the shared JWT secret and both report the same head — the sign your node is wired correctly.${learnUrl ? ` <a href="${escapeHtml(learnUrl)}" target="_blank" rel="noopener noreferrer">Learn more →</a>` : ""}</p>`
          : "";
      return `
        <li class="step-row ${isDone ? "step-done" : ""} ${isError ? "step-error" : ""}">
          <div class="step-head">${mark} <strong>${escapeHtml(s.title)}</strong></div>
          ${handshakeNote}
          ${lines.length ? `<pre class="step-log">${lines.map((l) => escapeHtml(l)).join("\n")}</pre>` : ""}
          ${errLine ? `<p class="error small">${escapeHtml(errLine)}</p>` : ""}
        </li>
      `;
    }).join("");

    const anyError = state.events.some((e) => e.err);
    const allDone =
      STEP_PLAN.every((s) => doneIds.has(s.id)) || state.events.some((e) => e.stepId === "handshake" && e.done);

    return `
      <section>
        <h2>5. Running setup</h2>
        <ol class="step-list">${items}</ol>
        ${
          allDone && !anyError
            ? `<p class="ok">Setup complete. <a href="#/dash/${encodeURIComponent(state.targetId)}">Open the dashboard →</a></p>`
            : ""
        }
        ${anyError ? `<button class="btn" data-action="start-setup">Retry setup</button>` : ""}
      </section>
    `;
  }

  function handleAction(action: string, el: HTMLElement): void {
    switch (action) {
      case "pick-network":
        state.chainId = Number(el.dataset.chainId);
        state.execId = null;
        state.beaconId = null;
        render();
        break;
      case "goto-network":
        state.step = "network";
        render();
        break;
      case "goto-clients":
        if (state.chainId === null) return;
        state.step = "clients";
        render();
        break;
      case "goto-mode":
        state.step = "mode";
        render();
        void probeDisk();
        break;
      case "goto-review":
        readModeInputs();
        if (
          state.execHTTPPortError ||
          state.beaconHTTPPortError ||
          state.execP2PPortError ||
          state.rpcBindAddrError ||
          state.checkpointUrlError
        ) {
          render();
          break;
        }
        state.step = "review";
        render();
        break;
      case "start-setup":
        void startSetup();
        break;
    }
  }

  function readModeInputs(): void {
    const radios = root.querySelectorAll<HTMLInputElement>('input[name="mode"]');
    for (const r of Array.from(radios)) {
      if (r.checked) state.archive = r.value === "archive";
    }
    const dataDirInput = root.querySelector<HTMLInputElement>("#data-dir-input");
    const jwtPathInput = root.querySelector<HTMLInputElement>("#jwt-path-input");
    if (dataDirInput) state.dataDir = dataDirInput.value.trim();
    if (jwtPathInput) state.jwtPath = jwtPathInput.value.trim();

    const execHTTPPortInput = root.querySelector<HTMLInputElement>("#exec-http-port-input");
    const beaconHTTPPortInput = root.querySelector<HTMLInputElement>("#beacon-http-port-input");
    const execP2PPortInput = root.querySelector<HTMLInputElement>("#exec-p2p-port-input");
    if (execHTTPPortInput) state.execHTTPPort = execHTTPPortInput.value.trim();
    if (beaconHTTPPortInput) state.beaconHTTPPort = beaconHTTPPortInput.value.trim();
    if (execP2PPortInput) state.execP2PPort = execP2PPortInput.value.trim();

    const rpcBindAddrInput = root.querySelector<HTMLInputElement>("#rpc-bind-addr-input");
    if (rpcBindAddrInput) state.rpcBindAddr = rpcBindAddrInput.value.trim();

    const checkpointUrlInput = root.querySelector<HTMLInputElement>("#checkpoint-url-input");
    if (checkpointUrlInput) state.checkpointUrl = checkpointUrlInput.value.trim();

    state.execHTTPPortError = parsePort(state.execHTTPPort).error ?? null;
    state.beaconHTTPPortError = parsePort(state.beaconHTTPPort).error ?? null;
    state.execP2PPortError = parsePort(state.execP2PPort).error ?? null;
    state.rpcBindAddrError = parseBindAddr(state.rpcBindAddr).error ?? null;
    state.checkpointUrlError = state.checkpoint ? validateCheckpointUrl(state.checkpointUrl) : null;
  }

  // parseBindAddr validates an optional RPC bind address: empty means the
  // loopback default; otherwise it must be a valid IPv4/IPv6 literal (the
  // server enforces the same with net.ParseIP). Hostnames are rejected —
  // clients bind to addresses, not names.
  function parseBindAddr(raw: string): { addr?: string; error?: string } {
    if (!raw) return {};
    const v4 = /^(\d{1,3})\.(\d{1,3})\.(\d{1,3})\.(\d{1,3})$/.exec(raw);
    if (v4) {
      if (v4.slice(1).every((o) => Number(o) <= 255)) return { addr: raw };
      return { error: "Each part of an IPv4 address must be 0–255." };
    }
    // Loose IPv6 acceptance: hex groups and colons (and an optional zone).
    if (/^[0-9a-fA-F:]+(%[0-9a-zA-Z]+)?$/.test(raw) && raw.includes(":")) {
      return { addr: raw };
    }
    return { error: "Enter a valid IP address (e.g. your Tailscale 100.x.y.z), or leave blank for loopback." };
  }

  // VALID_PORT_RE matches only a plain, unsigned decimal integer — no
  // decimal point, sign, exponent, or leading/trailing whitespace (already
  // trimmed by the caller). Number.parseInt would happily read "8080.5" as
  // 8080 and silently swallow the invalid suffix, so raw input is checked
  // against this regex before any numeric parsing happens.
  const VALID_PORT_RE = /^\d+$/;

  // parsePort validates a raw port field string against the 1-65535 range
  // (0 is reserved for "use the server's default" and is expressed by
  // leaving the field blank, never by typing "0" — see the port fields'
  // "leave blank for default" help text). Returns a user-facing error
  // message when the value is present but invalid.
  function parsePort(raw: string): { port?: number; error?: string } {
    if (!raw) return {};
    if (!VALID_PORT_RE.test(raw)) {
      return { error: "Enter a whole number (no decimals, signs, or other characters)." };
    }
    const n = Number(raw);
    if (!Number.isInteger(n) || n < 1 || n > 65535) {
      return { error: "Port must be between 1 and 65535." };
    }
    return { port: n };
  }

  // portOverride returns raw's parsed port only if it's valid and differs
  // from def — the wizard never sends a port field equal to the server's
  // own default, since "not set" and "set to default" are indistinguishable
  // once persisted anyway (see api.ts's WireConfig comment). Invalid values
  // are dropped here defensively; readModeInputs is what surfaces the
  // inline field error and blocks navigation to review in the first place.
  function portOverride(raw: string, def: number): number | undefined {
    const { port } = parsePort(raw);
    if (port === undefined || port === def) return undefined;
    return port;
  }

  async function startSetup(): Promise<void> {
    if (state.chainId === null || !state.execId || !state.beaconId) return;
    state.starting = true;
    state.startError = null;
    render();

    const wire: api.StartSetupRequest = {
      ChainID: state.chainId,
      ExecID: state.execId,
      BeaconID: state.beaconId,
      Archive: state.archive,
    };
    if (state.dataDir) wire.DataDir = state.dataDir;
    if (state.jwtPath) wire.JWTPath = state.jwtPath;

    const execHTTPPort = portOverride(state.execHTTPPort, DEFAULT_EXEC_HTTP_PORT);
    const beaconHTTPPort = portOverride(state.beaconHTTPPort, DEFAULT_BEACON_HTTP_PORT);
    const execP2PPort = portOverride(state.execP2PPort, DEFAULT_EXEC_P2P_PORT);
    if (execHTTPPort !== undefined) wire.ExecHTTPPort = execHTTPPort;
    if (beaconHTTPPort !== undefined) wire.BeaconHTTPPort = beaconHTTPPort;
    if (execP2PPort !== undefined) wire.ExecP2PPort = execP2PPort;

    const { addr: rpcBindAddr } = parseBindAddr(state.rpcBindAddr);
    if (rpcBindAddr !== undefined) wire.RPCBindAddr = rpcBindAddr;

    if (!state.checkpoint) wire.NoCheckpoint = true;
    else if (state.checkpointUrl) wire.CheckpointURL = state.checkpointUrl;

    try {
      await api.startSetup(state.targetId, wire);
    } catch (err) {
      // 409 means a run is already in flight for this target — that's fine,
      // we just attach to its live stream below instead of starting a new
      // one.
      if (!(err instanceof api.ApiError && err.status === 409)) {
        state.starting = false;
        state.startError = String(err instanceof Error ? err.message : err);
        render();
        return;
      }
    }

    state.starting = false;
    state.step = "run";
    state.events = [];
    render();
    state.streamStop?.();
    state.streamStop = api.streamSetup(state.targetId, (ev) => {
      if (disposed) return;
      state.events.push(ev);
      if (state.step === "run") render();
    });
  }

  function wizardProgress(current: WizardStep): string {
    const steps: { id: WizardStep; label: string }[] = [
      { id: "network", label: "Network" },
      { id: "clients", label: "Clients" },
      { id: "mode", label: "Mode" },
      { id: "review", label: "Review" },
      { id: "run", label: "Run" },
    ];
    const order = steps.map((s) => s.id);
    const currentIdx = order.indexOf(current);
    return `
      <ol class="wizard-progress">
        ${steps
          .map((s, i) => {
            const cls = i === currentIdx ? "current" : i < currentIdx ? "past" : "future";
            return `<li class="${cls}">${escapeHtml(s.label)}</li>`;
          })
          .join("")}
      </ol>
    `;
  }

  return () => {
    disposed = true;
    state.streamStop?.();
  };
}
