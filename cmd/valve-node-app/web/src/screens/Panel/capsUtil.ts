// Pure folds over a gateway's probed capabilities — ported from panel.ts. No
// DOM, no api calls: given the cached probe, they answer "what can this chain
// (or this one endpoint) actually do", which the detail screens' capability
// meters render.
import type { EndpointCapabilities, GatewayCapabilities, CapabilityStatus } from "../../api";

const CAP_KEYS = ["http", "ws", "archive", "trace"];

// capStatusOf mirrors rpc.ts's statusOf: there is no "http" probe result
// because answering JSON-RPC over HTTP IS reachability, recorded as
// EndpointCapabilities.reachable rather than an eleventh method call.
export function capStatusOf(e: EndpointCapabilities, key: string): CapabilityStatus | undefined {
  if (key === "http") {
    if (e.unprobeable) return "inconclusive";
    return e.reachable ? "supported" : "unsupported";
  }
  return (e.capabilities ?? []).find((c) => c.key === key)?.status;
}

// unionCapabilities folds every probed upstream on this chain into one verdict
// per capability: supported if ANY upstream on the chain supports it — the
// network-level question a balanced/failover front answers with the best of its
// upstreams, not the worst.
export function unionCapabilities(
  caps: GatewayCapabilities | null | undefined,
  chainId: number,
  upstreamIds: string[],
): Record<string, string> {
  const endpoints = (caps?.endpoints ?? []).filter(
    (e) => e.chainId === chainId && upstreamIds.includes(e.upstream),
  );
  const out: Record<string, string> = {};
  for (const key of CAP_KEYS) {
    if (endpoints.some((e) => capStatusOf(e, key) === "supported")) out[key] = "supported";
  }
  return out;
}

// hasProbeFor reports whether the cached probe actually covers this target —
// this one upstream when upstreamId is given, or any upstream on the chain
// otherwise. It distinguishes "probed and supports nothing" (a real verdict)
// from "not probed yet" (a freshly-added endpoint the last probe never saw),
// which otherwise both render as an alarming all-unavailable meter.
export function hasProbeFor(
  caps: GatewayCapabilities | null | undefined,
  chainId: number,
  upstreamId?: string,
): boolean {
  return (caps?.endpoints ?? []).some(
    (e) => e.chainId === chainId && (upstreamId === undefined || e.upstream === upstreamId),
  );
}

// singleCapabilities is unionCapabilities narrowed to exactly one upstream — no
// folding, because there is only one endpoint's own verdict to show.
export function singleCapabilities(
  caps: GatewayCapabilities | null | undefined,
  chainId: number,
  upstreamId: string,
): Record<string, string> {
  const e = (caps?.endpoints ?? []).find((c) => c.chainId === chainId && c.upstream === upstreamId);
  if (!e) return {};
  const out: Record<string, string> = {};
  for (const key of CAP_KEYS) {
    if (capStatusOf(e, key) === "supported") out[key] = "supported";
  }
  return out;
}
