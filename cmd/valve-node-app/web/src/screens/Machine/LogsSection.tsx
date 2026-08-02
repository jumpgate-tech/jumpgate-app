// Machine page → Logs section — live log tail with severity coloring, an
// error feed panel, and an "Explain with AI" flow gated by a once-per-browser
// localStorage consent. Port of logs.ts's renderLogs.
//
// renderLogs's `disposed` guard and manual hits/streamStop bookkeeping become
// useTargets/useCatalog (React Query, shared with every other target-scoped
// screen — see hooks/target.ts) plus useLogStream, this section's own
// fetch-then-stream hook (hooks/logs.ts). The explain flow's imperative
// openModal/closeModal calls become one local state machine driving <Modal>
// dialogs, the same replacement Panel/Dialogs.tsx made for panel.ts.
//
// NOT yet wired into the Machine page — the machine composer task mounts
// this. It still renders its own <h1>/<Footer>, exactly like the legacy
// module does today inside machine.ts's section body; stripping those so the
// page reads as one document is the follow-up machine.ts's own comment calls
// out, not this task.
import { useEffect, useRef, useState } from "react";
import * as api from "../../api";
import { useCatalog, useTargets } from "../../hooks/target";
import { useLogStream } from "../../hooks/logs";
import { Badge } from "../../components/Badge";
import { Footer } from "../../components/Footer";
import { Modal } from "../Panel/Modal";
import {
  errorHits,
  explainCandidateLines,
  formatLogTime,
  hasExplainConsent,
  setExplainConsent,
  severityOf,
} from "./logsModel";

type ExplainState =
  | { kind: "closed" }
  | { kind: "consent"; lines: string[] }
  | { kind: "loading" }
  | { kind: "result"; text: string; sentExcerpt: string[] }
  | { kind: "no-provider" }
  | { kind: "error"; message: string };

function formatError(err: unknown): string {
  return err instanceof Error ? err.message : String(err);
}

export function LogsSection({ targetId }: { targetId: string }) {
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

  const { hits, loading: logsLoading, error: logsError } = useLogStream(targetId, wired);

  const [explainState, setExplainState] = useState<ExplainState>({ kind: "closed" });
  // mountedRef mirrors renderLogs's own `disposed` flag: guards against
  // writing explain-flow state after this section unmounted (the operator
  // collapsed the section, or navigated away) while a request was in flight.
  const mountedRef = useRef(true);
  useEffect(() => {
    return () => {
      mountedRef.current = false;
    };
  }, []);

  // tailRef mirrors renderAll's `tail.scrollTop = tail.scrollHeight`: keep
  // the live tail pinned to its newest line as hits arrive.
  const tailRef = useRef<HTMLDivElement>(null);
  useEffect(() => {
    const el = tailRef.current;
    if (el) el.scrollTop = el.scrollHeight;
  }, [hits]);

  function openExplainFlow(): void {
    const lines = explainCandidateLines(hits);
    if (!hasExplainConsent()) {
      setExplainState({ kind: "consent", lines });
      return;
    }
    void runExplain(lines);
  }

  async function runExplain(lines: string[]): Promise<void> {
    setExplainState({ kind: "loading" });
    try {
      const res = lines.length ? await api.explain(targetId, lines) : await api.explain(targetId);
      if (!mountedRef.current) return;
      setExplainState({ kind: "result", text: res.text, sentExcerpt: res.sentExcerpt });
    } catch (err) {
      if (!mountedRef.current) return;
      if (err instanceof api.ApiError && err.status === 409) {
        setExplainState({ kind: "no-provider" });
        return;
      }
      setExplainState({ kind: "error", message: formatError(err) });
    }
  }

  function proceedConsent(lines: string[]): void {
    setExplainConsent();
    void runExplain(lines);
  }

  function closeExplain(): void {
    setExplainState({ kind: "closed" });
  }

  const errs = errorHits(hits);

  return (
    <>
      <h1>Logs: {targetId}</h1>
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
        ) : logsError ? (
          <p className="error">Failed to load logs: {logsError}</p>
        ) : logsLoading ? (
          <p className="muted">Loading…</p>
        ) : (
          <div className="logs-layout">
            <section className="logs-tail">
              <div className="logs-tail-head">
                <h2>Live tail</h2>
                <button className="btn" type="button" onClick={openExplainFlow}>
                  Explain with AI
                </button>
              </div>
              <div className="log-lines" ref={tailRef}>
                {hits.map((h, i) => (
                  <LogLine key={`${i}-${h.signature}`} hit={h} />
                ))}
              </div>
            </section>
            <section className="logs-errors">
              <h2>
                Error feed <Badge text={String(errs.length)} kind={errs.length ? "bad" : "neutral"} />
              </h2>
              <div className="log-lines">
                {errs.length ? (
                  errs
                    .slice()
                    .reverse()
                    .map((h, i) => <LogLine key={`${i}-${h.signature}`} hit={h} />)
                ) : (
                  <p className="muted">No errors seen yet.</p>
                )}
              </div>
            </section>
          </div>
        )}
      </div>
      <Footer contextLabel={net?.Name} contextUrl={net?.LearnURL} />
      {explainState.kind === "consent" && (
        <ExplainConsentModal
          lines={explainState.lines}
          onCancel={closeExplain}
          onProceed={() => proceedConsent(explainState.lines)}
        />
      )}
      {explainState.kind === "loading" && (
        <Modal onClose={() => {}}>
          <h2>Explain with AI</h2>
          <p className="muted">Asking the AI provider…</p>
        </Modal>
      )}
      {explainState.kind === "result" && (
        <ExplainResultModal
          text={explainState.text}
          sentExcerpt={explainState.sentExcerpt}
          onClose={closeExplain}
        />
      )}
      {explainState.kind === "no-provider" && (
        <Modal onClose={closeExplain}>
          <h2>No AI provider configured</h2>
          <p>
            Set a provider and key in <a href="#/settings">Settings</a>, then try again.
          </p>
          <div className="modal-actions">
            <button className="btn" onClick={closeExplain}>
              Close
            </button>
          </div>
        </Modal>
      )}
      {explainState.kind === "error" && (
        <Modal onClose={closeExplain}>
          <h2>Explain failed</h2>
          <p className="error">{explainState.message}</p>
          <div className="modal-actions">
            <button className="btn" onClick={closeExplain}>
              Close
            </button>
          </div>
        </Modal>
      )}
    </>
  );
}

function LogLine({ hit }: { hit: api.Hit }) {
  const sev = severityOf(hit);
  return (
    <div className={`log-line log-${sev}`}>
      <span className="log-time">{formatLogTime(hit.at)}</span>
      <span className="log-unit">{hit.unit}</span>
      <span className="log-sev">{sev}</span>
      <span className="log-text">{hit.line}</span>
      {hit.explain && (
        <div className="log-explain">
          {hit.explain}
          {hit.learnUrl && (
            <>
              {" "}
              <a href={hit.learnUrl} target="_blank" rel="noopener noreferrer">
                learn →
              </a>
            </>
          )}
        </div>
      )}
    </div>
  );
}

function ExplainConsentModal({
  lines,
  onCancel,
  onProceed,
}: {
  lines: string[];
  onCancel: () => void;
  onProceed: () => void;
}) {
  return (
    <Modal onClose={onCancel}>
      <h2>Send logs to your AI provider?</h2>
      <p>
        The excerpt below will be sent to the AI provider configured in <a href="#/settings">Settings</a> to
        generate a plain-English explanation. This happens every time you click &quot;Explain with AI&quot;;
        this confirmation only shows once per browser.
      </p>
      {lines.length ? (
        <pre className="explain-excerpt">{lines.join("\n")}</pre>
      ) : (
        <p className="muted">
          No recent error lines are loaded yet — the server will auto-select its own recent error/critical
          lines instead.
        </p>
      )}
      <div className="modal-actions">
        <button className="btn btn-ghost" onClick={onCancel}>
          Cancel
        </button>
        <button className="btn btn-primary" onClick={onProceed}>
          Send to AI provider
        </button>
      </div>
    </Modal>
  );
}

function ExplainResultModal({
  text,
  sentExcerpt,
  onClose,
}: {
  text: string;
  sentExcerpt: string[];
  onClose: () => void;
}) {
  return (
    <Modal onClose={onClose}>
      <h2>Explanation</h2>
      <div className="explain-text">{text}</div>
      <details className="advanced">
        <summary>What was sent</summary>
        <pre className="explain-excerpt">
          {sentExcerpt.length ? sentExcerpt.join("\n") : "(no log lines — general question only)"}
        </pre>
      </details>
      <div className="modal-actions">
        <button className="btn" onClick={onClose}>
          Close
        </button>
      </div>
    </Modal>
  );
}
