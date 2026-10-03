// Machine page → Dashboard section — the live sync/peer/disk dashboard fed
// by the target's 5s monitor SSE stream, plus the day-2 operator controls:
// service start/stop/restart, per-service clear-and-resync (behind a
// typed-confirm modal), disk usage/size estimates, and endpoint
// reachability. Port of dashboard.ts's renderDashboard.
//
// renderDashboard's `disposed` guard and manual streamStop/pending
// bookkeeping become useTargets/useCatalog (React Query, shared with every
// other target-scoped screen — see hooks/target.ts) plus this section's own
// hooks/dashboard.ts (useMonitorStream for the SSE snapshot + smoothed rate,
// useDiskUsage/useEndpoints for the one-shot fetches with manual retry,
// useServiceActions for start/stop/restart, useClearService for the
// destructive clear). The imperative openModal/closeModal clear-confirm
// dance becomes one local <ClearServiceModal>.
//
// NOT yet wired into the Machine page — the machine composer task mounts
// this. It still renders its own <h1>/<Footer>, exactly like the legacy
// module does today inside machine.ts's section body; stripping those so the
// page reads as one document is the follow-up machine.ts's own comment calls
// out, not this task.
import { useState } from "react";
import type { UseQueryResult } from "@tanstack/react-query";
import * as api from "../../api";
import { copyToClipboard, fmtBytes, fmtInt, fmtPct } from "../../ui";
import { useCatalog, useTargets } from "../../hooks/target";
import {
  useClearService,
  useDiskUsage,
  useEndpoints,
  useMonitorStream,
  useServiceActions,
} from "../../hooks/dashboard";
import { Badge } from "../../components/Badge";
import { Footer } from "../../components/Footer";
import { Modal } from "../Panel/Modal";
import {
  beaconSyncStatus,
  clearConfirmValid,
  diskWarn as computeDiskWarn,
  execSyncStatus,
  SERVICE_LABEL,
  serviceActionDisabled,
  servicePct,
  storageAdvancing,
  syncETA,
  topStatus,
  type StatusBadge as StatusBadgeValue,
} from "./dashboardModel";

function formatError(err: unknown): string {
  return err instanceof Error ? err.message : String(err);
}

export function DashboardSection({ targetId }: { targetId: string }) {
  const targetsQuery = useTargets();
  const catalogQuery = useCatalog();
  const initLoading = targetsQuery.isLoading || catalogQuery.isLoading;
  const initErr = targetsQuery.error ?? catalogQuery.error;

  const target = targetsQuery.data?.find((t) => t.id === targetId);
  const wired = !!target?.wire;
  const net =
    wired && target?.wire
      ? catalogQuery.data?.networks.find((n) => n.ChainID === target.wire!.ChainID)
      : undefined;

  const { snapshot, execBlocksPerSec } = useMonitorStream(targetId, wired);
  const diskUsageQuery = useDiskUsage(targetId, wired);
  const endpointsQuery = useEndpoints(targetId, wired);
  const { pending, error: actionErr, run: runServiceAction } = useServiceActions(targetId);
  const clearMutation = useClearService(targetId);

  const [clearSvc, setClearSvc] = useState<api.ServiceID | null>(null);

  function closeClear(): void {
    setClearSvc(null);
    clearMutation.reset();
  }

  async function confirmClear(svc: api.ServiceID): Promise<void> {
    try {
      await clearMutation.mutateAsync(svc);
      setClearSvc(null);
    } catch {
      // Swallowed here — clearMutation.error drives the modal's inline
      // error message, mirroring runClear's own catch that keeps the modal
      // open and re-enables its confirm button.
    }
  }

  return (
    <>
      <h1>Dashboard: {targetId}</h1>
      <div>
        {initLoading ? (
          <p className="muted">Loading…</p>
        ) : initErr ? (
          <p className="error">Failed to load target: {formatError(initErr)}</p>
        ) : !target ? (
          <p className="error">
            Target &quot;{targetId}&quot; not found. <a href="#/targets">Back to targets</a>
          </p>
        ) : !wired ? (
          <p className="muted">
            This target hasn&apos;t completed setup yet.{" "}
            <a href={`#/setup/${encodeURIComponent(targetId)}`}>Run the setup wizard →</a>
          </p>
        ) : !snapshot ? (
          <p className="muted">Connecting…</p>
        ) : (
          <>
            <p className="dash-status">
              <StatusBadge status={topStatus(snapshot)} />
            </p>
            <div className="card-grid">
              <ServicesCard
                snapshot={snapshot}
                targetId={targetId}
                pending={pending}
                actionErr={actionErr}
                onAction={(svc, kind) => void runServiceAction(svc, kind)}
                onOpenClear={setClearSvc}
              />
              <ExecSyncCard snapshot={snapshot} execBlocksPerSec={execBlocksPerSec} />
              <BeaconSyncCard snapshot={snapshot} />
              <PeersCard snapshot={snapshot} />
              <StorageCard
                snapshot={snapshot}
                execBlocksPerSec={execBlocksPerSec}
                diskUsageQuery={diskUsageQuery}
              />
              <EndpointsCard endpointsQuery={endpointsQuery} />
            </div>
            <p className="muted small">Last updated {new Date(snapshot.at).toLocaleTimeString()}</p>
          </>
        )}
      </div>
      <Footer contextLabel={net?.Name} contextUrl={net?.LearnURL} />
      {clearSvc && (
        <ClearServiceModal
          svc={clearSvc}
          diskUsage={diskUsageQuery.data}
          pending={clearMutation.isPending}
          error={clearMutation.error ? `Clear failed: ${formatError(clearMutation.error)}` : null}
          onCancel={closeClear}
          onConfirm={() => void confirmClear(clearSvc)}
        />
      )}
    </>
  );
}

function StatusBadge({ status }: { status: StatusBadgeValue }) {
  return <Badge text={status.text} kind={status.kind} />;
}

function ExecSyncCard({
  snapshot,
  execBlocksPerSec,
}: {
  snapshot: api.Snapshot;
  execBlocksPerSec: number | null;
}) {
  const status = execSyncStatus(snapshot);
  const { lag, eta } = syncETA(snapshot, execBlocksPerSec);
  return (
    <div className="card">
      <h3>Execution sync</h3>
      <p>
        <StatusBadge status={status} />
      </p>
      <dl className="stat-list">
        <div>
          <dt>Local head</dt>
          <dd>{fmtInt(snapshot.execHead)}</dd>
        </div>
        <div>
          <dt>Reference head</dt>
          <dd>{lag !== null ? fmtInt(snapshot.refHead) : "unavailable"}</dd>
        </div>
        <div>
          <dt>Lag</dt>
          <dd>{lag !== null ? `${fmtInt(Math.max(lag, 0))} blocks` : "—"}</dd>
        </div>
        <div>
          <dt>ETA</dt>
          <dd>{eta}</dd>
        </div>
      </dl>
    </div>
  );
}

function BeaconSyncCard({ snapshot }: { snapshot: api.Snapshot }) {
  const status = beaconSyncStatus(snapshot);
  return (
    <div className="card">
      <h3>Beacon sync</h3>
      <p>
        <StatusBadge status={status} />
      </p>
      <dl className="stat-list">
        <div>
          <dt>Slot</dt>
          <dd>{fmtInt(snapshot.beaconSlot)}</dd>
        </div>
        <div>
          <dt>Distance</dt>
          <dd>{fmtInt(snapshot.beaconDistance)}</dd>
        </div>
      </dl>
    </div>
  );
}

function PeersCard({ snapshot }: { snapshot: api.Snapshot }) {
  return (
    <div className="card">
      <h3>Peers</h3>
      <dl className="stat-list">
        <div>
          <dt>Execution</dt>
          <dd>{fmtInt(snapshot.execPeers)}</dd>
        </div>
        <div>
          <dt>Beacon</dt>
          <dd>{fmtInt(snapshot.beaconPeers)}</dd>
        </div>
      </dl>
    </div>
  );
}

// StorageCard is the consolidated Disk + Storage card — see dashboard.ts's
// own storageCard comment for the full rationale (estimates ported from
// learn.valve.city, not live measurements; the disk-used% meter comes from
// the snapshot so it renders unconditionally while the rest depends on the
// separate /du fetch and handles its own loading/error state below it).
function StorageCard({
  snapshot,
  execBlocksPerSec,
  diskUsageQuery,
}: {
  snapshot: api.Snapshot;
  execBlocksPerSec: number | null;
  diskUsageQuery: UseQueryResult<api.DiskUsage>;
}) {
  const warn = computeDiskWarn(snapshot);
  const cardClass = warn ? "card card-warn" : "card";
  const diskSection = (
    <>
      <div className="meter">
        <div
          className={warn ? "meter-fill meter-warn" : "meter-fill"}
          style={{ width: `${Math.min(snapshot.diskUsedPct, 100)}%` }}
        />
      </div>
      <p>{fmtPct(snapshot.diskUsedPct)} used</p>
    </>
  );

  if (diskUsageQuery.error) {
    return (
      <div className={cardClass}>
        <h3>Storage</h3>
        {diskSection}
        <p className="error small">{formatError(diskUsageQuery.error)}</p>
        <button className="btn btn-ghost" type="button" onClick={() => void diskUsageQuery.refetch()}>
          Retry
        </button>
      </div>
    );
  }

  const du = diskUsageQuery.data;
  if (!du) {
    return (
      <div className={cardClass}>
        <h3>Storage</h3>
        {diskSection}
        <p className="muted">Loading…</p>
      </div>
    );
  }

  const execPct = servicePct(du.ExecBytes, du.ExpectedExecBytes);
  const beaconPct = servicePct(du.BeaconBytes, du.ExpectedBeaconBytes);
  const { lag, eta } = syncETA(snapshot, execBlocksPerSec);
  const advancing = storageAdvancing(lag, execBlocksPerSec);

  return (
    <div className={cardClass}>
      <h3>Storage</h3>
      {diskSection}
      <p className="muted small">Estimate — varies by client and pruning.</p>
      <p className="muted small">
        Execution — {fmtBytes(du.ExecBytes)} of ~{fmtBytes(du.ExpectedExecBytes)}
      </p>
      <div className="meter">
        <div className="meter-fill" style={{ width: `${execPct}%` }} />
      </div>
      {advancing && <p className="muted small">Estimated time remaining: {eta}</p>}
      <p className="muted small">
        Beacon — {fmtBytes(du.BeaconBytes)} of ~{fmtBytes(du.ExpectedBeaconBytes)}
      </p>
      <div className="meter">
        <div className="meter-fill" style={{ width: `${beaconPct}%` }} />
      </div>
      <dl className="stat-list">
        <div>
          <dt>Disk free</dt>
          <dd>{fmtBytes(du.DiskFreeBytes)}</dd>
        </div>
        <div>
          <dt>Sync (snapshot)</dt>
          <dd>{du.SyncLabel}</dd>
        </div>
        <div>
          <dt>Sync (genesis)</dt>
          <dd>{du.GenesisSyncLabel}</dd>
        </div>
      </dl>
    </div>
  );
}

// endpointsCard shows the local RPC URLs, live reachability dots (probed
// on-box), and — for SSH targets — a copyable tunnel command plus the
// spec's "local to the server" sentence.
function EndpointsCard({ endpointsQuery }: { endpointsQuery: UseQueryResult<api.EndpointInfo> }) {
  if (endpointsQuery.error) {
    return (
      <div className="card card-warn">
        <h3>Endpoints</h3>
        <p className="error small">{formatError(endpointsQuery.error)}</p>
        <button className="btn btn-ghost" type="button" onClick={() => void endpointsQuery.refetch()}>
          Retry
        </button>
      </div>
    );
  }

  const ep = endpointsQuery.data;
  if (!ep) {
    return (
      <div className="card">
        <h3>Endpoints</h3>
        <p className="muted">Loading…</p>
      </div>
    );
  }

  const chainWarn = ep.ExecReachable && !ep.ChainIDMatches;
  return (
    <div className="card">
      <h3>Endpoints</h3>
      <div className="endpoint-row">
        <ReachDot ok={ep.ExecReachable} />
        <code className="endpoint-url">{ep.ExecHTTP}</code>
        <CopyButton text={ep.ExecHTTP} />
      </div>
      <div className="endpoint-row">
        <ReachDot ok={ep.BeaconReachable} />
        <code className="endpoint-url">{ep.BeaconHTTP}</code>
        <CopyButton text={ep.BeaconHTTP} />
      </div>
      {chainWarn && (
        <p className="error small">
          Exec responded, but its chain id doesn&apos;t match this target&apos;s wire config.
        </p>
      )}
      {ep.Access === "ssh" && (
        <>
          <p className="muted small">
            These URLs are local to the server; use the tunnel or your own reverse proxy to reach them from
            elsewhere.
          </p>
          <div className="endpoint-row">
            <code className="endpoint-url">{ep.TunnelHint}</code>
            <CopyButton text={ep.TunnelHint} />
          </div>
        </>
      )}
    </div>
  );
}

function ReachDot({ ok }: { ok: boolean }) {
  return <span className={`dot dot-${ok ? "ok" : "bad"}`} />;
}

// CopyButton mirrors dashboard.ts's own copyButton(): swap the label to
// "Copied!"/"Copy failed" for 1.5s, then back. A ref-tracked timeout is
// cleared on unmount so a stray setTimeout never fires setState on an
// unmounted component (the React equivalent of copyButton's own `if
// (!disposed)` guard).
function CopyButton({ text }: { text: string }) {
  const [label, setLabel] = useState("Copy");

  async function onClick(): Promise<void> {
    const ok = await copyToClipboard(text);
    setLabel(ok ? "Copied!" : "Copy failed");
    setTimeout(() => setLabel("Copy"), 1500);
  }

  return (
    <button className="btn btn-ghost" type="button" onClick={() => void onClick()}>
      {label}
    </button>
  );
}

function ServicesCard({
  snapshot,
  targetId,
  pending,
  actionErr,
  onAction,
  onOpenClear,
}: {
  snapshot: api.Snapshot;
  targetId: string;
  pending: Record<api.ServiceID, api.ServiceActionKind | null>;
  actionErr: string | null;
  onAction: (svc: api.ServiceID, kind: api.ServiceActionKind) => void;
  onOpenClear: (svc: api.ServiceID) => void;
}) {
  return (
    <div className="card">
      <h3>Services</h3>
      <ServiceRow
        svc="exec"
        active={snapshot.execActive}
        busy={pending.exec}
        onAction={onAction}
        onOpenClear={onOpenClear}
      />
      <ServiceRow
        svc="beacon"
        active={snapshot.beaconActive}
        busy={pending.beacon}
        onAction={onAction}
        onOpenClear={onOpenClear}
      />
      {actionErr && <p className="error small">{actionErr}</p>}
      <p className="card-links">
        <a href={`#/logs/${encodeURIComponent(targetId)}`}>View logs →</a>{" "}
        <a href={`#/security/${encodeURIComponent(targetId)}`}>Security →</a>{" "}
        <a href={`#/diag/${encodeURIComponent(targetId)}`}>Diagnostics →</a>
      </p>
    </div>
  );
}

function ServiceRow({
  svc,
  active,
  busy,
  onAction,
  onOpenClear,
}: {
  svc: api.ServiceID;
  active: boolean;
  busy: api.ServiceActionKind | null;
  onAction: (svc: api.ServiceID, kind: api.ServiceActionKind) => void;
  onOpenClear: (svc: api.ServiceID) => void;
}) {
  const label = SERVICE_LABEL[svc];

  function actionButton(kind: api.ServiceActionKind, text: string) {
    const disabled = serviceActionDisabled(kind, active, busy);
    const isBusy = busy === kind;
    return (
      <button className="btn btn-ghost" type="button" disabled={disabled} onClick={() => onAction(svc, kind)}>
        {isBusy ? <span className="spinner" aria-label="working" /> : text}
      </button>
    );
  }

  return (
    <div className="service-row">
      <span>
        {label} <Badge text={active ? "active" : "down"} kind={active ? "ok" : "bad"} />
      </span>
      <div className="service-actions">
        {actionButton("start", "Start")}
        {actionButton("stop", "Stop")}
        {actionButton("restart", "Restart")}
        <button
          className="btn btn-danger"
          type="button"
          disabled={busy !== null}
          onClick={() => onOpenClear(svc)}
        >
          Clear…
        </button>
      </div>
    </div>
  );
}

// ClearServiceModal is the typed-confirm gate for the destructive clear —
// see dashboard.ts's own SIMPLIFICATION comment on openClearModal: the API
// doesn't expose the exact filesystem path(s) a clear deletes, so this shows
// the generic "<service> chain data under the node's data directory"
// description plus the current size from /du instead of the literal path.
function ClearServiceModal({
  svc,
  diskUsage,
  pending,
  error,
  onCancel,
  onConfirm,
}: {
  svc: api.ServiceID;
  diskUsage: api.DiskUsage | undefined;
  pending: boolean;
  error: string | null;
  onCancel: () => void;
  onConfirm: () => void;
}) {
  const [input, setInput] = useState("");
  const label = SERVICE_LABEL[svc];
  const size = diskUsage
    ? fmtBytes(svc === "exec" ? diskUsage.ExecBytes : diskUsage.BeaconBytes)
    : "unknown (disk usage hasn't loaded)";
  const valid = clearConfirmValid(input, svc);

  return (
    <Modal onClose={onCancel}>
      <h2>Clear {label} data</h2>
      <p className="error">
        This stops the {label.toLowerCase()} service, deletes its chain data under the node&apos;s data
        directory (current size: {size}), and starts it again. A full resync is required afterward.
      </p>
      <p>
        Type <code>{svc}</code> to confirm.
      </p>
      <input
        type="text"
        autoComplete="off"
        spellCheck={false}
        value={input}
        onChange={(e) => setInput(e.target.value)}
        autoFocus
      />
      {error && <p className="error small">{error}</p>}
      <div className="modal-actions">
        <button className="btn btn-ghost" type="button" onClick={onCancel}>
          Cancel
        </button>
        <button className="btn btn-danger" type="button" disabled={!valid || pending} onClick={onConfirm}>
          {pending ? "Clearing…" : "Clear and resync"}
        </button>
      </div>
    </Modal>
  );
}
