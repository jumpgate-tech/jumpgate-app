// #/targets — the local-machine card, the list of SSH targets, and the
// "add server over SSH" form.
import * as api from "./api";
import { badge, confirmModal, escapeHtml, footer, onAction } from "./ui";

const LOCAL_TARGET_ID = "local";

export function renderTargets(root: HTMLElement): () => void {
  let disposed = false;
  // The SSH add form is revealed on demand (the "Add a server" button)
  // rather than always shown, so the empty state stays a clean set of
  // guiding actions. Held across re-renders.
  let showSSHForm = false;
  let hostOS = "";
  let lastData: { targets: api.Target[]; catalog: api.Catalog } | null = null;

  root.innerHTML = `
    <h1>Machines</h1>
    <div id="targets-body"><p class="muted">Loading…</p></div>
    ${footer()}
  `;
  const body = root.querySelector<HTMLElement>("#targets-body")!;

  onAction(root, (action, el) => {
    void handleAction(action, el);
  });

  load();

  async function load(): Promise<void> {
    try {
      const [targets, catalog, host] = await Promise.all([api.listTargets(), api.getCatalog(), api.getHost()]);
      if (disposed) return;
      hostOS = host.os;
      renderBody(targets, catalog);
    } catch (err) {
      if (disposed) return;
      body.innerHTML = `<p class="error">Failed to load machines: ${escapeHtml(String(err))}</p>`;
    }
  }

  function rerender(): void {
    if (lastData) renderBody(lastData.targets, lastData.catalog);
  }

  function renderBody(targets: api.Target[], catalog: api.Catalog): void {
    lastData = { targets, catalog };
    // Local node setup needs a Linux host — gated on the SERVER's OS (what
    // valve-node-app runs on), not the browser's. On a non-Linux host local
    // setup can't complete, but the option is still offered (secondary,
    // with a caveat) rather than hidden, so it's never a dead end.
    const localViable = hostOS === "linux";

    // One unified list, local target(s) first, then SSH servers.
    const ordered = [...targets].sort((a, b) => (a.mode === "local" ? -1 : 0) - (b.mode === "local" ? -1 : 0));
    const list = ordered.length
      ? `<div class="card-grid">${ordered.map((t) => targetCard(t, catalog)).join("")}</div>`
      : `<div class="card empty-state"><p class="muted">No machines yet — pick an option below.</p></div>`;

    body.innerHTML = `
      <section class="section">
        <div class="section-head"><h2>Your machines</h2></div>
        ${list}
      </section>
      <section class="section">
        <div class="section-head"><h2>Add a machine</h2></div>
        ${addOptions(localViable)}
        ${showSSHForm ? sshFormMarkup() : ""}
      </section>
    `;
  }

  // addOptions renders the ways to add a machine as options that each carry
  // their own availability and the reason behind it.
  //
  // This used to be two bare buttons whose caveat was repeated in a paragraph,
  // a tooltip, and a confirm() dialog — so the only place the constraint was
  // stated up front was a modal that interrupted you to say what the screen
  // already said, after you had committed to the action. Availability belongs
  // on the option itself, visible before you choose.
  function addOptions(localViable: boolean): string {
    const ssh = `
      <div class="card">
        <h3>A server over SSH ${badge("Available", "ok")}</h3>
        <p class="muted small">
          Run a node on a remote Linux server.${localViable ? "" : " The only option that can finish setup from here."}
        </p>
        <div class="card-actions">
          <button class="btn${localViable ? " btn-ghost" : ""}" data-action="toggle-ssh">
            ${showSSHForm ? "Cancel" : "Add a server"}
          </button>
        </div>
      </div>
    `;

    const local = localViable
      ? `
        <div class="card">
          <h3>This machine ${badge("Available", "ok")}</h3>
          <p class="muted small">Run a node here, on the Linux host valve-node-app is running on.</p>
          <div class="card-actions">
            <button class="btn" data-action="add-local">Add this machine</button>
          </div>
        </div>
      `
      : `
        <div class="card card-warn">
          <h3>This machine${hostOS ? ` (${escapeHtml(hostOS)})` : ""} ${badge("Can't run a node", "warn")}</h3>
          <p class="muted small">
            Setup installs systemd units, uses apt, and needs root, so it only completes on a
            Linux host. valve-node-app runs here as your <strong>controller</strong>, driving
            nodes on other machines.
          </p>
          <div class="card-actions">
            <button class="btn btn-ghost" data-action="add-local">Add anyway — preview the wizard</button>
          </div>
        </div>
      `;

    // Lead with whichever option can actually complete setup.
    return `<div class="card-grid">${localViable ? local + ssh : ssh + local}</div>`;
  }

  async function handleAction(action: string, el: HTMLElement): Promise<void> {
    if (action === "add-local") {
      await addLocal();
      return;
    }
    if (action === "delete-target") {
      const id = el.dataset.id;
      if (!id) return;
      const ok = await confirmModal({
        title: "Remove machine",
        body: `Remove "${id}"? This only removes it from valve-node-app — anything already running on the machine keeps running, and its data is left alone.`,
        confirmLabel: "Remove",
        danger: true,
      });
      if (!ok) return;
      await deleteTarget(id);
      return;
    }
    if (action === "toggle-ssh") {
      showSSHForm = !showSSHForm;
      clearFormError();
      rerender();
      if (showSSHForm) root.querySelector<HTMLInputElement>("#ssh-host")?.focus();
      return;
    }
    if (action === "add-ssh") {
      await addSSH();
    }
  }

  // No confirm() here: the option card states the constraint before you
  // click, and its button is explicitly labelled "Add anyway — preview the
  // wizard", so a modal would only interrupt to repeat what you just read.
  async function addLocal(): Promise<void> {
    clearFormError();
    try {
      await api.addTarget({ id: LOCAL_TARGET_ID, mode: "local" });
      await load();
    } catch (err) {
      showFormError(err);
    }
  }

  async function deleteTarget(id: string): Promise<void> {
    try {
      await api.deleteTarget(id);
      await load();
    } catch (err) {
      showFormError(err);
    }
  }

  async function addSSH(): Promise<void> {
    const hostEl = root.querySelector<HTMLInputElement>("#ssh-host");
    const userEl = root.querySelector<HTMLInputElement>("#ssh-user");
    const keyEl = root.querySelector<HTMLInputElement>("#ssh-key");
    const portEl = root.querySelector<HTMLInputElement>("#ssh-port");
    const idEl = root.querySelector<HTMLInputElement>("#ssh-id");
    if (!hostEl || !userEl || !keyEl || !portEl || !idEl) return;

    const host = hostEl.value.trim();
    const user = userEl.value.trim();
    const keyPath = keyEl.value.trim();
    const portRaw = portEl.value.trim();
    const idRaw = idEl.value.trim();

    clearFormError();
    if (!host || !user || !keyPath) {
      showFormError(new Error("host, user, and key path are required"));
      return;
    }

    const id = idRaw || slugify(host);
    const ssh: api.SSHConfig = { Host: host, User: user, KeyPath: keyPath };
    if (portRaw) {
      const port = Number.parseInt(portRaw, 10);
      if (!Number.isFinite(port) || port <= 0) {
        showFormError(new Error("port must be a positive number"));
        return;
      }
      ssh.Port = port;
    }

    const submitBtn = root.querySelector<HTMLButtonElement>("#ssh-submit");
    if (submitBtn) {
      submitBtn.disabled = true;
      submitBtn.textContent = "Connecting…";
    }
    try {
      await api.addTarget({ id, mode: "ssh", ssh });
      showSSHForm = false;
      await load();
    } catch (err) {
      showFormError(err);
      if (submitBtn) {
        submitBtn.disabled = false;
        submitBtn.textContent = "Add server";
      }
    }
  }

  function showFormError(err: unknown): void {
    let box = root.querySelector<HTMLElement>("#targets-error");
    if (!box) {
      body.insertAdjacentHTML("afterbegin", `<p id="targets-error" class="error"></p>`);
      box = root.querySelector<HTMLElement>("#targets-error")!;
    }
    box.textContent = String(err instanceof Error ? err.message : err);
  }

  function clearFormError(): void {
    root.querySelector<HTMLElement>("#targets-error")?.remove();
  }

  return () => {
    disposed = true;
  };
}

function targetCard(t: api.Target, catalog: api.Catalog): string {
  const wire = t.wire;
  const modeLabel = t.mode === "local" ? "this machine" : "SSH";
  const location = t.mode === "ssh" && t.ssh ? `${escapeHtml(t.ssh.User)}@${escapeHtml(t.ssh.Host)}` : modeLabel;

  let statusLine: string;
  let actions: string;
  if (!wire) {
    statusLine = badge("not set up", "neutral");
    actions = `<a class="btn" href="#/setup/${encodeURIComponent(t.id)}">Run setup wizard</a>`;
  } else {
    const net = catalog.networks.find((n) => n.ChainID === wire.ChainID);
    const netName = net ? net.Name : `chain ${wire.ChainID}`;
    statusLine = `${badge(netName, "ok")} ${badge(wire.ExecID, "neutral")} ${badge(wire.BeaconID, "neutral")}${wire.Archive ? " " + badge("archive", "warn") : ""}`;
    actions = `
      <a class="btn" href="#/dash/${encodeURIComponent(t.id)}">Dashboard</a>
      <a class="btn" href="#/logs/${encodeURIComponent(t.id)}">Logs</a>
      <a class="btn btn-ghost" href="#/setup/${encodeURIComponent(t.id)}">Re-run setup</a>
    `;
  }

  return `
    <div class="card">
      <h2>${escapeHtml(t.id)}</h2>
      <p class="muted">${location}</p>
      <p>${statusLine}</p>
      <div class="card-actions">
        ${actions}
        <button class="btn btn-danger" data-action="delete-target" data-id="${escapeHtml(t.id)}">Remove</button>
      </div>
    </div>
  `;
}

function sshFormMarkup(): string {
  return `
    <form class="card" id="ssh-add-form" onsubmit="return false">
      <h3>Add server over SSH</h3>
      <label>
        Host
        <input id="ssh-host" type="text" placeholder="203.0.113.10" autocomplete="off" />
      </label>
      <label>
        User
        <input id="ssh-user" type="text" placeholder="root" autocomplete="off" />
      </label>
      <label>
        Private key path
        <input id="ssh-key" type="text" placeholder="/home/me/.ssh/id_ed25519" autocomplete="off" />
      </label>
      <label>
        Port <span class="muted">(optional, default 22)</span>
        <input id="ssh-port" type="text" inputmode="numeric" placeholder="22" autocomplete="off" />
      </label>
      <label>
        Target name <span class="muted">(optional, defaults to the host)</span>
        <input id="ssh-id" type="text" placeholder="my-node" autocomplete="off" />
      </label>
      <p class="muted small">
        The key never leaves this machine — only its path is stored, and the
        connection is dialed immediately so the host key can be pinned
        (trust-on-first-use) before it's saved.
      </p>
      <button class="btn" type="button" id="ssh-submit" data-action="add-ssh">Add server</button>
    </form>
  `;
}

function slugify(s: string): string {
  return s.toLowerCase().replace(/[^a-z0-9]+/g, "-").replace(/^-+|-+$/g, "") || "target";
}
