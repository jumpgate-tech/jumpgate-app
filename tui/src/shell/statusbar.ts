// The status strip: server <ver> [skew!] | vault <state> | boxes ... | ! N decisions
import type { Fleet, FleetRow } from "../api/types.ts";
import { sanitizeLine } from "../text.ts";

export type BoxHealth = "ok" | "warn" | "down";

export function boxHealth(r: FleetRow): BoxHealth {
  if (!r.reachable) return "down";
  if (r.error || r.statusError || r.alertCount > 0 || r.stale.status) return "warn";
  return "ok";
}

export interface StatusInput {
  connected: boolean;
  serverVersion?: string;
  myVersion: string;
  vaultState?: string;
  fleet?: Fleet;
  fleetStale?: boolean;
  decisions?: number;
}

export function statusLine(s: StatusInput): string {
  if (!s.connected) return "jumpgate | not connected";
  const ver = s.serverVersion === undefined ? "n/a" : sanitizeLine(s.serverVersion);
  const skew = s.serverVersion !== undefined && s.serverVersion !== s.myVersion ? " [skew!]" : "";
  const vault = s.vaultState === undefined ? "n/a" : sanitizeLine(s.vaultState);
  let boxes = "boxes n/a";
  if (s.fleet) {
    const n = { ok: 0, warn: 0, down: 0 };
    for (const r of s.fleet.rows) n[boxHealth(r)]++;
    boxes = `boxes ${s.fleet.rows.length} ${n.ok} ok/${n.warn} warn/${n.down} down${s.fleetStale ? " ~" : ""}`;
  }
  const dec = s.decisions === undefined ? "! n/a decisions" : `! ${s.decisions} decisions`;
  return `server ${ver}${skew} | vault ${vault} | ${boxes} | ${dec}`;
}
