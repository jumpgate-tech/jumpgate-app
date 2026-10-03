// Setup wizard → step 3: sync mode (checkpoint sync, execution snapshot,
// full vs. archive, data location, and the advanced port/path overrides).
// Port of wizard.ts's own renderModeStep — the single biggest step, so its
// blocks are split into focused subcomponents below rather than one giant
// JSX tree.
import type { Network } from "../../api";
import { fmtBytes } from "../../ui";
import {
  approxSize,
  archiveTierTB,
  ARCHIVE_SIZE_BASIS,
  checkFit,
  DEFAULT_BEACON_HTTP_PORT,
  DEFAULT_EXEC_HTTP_PORT,
  DEFAULT_EXEC_P2P_PORT,
  FULL_SIZE_BASIS,
  fullTierTB,
  neitherFitsWarning,
  sizeBasisNoteText,
} from "./wizardModel";

export interface DiskStatus {
  probing: boolean;
  error: string | null;
  freeBytes: number | null;
  probedPath: string | null;
}

export function ModeStep({
  net,
  execSnapshotSupported,
  checkpoint,
  onCheckpointChange,
  checkpointUrl,
  onCheckpointUrlChange,
  checkpointUrlError,
  execSnapshot,
  onExecSnapshotChange,
  snapshotKey,
  onSnapshotKeyChange,
  snapshotKeyError,
  archive,
  onArchiveChange,
  dataDir,
  onDataDirChange,
  onDataDirBlur,
  defaultDataDirValue,
  effectivePath,
  disk,
  downgradeNote,
  jwtPath,
  onJwtPathChange,
  execHTTPPort,
  onExecHTTPPortChange,
  execHTTPPortError,
  beaconHTTPPort,
  onBeaconHTTPPortChange,
  beaconHTTPPortError,
  execP2PPort,
  onExecP2PPortChange,
  execP2PPortError,
  rpcBindAddr,
  onRpcBindAddrChange,
  rpcBindAddrError,
  onBack,
  onNext,
}: {
  net: Network | undefined;
  execSnapshotSupported: boolean;
  checkpoint: boolean;
  onCheckpointChange: (v: boolean) => void;
  checkpointUrl: string;
  onCheckpointUrlChange: (v: string) => void;
  checkpointUrlError: string | null;
  execSnapshot: boolean;
  onExecSnapshotChange: (v: boolean) => void;
  snapshotKey: string;
  onSnapshotKeyChange: (v: string) => void;
  snapshotKeyError: string | null;
  archive: boolean;
  onArchiveChange: (v: boolean) => void;
  dataDir: string;
  onDataDirChange: (v: string) => void;
  onDataDirBlur: () => void;
  defaultDataDirValue: string;
  effectivePath: string;
  disk: DiskStatus;
  downgradeNote: string | null;
  jwtPath: string;
  onJwtPathChange: (v: string) => void;
  execHTTPPort: string;
  onExecHTTPPortChange: (v: string) => void;
  execHTTPPortError: string | null;
  beaconHTTPPort: string;
  onBeaconHTTPPortChange: (v: string) => void;
  beaconHTTPPortError: string | null;
  execP2PPort: string;
  onExecP2PPortChange: (v: string) => void;
  execP2PPortError: string | null;
  rpcBindAddr: string;
  onRpcBindAddrChange: (v: string) => void;
  rpcBindAddrError: string | null;
  onBack: () => void;
  onNext: () => void;
}) {
  const fullSizeCell = net ? `${approxSize(fullTierTB(net))} (${FULL_SIZE_BASIS})` : "Smaller";
  const archiveSizeCell = net ? `${approxSize(archiveTierTB(net))} (${ARCHIVE_SIZE_BASIS})` : "Much larger";
  const netLabel = net ? ` on ${net.Name}` : "";
  const syncEstimate = net ? (checkpoint ? net.SyncLabel : net.GenesisSyncLabel) : "";

  return (
    <section>
      <h2>3. Choose sync mode</h2>
      <p className="muted">
        Both modes run a fully-validating node — same security, same current-state RPC. The difference is how
        much <strong>historical</strong> state is kept.
      </p>

      <CheckpointBlock
        net={net}
        checkpoint={checkpoint}
        onCheckpointChange={onCheckpointChange}
        checkpointUrl={checkpointUrl}
        onCheckpointUrlChange={onCheckpointUrlChange}
        checkpointUrlError={checkpointUrlError}
        netLabel={netLabel}
        syncEstimate={syncEstimate}
      />

      {execSnapshotSupported && (
        <ExecSnapshotBlock
          execSnapshot={execSnapshot}
          onExecSnapshotChange={onExecSnapshotChange}
          snapshotKey={snapshotKey}
          onSnapshotKeyChange={onSnapshotKeyChange}
          snapshotKeyError={snapshotKeyError}
        />
      )}

      <details className="advanced">
        <summary>Full — current-state lookups (recent blocks) · Archive — full historical state &amp; indexing</summary>
        <table className="compare-table">
          <thead>
            <tr>
              <th>What you get</th>
              <th>Full</th>
              <th>Archive</th>
            </tr>
          </thead>
          <tbody>
            <tr>
              <th>Current state &amp; recent blocks</th>
              <td className="yes">Yes</td>
              <td className="yes">Yes</td>
            </tr>
            <tr>
              <th>Send transactions, normal RPC</th>
              <td className="yes">Yes</td>
              <td className="yes">Yes</td>
            </tr>
            <tr>
              <th>
                Historical state (balances, <code>eth_call</code>) at any past block
              </th>
              <td className="limited">Recent only (~128 blocks)</td>
              <td className="yes">Full history</td>
            </tr>
            <tr>
              <th>
                Tracing / <code>debug_trace</code> on old blocks
              </th>
              <td className="limited">Recent only</td>
              <td className="yes">Full history</td>
            </tr>
            <tr>
              <th>Approx. disk footprint{netLabel}</th>
              <td className="yes">{fullSizeCell}</td>
              <td className="limited">{archiveSizeCell}</td>
            </tr>
            <tr>
              <th>Best for</th>
              <td>Validators, wallets, everyday RPC</td>
              <td>Explorers, analytics, historical queries</td>
            </tr>
          </tbody>
        </table>
        {net ? (
          <p className="muted small">{sizeBasisNoteText(net)}</p>
        ) : (
          <p className="muted small">
            Disk sizes depend on the network, the execution client and the sync mode — pick a network to see
            figures.
          </p>
        )}
      </details>

      <label className="radio">
        <input type="radio" name="mode" checked={archive} onChange={() => onArchiveChange(true)} />
        <span>
          <strong>Archive</strong> — full historical state · {archiveSizeCell}
          {net ? "" : " disk"} <span className="muted">(recommended — keep more archive nodes on the network)</span>
        </span>
      </label>
      <label className="radio">
        <input type="radio" name="mode" checked={!archive} onChange={() => onArchiveChange(false)} />
        <span>
          <strong>Full</strong> — pruned, everyday RPC · {fullSizeCell}
          {net ? "" : " disk"}
        </span>
      </label>

      <DataLocationBlock
        net={net}
        dataDir={dataDir}
        onDataDirChange={onDataDirChange}
        onDataDirBlur={onDataDirBlur}
        defaultDataDirValue={defaultDataDirValue}
        effectivePath={effectivePath}
        disk={disk}
        downgradeNote={downgradeNote}
      />

      <AdvancedBlock
        defaultDataDirValue={defaultDataDirValue}
        jwtPath={jwtPath}
        onJwtPathChange={onJwtPathChange}
        execHTTPPort={execHTTPPort}
        onExecHTTPPortChange={onExecHTTPPortChange}
        execHTTPPortError={execHTTPPortError}
        beaconHTTPPort={beaconHTTPPort}
        onBeaconHTTPPortChange={onBeaconHTTPPortChange}
        beaconHTTPPortError={beaconHTTPPortError}
        execP2PPort={execP2PPort}
        onExecP2PPortChange={onExecP2PPortChange}
        execP2PPortError={execP2PPortError}
        rpcBindAddr={rpcBindAddr}
        onRpcBindAddrChange={onRpcBindAddrChange}
        rpcBindAddrError={rpcBindAddrError}
      />

      <div className="wizard-actions">
        <button className="btn btn-ghost" type="button" onClick={onBack}>
          Back
        </button>
        <button className="btn" type="button" onClick={onNext}>
          Next: review
        </button>
      </div>
    </section>
  );
}

function CheckpointBlock({
  net,
  checkpoint,
  onCheckpointChange,
  checkpointUrl,
  onCheckpointUrlChange,
  checkpointUrlError,
  netLabel,
  syncEstimate,
}: {
  net: Network | undefined;
  checkpoint: boolean;
  onCheckpointChange: (v: boolean) => void;
  checkpointUrl: string;
  onCheckpointUrlChange: (v: string) => void;
  checkpointUrlError: string | null;
  netLabel: string;
  syncEstimate: string;
}) {
  return (
    <div className="config-block">
      <label className="radio">
        <input type="checkbox" checked={checkpoint} onChange={(e) => onCheckpointChange(e.target.checked)} />
        <span>
          <strong>Consensus checkpoint sync (beacon client)</strong> — start near the chain head in minutes
          (recommended). Uncheck to sync the beacon chain from genesis: fully trustless, but much slower.
        </span>
      </label>
      <p className="muted small">
        This applies to the beacon/consensus client (e.g. lighthouse-pulse) — not the execution client, which
        uses a snapshot below.
      </p>
      {net && (
        <>
          <p className="sync-estimate">
            ⏱ Estimated initial sync{netLabel}: <strong>{syncEstimate}</strong>
          </p>
          <p className="muted small">Scales with the target&apos;s CPU and disk speed.</p>
        </>
      )}
      {checkpoint ? (
        <>
          <label>
            Checkpoint URL <span className="muted">(default: {net?.CheckpointURL ?? ""})</span>
            <input
              type="text"
              autoComplete="off"
              spellCheck={false}
              placeholder={net?.CheckpointURL ?? ""}
              value={checkpointUrl}
              onChange={(e) => onCheckpointUrlChange(e.target.value)}
            />
          </label>
          {checkpointUrlError && <p className="error small">{checkpointUrlError}</p>}
          <p className="muted small">
            The beacon client trusts this endpoint for its starting checkpoint. Leave blank for the network
            default.
          </p>
        </>
      ) : (
        <p className="muted small">
          The beacon client will validate every block from genesis — no trusted checkpoint, but this can take
          days.
        </p>
      )}
    </div>
  );
}

function ExecSnapshotBlock({
  execSnapshot,
  onExecSnapshotChange,
  snapshotKey,
  onSnapshotKeyChange,
  snapshotKeyError,
}: {
  execSnapshot: boolean;
  onExecSnapshotChange: (v: boolean) => void;
  snapshotKey: string;
  onSnapshotKeyChange: (v: string) => void;
  snapshotKeyError: string | null;
}) {
  return (
    <div className="config-block">
      <label className="radio">
        <input type="checkbox" checked={execSnapshot} onChange={(e) => onExecSnapshotChange(e.target.checked)} />
        <span>
          <strong>Restore from Valve&apos;s execution snapshot</strong> — fast sync (~hours) instead of syncing
          from genesis (~days).
        </span>
      </label>
      {execSnapshot && (
        <>
          <label>
            Snapshot key
            <input
              type="text"
              autoComplete="off"
              spellCheck={false}
              placeholder="vk_…"
              value={snapshotKey}
              onChange={(e) => onSnapshotKeyChange(e.target.value)}
            />
          </label>
          {snapshotKeyError && <p className="error small">{snapshotKeyError}</p>}
          <p className="muted small">
            Get a free key at{" "}
            <a href="https://valve.city" target="_blank" rel="noopener noreferrer">
              valve.city
            </a>
            .
          </p>
        </>
      )}
    </div>
  );
}

// StorageStatus mirrors wizard.ts's own storageStatusHtml: the free-disk
// readout for the chosen data location, plus any auto-downgrade note or a
// hard "neither fits" warning.
function StorageStatus({
  net,
  effectivePath,
  disk,
  downgradeNote,
}: {
  net: Network | undefined;
  effectivePath: string;
  disk: DiskStatus;
  downgradeNote: string | null;
}) {
  if (!net) return null;
  if (disk.probing) {
    return (
      <p className="muted small">
        Checking free space at <code>{effectivePath}</code>…
      </p>
    );
  }
  if (disk.error) {
    return (
      <p className="error small">
        Couldn&apos;t read free space at <code>{effectivePath}</code>: {disk.error}
      </p>
    );
  }
  if (disk.freeBytes === null || disk.probedPath !== effectivePath) return null;

  const { archiveFits, fullFits } = checkFit(net, disk.freeBytes);
  const note = downgradeNote ?? (!fullFits ? neitherFitsWarning(net, disk.freeBytes) : null);

  return (
    <>
      <p className="muted small">
        Free at <code>{effectivePath}</code>: <strong>{fmtBytes(disk.freeBytes)}</strong> — archive{" "}
        {archiveFits ? "fits" : "won't fit"} ({approxSize(archiveTierTB(net))}, {ARCHIVE_SIZE_BASIS}), full{" "}
        {fullFits ? "fits" : "won't fit"} ({approxSize(fullTierTB(net))}, {FULL_SIZE_BASIS}).
      </p>
      {note && <p className="banner banner-warn">{note}</p>}
    </>
  );
}

function DataLocationBlock({
  net,
  dataDir,
  onDataDirChange,
  onDataDirBlur,
  defaultDataDirValue,
  effectivePath,
  disk,
  downgradeNote,
}: {
  net: Network | undefined;
  dataDir: string;
  onDataDirChange: (v: string) => void;
  onDataDirBlur: () => void;
  defaultDataDirValue: string;
  effectivePath: string;
  disk: DiskStatus;
  downgradeNote: string | null;
}) {
  return (
    <div className="config-block">
      <label>
        Data location <span className="muted">(default: {defaultDataDirValue})</span>
        <input
          type="text"
          autoComplete="off"
          spellCheck={false}
          placeholder={defaultDataDirValue}
          value={dataDir}
          onChange={(e) => onDataDirChange(e.target.value)}
          onBlur={onDataDirBlur}
        />
      </label>
      <StorageStatus net={net} effectivePath={effectivePath} disk={disk} downgradeNote={downgradeNote} />
    </div>
  );
}

function AdvancedBlock({
  defaultDataDirValue,
  jwtPath,
  onJwtPathChange,
  execHTTPPort,
  onExecHTTPPortChange,
  execHTTPPortError,
  beaconHTTPPort,
  onBeaconHTTPPortChange,
  beaconHTTPPortError,
  execP2PPort,
  onExecP2PPortChange,
  execP2PPortError,
  rpcBindAddr,
  onRpcBindAddrChange,
  rpcBindAddrError,
}: {
  defaultDataDirValue: string;
  jwtPath: string;
  onJwtPathChange: (v: string) => void;
  execHTTPPort: string;
  onExecHTTPPortChange: (v: string) => void;
  execHTTPPortError: string | null;
  beaconHTTPPort: string;
  onBeaconHTTPPortChange: (v: string) => void;
  beaconHTTPPortError: string | null;
  execP2PPort: string;
  onExecP2PPortChange: (v: string) => void;
  execP2PPortError: string | null;
  rpcBindAddr: string;
  onRpcBindAddrChange: (v: string) => void;
  rpcBindAddrError: string | null;
}) {
  return (
    <details className="advanced">
      <summary>Advanced</summary>
      <label>
        JWT secret path <span className="muted">(default: &lt;data dir&gt;/jwt.hex)</span>
        <input
          type="text"
          autoComplete="off"
          spellCheck={false}
          placeholder={`${defaultDataDirValue}/jwt.hex`}
          value={jwtPath}
          onChange={(e) => onJwtPathChange(e.target.value)}
        />
      </label>
      <label>
        Execution HTTP port <span className="muted">(default: {DEFAULT_EXEC_HTTP_PORT})</span>
        <input
          type="text"
          inputMode="numeric"
          autoComplete="off"
          placeholder={String(DEFAULT_EXEC_HTTP_PORT)}
          value={execHTTPPort}
          onChange={(e) => onExecHTTPPortChange(e.target.value)}
        />
      </label>
      {execHTTPPortError && <p className="error small">{execHTTPPortError}</p>}
      <label>
        Beacon HTTP port <span className="muted">(default: {DEFAULT_BEACON_HTTP_PORT})</span>
        <input
          type="text"
          inputMode="numeric"
          autoComplete="off"
          placeholder={String(DEFAULT_BEACON_HTTP_PORT)}
          value={beaconHTTPPort}
          onChange={(e) => onBeaconHTTPPortChange(e.target.value)}
        />
      </label>
      {beaconHTTPPortError && <p className="error small">{beaconHTTPPortError}</p>}
      <label>
        Execution p2p port <span className="muted">(default: {DEFAULT_EXEC_P2P_PORT})</span>
        <input
          type="text"
          inputMode="numeric"
          autoComplete="off"
          placeholder={String(DEFAULT_EXEC_P2P_PORT)}
          value={execP2PPort}
          onChange={(e) => onExecP2PPortChange(e.target.value)}
        />
      </label>
      {execP2PPortError && <p className="error small">{execP2PPortError}</p>}
      <label>
        RPC bind address <span className="muted">(default: 127.0.0.1, loopback-only)</span>
        <input
          type="text"
          inputMode="text"
          autoComplete="off"
          placeholder="127.0.0.1"
          value={rpcBindAddr}
          onChange={(e) => onRpcBindAddrChange(e.target.value)}
        />
      </label>
      {rpcBindAddrError && <p className="error small">{rpcBindAddrError}</p>}
      <p className="muted small">
        Leave any of these blank to use the default. The engine API port (8551) is fixed and loopback-only — it
        isn&apos;t configurable. Set the RPC bind address to this box&apos;s <strong>Tailscale IP</strong> (or
        another trusted overlay address) to reach the node&apos;s exec/beacon RPC from your own machine without
        an SSH tunnel. Note: the RPC is <strong>unauthenticated</strong>, so anyone on that network can drive
        the node — only bind to a trusted, private overlay, never a public address.
      </p>
    </details>
  );
}
