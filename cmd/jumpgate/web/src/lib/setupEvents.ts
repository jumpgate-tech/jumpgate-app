// setupEvents: how the web screens that follow a setup run (gateway and
// container provisioning) read its stream. The server ends every run with a
// terminal frame, {"stepId":"","done":true}, after the last step's own
// "run" done; it carries an error only when no step reported one (a canceled
// run). That frame is not a step: it never becomes a ": done" log line.
import type { SetupEvent } from "../api";

// FINAL_STEP is the id every gateway and container setup plan ends on.
export const FINAL_STEP = "run";

type Ev = Pick<SetupEvent, "stepId" | "line" | "done" | "err">;

// isTerminal: the run-ended frame, not a step's event.
export const isTerminal = (ev: Ev): boolean => ev.stepId === "" && !!ev.done;

// setupLine is the log line for one event, or null for the terminal frame
// with no error (the final step already said "run: done").
export function setupLine(ev: Ev): string | null {
  if (isTerminal(ev)) return ev.err ? `setup: ${ev.err}` : null;
  return ev.err ? `${ev.stepId}: ${ev.err}` : ev.line ? `${ev.stepId}: ${ev.line}` : `${ev.stepId}: done`;
}

// setupFinished: an error at any step, the final step reporting done, or the
// terminal frame (a run that ended before its final step).
export function setupFinished(ev: Ev): boolean {
  return !!ev.err || (ev.stepId === FINAL_STEP && !!ev.done) || isTerminal(ev);
}
