// #/machine/<id> — one page per machine. The four routes that used to be one
// screen each (#/setup, #/dash, #/logs, #/services) are now SECTIONS of this
// page: a one-line status that expands to the full detail on click. No tabs.
//
// This is deliberately subtraction, not a rewrite. Each section's detail is
// the EXISTING screen module (renderWizard / renderDashboard / renderLogs /
// renderServices), mounted unchanged into the section's expanded container the
// first time it is opened. Those modules own their own EventSource/timers and
// hand back a cleanup; this page keeps every mounted section's cleanup and
// calls them all when it unmounts, so navigating away closes their streams.
//
// The reused screens still render their own <h1>/footer inside a section —
// stripping those so the page reads as one document is a follow-up, called out
// in the plan, not this increment.
import * as api from "./api";
import { renderDashboard } from "./dashboard";
import { renderLogs } from "./logs";
import { renderServices } from "./services";
import { renderWizard } from "./wizard";
import { badge, escapeHtml, footer, onAction } from "./ui";

type Cleanup = () => void;

interface SectionDef {
  key: string;
  title: string;
  // status is the one-line summary shown on the collapsed row. It is
  // BEST-EFFORT from data already in hand (the target's wire config + the
  // catalog) — never a deep measured verdict. Whether a devnet container is
  // actually running, whether the node is synced, how many errors are in the
  // feed: those need live probes/streams and are a follow-up (the fleet
  // verdict stream and per-section verdicts), not this collapse.
  status: (t: api.Target, catalog: api.Catalog) => string;
  mount: (root: HTMLElement) => Cleanup;
}

export function renderMachine(root: HTMLElement, id: string): () => void {
  let disposed = false;

  // Each section is mounted LAZILY — its screen (and its stream) only starts
  // the first time the operator expands it. Once mounted we keep it mounted:
  // toggling the row just hides/shows the container. That is the simpler
  // correct option — no re-mount races, and log/scroll state survives a
  // collapse. The only cost is a background SSE stream per opened section,
  // and every one of them is torn down together in this page's cleanup below.
  const mounted = new Map<string, Cleanup>();

  root.innerHTML = `<h1>${escapeHtml(id)}</h1><div id="machine-body"><p class="muted">Loading…</p></div>`;
  const body = root.querySelector<HTMLElement>("#machine-body")!;

  onAction(root, (action, el) => {
    if (action === "toggle-section") toggleSection(el.dataset.section ?? "");
  });

  void load();

  async function load(): Promise<void> {
    let target: api.Target | undefined;
    let catalog: api.Catalog | undefined;
    try {
      const [targets, cat] = await Promise.all([api.listTargets(), api.getCatalog()]);
      target = targets.find((t) => t.id === id);
      catalog = cat;
    } catch (err) {
      if (disposed) return;
      body.innerHTML = `<p class="error">Failed to load machine: ${escapeHtml(String(err))}</p>`;
      return;
    }
    if (disposed) return;

    // Not a machine we know — the id in the hash is stale (removed, or a bad
    // deep link). Bounce to the list rather than render an empty page.
    if (!target) {
      location.hash = "#/targets";
      return;
    }

    renderShell(target, catalog!);
  }

  function renderShell(t: api.Target, catalog: api.Catalog): void {
    const modeLabel = t.mode === "local" ? "this machine" : "SSH";
    const location =
      t.mode === "ssh" && t.ssh ? `${escapeHtml(t.ssh.User)}@${escapeHtml(t.ssh.Host)}` : modeLabel;

    body.innerHTML = `
      <p class="muted">${location}</p>
      <p>${headerStatus(t, catalog)}</p>
      <div class="machine-sections">
        ${SECTIONS.map((s) => sectionRow(s, t, catalog)).join("")}
      </div>
      ${footer()}
    `;
  }

  // headerStatus mirrors targets.ts's card status line: the network + client
  // badges once set up, or "not set up" before. Same best-effort caveat as the
  // section statuses — it is the wire config, not a live health check.
  function headerStatus(t: api.Target, catalog: api.Catalog): string {
    const wire = t.wire;
    if (!wire) return badge("not set up", "neutral");
    const net = catalog.networks.find((n) => n.ChainID === wire.ChainID);
    const netName = net ? net.Name : `chain ${wire.ChainID}`;
    return `${badge(netName, "ok")} ${badge(wire.ExecID, "neutral")} ${badge(wire.BeaconID, "neutral")}${
      wire.Archive ? " " + badge("archive", "warn") : ""
    }`;
  }

  function sectionRow(s: SectionDef, t: api.Target, catalog: api.Catalog): string {
    return `
      <section class="card machine-section" data-section-card="${escapeHtml(s.key)}">
        <button type="button" class="machine-section-head" data-action="toggle-section"
                data-section="${escapeHtml(s.key)}" aria-expanded="false">
          <span class="machine-section-title">${escapeHtml(s.title)}</span>
          <span class="machine-section-status">${s.status(t, catalog)}</span>
          <span class="machine-section-caret" aria-hidden="true">▸</span>
        </button>
        <div class="machine-section-body" data-section-body="${escapeHtml(s.key)}" hidden></div>
      </section>
    `;
  }

  function toggleSection(key: string): void {
    const def = SECTIONS.find((s) => s.key === key);
    if (!def) return;
    const card = root.querySelector<HTMLElement>(`[data-section-card="${key}"]`);
    const bodyEl = root.querySelector<HTMLElement>(`[data-section-body="${key}"]`);
    const head = root.querySelector<HTMLElement>(`.machine-section-head[data-section="${key}"]`);
    if (!card || !bodyEl || !head) return;

    const willOpen = bodyEl.hidden;
    if (willOpen && !mounted.has(key)) {
      // A fresh child per section so the reused screen's delegated listeners
      // (attached to the node it is handed) are discarded with it on cleanup,
      // never stacked on this page's body — the same reasoning as main.ts.
      const child = document.createElement("div");
      bodyEl.appendChild(child);
      mounted.set(key, def.mount(child));
    }
    bodyEl.hidden = !willOpen;
    card.classList.toggle("open", willOpen);
    head.setAttribute("aria-expanded", String(willOpen));
  }

  // SECTIONS is the collapse itself: the four former routes, in the order the
  // direction doc puts them (set up, then watch, then read, then the devnet a
  // machine can always host).
  const SECTIONS: SectionDef[] = [
    {
      key: "setup",
      title: "Setup",
      status: (t) => (t.wire ? badge("set up", "ok") : badge("not set up", "neutral")),
      mount: (el) => renderWizard(el, id),
    },
    {
      key: "dashboard",
      title: "Dashboard",
      status: (t) =>
        t.wire
          ? `<span class="muted small">sync, peers, storage and endpoints — live</span>`
          : `<span class="muted small">available once this machine is set up</span>`,
      mount: (el) => renderDashboard(el, id),
    },
    {
      key: "logs",
      title: "Logs",
      status: (t) =>
        t.wire
          ? `<span class="muted small">live tail and error feed</span>`
          : `<span class="muted small">available once this machine is set up</span>`,
      mount: (el) => renderLogs(el, id),
    },
    {
      key: "services",
      title: "Devnet",
      // Offered on EVERY machine, set up or not — a devnet is a container, so
      // unlike a node it needs no Linux host and no root. This is the same
      // reasoning targets.ts used to justify the always-present Devnet link;
      // it now lives on the machine page instead.
      status: () => `<span class="muted small">throwaway chain — always available on this machine</span>`,
      mount: (el) => renderServices(el, id),
    },
  ];

  return () => {
    disposed = true;
    // Tear down every section we mounted — they own the EventSource/timers.
    for (const cleanup of mounted.values()) {
      try {
        cleanup();
      } catch {
        // One section's cleanup failing must not block the others.
      }
    }
    mounted.clear();
  };
}
