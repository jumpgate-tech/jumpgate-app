// The health watch's settings: its endpoints (one name=url per line, or
// name=beacon+url for a beacon API) and an optional webhook. The saved
// webhook is a secret the server never sends back (target.watchWebhookSet
// says one exists), so an empty webhook field means "keep it" and clearing
// it is its own choice.
import type { CatalogModule, SetModuleRequest } from "../../api";

export const WEBHOOK_PLACEHOLDER = "https://ntfy.sh/your-topic";
export const WEBHOOK_HELP = "Off by default. HTTPS only. Each new and cleared alert is posted as JSON.";

export function endpointLines(text: string): string[] {
  return text
    .split("\n")
    .map((l) => l.trim())
    .filter((l) => l !== "");
}

// watchRequest builds the PUT body for turning the watch on or saving its
// settings. Throws the message to show when the webhook is not https.
export function watchRequest(endpoints: string, webhook: string, clearWebhook: boolean): SetModuleRequest {
  const body: SetModuleRequest = { enabled: true, upstreams: endpointLines(endpoints) };
  const hook = webhook.trim();
  if (clearWebhook) {
    body.clearWebhook = true;
  } else if (hook !== "") {
    if (!/^https:\/\/[^/\s]+/.test(hook)) throw new Error("The webhook must be an https:// URL.");
    body.webhook = hook;
  }
  return body;
}

// chainAgnostic: a box module the catalog lists for chain 0 only covers the
// whole box and takes no chain (the server refuses one).
export function chainAgnostic(m: CatalogModule | undefined): boolean {
  return !!m && m.applies.length > 0 && m.applies.every((r) => r.chainId === 0);
}
