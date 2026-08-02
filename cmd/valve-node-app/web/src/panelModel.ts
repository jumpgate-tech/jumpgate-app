// Pure helpers for the panel. No DOM, no api calls — unit-tested in panelModel.test.ts.

import type { GatewayView } from "./api";

// endpointNameFromUrl derives a friendly default name from an endpoint URL:
// the second-level domain label ("publicnode" from rpc.publicnode.com), or the
// bare host for localhost / IPs, or "endpoint" if the URL won't parse.
export function endpointNameFromUrl(endpoint: string): string {
  let host: string;
  try {
    host = new URL(endpoint).hostname;
  } catch {
    return "endpoint";
  }
  if (!host) return "endpoint";
  if (host === "localhost" || /^[0-9.]+$/.test(host) || /^\[.*\]$/.test(host)) return host;
  const parts = host.split(".").filter(Boolean);
  if (parts.length <= 1) return host;
  // second-level label: publicnode.com → "publicnode"; infura.io → "infura".
  return parts[parts.length - 2];
}

export type PowerTone = "on" | "off" | "blocked";
export interface MasterState { tone: PowerTone; label: string; sub: string; actions: string[]; blocked?: string; }

// masterState maps a gateway to the master power button. tone drives the color
// (green/red/grey); actions/blocked come straight from the server so the button
// never offers an impossible transition.
export function masterState(gw: GatewayView | null): MasterState {
  if (!gw) return { tone: "off", label: "Not set up", sub: "Press to set up your endpoint", actions: [] };
  const actions = gw.actions ?? [];
  if (gw.blocked) return { tone: "blocked", label: "Unavailable", sub: gw.blocked, actions, blocked: gw.blocked };
  const count = gw.networks?.length ?? 0;
  if (gw.status.State === "running") {
    return { tone: "on", label: "Running", sub: `${count} network${count === 1 ? "" : "s"} served`, actions };
  }
  return { tone: "off", label: "Stopped", sub: count ? `${count} network${count === 1 ? "" : "s"} configured` : "Press to start", actions };
}

export type HealthClass = "stable" | "occasional" | "frequent" | "off";

// healthClass turns coarse signals into the dot's motion. Stillness = health, so
// a serviceable endpoint with rare slow requests is "stable" (no animation).
// Motion frequency tracks the slow-request rate. Thresholds: <10% stable,
// 10–40% occasional, >40% frequent. Not-serviceable while running is "frequent".
export function healthClass(input: { running: boolean; serviceable: boolean; slowRate?: number }): HealthClass {
  if (!input.running) return "off";
  if (!input.serviceable) return "frequent";
  const r = input.slowRate ?? 0;
  if (r > 0.4) return "frequent";
  if (r >= 0.1) return "occasional";
  return "stable";
}

export interface CapCell { key: string; label: string; lit: boolean; hot: boolean; }
const CAP_ORDER: { key: string; label: string; hot?: boolean }[] = [
  { key: "http", label: "HTTP" }, { key: "ws", label: "WS" },
  { key: "archive", label: "Archive", hot: true }, { key: "trace", label: "Trace" },
];

// capabilityCells folds probed capability statuses into the fixed-order meter.
// A cell is "lit" when supported; "hot" marks the standout (archive) when lit.
export function capabilityCells(statuses: Record<string, string>): CapCell[] {
  return CAP_ORDER.map(({ key, label, hot }) => {
    const lit = statuses[key] === "supported";
    return { key, label, lit, hot: !!hot && lit };
  });
}
