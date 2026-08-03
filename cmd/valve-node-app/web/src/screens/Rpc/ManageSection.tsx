// "Manage gateway" — the container chrome, collapsed by default so the top of
// the page stays the URLs. Inside: the lifecycle actions the SERVER permits
// (rendered verbatim from gw.actions), Analytics, re-probe, Settings, Forget,
// the base URL, the certificate-to-install note, any leftover container, and
// the settings form.
import type { OrphanedContainer, GatewayView, TrustCertResult } from "../../api";
import { internalCaPath, manageStatus, manualTrustCommand, osLabel } from "./rpcModel";
import { CopyButton } from "./CopyButton";
import { OrphanBanner } from "./OrphanBanner";
import { SettingsBlock, type SettingsValues } from "./SettingsBlock";

interface ActionButtonDef {
  label: string;
  title: string;
  className: string;
}

const ACTION_BUTTONS: Record<string, ActionButtonDef> = {
  start: { label: "Start", title: "Start the existing gateway container", className: "btn" },
  stop: { label: "Stop", title: "Stop the gateway. Its configuration is kept.", className: "btn btn-ghost" },
  restart: {
    label: "Restart",
    title: "Restart the gateway. This also clears its cached per-chain head, which is what a chain reset needs.",
    className: "btn btn-ghost",
  },
  create: { label: "Create gateway", title: "Create the container from the configuration below", className: "btn" },
  recreate: {
    label: "Re-create (apply config)",
    title:
      "Replace the container so the saved configuration takes effect. A container's port and mounts are fixed when it is created, so this is the only way to apply a change.",
    className: "btn btn-ghost",
  },
  wipe: { label: "Wipe…", title: "Destroy the gateway container and rebuild it", className: "btn btn-danger" },
};

export interface ManageProps {
  gw: GatewayView;
  open: boolean;
  onToggle: () => void;
  orphans: OrphanedContainer[];
  orphanErr: Record<string, string | null>;
  onDismissOrphan: (name: string) => void;
  // lifecycle
  busy: string | null;
  onAction: (action: string) => void;
  capsBusy: boolean;
  onReprobe: () => void;
  // settings
  settingsOpen: boolean;
  onToggleSettings: () => void;
  onSaveSettings: (values: SettingsValues) => void;
  onForget: () => void;
  // cert / trust
  hostOS: string;
  targetMode: string;
  trustBusy: boolean;
  trustResult: TrustCertResult | null;
  onTrust: () => void;
  // verify
  verifying: boolean;
  verifyResult: import("../../api").TlsVerification | null;
  verifyErr: string | null;
  onVerify: () => void;
}

function ActionButton({ action, busy, onAction }: { action: string; busy: string | null; onAction: (a: string) => void }) {
  const def = ACTION_BUTTONS[action];
  if (!def) return null;
  return (
    <button className={def.className} title={def.title} disabled={!!busy} onClick={() => onAction(action)}>
      {busy === action ? <span className="spinner" aria-label="working" /> : def.label}
    </button>
  );
}

function TrustResultLine({ r }: { r: TrustCertResult }) {
  if (r.ok) {
    return (
      <div className="strip-line strip-note">
        <span className="strip-text">{r.message}</span>
      </div>
    );
  }
  return (
    <div className="strip-line strip-warn">
      <span className="strip-text">{r.message}</span>
      {r.ranCommand ? (
        <>
          <code className="strip-cmd">{r.ranCommand}</code>
          <CopyButton value={r.ranCommand} />
        </>
      ) : null}
    </div>
  );
}

// CertBlock is the full "install this certificate" note — a note, not a warning
// (nothing is broken), so it carries no colour. It offers the one-click trust
// only for a LOCAL gateway, and always the manual command for the browsing
// device's OS.
function CertBlock({ gw, hostOS, targetMode, trustBusy, trustResult, onTrust }: {
  gw: GatewayView;
  hostOS: string;
  targetMode: string;
  trustBusy: boolean;
  trustResult: TrustCertResult | null;
  onTrust: () => void;
}) {
  const path = internalCaPath(gw);
  if (!path) return null;
  const local = targetMode === "local";
  const cmd = manualTrustCommand(hostOS, path, gw.id);
  return (
    <div className="strip">
      <div className="strip-line strip-note">
        <span className="strip-text">
          Served by Caddy's own certificate authority — the browser warns once, on every device that calls it, until
          that authority's root is trusted. The root is on {gw.placement.targetId} at:
        </span>
        <code className="strip-cmd">{path}</code>
        <CopyButton value={path} label="Copy path" />
      </div>
      {local ? (
        <div className="strip-line strip-note">
          <span className="strip-text">This gateway runs on this machine, so its root can be installed here in one click:</span>
          <button className="btn btn-tiny" disabled={trustBusy} onClick={onTrust}>
            {trustBusy ? <span className="spinner" aria-label="installing" /> : "Trust on this machine"}
          </button>
        </div>
      ) : null}
      {trustResult ? <TrustResultLine r={trustResult} /> : null}
      <div className="strip-line strip-note">
        <span className="strip-text">
          The certificate must be trusted on whatever device opens the URL —{" "}
          {local
            ? "if that is a different device (a phone, another laptop), copy the root above to it and run"
            : "this gateway runs elsewhere, so on the device you browse from run"}
          {hostOS ? ` (${osLabel(hostOS)})` : ""}:
        </span>
        <code className="strip-cmd">{cmd}</code>
        <CopyButton value={cmd} label="Copy command" />
      </div>
    </div>
  );
}

export function ManageSection(props: ManageProps) {
  const { gw, open } = props;
  const running = gw.status.State === "running";
  return (
    <section className={`card manage-section${open ? " open" : ""}`}>
      <button type="button" className="manage-head" aria-expanded={open} onClick={props.onToggle}>
        <span className="manage-title">Manage gateway</span>
        <span className="manage-status muted small">{manageStatus(gw, props.orphans.length)}</span>
        <span className="manage-caret" aria-hidden="true">
          ▸
        </span>
      </button>
      {open ? (
        <div className="manage-body">
          <div className="rpc-head-actions">
            {(gw.actions ?? []).map((a) => (
              <ActionButton key={a} action={a} busy={props.busy} onAction={props.onAction} />
            ))}
            <a
              className="btn btn-ghost"
              href={`#/analytics/${encodeURIComponent(gw.id)}`}
              title="Latency, failures and why eRPC is routing the way it is. This screen tells you something is off; that one tells you what."
            >
              Analytics
            </a>
            <button
              className="btn btn-ghost"
              title="Ask every endpoint what it can do, again. This opens real connections to them."
              disabled={props.capsBusy}
              onClick={props.onReprobe}
            >
              {props.capsBusy ? <span className="spinner" aria-label="probing" /> : "Re-probe"}
            </button>
            <button className="btn btn-ghost" onClick={props.onToggleSettings}>
              {props.settingsOpen ? "Close settings" : "Settings"}
            </button>
            <button
              className="btn btn-ghost"
              title="Remove this gateway from valve-node-app. Its container is left alone."
              onClick={props.onForget}
            >
              Forget…
            </button>
          </div>
          {running ? (
            <div className="rpc-head-url">
              <code className="endpoint-url">{gw.baseUrl}</code>
              <CopyButton value={gw.baseUrl} label="Copy base" />
              <span className="muted small">
                a chain is addressed by path, e.g. <code>{(gw.networks ?? [])[0]?.path ?? "/main/evm/<chainId>"}</code>
              </span>
            </div>
          ) : (
            <p className="muted small">
              Not serving — it will answer on <code>{gw.baseUrl}</code> once it is running.
            </p>
          )}
          <CertBlock
            gw={gw}
            hostOS={props.hostOS}
            targetMode={props.targetMode}
            trustBusy={props.trustBusy}
            trustResult={props.trustResult}
            onTrust={props.onTrust}
          />
          {props.orphans.map((o) => (
            <OrphanBanner key={o.containerName} orphan={o} error={props.orphanErr[o.containerName]} onDismiss={props.onDismissOrphan} />
          ))}
          {props.settingsOpen ? (
            <SettingsBlock
              gw={gw}
              onSave={props.onSaveSettings}
              verifying={props.verifying}
              verifyResult={props.verifyResult}
              verifyErr={props.verifyErr}
              onVerify={props.onVerify}
            />
          ) : null}
        </div>
      ) : null}
    </section>
  );
}
