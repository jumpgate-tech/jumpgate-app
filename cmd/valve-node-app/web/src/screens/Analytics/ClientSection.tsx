// Section 1: "What your clients experienced" — from erpc_network_*, the only
// family the state poller does not touch. Port of analytics.ts's clientSection.
import type { GatewayAnalytics } from "../../api";
import { ChainCard } from "./ChainCard";
import type { Reading } from "./analyticsModel";

export function ClientSection({ data, history }: { data: GatewayAnalytics; history: Reading[] }) {
  const nets = data.networks ?? [];
  return (
    <section className="an-section">
      <h2>What your clients experienced</h2>
      <p className="muted small">
        Counted on the path a client&apos;s request actually takes. The gateway&apos;s own
        block-tracking poller does not appear here at all, which is what makes these numbers
        about your users rather than about the gateway.
      </p>
      {nets.length === 0 ? (
        <div className="card">
          <p className="muted">This gateway fronts no chains yet.</p>
        </div>
      ) : (
        nets.map((n) => <ChainCard key={n.chainId} n={n} history={history} />)
      )}
    </section>
  );
}
