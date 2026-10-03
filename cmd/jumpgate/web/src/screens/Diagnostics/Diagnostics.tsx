// #/diag/:id — the network diagnostics ladder from internal/ops's
// NetworkDiagnostics. Checks run in order and stop at the first failure, so
// the report reads "check, check, check — failed HERE". Runs happen two
// ways: automatically on the server when a journal error signature fires or
// a connection fails (inactive service, zero peers), and manually from the
// button here. This screen shows the latest report either way. Every probe
// is read-only on the server side — this screen never sends a mutating
// command, only ever copies a suggested fix to the clipboard for the
// operator to review and run themselves.
//
// Port of diag.ts. Its `disposed` guard and manual report/loaded/running
// bookkeeping are gone: useTargets/useCatalog + useLatestDiagnostics (React
// Query) own the "what's the latest report" state and its loading/error
// state, useRunDiagnostics (a mutation that writes its result straight into
// the latest-report query's cache) owns the "run it now" action, and
// unmounting cancels in-flight work the framework's way.
import { useParams } from "react-router-dom";
import { useCatalog, useTargets } from "../../hooks/target";
import { useLatestDiagnostics, useRunDiagnostics } from "../../hooks/diagnostics";
import { Footer } from "../../components/Footer";
import { DiagCheckList } from "./DiagCheckList";
import { failedCheckTitle } from "./diagModel";

function formatError(err: unknown): string {
  return err instanceof Error ? err.message : String(err);
}

export function Diagnostics() {
  const { id } = useParams<{ id: string }>();
  const targetId = id ?? "";

  const targetsQuery = useTargets();
  const catalogQuery = useCatalog();
  const initLoading = targetsQuery.isLoading || catalogQuery.isLoading;
  const initErr = targetsQuery.error ?? catalogQuery.error;

  const target = targetsQuery.data?.find((t) => t.id === targetId);
  const wired = !!target?.wire;

  const latestQuery = useLatestDiagnostics(targetId, wired);
  const runMutation = useRunDiagnostics(targetId);
  const running = runMutation.isPending;

  const report = latestQuery.data;
  // `loaded` mirrors diag.ts's own `loaded` flag: true once a report has
  // ever loaded (report can legitimately be `null` — "no run yet" is not an
  // error). React Query keeps the last successful `data` across a failed
  // refetch, exactly like the legacy flag that was only ever set, never
  // reset.
  const loaded = report !== undefined;
  // Once a manual run has been attempted, its own error (or lack of one)
  // fully replaces the initial fetch's error — mirroring run()'s
  // `loadErr = null` at the top of every run, same as diag.ts sharing one
  // `loadErr` variable between init() and run().
  const loadErr = runMutation.isIdle
    ? latestQuery.error
      ? formatError(latestQuery.error)
      : null
    : runMutation.isError
      ? formatError(runMutation.error)
      : null;

  const net =
    wired && target?.wire
      ? catalogQuery.data?.networks.find((n) => n.ChainID === target.wire!.ChainID)
      : undefined;

  return (
    <>
      <h1>Network diagnostics: {targetId}</h1>
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
                Checks run in order and stop at the first failure — the last item is where your
                node&apos;s network stack breaks. Diagnostics also run automatically when an error
                shows up in the logs or a connection fails (service down, zero peers); the latest
                result is shown here. All probes are read-only — nothing is ever changed
                automatically.
              </p>
              <button className="btn" type="button" disabled={running} onClick={() => runMutation.mutate()}>
                {running ? "Running…" : "Run diagnostics"}
              </button>
            </div>
            {loadErr && <p className="error">{loadErr}</p>}
            {!loaded && !loadErr ? (
              <p className="muted">Loading…</p>
            ) : !report ? (
              <p className="muted">
                No diagnostics have run yet for this target. Run them now, or they&apos;ll run on
                their own the next time something goes wrong.
              </p>
            ) : (
              <>
                <p className="muted small">
                  Last run {new Date(report.at).toLocaleString()} — trigger: {report.trigger}
                </p>
                {report.failedId ? (
                  <p>
                    <strong>Failed at: {failedCheckTitle(report)}.</strong>{" "}
                    <span className="muted small">
                      Later checks were skipped — fix this first, then re-run.
                    </span>
                  </p>
                ) : (
                  <p>
                    <strong>All checks passed.</strong>
                  </p>
                )}
                <DiagCheckList report={report} />
              </>
            )}
          </>
        )}
      </div>
      <Footer contextLabel={net?.Name} contextUrl={net?.LearnURL} />
    </>
  );
}
