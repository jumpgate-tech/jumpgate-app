// #/security/:id — the firewall/exposure checklist from internal/ops's
// FirewallChecklist: status chips, expandable why/fix per item (with a copy
// button on the fix command), and a "Re-run checks" button that re-probes
// live. Every probe here is read-only on the server side — this screen never
// sends a mutating command, only ever copies a suggested one to the
// clipboard for the operator to review and run themselves.
//
// Port of security.ts. Its `disposed` guard, manual load()/render()
// bookkeeping and innerHTML templating are gone: useTargets/useCatalog +
// useFirewallChecklist (React Query) own the data and its loading/error
// state, and unmounting cancels in-flight work the framework's way.
import { useParams } from "react-router-dom";
import { useCatalog, useTargets } from "../../hooks/target";
import { useFirewallChecklist } from "../../hooks/security";
import { Footer } from "../../components/Footer";
import { CheckList } from "./CheckList";

function formatError(err: unknown): string {
  return err instanceof Error ? err.message : String(err);
}

export function Security() {
  const { id } = useParams<{ id: string }>();
  const targetId = id ?? "";

  const targetsQuery = useTargets();
  const catalogQuery = useCatalog();
  const initLoading = targetsQuery.isLoading || catalogQuery.isLoading;
  const initErr = targetsQuery.error ?? catalogQuery.error;

  const target = targetsQuery.data?.find((t) => t.id === targetId);
  const wired = !!target?.wire;

  const checklistQuery = useFirewallChecklist(targetId, wired);
  const items = checklistQuery.data;
  // `loaded` mirrors security.ts's own `loaded` flag: true once a checklist
  // has ever loaded, and — since React Query keeps the last successful
  // `data` across a failed refetch — it STAYS true through a re-run that
  // fails, exactly like the legacy flag that was only ever set, never reset.
  const loaded = items !== undefined;
  const loading = checklistQuery.isLoading;
  const fetching = checklistQuery.isFetching;
  const loadErr = checklistQuery.error ? formatError(checklistQuery.error) : null;

  const net =
    wired && target?.wire
      ? catalogQuery.data?.networks.find((n) => n.ChainID === target.wire!.ChainID)
      : undefined;

  return (
    <>
      <h1>Security: {targetId}</h1>
      <div>
        {initLoading ? (
          <p className="muted">Loading…</p>
        ) : initErr ? (
          <p className="error">Failed to load target: {String(initErr)}</p>
        ) : !target ? (
          <p className="error">
            Target &quot;{targetId}&quot; not found. <a href="#/targets">Back to targets</a>
          </p>
        ) : !target.wire ? (
          <p className="muted">
            This target hasn&apos;t completed setup yet.{" "}
            <a href={`#/setup/${encodeURIComponent(targetId)}`}>Run the setup wizard →</a>
          </p>
        ) : (
          <>
            <p>
              <a href={`#/dash/${encodeURIComponent(targetId)}`}>← Back to dashboard</a>
            </p>
            <div className="section-head">
              <p className="muted small">
                Every check here is a live, read-only probe run on the target — nothing is ever
                changed automatically. Each &quot;Fix&quot; is a copy-paste command for you to review
                and run yourself.
              </p>
              <button
                className="btn"
                type="button"
                disabled={fetching}
                onClick={() => void checklistQuery.refetch()}
              >
                {fetching ? "Re-running…" : "Re-run checks"}
              </button>
            </div>
            {loadErr && <p className="error">{loadErr}</p>}
            {!loaded && loading ? (
              <p className="muted">Loading…</p>
            ) : items && items.length > 0 ? (
              <CheckList items={items} />
            ) : loaded ? (
              <p className="muted">No checks returned.</p>
            ) : null}
          </>
        )}
      </div>
      <Footer contextLabel={net?.Name} contextUrl={net?.LearnURL} />
    </>
  );
}
