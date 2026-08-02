// Pure derivations for the Services section — kept apart from
// ServicesSection.tsx so they're testable without rendering. Mirrors
// services.ts's own constants and inline helpers: the service blurb, the
// action-button presentation, the state badge, the exit-code line, the
// devnet config summary/validation, the docker-banner text, the
// no-endpoints message, and the wipe typed-confirm gate.
import type { ContainerServiceID, ContainerView, DevnetConfig, DockerView } from "../../api";

// The step id both container plans end with (PlanDevnet: preflight, run;
// PlanGateway: preflight, config, run). The setup event stream has no
// terminal frame, so this is what tells a provisioning run's progress log
// that it is finished and the cards can be re-read. Mirrors services.ts's
// own FINAL_STEP.
export const FINAL_STEP = "run";

export const SERVICE_BLURB: Record<ContainerServiceID, string> = {
  devnet:
    "A throwaway chain that runs entirely on this machine: reth in --dev mode, sealing a block on a timer from its own genesis. Nothing to sync, nothing on disk outside the container.",
};

export interface ActionButtonDef {
  label: string;
  title: string;
  className: string;
}

// ACTION_BUTTONS is the presentation of the server's action ids. Labels say
// what will happen, not what the verb is called internally: "recreate" is
// destructive-ish (the container is replaced) and its label has to admit
// that, because it is the only way an edited port or block time ever takes
// effect.
export const ACTION_BUTTONS: Record<string, ActionButtonDef> = {
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

// actionLabel mirrors actionButton's own label override: "create" reads as
// "Create devnet"/"Create gateway" rather than the bare "Create" in
// ACTION_BUTTONS, so the button says what it creates.
export function actionLabel(svc: ContainerServiceID, action: string): string {
  const def = ACTION_BUTTONS[action];
  if (!def) return "";
  if (action === "create") return `Create ${svc === "devnet" ? "devnet" : "gateway"}`;
  return def.label;
}

export type BadgeKind = "ok" | "bad" | "warn" | "neutral";

export interface StatusBadge {
  text: string;
  kind: BadgeKind;
}

// stateBadge mirrors serviceCard's own stateBadge switch.
export function stateBadge(v: ContainerView): StatusBadge {
  switch (v.status.State) {
    case "running":
      return { text: "running", kind: "ok" };
    case "created-but-stopped":
      return { text: "stopped", kind: "warn" };
    case "not-created":
      return { text: "not created", kind: "neutral" };
    default:
      return { text: "unknown", kind: "bad" };
  }
}

// exitCodeLine mirrors services.ts's own function of the same name: a
// non-zero exit code is shown only where it means something — on a
// container that stopped. 137 is a kill (often the OOM killer), 0 is a
// deliberate stop, and on a running container it is stale noise. Returns
// null when nothing should be shown.
export function exitCodeLine(v: ContainerView): string | null {
  if (v.status.State !== "created-but-stopped" || v.status.ExitCode === 0) return null;
  const oom = v.status.ExitCode === 137 ? " (137 is a kill — most often the machine ran out of memory)" : "";
  return `It exited with code ${v.status.ExitCode}${oom}.`;
}

// showNoEndpointsMessage mirrors endpointsBlock's empty-but-running case:
// eps.length === 0 shows a message only while running; while stopped/not
// created it shows nothing at all.
export function showNoEndpointsMessage(v: ContainerView): boolean {
  const eps = v.endpoints ?? [];
  return eps.length === 0 && v.status.State === "running";
}

// devnetSummary mirrors services.ts's own devnetSummary — the collapsed
// configuration line shown when the editor is closed.
export function devnetSummary(v: ContainerView): string {
  const d = v.devnet;
  if (!d) return "—";
  return `Chain ${d.ChainID} · a block every ${d.BlockTime} · JSON-RPC on ${d.BindAddr}:${d.HTTPPort} · WebSocket on ${d.BindAddr}:${d.WSPort}`;
}

// devnetConfigError mirrors saveConfig's own client-side check: it exists
// only to answer instantly, and the server validates the same thing — its
// verdict is the one that counts.
export function devnetConfigError(config: DevnetConfig): string | null {
  if (config.HTTPPort === config.WSPort) {
    return "JSON-RPC and WebSocket cannot share a port — docker would accept both mappings and then fail to start the container.";
  }
  return null;
}

// wipeConfirmValid mirrors openWipeModal's typed-confirm gate: the input
// must equal the service id exactly (after trimming surrounding whitespace).
export function wipeConfirmValid(input: string, svc: string): boolean {
  return input.trim() === svc;
}

// dockerOk / dockerBannerTitle mirror dockerBanner's own branching: both
// services depend on a single reachable docker engine, so a failure is
// reported once rather than separately per card.
export function dockerOk(d: DockerView): boolean {
  return d.present && d.reachable && !d.hint;
}

export function dockerBannerTitle(d: DockerView): string {
  return !d.present ? "No docker engine on this machine" : "Docker is installed, but no engine answered";
}

// parseIntOr mirrors services.ts's own int() helper used to read a draft
// port field back from an input: an unparseable/non-finite value keeps the
// previous value rather than corrupting the draft.
export function parseIntOr(value: string, fallback: number): number {
  const n = Number.parseInt(value.trim(), 10);
  return Number.isFinite(n) ? n : fallback;
}

// provisionLine mirrors provision()'s own per-event line formatting inside
// its streamSetup callback.
export function provisionLine(ev: { stepId: string; line?: string; err?: string }): string {
  return ev.err ? `${ev.stepId}: ${ev.err}` : ev.line ? `${ev.stepId}: ${ev.line}` : `${ev.stepId}: done`;
}

// provisionFinished mirrors provision()'s own finish condition: an error at
// any step, or the plan's final step reporting done.
export function provisionFinished(ev: { stepId: string; done?: boolean; err?: string }): boolean {
  return !!ev.err || (ev.stepId === FINAL_STEP && !!ev.done);
}
