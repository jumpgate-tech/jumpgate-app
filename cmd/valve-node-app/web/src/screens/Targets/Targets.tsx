// #/targets — the local-machine card, the list of SSH targets, and the
// "add server over SSH" form. Port of targets.ts.
//
// Its `disposed` guard and manual load()/render() bookkeeping are gone:
// useTargets/useCatalog/useHost (React Query) own the "what's out there"
// state and its loading/error state, useAddTarget/useDeleteTarget (mutations
// that invalidate the targets list on success) own the write actions, and
// unmounting cancels in-flight work the framework's way. The delete
// confirmation is the shared <ConfirmDialog> (Panel/Dialogs.tsx) instead of
// ui.ts's imperative openModal/confirmModal.
import { useState } from "react";
import type * as api from "../../api";
import { Footer } from "../../components/Footer";
import { useAddTarget, useCatalog, useDeleteTarget, useHost, useTargets } from "../../hooks/target";
import { ConfirmDialog } from "../Panel/Dialogs";
import { AddOptions } from "./AddOptions";
import { FleetVerdict } from "./FleetVerdict";
import { SSHForm, type SSHFormValues } from "./SSHForm";
import { TargetCard } from "./TargetCard";
import { orderTargets, slugify } from "./targetsModel";

// This machine's fixed target id — it can only ever be added once.
const LOCAL_TARGET_ID = "local";

// formatFormError mirrors showFormError's message: an Error's own .message,
// or String(err) for anything else — distinct from the initial-load error
// below, which shows the raw String(err) the way every other converted
// screen's "Failed to load ..." line does.
function formatFormError(err: unknown): string {
  return err instanceof Error ? err.message : String(err);
}

export function Targets() {
  const targetsQuery = useTargets();
  const catalogQuery = useCatalog();
  const hostQuery = useHost();
  const addTarget = useAddTarget();
  const deleteTarget = useDeleteTarget();

  // The SSH add form is revealed on demand (the "Add a server" button)
  // rather than always shown, so the empty state stays a clean set of
  // guiding actions.
  const [showSSHForm, setShowSSHForm] = useState(false);
  const [formError, setFormError] = useState<string | null>(null);
  const [pendingRemoveId, setPendingRemoveId] = useState<string | null>(null);

  const loading = targetsQuery.isLoading || catalogQuery.isLoading || hostQuery.isLoading;
  const loadErr = targetsQuery.error ?? catalogQuery.error ?? hostQuery.error;
  const targets = targetsQuery.data;
  const catalog = catalogQuery.data;
  const hostOS = hostQuery.data?.os ?? "";

  // Local node setup needs a Linux host — gated on the SERVER's OS (what
  // valve-node-app runs on), not the browser's. On a non-Linux host local
  // setup can't complete, but the option is still offered (secondary, with a
  // caveat) rather than hidden, so it's never a dead end.
  const localViable = hostOS === "linux";
  const hasLocal = targets?.some((t) => t.mode === "local") ?? false;

  // No confirm() here: the option card states the constraint before you
  // click, and its button is explicitly labelled "Add anyway — preview the
  // wizard", so a modal would only interrupt to repeat what you just read.
  async function addLocal(): Promise<void> {
    setFormError(null);
    try {
      await addTarget.mutateAsync({ id: LOCAL_TARGET_ID, mode: "local" });
    } catch (err) {
      setFormError(formatFormError(err));
    }
  }

  async function addSSH(values: SSHFormValues): Promise<void> {
    const host = values.host.trim();
    const user = values.user.trim();
    const keyPath = values.keyPath.trim();
    const portRaw = values.port.trim();
    const idRaw = values.id.trim();

    setFormError(null);
    if (!host || !user || !keyPath) {
      setFormError("host, user, and key path are required");
      return;
    }

    const id = idRaw || slugify(host);
    const ssh: api.SSHConfig = { Host: host, User: user, KeyPath: keyPath };
    if (portRaw) {
      const port = Number.parseInt(portRaw, 10);
      if (!Number.isFinite(port) || port <= 0) {
        setFormError("port must be a positive number");
        return;
      }
      ssh.Port = port;
    }

    try {
      await addTarget.mutateAsync({ id, mode: "ssh", ssh });
      setShowSSHForm(false);
    } catch (err) {
      setFormError(formatFormError(err));
    }
  }

  async function confirmRemove(): Promise<void> {
    if (!pendingRemoveId) return;
    const id = pendingRemoveId;
    setPendingRemoveId(null);
    try {
      await deleteTarget.mutateAsync(id);
    } catch (err) {
      setFormError(formatFormError(err));
    }
  }

  const ordered = targets ? orderTargets(targets) : [];

  return (
    <>
      <h1>Machines</h1>
      <div>
        {loading ? (
          <p className="muted">Loading…</p>
        ) : loadErr ? (
          <p className="error">Failed to load machines: {String(loadErr)}</p>
        ) : targets && catalog ? (
          <>
            {formError && <p className="error">{formError}</p>}
            <FleetVerdict targets={targets} catalog={catalog} />
            <section className="section">
              <div className="section-head">
                <h2>Your machines</h2>
              </div>
              {ordered.length ? (
                <div className="card-grid">
                  {ordered.map((t) => (
                    <TargetCard
                      key={t.id}
                      target={t}
                      catalog={catalog}
                      canRunNode={t.mode !== "local" || localViable}
                      hostOS={hostOS}
                      onRemove={setPendingRemoveId}
                    />
                  ))}
                </div>
              ) : (
                <div className="card empty-state">
                  <p className="muted">No machines yet — pick an option below.</p>
                </div>
              )}
            </section>
            <section className="section">
              <div className="section-head">
                <h2>Add a machine</h2>
              </div>
              <AddOptions
                localViable={localViable}
                hasLocal={hasLocal}
                hostOS={hostOS}
                showSSHForm={showSSHForm}
                onToggleSSH={() => {
                  setFormError(null);
                  setShowSSHForm((v) => !v);
                }}
                onAddLocal={() => void addLocal()}
              />
              {showSSHForm && <SSHForm submitting={addTarget.isPending} onSubmit={(v) => void addSSH(v)} />}
            </section>
          </>
        ) : null}
      </div>
      <Footer />
      {pendingRemoveId && (
        <ConfirmDialog
          title="Remove machine"
          body={`Remove "${pendingRemoveId}"? This only removes it from valve-node-app — anything already running on the machine keeps running, and its data is left alone.`}
          confirmLabel="Remove"
          danger
          onConfirm={() => void confirmRemove()}
          onCancel={() => setPendingRemoveId(null)}
        />
      )}
    </>
  );
}
