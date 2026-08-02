// Machine page → Services section — the container-backed devnet a machine
// hosts: status, start/stop/restart, create/re-create, wipe, and its saved
// configuration, in one section. Port of services.ts's renderServices.
//
// Unlike Logs/Dashboard, this section does NOT gate on the target having
// completed setup — see services.ts's own header comment: "Offered on EVERY
// machine, set up or not... a devnet is a container, so unlike a node it
// needs no Linux host and no root." It also never reads useTargets/
// useCatalog: services.ts's renderServices takes only targetId and talks
// straight to the containers API, so this component does the same.
//
// renderServices's `disposed` guard and manual busy/actionErr/activity/
// streamStop bookkeeping become hooks/services.ts's useContainers (a plain
// React Query, re-read after every action settles — never on a timer, since
// nothing here polls) plus useContainerOps (the shared busy/error/activity
// state and the provisioning stream) and useSaveContainerConfig/
// useWipeContainer. The imperative openModal/closeModal wipe dance becomes
// one local <WipeModal>/<WipeResultModal> pair, the same replacement
// Panel/Dialogs.tsx made for panel.ts.
//
// The config editor's readDrafts()-on-every-structural-edit dance has no
// counterpart here: services.ts needed it because a re-render from an
// unrelated card (a busy spinner ticking, say) would otherwise blow away
// whatever was mid-typed in the form. React's own controlled-input state
// (the draft lives in useState, not the DOM) makes that unnecessary — a
// SIMPLIFICATION over the legacy DOM-read-back approach, not a behavior
// change.
//
// NOT yet wired into the Machine page — the machine composer task mounts
// this. It still renders its own <h1>/<Footer>, exactly like the legacy
// module does today inside machine.ts's section body; stripping those so the
// page reads as one document is the follow-up machine.ts's own comment calls
// out, not this task.
import { useState } from "react";
import * as api from "../../api";
import { copyToClipboard } from "../../ui";
import { useContainerOps, useContainers, useSaveContainerConfig, useWipeContainer } from "../../hooks/services";
import { Badge } from "../../components/Badge";
import { Footer } from "../../components/Footer";
import { Modal } from "../Panel/Modal";
import {
  ACTION_BUTTONS,
  actionLabel,
  devnetConfigError,
  devnetSummary,
  dockerBannerTitle,
  dockerOk,
  exitCodeLine,
  parseIntOr,
  SERVICE_BLURB,
  showNoEndpointsMessage,
  stateBadge,
  wipeConfirmValid,
} from "./servicesModel";

function formatError(err: unknown): string {
  return err instanceof Error ? err.message : String(err);
}

type WipeState =
  | { kind: "closed" }
  | { kind: "confirm"; svc: api.ContainerServiceID; pending: boolean; error: string | null }
  | { kind: "result"; svc: api.ContainerServiceID; result: api.WipeResult };

export function ServicesSection({ targetId }: { targetId: string }) {
  const containersQuery = useContainers(targetId);
  const ops = useContainerOps(targetId);
  const saveMutation = useSaveContainerConfig(targetId);
  const wipeMutation = useWipeContainer(targetId);

  const [openConfig, setOpenConfig] = useState<Record<string, boolean>>({});
  const [devnetDraft, setDevnetDraft] = useState<api.DevnetConfig | null>(null);
  const [saveErr, setSaveErr] = useState<Record<string, string | null>>({});
  const [saveNote, setSaveNote] = useState<Record<string, string | null>>({});
  const [wipe, setWipe] = useState<WipeState>({ kind: "closed" });

  const services = containersQuery.data?.services ?? [];
  const viewOf = (svc: string) => services.find((s) => s.id === svc);

  function toggleConfig(v: api.ContainerView): void {
    const willOpen = !openConfig[v.id];
    setOpenConfig((o) => ({ ...o, [v.id]: willOpen }));
    setSaveErr((e) => ({ ...e, [v.id]: null }));
    setSaveNote((n) => ({ ...n, [v.id]: null }));
    // Deep-copied from the view so an abandoned edit cannot leak into what
    // the card displays — mirrors toggleConfig's own draft seeding.
    if (willOpen && v.devnet) setDevnetDraft({ ...v.devnet });
  }

  async function saveConfig(svc: api.ContainerServiceID): Promise<void> {
    setSaveErr((e) => ({ ...e, [svc]: null }));
    setSaveNote((n) => ({ ...n, [svc]: null }));
    if (!devnetDraft) return;

    const validationErr = devnetConfigError(devnetDraft);
    if (validationErr) {
      setSaveErr((e) => ({ ...e, [svc]: validationErr }));
      return;
    }

    const wasRunning = viewOf(svc)?.status.State === "running";
    try {
      await saveMutation.mutateAsync({ svc, config: devnetDraft });
    } catch (err) {
      setSaveErr((e) => ({ ...e, [svc]: formatError(err) }));
      return;
    }
    setOpenConfig((o) => ({ ...o, [svc]: false }));
    setSaveNote((n) => ({
      ...n,
      [svc]: wasRunning
        ? "Saved. The running container still has the old settings — press “Re-create (apply config)” to put them into effect."
        : "Saved.",
    }));
  }

  function handleAction(v: api.ContainerView, action: string): void {
    if (action === "create" || action === "recreate") {
      void ops.provision(v.id);
      return;
    }
    if (action === "wipe") {
      setWipe({ kind: "confirm", svc: v.id, pending: false, error: null });
      return;
    }
    void ops.run(v.id, action as api.ContainerActionKind);
  }

  async function confirmWipe(): Promise<void> {
    if (wipe.kind !== "confirm") return;
    const svc = wipe.svc;
    setWipe({ kind: "confirm", svc, pending: true, error: null });
    try {
      const result = await wipeMutation.mutateAsync(svc);
      setWipe({ kind: "result", svc, result });
    } catch (err) {
      setWipe({ kind: "confirm", svc, pending: false, error: `Wipe failed: ${formatError(err)}` });
    }
  }

  // Both the confirm-cancel and result-close paths reload the list — mirrors
  // services.ts's own modal callback, which calls `void load()` on every
  // "cancel"/"close" action regardless of whether a wipe actually happened.
  function closeWipe(): void {
    setWipe({ kind: "closed" });
    void containersQuery.refetch();
  }

  return (
    <>
      <div className="page-head">
        <h1>Services: {targetId}</h1>
        <button className="btn btn-ghost" type="button" onClick={() => void containersQuery.refetch()}>
          Refresh
        </button>
      </div>
      <p className="muted">
        The throwaway chain this machine can host. It is independent of any node setup — a machine can run a
        devnet, a node, both, or neither. The RPC gateway in front of it lives on the{" "}
        <a href="#/rpc">RPC</a> screen, because it fronts chains across every machine rather than belonging to
        this one.
      </p>
      <div>
        {containersQuery.isError ? (
          <p className="error">Could not read this machine&apos;s services: {formatError(containersQuery.error)}</p>
        ) : !containersQuery.data ? (
          <p className="muted">Loading…</p>
        ) : (
          <>
            <DockerBanner docker={containersQuery.data.docker} />
            <div className="card-grid card-grid-wide">
              {services.map((v) => (
                <ServiceCard
                  key={v.id}
                  v={v}
                  busy={ops.busy[v.id] ?? null}
                  error={ops.error[v.id] ?? null}
                  activity={ops.activity[v.id] ?? []}
                  onAction={(action) => handleAction(v, action)}
                  configOpen={!!openConfig[v.id]}
                  onToggleConfig={() => toggleConfig(v)}
                  draft={devnetDraft}
                  onDraftChange={setDevnetDraft}
                  onSaveConfig={() => void saveConfig(v.id)}
                  saveErr={saveErr[v.id] ?? null}
                  saveNote={saveNote[v.id] ?? null}
                  saving={saveMutation.isPending}
                />
              ))}
            </div>
          </>
        )}
      </div>
      <Footer />
      {wipe.kind === "confirm" && (
        <WipeModal
          v={viewOf(wipe.svc)}
          allViews={services}
          pending={wipe.pending}
          error={wipe.error}
          onCancel={closeWipe}
          onConfirm={() => void confirmWipe()}
        />
      )}
      {wipe.kind === "result" && (
        <WipeResultModal v={viewOf(wipe.svc)} svc={wipe.svc} allViews={services} result={wipe.result} onClose={closeWipe} />
      )}
    </>
  );
}

function DockerBanner({ docker }: { docker: api.DockerView }) {
  if (dockerOk(docker)) {
    return (
      <p className="muted small">
        Docker: {docker.flavor}
        {docker.serverVersion ? ` ${docker.serverVersion}` : ""} · reachable
      </p>
    );
  }
  return (
    <div className="banner banner-bad">
      <strong>{dockerBannerTitle(docker)}</strong> — both of these services are containers, so nothing here can
      start until that is fixed.
      {docker.detail && <div className="small">{docker.detail}</div>}
      {docker.hint && <div className="small">{docker.hint}</div>}
    </div>
  );
}

function ServiceCard({
  v,
  busy,
  error,
  activity,
  onAction,
  configOpen,
  onToggleConfig,
  draft,
  onDraftChange,
  onSaveConfig,
  saveErr,
  saveNote,
  saving,
}: {
  v: api.ContainerView;
  busy: string | null;
  error: string | null;
  activity: string[];
  onAction: (action: string) => void;
  configOpen: boolean;
  onToggleConfig: () => void;
  draft: api.DevnetConfig | null;
  onDraftChange: (d: api.DevnetConfig) => void;
  onSaveConfig: () => void;
  saveErr: string | null;
  saveNote: string | null;
  saving: boolean;
}) {
  const badge = stateBadge(v);
  const warnings = v.warnings ?? [];
  const exitLine = exitCodeLine(v);

  return (
    <div className="card">
      <div className="service-head">
        <h2>{v.label}</h2>
        <Badge text={badge.text} kind={badge.kind} />
      </div>
      <p className="muted small">{SERVICE_BLURB[v.id] ?? ""}</p>

      {v.error && (
        <div className="banner banner-bad">
          <strong>This service could not be read.</strong>
          <div className="small">{v.error}</div>
          {v.hint && <div className="small">{v.hint}</div>}
        </div>
      )}
      {v.blocked && <div className="banner banner-warn">{v.blocked}</div>}
      {warnings.map((wmsg, i) => (
        <div className="banner banner-warn" key={i}>
          {wmsg}
        </div>
      ))}

      <dl className="stat-list">
        <div>
          <dt>Container</dt>
          <dd>
            <code>{v.containerName}</code>
          </dd>
        </div>
        <div>
          <dt>Image</dt>
          <dd>{v.status.Image ? <code>{v.status.Image}</code> : "—"}</dd>
        </div>
      </dl>
      {exitLine && <p className="muted small">{exitLine}</p>}

      <EndpointsBlock v={v} />

      <div className="card-actions">
        {(v.actions ?? []).map((action) => {
          const def = ACTION_BUTTONS[action];
          if (!def) return null;
          const isBusy = busy === action;
          return (
            <button
              key={action}
              className={def.className}
              type="button"
              title={def.title}
              disabled={!!busy}
              onClick={() => onAction(action)}
            >
              {isBusy ? <span className="spinner" aria-label="working" /> : actionLabel(v.id, action)}
            </button>
          );
        })}
      </div>
      {error && <p className="error small">{error}</p>}
      {activity.length > 0 && (
        <div className="config-block">
          <p className="muted small">Provisioning</p>
          <pre className="step-log">{activity.join("\n")}</pre>
        </div>
      )}

      <ConfigSection
        v={v}
        open={configOpen}
        onToggle={onToggleConfig}
        draft={draft}
        onDraftChange={onDraftChange}
        onSave={onSaveConfig}
        saveErr={saveErr}
        saveNote={saveNote}
        saving={saving}
      />
    </div>
  );
}

function EndpointsBlock({ v }: { v: api.ContainerView }) {
  const eps = v.endpoints ?? [];
  if (eps.length === 0) {
    if (showNoEndpointsMessage(v)) {
      return <p className="muted small">No endpoint to show — this gateway has no chains configured yet.</p>;
    }
    return null;
  }
  return (
    <>
      {eps.map((ep) => (
        <EndpointRow key={ep.url} label={ep.label} url={ep.url} />
      ))}
    </>
  );
}

function EndpointRow({ label, url }: { label: string; url: string }) {
  const [copyLabel, setCopyLabel] = useState("Copy");

  async function onClick(): Promise<void> {
    const ok = await copyToClipboard(url);
    setCopyLabel(ok ? "Copied!" : "Copy failed");
    setTimeout(() => setCopyLabel("Copy"), 1500);
  }

  return (
    <div className="endpoint-row">
      <span className="dot dot-ok" />
      <span className="muted small">{label}</span>
      <code className="endpoint-url">{url}</code>
      <button className="btn btn-ghost" type="button" onClick={() => void onClick()}>
        {copyLabel}
      </button>
    </div>
  );
}

function ConfigSection({
  v,
  open,
  onToggle,
  draft,
  onDraftChange,
  onSave,
  saveErr,
  saveNote,
  saving,
}: {
  v: api.ContainerView;
  open: boolean;
  onToggle: () => void;
  draft: api.DevnetConfig | null;
  onDraftChange: (d: api.DevnetConfig) => void;
  onSave: () => void;
  saveErr: string | null;
  saveNote: string | null;
  saving: boolean;
}) {
  return (
    <div className="config-block">
      <div className="service-head">
        <p className="muted small">Configuration{v.configured ? "" : " (nothing saved yet — these are the defaults)"}</p>
        <button className="btn btn-ghost" type="button" onClick={onToggle}>
          {open ? "Close" : "Edit"}
        </button>
      </div>
      {open ? (
        <DevnetForm draft={draft} onChange={onDraftChange} onSave={onSave} saving={saving} />
      ) : (
        <p className="small">{devnetSummary(v)}</p>
      )}
      {saveErr && <p className="error small">{saveErr}</p>}
      {saveNote && <p className="muted small">{saveNote}</p>}
    </div>
  );
}

function DevnetForm({
  draft,
  onChange,
  onSave,
  saving,
}: {
  draft: api.DevnetConfig | null;
  onChange: (d: api.DevnetConfig) => void;
  onSave: () => void;
  saving: boolean;
}) {
  if (!draft) return null;
  return (
    <>
      <label>
        Block time <span className="muted">— how often the chain seals a block</span>
        <input
          type="text"
          autoComplete="off"
          spellCheck={false}
          value={draft.BlockTime}
          onChange={(e) => onChange({ ...draft, BlockTime: e.target.value })}
        />
      </label>
      <label>
        JSON-RPC port
        <input
          type="text"
          inputMode="numeric"
          autoComplete="off"
          value={draft.HTTPPort}
          onChange={(e) => onChange({ ...draft, HTTPPort: parseIntOr(e.target.value, draft.HTTPPort) })}
        />
      </label>
      <label>
        WebSocket port
        <input
          type="text"
          inputMode="numeric"
          autoComplete="off"
          value={draft.WSPort}
          onChange={(e) => onChange({ ...draft, WSPort: parseIntOr(e.target.value, draft.WSPort) })}
        />
      </label>
      <label>
        Bind address{" "}
        <span className="muted">— 127.0.0.1 keeps it on this machine; 0.0.0.0 exposes it to your network</span>
        <input
          type="text"
          autoComplete="off"
          spellCheck={false}
          value={draft.BindAddr}
          onChange={(e) => onChange({ ...draft, BindAddr: e.target.value })}
        />
      </label>
      <p className="muted small">
        The chain id is fixed at {draft.ChainID}: reth&apos;s --dev genesis is baked into the image, and serving
        another id would need a custom genesis this app does not render.
      </p>
      <div className="card-actions">
        <button className="btn" type="button" disabled={saving} onClick={onSave}>
          {saving ? "Saving…" : "Save configuration"}
        </button>
      </div>
    </>
  );
}

// WipeModal mirrors openWipeModal: it says three things before it will
// accept a confirmation — what is destroyed, what is restarted as a
// consequence, and why that restart is not optional.
function WipeModal({
  v,
  allViews,
  pending,
  error,
  onCancel,
  onConfirm,
}: {
  v: api.ContainerView | undefined;
  allViews: api.ContainerView[];
  pending: boolean;
  error: string | null;
  onCancel: () => void;
  onConfirm: () => void;
}) {
  const [input, setInput] = useState("");
  if (!v) return null;
  const fronts = (v.restartsOnWipe ?? []).map((id) => allViews.find((s) => s.id === id)?.label ?? id);
  const valid = wipeConfirmValid(input, v.id);

  return (
    <Modal onClose={onCancel}>
      <h2>Wipe {v.label}</h2>
      <p className="error">This deletes {v.wipeDiscards}</p>
      {fronts.length > 0 && (
        <p>
          It also restarts what sits in front of it: {fronts.join(", ")}. That restart is required, not tidy-up:
          eRPC only ever moves a chain&apos;s head forward, so once this chain restarts at block 0 the gateway
          would keep advertising the old head — answering for blocks the chain no longer has — until the new
          chain grew past it.
        </p>
      )}
      <p>
        Type <code>{v.id}</code> to confirm.
      </p>
      <input type="text" autoComplete="off" spellCheck={false} value={input} onChange={(e) => setInput(e.target.value)} autoFocus />
      {error && <p className="error small">{error}</p>}
      <div className="modal-actions">
        <button className="btn btn-ghost" type="button" onClick={onCancel}>
          Cancel
        </button>
        <button className="btn btn-danger" type="button" disabled={!valid || pending} onClick={onConfirm}>
          {pending ? "Wiping…" : `Wipe ${v.id}`}
        </button>
      </div>
    </Modal>
  );
}

// WipeResultModal mirrors showWipeResult: an account of what actually
// happened, including the cascade — a restart nobody can see is the same
// silent behaviour this whole feature exists to make visible.
function WipeResultModal({
  v,
  svc,
  allViews,
  result,
  onClose,
}: {
  v: api.ContainerView | undefined;
  svc: api.ContainerServiceID;
  allViews: api.ContainerView[];
  result: api.WipeResult;
  onClose: () => void;
}) {
  const label = (id: string): string => allViews.find((s) => s.id === id)?.label ?? id;
  const lines: string[] = [];
  lines.push(result.report.ContainerRemoved ? "Container removed." : "There was no container to remove.");
  for (const vol of result.report.VolumesRemoved ?? []) lines.push(`Volume ${vol} deleted.`);
  for (const vol of result.report.VolumesAbsent ?? []) lines.push(`Volume ${vol} was already gone.`);
  if (result.report.Recreated) lines.push("Container re-created from your saved configuration.");

  const cascaded = (result.report.Cascaded ?? []).map(label);
  const skipped = (result.report.CascadeSkipped ?? []).map(label);

  return (
    <Modal onClose={onClose}>
      <h2>{v?.label ?? svc} wiped</h2>
      <ul className="plain-list">
        {lines.map((l, i) => (
          <li key={i}>{l}</li>
        ))}
      </ul>
      {cascaded.length > 0 && (
        <p className="ok">
          Restarted in front of it: {cascaded.join(", ")} — its cached head was cleared, so it now reports this
          chain&apos;s real height rather than the one from before the wipe.
        </p>
      )}
      {skipped.length > 0 && (
        <p className="muted small">
          Not restarted (they were not running, so they held no stale head): {skipped.join(", ")}.
        </p>
      )}
      {result.error && (
        <>
          <p className="error">
            The wipe itself succeeded, but a service in front of it could not be restarted — it is now serving a
            head this chain no longer has. Restart it by hand.
          </p>
          <p className="error small">{result.error}</p>
        </>
      )}
      <div className="modal-actions">
        <button className="btn" type="button" onClick={onClose}>
          Close
        </button>
      </div>
    </Modal>
  );
}
