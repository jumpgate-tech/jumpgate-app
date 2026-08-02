// Pure helpers for the panel. No DOM, no api calls — unit-tested in panelModel.test.ts.

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
