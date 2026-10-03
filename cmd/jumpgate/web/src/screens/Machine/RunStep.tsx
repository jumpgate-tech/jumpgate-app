// Setup wizard → step 5: live progress through STEP_PLAN, driven by the
// setup event stream. Port of wizard.ts's own renderRunStep — the
// done/error/line derivations live in wizardModel.ts (doneStepIds,
// erroredStepIds, stepLines, stepErrorLine, runAllDone, runAnyError) so the
// finish condition (byte-exact with wizard.ts's own allDone) is unit-tested
// apart from this rendering.
import type { Network, SetupEvent } from "../../api";
import { Badge } from "../../components/Badge";
import {
  doneStepIds,
  erroredStepIds,
  runAllDone,
  runAnyError,
  STEP_PLAN,
  stepErrorLine,
  stepLines,
} from "./wizardModel";

export function RunStep({
  targetId,
  net,
  events,
  startError,
  onRetry,
}: {
  targetId: string;
  net: Network | undefined;
  events: SetupEvent[];
  startError: string | null;
  onRetry: () => void;
}) {
  const learnUrl = net?.LearnURL;
  const doneIds = doneStepIds(events);
  const erroredIds = erroredStepIds(events);
  const allDone = runAllDone(events);
  const anyError = runAnyError(events);

  return (
    <section>
      <h2>5. Running setup</h2>
      <ol className="step-list">
        {STEP_PLAN.map((s) => {
          const isDone = doneIds.has(s.id);
          const isError = erroredIds.has(s.id);
          const lines = stepLines(events, s.id);
          const errLine = stepErrorLine(events, s.id);
          return (
            <li key={s.id} className={`step-row ${isDone ? "step-done" : ""} ${isError ? "step-error" : ""}`}>
              <div className="step-head">
                {isError ? (
                  <Badge text="failed" kind="bad" />
                ) : isDone ? (
                  <Badge text="done" kind="ok" />
                ) : (
                  <Badge text="pending" kind="neutral" />
                )}{" "}
                <strong>{s.title}</strong>
              </div>
              {s.id === "handshake" && (
                <p className="muted small">
                  &quot;Talking&quot; means the beacon client can reach the execution client&apos;s Engine API
                  over the shared JWT secret and both report the same head — the sign your node is wired
                  correctly.
                  {learnUrl && (
                    <>
                      {" "}
                      <a href={learnUrl} target="_blank" rel="noopener noreferrer">
                        Learn more →
                      </a>
                    </>
                  )}
                </p>
              )}
              {lines.length > 0 && <pre className="step-log">{lines.join("\n")}</pre>}
              {errLine && <p className="error small">{errLine}</p>}
            </li>
          );
        })}
      </ol>
      {allDone && !anyError && (
        <p className="ok">
          Setup complete.{" "}
          <a href={`#/dash/${encodeURIComponent(targetId)}`}>Open the dashboard →</a>
        </p>
      )}
      {startError && <p className="error">{startError}</p>}
      {anyError && (
        <button className="btn" type="button" onClick={onRetry}>
          Retry setup
        </button>
      )}
    </section>
  );
}
