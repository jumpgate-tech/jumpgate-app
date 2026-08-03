// Setup wizard → step 4: review the WireConfig about to be POSTed, plus the
// fixed step sequence setup always runs. Port of wizard.ts's own
// renderReviewStep.
//
// DEVIATION FROM THE BRIEF (carried over from wizard.ts, see its own header
// comment): there is no API that renders the exact units ahead of time, so
// the step list below is STEP_PLAN — a client-side mirror of
// internal/setup/steps.go's fixed sequence — not something the API returns.
import type { Network } from "../../api";
import { nonDefaultPorts, parseBindAddr, STEP_PLAN } from "./wizardModel";

export function ReviewStep({
  targetId,
  net,
  chainId,
  execId,
  beaconId,
  archive,
  dataDir,
  jwtPath,
  checkpoint,
  checkpointUrl,
  execHTTPPort,
  beaconHTTPPort,
  execP2PPort,
  rpcBindAddr,
  startError,
  starting,
  onBack,
  onStart,
}: {
  targetId: string;
  net: Network | undefined;
  chainId: number;
  execId: string;
  beaconId: string;
  archive: boolean;
  dataDir: string;
  jwtPath: string;
  checkpoint: boolean;
  checkpointUrl: string;
  execHTTPPort: string;
  beaconHTTPPort: string;
  execP2PPort: string;
  rpcBindAddr: string;
  startError: string | null;
  starting: boolean;
  onBack: () => void;
  onStart: () => void;
}) {
  const ports = nonDefaultPorts({ execHTTPPort, beaconHTTPPort, execP2PPort });
  const { addr: boundAddr } = parseBindAddr(rpcBindAddr);

  return (
    <section>
      <h2>4. Review</h2>
      <table className="review-table">
        <tbody>
          <tr>
            <th>Target</th>
            <td>{targetId}</td>
          </tr>
          <tr>
            <th>Network</th>
            <td>
              {net?.Name ?? String(chainId)} (chain {chainId})
            </td>
          </tr>
          <tr>
            <th>Execution client</th>
            <td>{execId}</td>
          </tr>
          <tr>
            <th>Beacon client</th>
            <td>{beaconId}</td>
          </tr>
          <tr>
            <th>Mode</th>
            <td>{archive ? "Archive" : "Full"}</td>
          </tr>
          <tr>
            <th>Data directory</th>
            <td>
              <code>{dataDir}</code>
            </td>
          </tr>
          <tr>
            <th>JWT secret path</th>
            <td>
              <code>{jwtPath}</code>
            </td>
          </tr>
          <tr>
            <th>Checkpoint sync</th>
            <td>{checkpoint ? <code>{checkpointUrl || net?.CheckpointURL || ""}</code> : "off — syncing from genesis"}</td>
          </tr>
          {ports.length > 0 && (
            <tr>
              <th>Non-default ports</th>
              <td>{ports.map((p) => `${p.label} ${p.port}`).join(", ")}</td>
            </tr>
          )}
          {boundAddr && (
            <tr>
              <th>RPC bind address</th>
              <td>
                <code>{boundAddr}</code>{" "}
                <span className="muted">(reachable off-box — unauthenticated, keep it on a trusted overlay)</span>
              </td>
            </tr>
          )}
        </tbody>
      </table>
      <p className="muted small">
        There is no preview API for the exact files/units that will be written — the list below is the fixed
        step sequence setup always runs; the actual commands and file contents stream live once you start.
      </p>
      <ol className="step-preview">
        {STEP_PLAN.map((s) => (
          <li key={s.id}>{s.title}</li>
        ))}
      </ol>
      {startError && <p className="error">{startError}</p>}
      <div className="wizard-actions">
        <button className="btn btn-ghost" type="button" onClick={onBack}>
          Back
        </button>
        <button className="btn btn-primary" type="button" disabled={starting} onClick={onStart}>
          {starting ? "Starting…" : "Start setup"}
        </button>
      </div>
    </section>
  );
}
