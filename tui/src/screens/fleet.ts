// Fleet (D, slice part): one row per box; Enter opens node detail.
import type { Fleet, FleetRow } from "../api/types.ts";
import type { App, Screen } from "../shell/app.ts";
import type { Key } from "../shell/keys.ts";
import { boxHealth } from "../shell/statusbar.ts";
import { pad, sanitizeLine } from "../text.ts";

const COLS = [
  ["name", 14],
  ["gate", 9],
  ["sync", 12],
  ["peers", 8],
  ["disk", 16],
  ["state", 5],
] as const;

const gb = (b: number) => Math.round(b / 1e9);

function cells(r: FleetRow, stale: boolean): string[] {
  const s = stale || r.stale.status ? "~" : "";
  const st = r.status;
  const sync = st ? `${s}${st.overall}${st.headLag ? ` -${st.headLag}` : ""}` : "n/a";
  const peers = st ? `${s}${st.exec.peers}/${st.beacon.peers}` : "n/a";
  const d = r.disk;
  const ds = stale || r.stale.disk ? "~" : "";
  const disk = d ? `${ds}${gb(d.execBytes + d.beaconBytes)}/${gb(d.execBytes + d.beaconBytes + d.freeBytes)} GB` : "n/a";
  return [sanitizeLine(r.id), r.agent ? sanitizeLine(r.agent) : "n/a", sync, peers, disk, boxHealth(r)];
}

/** fleetRows is the table: a header, then one line per box. */
export function fleetRows(f: Fleet, stale: boolean): string[] {
  const line = (vals: string[]) =>
    vals.map((v, i) => (i === vals.length - 1 ? v : pad(v, COLS[i]![1]))).join(" ").trimEnd();
  return [line(COLS.map((c) => c[0])), ...f.rows.map((r) => line(cells(r, stale)))];
}

export class FleetScreen implements Screen {
  readonly id = "fleet";
  readonly title = "Fleet";
  private sel = 0;

  constructor(private readonly app: App) {}

  view(): string[] {
    const f = this.app.state.fleet;
    if (!this.app.state.conn) return ["Not connected."];
    if (!f) return ["Waiting for the fleet..."];
    if (f.rows.length === 0) return ["No boxes yet. Add one with `jumpgate hosts add`."];
    this.sel = Math.min(this.sel, f.rows.length - 1);
    const [head, ...rows] = fleetRows(f, this.app.state.fleetStale);
    // The vault state and each box's alerts are on the status strip and the
    // Alerts screen: the table does not repeat them.
    return ["  " + head, ...rows.map((r, i) => (i === this.sel ? "> " : "  ") + r)];
  }

  key(k: Key): boolean {
    const rows = this.app.state.fleet?.rows ?? [];
    if (!rows.length) return false;
    if (k.name === "up") return (this.sel = Math.max(0, this.sel - 1)), true;
    if (k.name === "down") return (this.sel = Math.min(rows.length - 1, this.sel + 1)), true;
    if (k.name === "return") {
      this.app.go("node", rows[this.sel]!.id);
      return true;
    }
    return false;
  }

  hints(): string {
    return "up/down select  enter open";
  }
}
