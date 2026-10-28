import type { Args } from "../main.ts";

// start runs the dashboard. Filled in by the shell task.
export async function start(_args: Args): Promise<number> {
  process.stderr.write("jumpgate-tui: the dashboard is not built yet\n");
  return 1;
}
