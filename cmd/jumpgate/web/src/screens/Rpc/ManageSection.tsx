// "Manage gateway" — the container chrome, collapsed by default so the top of
// the page stays the URLs. Inside: the lifecycle actions the SERVER permits
// (rendered verbatim from gw.actions), Analytics, re-probe, Settings, Forget,
// the base URL, any leftover container, and the settings form. Cert-trust is
// NOT here — it is a one-time, gateway-level action that lives beside the
// base URL in <GatewayIdentity>, not duplicated in this section too.
import type { OrphanedContainer, GatewayView } from "../../api";
import { manageStatus } from "./rpcModel";
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
