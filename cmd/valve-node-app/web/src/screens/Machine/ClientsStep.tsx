// Setup wizard → step 2: choose the execution/beacon client pair. Port of
// wizard.ts's own renderClientsStep. Only the clients the chosen network's
// catalog entry actually offers are selectable — the parent (SetupWizard)
// is responsible for keeping execId/beaconId synced to a valid choice
// whenever the network changes (wizardModel.resolveClientId), so this
// component can assume whatever it's handed is already valid.
import type { Catalog, CatalogClient } from "../../api";
import { clientOptionLabel, clientRepoDisplay } from "./wizardModel";

function ClientSourceLine({ client }: { client: CatalogClient | undefined }) {
  if (!client) return null;
  return (
    <p className="muted small">
      Source:{" "}
      <a href={client.repo} target="_blank" rel="noopener noreferrer">
        {clientRepoDisplay(client.repo)}
      </a>
    </p>
  );
}

export function ClientsStep({
  catalog,
  chainId,
  execId,
  beaconId,
  onExecChange,
  onBeaconChange,
  onBack,
  onNext,
}: {
  catalog: Catalog;
  chainId: number;
  execId: string | null;
  beaconId: string | null;
  onExecChange: (id: string) => void;
  onBeaconChange: (id: string) => void;
  onBack: () => void;
  onNext: () => void;
}) {
  const net = catalog.networks.find((n) => n.ChainID === chainId);
  if (!net) return <p className="error">Unknown network.</p>;

  const execClient = execId ? catalog.clients.find((c) => c.id === execId) : undefined;
  const beaconClient = beaconId ? catalog.clients.find((c) => c.id === beaconId) : undefined;

  return (
    <section>
      <h2>2. Choose your client pair</h2>
      <p className="muted">Only combinations known to work on {net.Name} are offered.</p>
      <p className="muted small">
        The <strong>provider</strong> shown for each client is the org that publishes it — some are the
        original upstream team, others are forks. Check the source if you only want to run a client from a
        particular team.
      </p>
      <label>
        Execution client
        <select value={execId ?? ""} onChange={(e) => onExecChange(e.target.value)}>
          {net.ExecClients.map((id) => (
            <option key={id} value={id}>
              {clientOptionLabel(id, catalog)}
            </option>
          ))}
        </select>
      </label>
      <ClientSourceLine client={execClient} />
      <label>
        Beacon client
        <select value={beaconId ?? ""} onChange={(e) => onBeaconChange(e.target.value)}>
          {net.BeaconClients.map((id) => (
            <option key={id} value={id}>
              {clientOptionLabel(id, catalog)}
            </option>
          ))}
        </select>
      </label>
      <ClientSourceLine client={beaconClient} />
      <div className="wizard-actions">
        <button className="btn btn-ghost" type="button" onClick={onBack}>
          Back
        </button>
        <button className="btn" type="button" onClick={onNext}>
          Next: mode
        </button>
      </div>
    </section>
  );
}
