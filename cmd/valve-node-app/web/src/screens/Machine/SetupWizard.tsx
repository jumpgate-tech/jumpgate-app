// Machine page → Setup section — the guided node-setup wizard: pick a
// network, pick a valid exec/beacon client pair, pick full vs. archive + a
// data dir, review, then run with live SSE step progress. Port of
// wizard.ts's own renderWizard.
//
// renderWizard's `disposed` guard and the single mutable `state` object
// become useCatalog/useTargets (React Query, shared with every other
// target-scoped screen — see hooks/target.ts, mirroring wizard.ts's own
// load()'s Promise.all([getCatalog(), listTargets()])) plus this screen's
// own hooks/wizard.ts (useDiskProbe for the on-demand free-space probe,
// useSetupRun for the POST-then-stream provisioning run) and a set of plain
// useState fields for the form itself — every wizard.ts form field becomes
// a CONTROLLED React input here, so there is no DOM-read-back
// (readModeInputs) step: the current field values already ARE the state.
//
// Two render-time corrections in wizard.ts become explicit effects instead
// of "recompute this every render" logic, since a React component can't
// mutate state mid-render the way the legacy renderClientsStep did:
//   - which exec/beacon client is selected defaults (and re-defaults, on a
//     network change) to the network's first offered client — see the
//     "keep the client pair valid for the chosen network" effect below.
//   - an existing target's saved wire config seeds the form once, the first
//     time the targets list loads — see the "prefill from an existing wire
//     config" effect below, run at most once via appliedInitialRef.
//
// NOT yet wired into the Machine page — the machine composer task mounts
// this. It still renders its own <h1>/<Footer>, exactly like the legacy
// module does today inside machine.ts's section body; stripping those so the
// page reads as one document is the follow-up machine.ts's own comment calls
// out, not this task.
import { useEffect, useMemo, useRef, useState } from "react";
import { useCatalog, useTargets } from "../../hooks/target";
import { useDiskProbe, useSetupRun } from "../../hooks/wizard";
import { Footer } from "../../components/Footer";
import { NetworkStep } from "./NetworkStep";
import { ClientsStep } from "./ClientsStep";
import { ModeStep } from "./ModeStep";
import { ReviewStep } from "./ReviewStep";
import { RunStep } from "./RunStep";
import {
  buildStartSetupRequest,
  defaultDataDir,
  defaultJwtPath,
  evaluateFit,
  hasModeFieldErrors,
  resolveClientId,
  validateModeFields,
  WIZARD_STEPS,
  wizardStepClass,
  type WizardStep,
} from "./wizardModel";

function formatError(err: unknown): string {
  return err instanceof Error ? err.message : String(err);
}

export function SetupWizard({ targetId }: { targetId: string }) {
  const catalogQuery = useCatalog();
  const targetsQuery = useTargets();
  const diskProbe = useDiskProbe(targetId);
  const setupRun = useSetupRun(targetId);

  const [step, setStep] = useState<WizardStep>("network");
  const [chainId, setChainId] = useState<number | null>(369);
  const [execId, setExecId] = useState<string | null>(null);
  const [beaconId, setBeaconId] = useState<string | null>(null);
  const [archive, setArchive] = useState(true);
  const [dataDir, setDataDir] = useState("");
  const [jwtPath, setJwtPath] = useState("");
  const [execHTTPPort, setExecHTTPPort] = useState("");
  const [beaconHTTPPort, setBeaconHTTPPort] = useState("");
  const [execP2PPort, setExecP2PPort] = useState("");
  const [rpcBindAddr, setRpcBindAddr] = useState("");
  const [downgradeNote, setDowngradeNote] = useState<string | null>(null);
  const [checkpoint, setCheckpoint] = useState(true);
  const [checkpointUrl, setCheckpointUrl] = useState("");
  const [execSnapshot, setExecSnapshot] = useState(false);
  const [snapshotKey, setSnapshotKey] = useState("");

  const appliedInitialRef = useRef(false);

  // Prefill from an existing wire config — mirrors load()'s own
  // `existing?.wire` branch, applied at most once (a later refetch of the
  // targets list, e.g. after this same run completes, must not stomp on
  // whatever the operator has since typed).
  useEffect(() => {
    if (appliedInitialRef.current || !targetsQuery.data) return;
    appliedInitialRef.current = true;
    const existing = targetsQuery.data.find((t) => t.id === targetId);
    if (!existing?.wire) return;
    setChainId(existing.wire.ChainID);
    setExecId(existing.wire.ExecID);
    setBeaconId(existing.wire.BeaconID);
    setArchive(existing.wire.Archive);
    if (existing.wire.ExecHTTPPort) setExecHTTPPort(String(existing.wire.ExecHTTPPort));
    if (existing.wire.BeaconHTTPPort) setBeaconHTTPPort(String(existing.wire.BeaconHTTPPort));
    if (existing.wire.ExecP2PPort) setExecP2PPort(String(existing.wire.ExecP2PPort));
    if (existing.wire.RPCBindAddr) setRpcBindAddr(existing.wire.RPCBindAddr);
  }, [targetsQuery.data, targetId]);

  const net = catalogQuery.data?.networks.find((n) => n.ChainID === chainId);

  // Keep the client pair valid for the chosen network — mirrors
  // renderClientsStep's own per-render `if (... !net.ExecClients.includes
  // (state.execId)) state.execId = net.ExecClients[0] ?? null` correction,
  // run as an effect (once when the catalog loads, and again on every
  // network change) instead of during render.
  useEffect(() => {
    if (!net) return;
    setExecId((cur) => resolveClientId(cur, net.ExecClients));
    setBeaconId((cur) => resolveClientId(cur, net.BeaconClients));
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [net?.ChainID, catalogQuery.data]);

  const defaultDataDirValue = defaultDataDir(chainId);
  const effectivePath = (dataDir || defaultDataDirValue).trim();

  async function probeAndEvaluate(): Promise<void> {
    const outcome = await diskProbe.probe(effectivePath);
    if (outcome.freeBytes === undefined) return;
    const result = evaluateFit(net, outcome.freeBytes, archive, effectivePath);
    setArchive(result.archive);
    setDowngradeNote(result.downgradeNote);
  }

  const modeErrors = useMemo(
    () =>
      validateModeFields({
        execHTTPPort,
        beaconHTTPPort,
        execP2PPort,
        rpcBindAddr,
        checkpoint,
        checkpointUrl,
        execSnapshot,
        snapshotKey,
      }),
    [execHTTPPort, beaconHTTPPort, execP2PPort, rpcBindAddr, checkpoint, checkpointUrl, execSnapshot, snapshotKey],
  );

  function handlePickNetwork(id: number): void {
    setChainId(id);
    setExecId(null);
    setBeaconId(null);
  }

  function handleGotoClients(): void {
    if (chainId === null) return;
    setStep("clients");
  }

  function handleGotoMode(): void {
    setStep("mode");
    void probeAndEvaluate();
  }

  function handleGotoReview(): void {
    if (hasModeFieldErrors(modeErrors)) return;
    setStep("review");
  }

  async function handleStartSetup(): Promise<void> {
    if (chainId === null || !execId || !beaconId) return;
    const wire = buildStartSetupRequest({
      chainId,
      execId,
      beaconId,
      archive,
      dataDir,
      jwtPath,
      execHTTPPort,
      beaconHTTPPort,
      execP2PPort,
      rpcBindAddr,
      checkpoint,
      checkpointUrl,
      execSnapshot,
      snapshotKey,
    });
    const ok = await setupRun.start(wire);
    if (ok) setStep("run");
  }

  const initLoading = catalogQuery.isLoading || targetsQuery.isLoading;
  const initErr = catalogQuery.error ?? targetsQuery.error;
  const reviewDataDir = dataDir || defaultDataDirValue;
  const reviewJwtPath = jwtPath || defaultJwtPath(reviewDataDir);
  const execSnapshotSupported = execId
    ? (catalogQuery.data?.clients.find((c) => c.id === execId)?.snapshotSupported ?? false)
    : false;

  return (
    <>
      <h1>Setup: {targetId}</h1>
      <div>
        {initLoading ? (
          <p className="muted">Loading catalog…</p>
        ) : initErr ? (
          <p className="error">Failed to load: {formatError(initErr)}</p>
        ) : !catalogQuery.data ? null : (
          <>
            <WizardProgress current={step} />
            {step === "network" && (
              <NetworkStep
                catalog={catalogQuery.data}
                chainId={chainId}
                onPick={handlePickNetwork}
                onNext={handleGotoClients}
              />
            )}
            {step === "clients" && chainId !== null && (
              <ClientsStep
                catalog={catalogQuery.data}
                chainId={chainId}
                execId={execId}
                beaconId={beaconId}
                onExecChange={setExecId}
                onBeaconChange={setBeaconId}
                onBack={() => setStep("network")}
                onNext={handleGotoMode}
              />
            )}
            {step === "mode" && (
              <ModeStep
                net={net}
                execSnapshotSupported={execSnapshotSupported}
                checkpoint={checkpoint}
                onCheckpointChange={setCheckpoint}
                checkpointUrl={checkpointUrl}
                onCheckpointUrlChange={setCheckpointUrl}
                checkpointUrlError={modeErrors.checkpointUrlError}
                execSnapshot={execSnapshot}
                onExecSnapshotChange={setExecSnapshot}
                snapshotKey={snapshotKey}
                onSnapshotKeyChange={setSnapshotKey}
                snapshotKeyError={modeErrors.snapshotKeyError}
                archive={archive}
                onArchiveChange={setArchive}
                dataDir={dataDir}
                onDataDirChange={setDataDir}
                onDataDirBlur={() => void probeAndEvaluate()}
                defaultDataDirValue={defaultDataDirValue}
                effectivePath={effectivePath}
                disk={{
                  probing: diskProbe.probing,
                  error: diskProbe.error,
                  freeBytes: diskProbe.freeBytes,
                  probedPath: diskProbe.probedPath,
                }}
                downgradeNote={downgradeNote}
                jwtPath={jwtPath}
                onJwtPathChange={setJwtPath}
                execHTTPPort={execHTTPPort}
                onExecHTTPPortChange={setExecHTTPPort}
                execHTTPPortError={modeErrors.execHTTPPortError}
                beaconHTTPPort={beaconHTTPPort}
                onBeaconHTTPPortChange={setBeaconHTTPPort}
                beaconHTTPPortError={modeErrors.beaconHTTPPortError}
                execP2PPort={execP2PPort}
                onExecP2PPortChange={setExecP2PPort}
                execP2PPortError={modeErrors.execP2PPortError}
                rpcBindAddr={rpcBindAddr}
                onRpcBindAddrChange={setRpcBindAddr}
                rpcBindAddrError={modeErrors.rpcBindAddrError}
                onBack={() => setStep("clients")}
                onNext={handleGotoReview}
              />
            )}
            {step === "review" && chainId !== null && execId && beaconId && (
              <ReviewStep
                targetId={targetId}
                net={net}
                chainId={chainId}
                execId={execId}
                beaconId={beaconId}
                archive={archive}
                dataDir={reviewDataDir}
                jwtPath={reviewJwtPath}
                checkpoint={checkpoint}
                checkpointUrl={checkpointUrl}
                execHTTPPort={execHTTPPort}
                beaconHTTPPort={beaconHTTPPort}
                execP2PPort={execP2PPort}
                rpcBindAddr={rpcBindAddr}
                startError={setupRun.startError}
                starting={setupRun.starting}
                onBack={() => setStep("mode")}
                onStart={() => void handleStartSetup()}
              />
            )}
            {step === "run" && (
              <RunStep
                targetId={targetId}
                net={net}
                events={setupRun.events}
                startError={setupRun.startError}
                onRetry={() => void handleStartSetup()}
              />
            )}
          </>
        )}
      </div>
      <Footer contextLabel={net?.Name} contextUrl={net?.LearnURL} />
    </>
  );
}

// WizardProgress mirrors wizard.ts's own wizardProgress: a 5-step rail
// classed past/current/future relative to the active step.
function WizardProgress({ current }: { current: WizardStep }) {
  return (
    <ol className="wizard-progress">
      {WIZARD_STEPS.map((s) => (
        <li key={s.id} className={wizardStepClass(current, s.id)}>
          {s.label}
        </li>
      ))}
    </ol>
  );
}
