// One leftover container a merge stopped managing but did NOT stop. It keeps
// its own strip (it belongs to no chain), states the exact `docker rm -f` to
// run by hand, and offers Dismiss — which forgets the record only, never
// touching the container (matching this app's rule that it never stops a
// container it did not start).
import type { OrphanedContainer } from "../../api";
import { AttentionLine } from "./AttentionStrip";
import { CopyButton } from "./CopyButton";

export function OrphanBanner({
  orphan,
  error,
  onDismiss,
}: {
  orphan: OrphanedContainer;
  error?: string | null;
  onDismiss: (name: string) => void;
}) {
  const cmd = `docker rm -f ${orphan.containerName}`;
  return (
    <div className="strip">
      <AttentionLine
        line={{
          tone: "warn",
          text:
            `${orphan.containerName} is still running on ${orphan.targetId}. Its chains were folded into ` +
            `${orphan.mergedInto}, but valve-node-app does not stop containers it did not start.`,
          cmd,
        }}
      />
      {error ? <AttentionLine line={{ tone: "bad", text: error }} /> : null}
      <div className="strip-line strip-note">
        <button className="btn btn-ghost btn-tiny" onClick={() => onDismiss(orphan.containerName)}>
          Dismiss this record
        </button>
        <span className="muted small">Forgets the record only — the container is never touched from here.</span>
        <CopyButton value={cmd} />
      </div>
    </div>
  );
}
