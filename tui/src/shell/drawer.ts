// The drawer: running jobs and pending decisions. It never opens itself.
import type { RunningJob } from "../api/types.ts";
import { sanitizeLine } from "../text.ts";

export function drawerLines(running: RunningJob[] | undefined): string[] {
  if (!running) return ["n/a"];
  if (running.length === 0) return ["none running"];
  return running.map((j) => {
    const mark = j.needsDecision ? "! " : "  ";
    const who = sanitizeLine(`${j.target}/${j.id}`);
    return `${mark}${who} ${sanitizeLine(j.kind)} step ${j.step + 1}/${j.total} ${sanitizeLine(j.state)}`;
  });
}

/**
 * minimizedLine is the drawer folded into the hints row: how many jobs run.
 * Pending decisions are counted once, on the status strip.
 */
export function minimizedLine(running: RunningJob[] | undefined): string {
  return running && running.length > 0 ? `[${running.length} running]` : "";
}
