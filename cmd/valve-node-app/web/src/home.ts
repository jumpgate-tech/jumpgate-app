// The capability-detected landing (the empty-hash / #/home route).
//
// The app used to open on the machines view — the node-operator flow — for
// everyone, including GUI users on macOS/Windows who can NEVER run a node
// (setup installs systemd units, uses apt and needs root, so it only completes
// on a Linux host). For them the only usable path is eRPC: a managed RPC
// endpoint that aggregates Valve (via the shared vk_demo key) and the chain's
// known-set public upstreams with automatic failover, needing NO node. This
// screen decides which of those two audiences is in front of us and opens on
// the surface that audience can actually act on.
//
// It owns the DECISION, not the screens: a node-capable fleet is sent to the
// existing machines view (targets.ts) and a non-capable one gets an eRPC-first
// hero that links into the existing eRPC screen (rpc.ts) and, secondarily, the
// SSH-add flow. Neither of those screens is rebuilt here.
import * as api from "./api";
import { badge, escapeHtml, footer } from "./ui";
import { computeFleetVerdict, renderVerdictLine } from "./verdict";

// isNodeCapableFleet is the whole capability signal, pure so it can be reasoned
// about (and unit-tested) on its own: (targets, hostOS) → boolean.
//
// A single MACHINE can run a node when it is an SSH target (a remote Linux
// server by construction) or a local target on a Linux host — local setup only
// completes where valve-node-app itself runs on Linux. The FLEET is
// node-capable when at least one such machine already exists, OR the host OS is
// Linux (so even with no machines yet, local setup is still possible here).
//
// The local-target case is written out even though `hostOS === "linux"` already
// implies it, because the rule the app reasons about is per-machine and stating
// it here is what keeps this function readable against that rule rather than a
// simplified equivalent.
export function isNodeCapableFleet(targets: api.Target[], hostOS: string): boolean {
  const hostIsLinux = hostOS === "linux";
  const hasNodeCapableMachine = targets.some(
    (t) => t.mode === "ssh" || (t.mode === "local" && hostIsLinux),
  );
  return hasNodeCapableMachine || hostIsLinux;
}

export function renderHome(root: HTMLElement): () => void {
  let disposed = false;

  root.innerHTML = `<div id="home-body"><p class="muted">Loading…</p></div>`;
  const body = root.querySelector<HTMLElement>("#home-body")!;

  void load();

  async function load(): Promise<void> {
    let targets: api.Target[];
    let catalog: api.Catalog;
    let hostOS: string;
    try {
      const [t, c, host] = await Promise.all([api.listTargets(), api.getCatalog(), api.getHost()]);
      targets = t;
      catalog = c;
      hostOS = host.os;
    } catch (err) {
      if (disposed) return;
      body.innerHTML = `<p class="error">Failed to load: ${escapeHtml(String(err))}</p>`;
      return;
    }
    if (disposed) return;

    // A node-capable fleet keeps the machines view as its home — the operator
    // flow is the right one for them, and it already carries the fleet verdict
    // and the add-a-machine options. Redirecting (rather than composing the
    // view here) keeps one home per audience and lights the correct nav item.
    if (isNodeCapableFleet(targets, hostOS)) {
      location.hash = "#/targets";
      return;
    }

    renderERPCFirst(targets, catalog);
  }

  // renderERPCFirst is the landing for a controller that cannot run a node:
  // eRPC is the hero and the only path offered up front, with running-your-own
  // demoted to a clearly-labelled secondary card. The client / snapshot /
  // checkpoint questions of the node wizard are deliberately unreachable from
  // here — the only node link is the SSH-add flow on #/targets, never the
  // wizard — so a user who cannot act on those questions is never asked them.
  function renderERPCFirst(targets: api.Target[], catalog: api.Catalog): void {
    body.innerHTML = `
      <h1>valve-node-app</h1>
      <div id="fleet-verdict"></div>
      <section class="section">
        <div class="section-head"><h2>Your RPC endpoint</h2></div>
        <div class="card hero-card">
          <h3>Get an RPC endpoint — no node required ${badge("recommended", "ok")}</h3>
          <p class="muted">
            eRPC is a managed endpoint that aggregates Valve — via the shared
            <code>vk_demo</code> key — and the chain's known-set public upstreams behind one
            URL, with automatic failover between them. It runs as a container here; you never
            run, sync or babysit a node.
          </p>
          <div class="card-actions">
            <a class="btn btn-primary" href="#/rpc">Set up my endpoint →</a>
          </div>
        </div>
      </section>
      <section class="section">
        <div class="section-head"><h2>Run your own node</h2></div>
        <div class="card card-warn">
          <h3>Run your own node ${badge("needs a Linux server", "warn")}</h3>
          <p class="muted small">
            Node setup installs systemd units, uses apt and needs root, so it only completes on
            a Linux server — not on this machine. Add one over SSH and valve-node-app will drive
            the node on it from here.
          </p>
          <div class="card-actions">
            <a class="btn btn-ghost" href="#/targets">Add a Linux server →</a>
          </div>
        </div>
      </section>
      ${footer()}
    `;

    // The fleet verdict stays at the top of whichever screen is home — the app
    // opens saying what needs attention or that nothing does, here just as on
    // the machines view.
    const verdictEl = body.querySelector<HTMLElement>("#fleet-verdict");
    if (verdictEl) renderVerdictLine(verdictEl, computeFleetVerdict(targets, catalog));
  }

  return () => {
    disposed = true;
  };
}
