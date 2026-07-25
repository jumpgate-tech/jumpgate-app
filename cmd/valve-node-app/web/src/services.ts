// #/services/<id> — the container-backed service a machine hosts: its local
// devnet. Status, start/stop/restart, create/re-create, reset, wipe, and its
// configuration, in one screen.
//
// The eRPC gateway used to share this screen and deliberately no longer does.
// A devnet is a chain running in a container on THIS box and can be nowhere
// else, so it belongs to the machine. A gateway fronts chains all over the
// fleet, so the machine it runs on is one field of it — it lives on the
// top-level RPC screen (#/rpc).
//
// Three rules shape this file, all of them learned the hard way elsewhere in
// this app:
//
//  1. Availability is shown ON the thing you interact with. Every card states
//     what it is, what state it is in, and what can be done to it right now.
//     The server decides that list (containers.go's availableActions), so a
//     button that ops would reject is never rendered — and when the list is
//     empty the card says WHY instead of leaving a row of dead buttons.
//
//  2. Nothing here uses a native dialog. The wipe confirmation is a modal
//     with a typed confirmation, matching the dashboard's clear-and-resync
//     gate.
//
//  3. A wipe reports what it did, including the services it restarted.
//     ops.WipeService bounces anything fronting a wiped chain because eRPC's
//     per-network head only ever moves forward, so a reset chain leaves the
//     gateway advertising blocks that no longer exist. That restart is the
//     one part of a wipe an operator cannot see happening, so the modal says
//     it will happen beforehand and reports that it did afterwards.
import * as api from "./api";
import {
  badge,
  closeModal,
  copyToClipboard,
  dot,
  escapeHtml,
  footer,
  modalBody,
  onAction,
  openModal,
} from "./ui";

// The step id both container plans end with (PlanDevnet: preflight, run;
// PlanGateway: preflight, config, run). The setup event stream has no
// terminal frame, so this is what tells a provisioning run's progress log
// that it is finished and the cards can be re-read.
const FINAL_STEP = "run";

const SERVICE_BLURB: Record<api.ContainerServiceID, string> = {
  devnet:
    "A throwaway chain that runs entirely on this machine: reth in --dev mode, sealing a block on a timer from its own genesis. Nothing to sync, nothing on disk outside the container.",
};

interface ActionButton {
  label: string;
  title: string;
  className: string;
}

// ACTION_BUTTONS is the presentation of the server's action ids. Labels say
// what will happen, not what the verb is called internally: "recreate" is
// destructive-ish (the container is replaced) and its label has to admit
// that, because it is the only way an edited port or block time ever takes
// effect.
const ACTION_BUTTONS: Record<string, ActionButton> = {
  start: { label: "Start", title: "Start the existing container", className: "btn" },
  stop: { label: "Stop", title: "Stop the container; it keeps its data", className: "btn btn-ghost" },
  restart: {
    label: "Restart",
    title: "Restart the container. Data is kept, so no head ever moves backwards — nothing in front of it is touched.",
    className: "btn btn-ghost",
  },
  create: { label: "Create", title: "Create the container from the configuration below", className: "btn" },
  recreate: {
    label: "Re-create (apply config)",
    title:
      "Replace the container so the saved configuration takes effect. Ports, mounts and the command line are fixed when a container is created, so this is the only way to apply a change.",
    className: "btn btn-ghost",
  },
  wipe: { label: "Wipe…", title: "Destroy this service's data and rebuild it", className: "btn btn-danger" },
};

export function renderServices(root: HTMLElement, targetId: string): () => void {
  let disposed = false;
  let data: api.ContainersResponse | null = null;
  let loadErr: string | null = null;

  // busy holds the in-flight action per service, so its card can show a
  // spinner and disable the rest of its buttons. State always comes from a
  // re-read afterwards, never from the action's own response.
  const busy: Record<string, string | null> = { devnet: null };
  const actionErr: Record<string, string | null> = { devnet: null };

  // Provisioning progress, per service. Kept here rather than in the DOM so a
  // re-render (triggered by any other card) does not lose it.
  const activity: Record<string, string[]> = { devnet: [] };
  let streamStop: (() => void) | null = null;

  // Draft configs. They exist only while an editor is open, and are the
  // single source of truth for it: structural edits (add/remove an upstream)
  // re-render the form, so anything typed is read back into the draft first.
  const open: Record<string, boolean> = { devnet: false };
  let devnetDraft: api.DevnetConfig | null = null;
  const saveErr: Record<string, string | null> = { devnet: null };
  const saveNote: Record<string, string | null> = { devnet: null };

  root.innerHTML = `
    <div class="page-head">
      <h1>Services: ${escapeHtml(targetId)}</h1>
      <button class="btn btn-ghost" data-action="refresh">Refresh</button>
    </div>
    <p class="muted">
      The throwaway chain this machine can host. It is independent of any node
      setup — a machine can run a devnet, a node, both, or neither. The RPC
      gateway in front of it lives on the <a href="#/rpc">RPC</a> screen, because
      it fronts chains across every machine rather than belonging to this one.
    </p>
    <div id="services-body"><p class="muted">Loading…</p></div>
    ${footer()}
  `;
  const body = root.querySelector<HTMLElement>("#services-body")!;

  onAction(root, (action, el) => {
    void handleAction(action, el);
  });

  void load();

  // --- data ---------------------------------------------------------------

  async function load(): Promise<void> {
    try {
      const next = await api.getContainers(targetId);
      if (disposed) return;
      data = next;
      loadErr = null;
    } catch (err) {
      if (disposed) return;
      data = null;
      loadErr = message(err);
    }
    render();
  }

  function viewOf(svc: string): api.ContainerView | undefined {
    return data?.services.find((s) => s.id === svc);
  }

  // --- render -------------------------------------------------------------

  function render(): void {
    if (disposed) return;
    if (loadErr) {
      body.innerHTML = `<p class="error">Could not read this machine's services: ${escapeHtml(loadErr)}</p>`;
      return;
    }
    if (!data) {
      body.innerHTML = `<p class="muted">Loading…</p>`;
      return;
    }
    body.innerHTML = `
      ${dockerBanner(data.docker)}
      <div class="card-grid card-grid-wide">
        ${data.services.map(serviceCard).join("")}
      </div>
    `;
  }

  // dockerBanner states the one fact both services depend on, once. Without
  // it every card would separately report a failure that has a single cause
  // and a single fix.
  function dockerBanner(d: api.DockerView): string {
    if (d.present && d.reachable && !d.hint) {
      return `<p class="muted small">Docker: ${escapeHtml(d.flavor)}${d.serverVersion ? ` ${escapeHtml(d.serverVersion)}` : ""} · reachable</p>`;
    }
    const title = !d.present ? "No docker engine on this machine" : "Docker is installed, but no engine answered";
    return `
      <div class="banner banner-bad">
        <strong>${escapeHtml(title)}</strong> — both of these services are containers, so nothing here can start until that is fixed.
        ${d.detail ? `<div class="small">${escapeHtml(d.detail)}</div>` : ""}
        ${d.hint ? `<div class="small">${escapeHtml(d.hint)}</div>` : ""}
      </div>
    `;
  }

  function serviceCard(v: api.ContainerView): string {
    const warnings = v.warnings ?? [];
    return `
      <div class="card">
        <div class="service-head">
          <h2>${escapeHtml(v.label)}</h2>
          ${stateBadge(v)}
        </div>
        <p class="muted small">${escapeHtml(SERVICE_BLURB[v.id] ?? "")}</p>

        ${v.error ? errorBlock(v) : ""}
        ${v.blocked ? `<div class="banner banner-warn">${escapeHtml(v.blocked)}</div>` : ""}
        ${warnings.map((wmsg) => `<div class="banner banner-warn">${escapeHtml(wmsg)}</div>`).join("")}

        <dl class="stat-list">
          <div><dt>Container</dt><dd><code>${escapeHtml(v.containerName)}</code></dd></div>
          <div><dt>Image</dt><dd>${v.status.Image ? `<code>${escapeHtml(v.status.Image)}</code>` : "—"}</dd></div>
        </dl>
        ${exitCodeLine(v)}

        ${endpointsBlock(v)}

        <div class="card-actions">
          ${(v.actions ?? []).map((a) => actionButton(v, a)).join("")}
        </div>
        ${actionErr[v.id] ? `<p class="error small">${escapeHtml(actionErr[v.id]!)}</p>` : ""}
        ${activityBlock(v)}

        ${configSection(v)}
      </div>
    `;
  }

  function stateBadge(v: api.ContainerView): string {
    switch (v.status.State) {
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

  function errorBlock(v: api.ContainerView): string {
    return `
      <div class="banner banner-bad">
        <strong>This service could not be read.</strong>
        <div class="small">${escapeHtml(v.error ?? "")}</div>
        ${v.hint ? `<div class="small">${escapeHtml(v.hint)}</div>` : ""}
      </div>
    `;
  }

  // A non-zero exit code is shown only where it means something: on a
  // container that stopped. 137 is a kill (often the OOM killer), 0 is a
  // deliberate stop — and on a running container it is stale noise.
  function exitCodeLine(v: api.ContainerView): string {
    if (v.status.State !== "created-but-stopped" || v.status.ExitCode === 0) return "";
    const oom = v.status.ExitCode === 137 ? " (137 is a kill — most often the machine ran out of memory)" : "";
    return `<p class="muted small">It exited with code ${v.status.ExitCode}${oom}.</p>`;
  }

  // Endpoints are shown only while the service is running, and are the ports
  // the CONTAINER is actually published on — not the ones the saved config
  // asks for. A URL for a stopped service is an invitation to a connection
  // refused; a URL from an unapplied config is worse, because it looks right.
  function endpointsBlock(v: api.ContainerView): string {
    const eps = v.endpoints ?? [];
    if (eps.length === 0) {
      if (v.status.State === "running") {
        return `<p class="muted small">No endpoint to show — this gateway has no chains configured yet.</p>`;
      }
      return "";
    }
    return eps
      .map(
        (ep) => `
        <div class="endpoint-row">
          ${dot("ok")}
          <span class="muted small">${escapeHtml(ep.label)}</span>
          <code class="endpoint-url">${escapeHtml(ep.url)}</code>
          <button class="btn btn-ghost" data-action="copy" data-copy="${escapeHtml(ep.url)}">Copy</button>
        </div>`,
      )
      .join("");
  }

  function actionButton(v: api.ContainerView, action: string): string {
    const def = ACTION_BUTTONS[action];
    if (!def) return "";
    const running = busy[v.id];
    const label = action === "create" ? `Create ${v.id === "devnet" ? "devnet" : "gateway"}` : def.label;
    return `
      <button class="${def.className}" data-action="svc-${action}" data-svc="${escapeHtml(v.id)}"
              title="${escapeHtml(def.title)}" ${running ? "disabled" : ""}>
        ${running === action ? `<span class="spinner" aria-label="working"></span>` : escapeHtml(label)}
      </button>
    `;
  }

  function activityBlock(v: api.ContainerView): string {
    const lines = activity[v.id] ?? [];
    if (lines.length === 0) return "";
    return `
      <div class="config-block">
        <p class="muted small">Provisioning</p>
        <pre class="step-log">${escapeHtml(lines.join("\n"))}</pre>
      </div>
    `;
  }

  // --- configuration editors ----------------------------------------------

  function configSection(v: api.ContainerView): string {
    const isOpen = open[v.id];
    const summary = devnetSummary(v);
    return `
      <div class="config-block">
        <div class="service-head">
          <p class="muted small">Configuration${v.configured ? "" : " (nothing saved yet — these are the defaults)"}</p>
          <button class="btn btn-ghost" data-action="toggle-config" data-svc="${escapeHtml(v.id)}">
            ${isOpen ? "Close" : "Edit"}
          </button>
        </div>
        ${isOpen ? editorFor(v) : `<p class="small">${summary}</p>`}
        ${saveErr[v.id] ? `<p class="error small">${escapeHtml(saveErr[v.id]!)}</p>` : ""}
        ${saveNote[v.id] ? `<p class="muted small">${escapeHtml(saveNote[v.id]!)}</p>` : ""}
      </div>
    `;
  }

  function devnetSummary(v: api.ContainerView): string {
    const d = v.devnet;
    if (!d) return "—";
    return `Chain ${d.ChainID} · a block every ${escapeHtml(d.BlockTime)} · JSON-RPC on ${escapeHtml(d.BindAddr)}:${d.HTTPPort} · WebSocket on ${escapeHtml(d.BindAddr)}:${d.WSPort}`;
  }

  function editorFor(_v: api.ContainerView): string {
    return devnetForm();
  }

  function devnetForm(): string {
    const d = devnetDraft;
    if (!d) return "";
    return `
      <label>
        Block time <span class="muted">— how often the chain seals a block</span>
        <input type="text" id="dev-blocktime" value="${escapeHtml(d.BlockTime)}" autocomplete="off" spellcheck="false" />
      </label>
      <label>
        JSON-RPC port
        <input type="text" inputmode="numeric" id="dev-http" value="${d.HTTPPort}" autocomplete="off" />
      </label>
      <label>
        WebSocket port
        <input type="text" inputmode="numeric" id="dev-ws" value="${d.WSPort}" autocomplete="off" />
      </label>
      <label>
        Bind address <span class="muted">— 127.0.0.1 keeps it on this machine; 0.0.0.0 exposes it to your network</span>
        <input type="text" id="dev-bind" value="${escapeHtml(d.BindAddr)}" autocomplete="off" spellcheck="false" />
      </label>
      <p class="muted small">
        The chain id is fixed at ${d.ChainID}: reth's --dev genesis is baked into the image, and serving another id
        would need a custom genesis this app does not render.
      </p>
      <div class="card-actions">
        <button class="btn" data-action="save-config" data-svc="devnet">Save configuration</button>
      </div>
    `;
  }

  // readDrafts pulls whatever is currently typed into the open editors back
  // into the draft objects. Every structural edit calls it first, so adding
  // an upstream never discards the port you just changed.
  function readDrafts(): void {
    if (open.devnet && devnetDraft) {
      devnetDraft.BlockTime = text("#dev-blocktime", devnetDraft.BlockTime);
      devnetDraft.HTTPPort = int("#dev-http", devnetDraft.HTTPPort);
      devnetDraft.WSPort = int("#dev-ws", devnetDraft.WSPort);
      devnetDraft.BindAddr = text("#dev-bind", devnetDraft.BindAddr);
    }
  }

  function text(selector: string, fallback: string): string {
    const el = root.querySelector<HTMLInputElement>(selector);
    return el ? el.value.trim() : fallback;
  }

  function int(selector: string, fallback: number): number {
    const el = root.querySelector<HTMLInputElement>(selector);
    if (!el) return fallback;
    const n = Number.parseInt(el.value.trim(), 10);
    return Number.isFinite(n) ? n : fallback;
  }

  // --- actions ------------------------------------------------------------

  async function handleAction(action: string, el: HTMLElement): Promise<void> {
    const svc = (el.dataset.svc ?? "") as api.ContainerServiceID;
    switch (action) {
      case "refresh":
        await load();
        return;
      case "copy":
        if (el.dataset.copy) await copyButton(el, el.dataset.copy);
        return;
      case "svc-start":
      case "svc-stop":
      case "svc-restart":
        await runAction(svc, action.slice("svc-".length) as api.ContainerActionKind);
        return;
      case "svc-create":
      case "svc-recreate":
        await provision(svc);
        return;
      case "svc-wipe":
        openWipeModal(svc);
        return;
      case "toggle-config":
        toggleConfig(svc);
        return;
      case "save-config":
        await saveConfig(svc);
        return;
      default:
        return;
    }
  }

  async function runAction(svc: api.ContainerServiceID, kind: api.ContainerActionKind): Promise<void> {
    if (busy[svc]) return;
    busy[svc] = kind;
    actionErr[svc] = null;
    render();
    try {
      await api.containerAction(targetId, svc, kind);
    } catch (err) {
      actionErr[svc] = `${kind} failed: ${message(err)}${hintOf(err)}`;
    }
    busy[svc] = null;
    await load();
  }

  // provision runs the service's setup plan and follows the target's setup
  // event stream — the same one the node wizard uses. There is one run slot
  // per target, so a 409 here means something else is already provisioning
  // this machine, and saying so is more useful than a generic failure.
  async function provision(svc: api.ContainerServiceID): Promise<void> {
    if (busy[svc]) return;
    busy[svc] = "create";
    actionErr[svc] = null;
    activity[svc] = ["starting…"];
    render();

    try {
      await api.provisionContainer(targetId, svc);
    } catch (err) {
      actionErr[svc] = `${message(err)}${hintOf(err)}`;
      activity[svc] = [];
      busy[svc] = null;
      render();
      return;
    }

    streamStop?.();
    streamStop = api.streamSetup(targetId, (ev) => {
      if (disposed) return;
      const line = ev.err ? `${ev.stepId}: ${ev.err}` : ev.line ? `${ev.stepId}: ${ev.line}` : `${ev.stepId}: done`;
      activity[svc] = [...(activity[svc] ?? []).filter((l) => l !== "starting…"), line];
      const finished = !!ev.err || (ev.stepId === FINAL_STEP && !!ev.done);
      if (finished) {
        streamStop?.();
        streamStop = null;
        busy[svc] = null;
        if (ev.err) actionErr[svc] = "Provisioning failed — see the log below.";
        void load();
        return;
      }
      render();
    });
  }

  function toggleConfig(svc: api.ContainerServiceID): void {
    readDrafts();
    open[svc] = !open[svc];
    saveErr[svc] = null;
    saveNote[svc] = null;
    if (open[svc]) {
      const v = viewOf(svc);
      // Deep-copied from the view so an abandoned edit cannot leak into what
      // the cards display.
      if (v?.devnet) devnetDraft = { ...v.devnet };
    }
    render();
  }

  async function saveConfig(svc: api.ContainerServiceID): Promise<void> {
    readDrafts();
    saveErr[svc] = null;
    saveNote[svc] = null;

    const config = devnetDraft;
    if (!config) return;

    {
      // Checked here only to answer instantly; the server validates the same
      // things and its verdict is the one that counts.
      if (config.HTTPPort === config.WSPort) {
        saveErr[svc] = "JSON-RPC and WebSocket cannot share a port — docker would accept both mappings and then fail to start the container.";
        render();
        return;
      }
    }

    try {
      await api.putContainerConfig(targetId, svc, config);
    } catch (err) {
      saveErr[svc] = message(err);
      render();
      return;
    }

    const wasRunning = viewOf(svc)?.status.State === "running";
    open[svc] = false;
    saveNote[svc] = wasRunning
      ? "Saved. The running container still has the old settings — press “Re-create (apply config)” to put them into effect."
      : "Saved.";
    await load();
  }

  // --- wipe ---------------------------------------------------------------

  // The wipe modal says three things before it will accept a confirmation:
  // what is destroyed, what is restarted as a consequence, and why that
  // restart is not optional. The last one is the part that is invisible
  // everywhere else — see this file's header.
  function openWipeModal(svc: api.ContainerServiceID): void {
    const v = viewOf(svc);
    if (!v) return;
    const fronts = (v.restartsOnWipe ?? []).map((id) => viewOf(id)?.label ?? id);

    openModal(
      `
        <h2>Wipe ${escapeHtml(v.label)}</h2>
        <p class="error">This deletes ${escapeHtml(v.wipeDiscards)}</p>
        ${
          fronts.length
            ? `<p>It also restarts what sits in front of it: ${escapeHtml(fronts.join(", "))}.
                 That restart is required, not tidy-up: eRPC only ever moves a chain's head forward, so once this chain
                 restarts at block 0 the gateway would keep advertising the old head — answering for blocks the chain no
                 longer has — until the new chain grew past it.</p>`
            : ""
        }
        <p>Type <code>${escapeHtml(svc)}</code> to confirm.</p>
        <input type="text" id="wipe-confirm-input" autocomplete="off" spellcheck="false" />
        <div class="modal-actions">
          <button class="btn btn-ghost" data-modal-action="cancel">Cancel</button>
          <button class="btn btn-danger" data-modal-action="confirm" id="wipe-confirm-btn" disabled>Wipe ${escapeHtml(svc)}</button>
        </div>
      `,
      (action) => {
        if (action === "cancel" || action === "close") {
          closeModal();
          void load();
          return;
        }
        if (action === "confirm") void runWipe(svc);
      },
    );

    const input = document.getElementById("wipe-confirm-input") as HTMLInputElement | null;
    const btn = document.getElementById("wipe-confirm-btn") as HTMLButtonElement | null;
    input?.addEventListener("input", () => {
      if (btn) btn.disabled = input.value.trim() !== svc;
    });
    input?.focus();
  }

  async function runWipe(svc: api.ContainerServiceID): Promise<void> {
    const btn = document.getElementById("wipe-confirm-btn") as HTMLButtonElement | null;
    if (btn) {
      btn.disabled = true;
      btn.textContent = "Wiping…";
    }
    let result: api.WipeResult;
    try {
      result = await api.wipeContainer(targetId, svc);
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
        btn.textContent = `Wipe ${svc}`;
      }
      return;
    }
    showWipeResult(svc, result);
  }

  // showWipeResult replaces the modal with an account of what actually
  // happened. The cascade lines are the point: a restart nobody can see is
  // the same silent behaviour this whole feature exists to make visible.
  function showWipeResult(svc: api.ContainerServiceID, r: api.WipeResult): void {
    const v = viewOf(svc);
    const label = (id: string): string => viewOf(id)?.label ?? id;
    const lines: string[] = [];
    lines.push(r.report.ContainerRemoved ? "Container removed." : "There was no container to remove.");
    for (const vol of r.report.VolumesRemoved ?? []) lines.push(`Volume ${vol} deleted.`);
    for (const vol of r.report.VolumesAbsent ?? []) lines.push(`Volume ${vol} was already gone.`);
    if (r.report.Recreated) lines.push("Container re-created from your saved configuration.");

    const cascaded = (r.report.Cascaded ?? []).map(label);
    const skipped = (r.report.CascadeSkipped ?? []).map(label);

    openModal(
      `
        <h2>${escapeHtml(v?.label ?? svc)} wiped</h2>
        <ul class="plain-list">${lines.map((l) => `<li>${escapeHtml(l)}</li>`).join("")}</ul>
        ${
          cascaded.length
            ? `<p class="ok">Restarted in front of it: ${escapeHtml(cascaded.join(", "))} — its cached head was cleared, so it now reports this chain's real height rather than the one from before the wipe.</p>`
            : ""
        }
        ${
          skipped.length
            ? `<p class="muted small">Not restarted (they were not running, so they held no stale head): ${escapeHtml(skipped.join(", "))}.</p>`
            : ""
        }
        ${
          r.error
            ? `<p class="error">The wipe itself succeeded, but a service in front of it could not be restarted — it is now serving a head this chain no longer has. Restart it by hand.</p>
               <p class="error small">${escapeHtml(r.error)}</p>`
            : ""
        }
        <div class="modal-actions">
          <button class="btn" data-modal-action="close">Close</button>
        </div>
      `,
      (action) => {
        if (action === "close" || action === "cancel") {
          closeModal();
          void load();
        }
      },
    );
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
