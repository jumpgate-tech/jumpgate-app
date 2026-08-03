// Pure derivations for the Setup wizard section — kept apart from
// SetupWizard.tsx (and its step components) so the fit-check math, the step
// plan and the request-building logic are testable without rendering.
// Mirrors wizard.ts's own module-scope constants and inline helper
// functions; every constant below is copied byte-exact from wizard.ts, not
// re-derived, so a change there has to be made here too (see wizard.ts's own
// comments on FIT_MARGIN/FULL_TIER_FRACTION for why they can't just be read
// from the API).
import type { Catalog, Network, SetupEvent, StartSetupRequest } from "../../api";

// ---------------------------------------------------------------------
// wizard steps
// ---------------------------------------------------------------------

export type WizardStep = "network" | "clients" | "mode" | "review" | "run";

// STEP_PLAN mirrors internal/setup/steps.go's Plan() — a fixed sequence
// regardless of which clients are chosen (only the titles vary slightly,
// which we don't try to reproduce; the real titles come from the SSE
// stream's stepId once the run starts).
export const STEP_PLAN: { id: string; title: string }[] = [
  { id: "preflight", title: "Preflight checks" },
  { id: "toolchain", title: "Ensure git + build toolchains" },
  { id: "install-exec", title: "Install execution client" },
  { id: "install-beacon", title: "Install beacon client" },
  { id: "wire", title: "Write JWT secret and systemd units" },
  { id: "start", title: "Start execution and beacon services" },
  { id: "handshake", title: "Verify execution/beacon handshake" },
];

// WIZARD_STEPS drives the progress rail (wizardProgress in wizard.ts).
export const WIZARD_STEPS: { id: WizardStep; label: string }[] = [
  { id: "network", label: "Network" },
  { id: "clients", label: "Clients" },
  { id: "mode", label: "Mode" },
  { id: "review", label: "Review" },
  { id: "run", label: "Run" },
];

// wizardStepClass mirrors wizardProgress's own i === currentIdx / i <
// currentIdx ternary.
export function wizardStepClass(current: WizardStep, step: WizardStep): "current" | "past" | "future" {
  const order = WIZARD_STEPS.map((s) => s.id);
  const currentIdx = order.indexOf(current);
  const idx = order.indexOf(step);
  return idx === currentIdx ? "current" : idx < currentIdx ? "past" : "future";
}

// Port defaults mirror internal/catalog/units.go's defaultExecHTTPPort /
// defaultBeaconHTTPPort / defaultExecP2PPort — the wizard only sends a port
// field to the server when it differs from its default (see
// buildStartSetupRequest).
export const DEFAULT_EXEC_HTTP_PORT = 8545;
export const DEFAULT_BEACON_HTTP_PORT = 5052;
export const DEFAULT_EXEC_P2P_PORT = 30303;

export const NETWORK_ORDER = [369, 943, 1];
export const NETWORK_BADGE: Record<number, string> = {
  369: "default",
  943: "practise here first",
};

export function defaultDataDir(chainId: number | null): string {
  return chainId !== null ? `/var/lib/valve-node-app/${chainId}` : "";
}

export function defaultJwtPath(dataDir: string): string {
  return `${dataDir}/jwt.hex`;
}

// ---------------------------------------------------------------------
// fit-check / sizing math — CRITICAL: byte-exact with wizard.ts
// ---------------------------------------------------------------------

// approxSize formats a decimal-TB dataset estimate for display, dropping to
// GB below a terabyte. Kept in decimal (not binary) so the figures match
// learn.valve.city's snapshot sizes and the Go catalog's ExpectedBytes.
export function approxSize(sizeTB: number): string {
  if (sizeTB <= 0) return "—";
  if (sizeTB >= 1) return `~${sizeTB.toFixed(1)} TB`;
  return `~${Math.round(sizeTB * 1000)} GB`;
}

// FIT_MARGIN matches the server preflight's 10% headroom over the raw
// dataset estimate.
export const FIT_MARGIN = 1.1;

// FULL_TIER_FRACTION mirrors Go's catalog.fullTierFraction — how much of
// the reth snapshot size we quote for the full(pruned) tier. It is an
// UNSOURCED PLACEHOLDER on both sides: no measurement backs it, for any
// client. It is duplicated here (rather than derived from the API) only so
// the fit maths the browser does agrees with the server's preflight floor
// exactly; if the Go constant changes, change this one too.
export const FULL_TIER_FRACTION = 0.5;

export const ARCHIVE_SIZE_BASIS = "Valve reth snapshot";
export const FULL_SIZE_BASIS = "rough estimate";

// archiveTierTB / fullTierTB are the single place these two figures are
// derived, so the accordion, the radios, the disk readout and the downgrade
// note can never quote different numbers.
export function archiveTierTB(net: Network): number {
  return net.SnapshotSizeTB;
}
export function fullTierTB(net: Network): number {
  return net.SnapshotSizeTB * FULL_TIER_FRACTION;
}

// sizeBasisNoteText mirrors wizard.ts's own sizeBasisNote — the caveat
// spelled out once per step so the qualifiers on the individual numbers
// aren't cryptic.
export function sizeBasisNoteText(net: Network): string {
  return `${approxSize(archiveTierTB(net))} is the measured size of Valve's reth snapshot for ${net.Name}, at the block height it was cut. The full-tier figure is a rough estimate derived from it, not a measurement, and neither figure is client-specific — go-pulse, erigon-pulse and geth store the chain differently. Treat both as a sanity check on disk size, not a promise.`;
}

// tierNeeds applies the 10% headroom margin to both tiers' raw byte
// estimates — the exact figures a fit check is measured against.
export function tierNeeds(net: Network): { archive: number; full: number } {
  return {
    archive: archiveTierTB(net) * 1e12 * FIT_MARGIN,
    full: fullTierTB(net) * 1e12 * FIT_MARGIN,
  };
}

export interface FitCheck {
  archiveFits: boolean;
  fullFits: boolean;
  needs: { archive: number; full: number };
}

// checkFit mirrors storageStatusHtml's inline archiveFits/fullFits
// computation.
export function checkFit(net: Network, freeBytes: number): FitCheck {
  const needs = tierNeeds(net);
  return { archiveFits: freeBytes >= needs.archive, fullFits: freeBytes >= needs.full, needs };
}

export interface FitEvaluation {
  archive: boolean;
  downgradeNote: string | null;
}

// evaluateFit mirrors wizard.ts's own evaluateFit: if archive is selected
// but the location can't hold it (while full would fit), downgrade to full
// and return a note explaining why. Only meant to be applied right after a
// disk probe resolves — never as a reaction to the operator manually
// re-picking archive, so it never fights a deliberate choice.
export function evaluateFit(
  net: Network | undefined,
  freeBytes: number | null,
  archive: boolean,
  path: string,
): FitEvaluation {
  if (!net || freeBytes === null) return { archive, downgradeNote: null };
  const { archiveFits, fullFits } = checkFit(net, freeBytes);
  if (archive && !archiveFits && fullFits) {
    return {
      archive: false,
      downgradeNote: `Not enough space at ${path} for archive (${approxSize(archiveTierTB(net))}, ${ARCHIVE_SIZE_BASIS}) — switched to Full (${approxSize(fullTierTB(net))}, ${FULL_SIZE_BASIS}). Pick a location with more room to run archive.`,
    };
  }
  return { archive, downgradeNote: null };
}

// neitherFitsWarning mirrors storageStatusHtml's hard "neither fits"
// warning, shown only when there is no downgrade note already covering the
// situation.
export function neitherFitsWarning(net: Network, freeBytes: number): string | null {
  const { fullFits } = checkFit(net, freeBytes);
  if (fullFits) return null;
  return `Neither full (${approxSize(fullTierTB(net))}, ${FULL_SIZE_BASIS}) nor archive (${approxSize(archiveTierTB(net))}, ${ARCHIVE_SIZE_BASIS}) fits the free space here — choose a location with more room.`;
}

// ---------------------------------------------------------------------
// client picker
// ---------------------------------------------------------------------

// clientProvider extracts the publishing org from a client's source repo
// (e.g. https://github.com/valve-tech/reth → "valve-tech"), so the picker
// shows who actually provides each client — upstream team vs a fork.
export function clientProvider(repo: string): string {
  const parts = repo.split("/");
  return parts.length >= 4 ? parts[3] : repo;
}

export function clientOptionLabel(id: string, catalog: Catalog): string {
  const client = catalog.clients.find((c) => c.id === id);
  return client ? `${client.id} — ${clientProvider(client.repo)}` : id;
}

// clientRepoDisplay mirrors clientSourceLine's `shown` — the repo URL with
// its scheme stripped for display.
export function clientRepoDisplay(repo: string): string {
  return repo.replace(/^https?:\/\//, "");
}

// resolveClientId mirrors renderClientsStep's own per-render correction:
// if the current pick isn't valid for the network, fall back to that
// network's first offered client (or null if it offers none).
export function resolveClientId(current: string | null, offered: string[]): string | null {
  if (current !== null && offered.includes(current)) return current;
  return offered[0] ?? null;
}

// ---------------------------------------------------------------------
// field validation
// ---------------------------------------------------------------------

// validateCheckpointUrl mirrors the server's check: empty is fine (network
// default), otherwise it must be an http(s) URL.
export function validateCheckpointUrl(raw: string): string | null {
  if (!raw) return null;
  if (!/^https?:\/\/.+/i.test(raw)) {
    return "Enter an http(s) URL, or leave blank for the network default.";
  }
  return null;
}

// parseBindAddr validates an optional RPC bind address: empty means the
// loopback default; otherwise it must be a valid IPv4/IPv6 literal (the
// server enforces the same with net.ParseIP). Hostnames are rejected —
// clients bind to addresses, not names.
export function parseBindAddr(raw: string): { addr?: string; error?: string } {
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
export const VALID_PORT_RE = /^\d+$/;

// parsePort validates a raw port field string against the 1-65535 range
// (0 is reserved for "use the server's default" and is expressed by
// leaving the field blank, never by typing "0" — see the port fields'
// "leave blank for default" help text). Returns a user-facing error
// message when the value is present but invalid.
export function parsePort(raw: string): { port?: number; error?: string } {
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
// from def — the wizard never sends a port field equal to the server's own
// default, since "not set" and "set to default" are indistinguishable once
// persisted anyway (see api.ts's WireConfig comment). Invalid values are
// dropped here defensively; validateModeFields is what surfaces the inline
// field error and blocks navigation to review in the first place.
export function portOverride(raw: string, def: number): number | undefined {
  const { port } = parsePort(raw);
  if (port === undefined || port === def) return undefined;
  return port;
}

export interface ModeFields {
  execHTTPPort: string;
  beaconHTTPPort: string;
  execP2PPort: string;
  rpcBindAddr: string;
  checkpoint: boolean;
  checkpointUrl: string;
  execSnapshot: boolean;
  snapshotKey: string;
}

export interface ModeFieldErrors {
  execHTTPPortError: string | null;
  beaconHTTPPortError: string | null;
  execP2PPortError: string | null;
  rpcBindAddrError: string | null;
  checkpointUrlError: string | null;
  snapshotKeyError: string | null;
}

// validateModeFields mirrors readModeInputs's own error computations —
// wizard.ts recomputes these by re-reading the DOM on every trigger, so a
// controlled-input equivalent recomputes them from current field state
// instead (a SIMPLIFICATION over the DOM-read-back dance, not a behavior
// change: the same inputs produce the same errors).
export function validateModeFields(fields: ModeFields): ModeFieldErrors {
  return {
    execHTTPPortError: parsePort(fields.execHTTPPort).error ?? null,
    beaconHTTPPortError: parsePort(fields.beaconHTTPPort).error ?? null,
    execP2PPortError: parsePort(fields.execP2PPort).error ?? null,
    rpcBindAddrError: parseBindAddr(fields.rpcBindAddr).error ?? null,
    checkpointUrlError: fields.checkpoint ? validateCheckpointUrl(fields.checkpointUrl) : null,
    snapshotKeyError:
      fields.execSnapshot && !fields.snapshotKey.trim()
        ? "A free snapshot key is required (get one at valve.city)."
        : null,
  };
}

export function hasModeFieldErrors(errors: ModeFieldErrors): boolean {
  return Object.values(errors).some((e) => e !== null);
}

// ---------------------------------------------------------------------
// start-setup request
// ---------------------------------------------------------------------

export interface StartSetupFields {
  chainId: number;
  execId: string;
  beaconId: string;
  archive: boolean;
  dataDir: string;
  jwtPath: string;
  execHTTPPort: string;
  beaconHTTPPort: string;
  execP2PPort: string;
  rpcBindAddr: string;
  checkpoint: boolean;
  checkpointUrl: string;
  execSnapshot: boolean;
  snapshotKey: string;
}

// buildStartSetupRequest mirrors startSetup's own wire-object construction
// byte-for-byte: DataDir/JWTPath are only sent when non-empty, a port is
// only sent when it parses AND differs from the server's default, an RPC
// bind address only when parseBindAddr accepts it, NoCheckpoint/
// CheckpointURL are mutually exclusive on the checkpoint toggle, and
// ExecSnapshot/SnapshotKey are only sent when the operator opted in.
export function buildStartSetupRequest(fields: StartSetupFields): StartSetupRequest {
  const wire: StartSetupRequest = {
    ChainID: fields.chainId,
    ExecID: fields.execId,
    BeaconID: fields.beaconId,
    Archive: fields.archive,
  };
  if (fields.dataDir) wire.DataDir = fields.dataDir;
  if (fields.jwtPath) wire.JWTPath = fields.jwtPath;

  const execHTTPPort = portOverride(fields.execHTTPPort, DEFAULT_EXEC_HTTP_PORT);
  const beaconHTTPPort = portOverride(fields.beaconHTTPPort, DEFAULT_BEACON_HTTP_PORT);
  const execP2PPort = portOverride(fields.execP2PPort, DEFAULT_EXEC_P2P_PORT);
  if (execHTTPPort !== undefined) wire.ExecHTTPPort = execHTTPPort;
  if (beaconHTTPPort !== undefined) wire.BeaconHTTPPort = beaconHTTPPort;
  if (execP2PPort !== undefined) wire.ExecP2PPort = execP2PPort;

  const { addr: rpcBindAddr } = parseBindAddr(fields.rpcBindAddr);
  if (rpcBindAddr !== undefined) wire.RPCBindAddr = rpcBindAddr;

  if (!fields.checkpoint) wire.NoCheckpoint = true;
  else if (fields.checkpointUrl) wire.CheckpointURL = fields.checkpointUrl;

  if (fields.execSnapshot) {
    wire.ExecSnapshot = true;
    wire.SnapshotKey = fields.snapshotKey;
  }

  return wire;
}

// ---------------------------------------------------------------------
// review step
// ---------------------------------------------------------------------

// nonDefaultPorts mirrors renderReviewStep's portsRow: the list of
// non-default ports to summarize, in exec-HTTP/beacon-HTTP/exec-p2p order,
// empty when every port is left at its default.
export function nonDefaultPorts(fields: {
  execHTTPPort: string;
  beaconHTTPPort: string;
  execP2PPort: string;
}): { label: string; port: number }[] {
  const out: { label: string; port: number }[] = [];
  const execHTTPPort = portOverride(fields.execHTTPPort, DEFAULT_EXEC_HTTP_PORT);
  const beaconHTTPPort = portOverride(fields.beaconHTTPPort, DEFAULT_BEACON_HTTP_PORT);
  const execP2PPort = portOverride(fields.execP2PPort, DEFAULT_EXEC_P2P_PORT);
  if (execHTTPPort !== undefined) out.push({ label: "exec HTTP", port: execHTTPPort });
  if (beaconHTTPPort !== undefined) out.push({ label: "beacon HTTP", port: beaconHTTPPort });
  if (execP2PPort !== undefined) out.push({ label: "exec p2p", port: execP2PPort });
  return out;
}

// ---------------------------------------------------------------------
// run step — step/stream sequence
// ---------------------------------------------------------------------

// doneStepIds / erroredStepIds mirror renderRunStep's own doneIds/
// erroredIds sets.
export function doneStepIds(events: SetupEvent[]): Set<string> {
  return new Set(events.filter((e) => e.done).map((e) => e.stepId));
}

export function erroredStepIds(events: SetupEvent[]): Set<string> {
  return new Set(events.filter((e) => e.err).map((e) => e.stepId));
}

// stepLines mirrors renderRunStep's linesByStep, sliced to the last 5 lines
// per step at read time.
export function stepLines(events: SetupEvent[], stepId: string): string[] {
  const lines: string[] = [];
  for (const e of events) {
    if (e.stepId === stepId && e.line) lines.push(e.line);
  }
  return lines.slice(-5);
}

// stepErrorLine mirrors renderRunStep's own errLine lookup: the FIRST
// error event for a step, matching Array.prototype.find.
export function stepErrorLine(events: SetupEvent[], stepId: string): string | undefined {
  return events.find((e) => e.stepId === stepId && e.err)?.err;
}

// runAllDone mirrors renderRunStep's own allDone: every step in STEP_PLAN
// has reported done, OR the plan's terminal step (handshake) alone has.
// The second clause is a strict superset of the first (handshake is itself
// one of STEP_PLAN's steps), so in practice this reduces to "has handshake
// reported done" — kept as the exact two-clause form wizard.ts uses rather
// than simplified away, so a future change to either side here matches a
// future change there.
export function runAllDone(events: SetupEvent[]): boolean {
  const doneIds = doneStepIds(events);
  return STEP_PLAN.every((s) => doneIds.has(s.id)) || events.some((e) => e.stepId === "handshake" && e.done);
}

// runAnyError mirrors renderRunStep's own anyError.
export function runAnyError(events: SetupEvent[]): boolean {
  return events.some((e) => !!e.err);
}
