// alerts: a port of internal/tui/alerts.go. Alerts are derived from the fleet
// (and gateways when known): FAIL first, then by box and text.
import type { Fleet } from "../api/types.ts";
import { maskText } from "../maskurl.ts";
import { sanitizeLine } from "../text.ts";

export type Sev = "warn" | "FAIL";
export interface Alert {
  sev: Sev;
  box: string;
  text: string;
}
export interface GatewayAlertInput {
  id: string;
  label?: string;
  status?: { State?: string };
  warnings?: string[];
}

const DISK_WARN = 85;
const DISK_FAIL = 95;
const HEAD_LAG_WARN = 64;
const alertText = (s: string) => maskText(sanitizeLine(s));

export function formatAge(ms: number): string {
  const s = Math.floor(ms / 1000);
  if (ms < 1000) return "0s";
  if (s < 60) return `${s}s`;
  if (s < 3600) return `${Math.floor(s / 60)}m`;
  if (s < 48 * 3600) return `${Math.floor(s / 3600)}h`;
  return `${Math.floor(s / 86400)}d`;
}

export function alerts(fleet: Fleet, now: Date, gateways?: GatewayAlertInput[]): Alert[] {
  const out: Alert[] = [];
  const add = (sev: Sev, box: string, text: string) => out.push({ sev, box: sanitizeLine(box), text });
  const every = fleet.intervalSeconds * 1000;
  for (const r of fleet.rows) {
    if (!r.reachable && r.lastSeen) {
      add("FAIL", r.id, `down ${formatAge(now.getTime() - Date.parse(r.lastSeen))}`);
      continue;
    }
    if (r.error && r.error.code !== "target_not_set_up") {
      let text = alertText(r.error.error);
      if (r.error.code) text = `${sanitizeLine(r.error.code)}: ${text}`;
      add("FAIL", r.id, text);
    }
    for (const [name, e] of [["status", r.statusError], ["disk", r.diskError], ["firewall", r.firewallError]] as const)
      if (e) add("FAIL", r.id, `${name}: ${alertText(e.error)}`);
    const at = r.status ? Date.parse(r.status.at) : NaN;
    const statusOld = r.status && at > 0 && every > 0 && now.getTime() - at > 2 * every;
    if (r.stale.status || statusOld) add("warn", r.id, "status stale");
    if (r.stale.disk) add("warn", r.id, "disk stale");
    if (r.stale.firewall) add("warn", r.id, "firewall stale");
    if (r.firewall?.grade === "fail") add("FAIL", r.id, "firewall fail");
    else if (r.firewall?.grade === "warn") add("warn", r.id, "firewall warn");
    if (r.disk?.fit === "short") add("FAIL", r.id, "disk short");
    else if (r.disk?.fit === "tight") add("warn", r.id, "disk tight");
    const st = r.status;
    if (st) {
      const p = st.diskUsedPct;
      if (p !== undefined && p >= DISK_WARN) add(p >= DISK_FAIL ? "FAIL" : "warn", r.id, `disk ${p.toFixed(0)}% used`);
      if (st.exec.state && st.exec.state !== "unavailable" && st.exec.peers === 0) add("warn", r.id, "exec peers 0");
      if (st.headLag !== undefined && st.headLag > HEAD_LAG_WARN) add("warn", r.id, `head lag ${st.headLag} blk`);
    }
    if (r.jobs && r.jobs.needsDecision > 0) add("warn", r.id, `${r.jobs.needsDecision} needs decision`);
  }
  for (const g of gateways ?? []) {
    const name = g.label || g.id;
    const state = g.status?.State;
    if (state && state !== "running") add("FAIL", name, `gateway ${alertText(state)}`);
    for (const w of g.warnings ?? []) add("warn", name, alertText(w));
  }
  const rank = (s: Sev) => (s === "FAIL" ? 1 : 0);
  return out.sort((a, b) => rank(b.sev) - rank(a.sev) || a.box.localeCompare(b.box) || a.text.localeCompare(b.text));
}

export const alertLine = (a: Alert) => `${a.sev === "FAIL" ? "FAIL" : "warn"} ${a.box}: ${a.text}`;
