// The drawer: running jobs and pending decisions. It never opens itself.
import type { RunningJob } from "../api/types.ts";
import { sanitizeLine, stepOf } from "../text.ts";

export function drawerLines(running: RunningJob[] | undefined): string[] {
  if (!running) return ["jobs n/a"];
  if (running.length === 0) return ["no jobs running"];
  return running.map((j) => {
    const mark = j.needsDecision ? "! " : "  ";
    const who = sanitizeLine(`${j.target}/${j.id}`);
    return `${mark}${who} ${sanitizeLine(j.kind)} ${stepOf(j.step, j.total)} ${sanitizeLine(j.state)}`;
  });
}

/** minimizedLine is the drawer folded into the hints row. */
export function minimizedLine(running: RunningJob[] | undefined, decisions: number | undefined): string {
  if (decisions && decisions > 0) return `[! ${decisions} needs decision]`;
  if (running && running.length > 0) return `[${running.length} running]`;
  return "";
}
