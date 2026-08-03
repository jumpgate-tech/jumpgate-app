// Setup wizard → step 1: choose a network. Port of wizard.ts's own
// renderNetworkStep: one selectable card per network in NETWORK_ORDER that
// the catalog actually has, tagged with NETWORK_BADGE where the wizard
// wants an operator-facing hint (369 as the default, 943 as "practise here
// first").
import type { Catalog } from "../../api";
import { Badge, type BadgeKind } from "../../components/Badge";
import { NETWORK_BADGE, NETWORK_ORDER } from "./wizardModel";

export function NetworkStep({
  catalog,
  chainId,
  onPick,
  onNext,
}: {
  catalog: Catalog;
  chainId: number | null;
  onPick: (chainId: number) => void;
  onNext: () => void;
}) {
  return (
    <section>
      <h2>1. Choose a network</h2>
      <div className="card-grid">
        {NETWORK_ORDER.map((id) => {
          const net = catalog.networks.find((n) => n.ChainID === id);
          if (!net) return null;
          const selected = chainId === id;
          const tag = NETWORK_BADGE[id];
          return (
            <button
              key={id}
              type="button"
              className={`card card-selectable ${selected ? "selected" : ""}`}
              onClick={() => onPick(id)}
            >
              <h3>
                {net.Name} <span className="muted">(chain {id})</span>
              </h3>
              {tag && <Badge text={tag} kind={(id === 369 ? "ok" : "warn") as BadgeKind} />}
            </button>
          );
        })}
      </div>
      <div className="wizard-actions">
        <button className="btn" type="button" disabled={chainId === null} onClick={onNext}>
          Next: clients
        </button>
      </div>
    </section>
  );
}
