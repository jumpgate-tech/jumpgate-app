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
import { badge, escapeHtml, footer, onAction } from "./ui";
import { computeFleetVerdict, renderVerdictLine } from "./verdict";

// FINAL_STEP is the id every gateway provision plan ends on (preflight, config,
// run) — the same constant rpc.ts and services.ts follow their setup streams by.
const FINAL_STEP = "run";

// SETUP_CHAINS are the chains the zero-machine one-click endpoint comes up
// serving: Ethereum and PulseChain, by valve's own measured known set. The
// devnet (1337) is deliberately absent — this flow builds a real public
// endpoint, not a scratch chain.
export const SETUP_CHAINS = [
  { chainId: 1, name: "Ethereum" },
  { chainId: 369, name: "PulseChain" },
];

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

  let setupBusy = false;
  onAction(root, (action) => {
    if (action === "setup-endpoint") void setupEndpoint();
  });

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
            <button class="btn btn-primary" data-action="setup-endpoint">Set up my endpoint →</button>
          </div>
          <div id="setup-progress" aria-live="polite"></div>
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

  // setupEndpoint is the zero-machine one-click: it turns "Set up my endpoint"
  // into a live eRPC gateway on this machine, so a user with no machine and no
  // node lands on a working URL rather than the RPC screen's "add a machine
  // first" dead end.
  //
  // It reuses the ordinary building blocks rather than a special path —
  // addTarget, createGateway, the known-set route, provisionGateway and the
  // shared setup stream — so nothing here is a second way to provision a
  // gateway that could drift from the one the RPC screen uses. Every step that
  // can fail (no Docker, a create refused, a stream that errors) surfaces its
  // reason rather than silently doing nothing.
  async function setupEndpoint(): Promise<void> {
    if (setupBusy) return;
    setupBusy = true;

    const btn = body.querySelector<HTMLButtonElement>('[data-action="setup-endpoint"]');
    const prog = body.querySelector<HTMLElement>("#setup-progress");
    if (btn) btn.disabled = true;
    const say = (html: string): void => {
      if (prog) prog.innerHTML = html;
    };
    const working = (text: string): void =>
      say(`<p class="muted small"><span class="spinner" aria-label="working"></span> ${escapeHtml(text)}</p>`);
    const fail = (msg: string, hint?: string): void => {
      setupBusy = false;
      if (btn) btn.disabled = false;
      say(`<p class="error small">${escapeHtml(msg)}${hint ? ` — ${escapeHtml(hint)}` : ""}</p>`);
    };

    working("Preparing your endpoint…");

    // 1. Register this machine as a local target if it is not already one.
    try {
      const targets = await api.listTargets();
      if (!targets.some((t) => t.id === "local")) {
        await api.addTarget({ id: "local", mode: "local" });
      }
    } catch (err) {
      fail(`Could not register this machine: ${errMsg(err)}`, errHint(err));
      return;
    }
    if (disposed) return;

    // 2. A gateway is a container, so this needs a reachable Docker engine.
    //    This is the capability check the flow degrades on — clearly, with the
    //    engine's own hint, rather than failing later mid-provision.
    try {
      const c = await api.getContainers("local");
      if (!c.docker.reachable) {
        fail(
          c.docker.detail || "A gateway runs as a container, and no Docker engine answered on this machine.",
          c.docker.hint || "Start Docker Desktop, OrbStack or colima, then try again.",
        );
        return;
      }
    } catch (err) {
      fail(`Could not check Docker on this machine: ${errMsg(err)}`, errHint(err));
      return;
    }
    if (disposed) return;

    // 3. If this machine already runs a gateway, there is nothing to create —
    //    go straight to it.
    let gid = "default";
    try {
      const mine = ((await api.getGateways()).gateways ?? []).find((g) => g.placement?.targetId === "local");
      if (mine) {
        location.hash = "#/rpc";
        return;
      }
    } catch {
      // Could not list — fall through and let createGateway be the arbiter.
    }
    if (disposed) return;

    // 4. Create the gateway, fronted by the internal-CA HTTPS so it comes up on
    //    the localhost-valaxy URL — consistent with turning HTTPS on by hand.
    //    The server fills the hostname; HTTPSPort 0 means 443.
    working("Creating the gateway…");
    try {
      const created = await api.createGateway({
        id: gid,
        placement: { targetId: "local", backend: "docker" },
        config: internalTLSConfig([]),
      });
      gid = created.id;
    } catch (err) {
      fail(`Could not create the gateway: ${errMsg(err)}`, errHint(err));
      return;
    }
    if (disposed) return;

    // 5. Add valve's known set for each chain via the known-set route — the
    //    same vetted, measured set the RPC screen's "Add valve's set…" adds.
    working("Adding Ethereum and PulseChain endpoints…");
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
      } catch (err) {
        fail(`Could not read valve's set for chain ${chainId}: ${errMsg(err)}`, errHint(err));
        return;
      }
    }
    if (disposed) return;
    if (networks.length === 0) {
      fail("valve has no measured endpoints for Ethereum or PulseChain right now, so there was nothing to add.");
      return;
    }
    try {
      await api.putGatewayConfig(gid, internalTLSConfig(networks));
    } catch (err) {
      fail(`Could not save the endpoints: ${errMsg(err)}`, errHint(err));
      return;
    }
    if (disposed) return;

    // 6. Provision the container, following the placement machine's setup
    //    stream exactly as the RPC screen's create does. On success, land on
    //    #/rpc where the URLs are shown.
    working("Starting the gateway… the first run pulls the eRPC and Caddy images.");
    let started: { targetId: string };
    try {
      started = await api.provisionGateway(gid);
    } catch (err) {
      fail(`Could not start the gateway: ${errMsg(err)}`, errHint(err));
      return;
    }
    const stop = api.streamSetup(started.targetId, (ev) => {
      if (disposed) {
        stop();
        return;
      }
      const line = ev.err
        ? `${ev.stepId}: ${ev.err}`
        : ev.line
          ? `${ev.stepId}: ${ev.line}`
          : `${ev.stepId}: done`;
      working(line);
      const finished = !!ev.err || (ev.stepId === FINAL_STEP && !!ev.done);
      if (!finished) return;
      stop();
      if (ev.err) {
        // The gateway exists and is configured; the log lives on #/rpc, so send
        // the operator there to read it and retry rather than trapping them here.
        fail("The gateway was created but did not start — open RPC to see the log and retry.");
        setTimeout(() => {
          if (!disposed) location.hash = "#/rpc";
        }, 1500);
        return;
      }
      location.hash = "#/rpc";
    });
  }

  return () => {
    disposed = true;
  };
}

// internalTLSConfig is the gateway config the one-click setup flow creates and
// updates: an eRPC on 4000 fronted by Caddy's internal CA (HTTPSPort 0 → 443).
// Hostname is left empty on purpose — the server fills a name whose wildcard
// already resolves to loopback. Exported so the Easy-Button panel's own
// one-click setup (panel.ts) builds the exact same config rather than a
// second, driftable version of it.
export function internalTLSConfig(networks: api.GatewayNetwork[]): api.GatewayConfig {
  return {
    ProjectID: "main",
    BindAddr: "127.0.0.1",
    Port: 4000,
    Networks: networks,
    TLS: {
      Enabled: true,
      Hostname: "",
      CertSource: "internal",
      CertFile: "",
      KeyFile: "",
      HTTPSPort: 0,
      BindAddr: "",
      ImageRef: "",
    },
  };
}

function errMsg(err: unknown): string {
  return err instanceof Error ? err.message : String(err);
}

// errHint surfaces the server's operator-facing hint verbatim — it is written
// for exactly this moment (e.g. "start Docker Desktop / OrbStack / colima").
function errHint(err: unknown): string | undefined {
  return err instanceof api.ApiError ? err.hint : undefined;
}
