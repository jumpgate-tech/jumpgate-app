// One machine's gateway — the per-gateway container. It owns everything scoped
// to a single gateway: its traffic + capability reads (a hook per card, which
// is why this is a component and not a loop in the parent), its lifecycle and
// config flows, and every dialog those flows open. The parent (Rpc) owns only
// the fleet-level concerns (the gateway list, add-a-gateway, loose orphans).
//
// The render order mirrors the legacy gatewayBlock: the quiet identity line,
// the one attention strip, the provisioning log, the chains (the whole point),
// then the collapsed "Manage gateway".
import { useEffect, useState } from "react";
import * as api from "../../api";
import type {
  ChainlistResult,
  GatewayView,
  KnownSetResponse,
  NetworkPreset,
  TargetSummary,
  UpstreamSource,
  WipeResult,
} from "../../api";
import { usePutGatewayConfig, useGatewayCapabilities, useWipeGateway } from "../../hooks/gateway";
import {
  useDeleteGateway,
  useDiscoverEndpoints,
  useDismissOrphan,
  useGatewayOps,
  useGatewayTraffic,
  useKnownSet,
  useResetDevnet,
  useTrustCert,
  useVerifyTls,
} from "../../hooks/rpc";
import { ConfirmDialog } from "../Panel/Dialogs";
import { GatewayIdentity } from "./GatewayIdentity";
import { AttentionStrip } from "./AttentionStrip";
import { NetworksPanel } from "./Chains";
import { ManageSection } from "./ManageSection";
import type { SettingsValues } from "./SettingsBlock";
import {
  AddChainDialog,
  AddEndpointDialog,
  CustomChainDialog,
  DiscoverDialog,
  KnownSetDialog,
  ManualEndpointDialog,
  MessageDialog,
  ResetResultDialog,
  WipeDialog,
  WipeResultDialog,
} from "./RpcDialogs";
import {
  attentionLines,
  mergePendingNetworks,
  pruneEmptyNetworks,
  storedConfig,
  targetModeOf,
  withDevnetUpstream,
  withExternalUpstreams,
  withManagedUpstream,
  withoutChain,
  withoutEndpoint,
} from "./rpcModel";

type Dialog =
  | { kind: "add-chain" }
  | { kind: "custom-chain" }
  | { kind: "add-endpoint"; chainId: number }
  | { kind: "manual-endpoint"; chainId: number }
  | { kind: "known-set"; chainId: number; set: KnownSetResponse | null; error: string | null }
  | { kind: "discover"; chainId: number; result: ChainlistResult | null }
  | { kind: "forget" }
  | { kind: "remove-chain"; chainId: number; name: string }
  | { kind: "remove-endpoint"; chainId: number; upstreamId: string }
  | { kind: "reset-confirm"; chainId: number; upstreamId: string; targetId: string }
  | { kind: "create-devnet-first"; chainId: number }
  | { kind: "wipe" }
  | { kind: "wipe-result"; result: WipeResult }
  | { kind: "reset-result"; targetId: string; result: WipeResult };

function message(e: unknown): string {
  return e instanceof Error ? e.message : String(e);
}

export function GatewayCard({
  gw,
  targets,
  sources,
  presets,
  orphans,
  showMachine,
  hostOS,
}: {
  gw: GatewayView;
  targets: TargetSummary[];
  sources: UpstreamSource[];
  presets: NetworkPreset[];
  orphans: api.OrphanedContainer[];
  showMachine: boolean;
  hostOS: string;
}) {
  const gid = gw.id;
  const traffic = useGatewayTraffic(gid);
  const caps = useGatewayCapabilities(gid, true);
  const ops = useGatewayOps(gid);
  const putCfg = usePutGatewayConfig();
  const wipeMut = useWipeGateway();
  const verifyMut = useVerifyTls();
  const trustMut = useTrustCert();
  const forgetMut = useDeleteGateway();
  const dismissMut = useDismissOrphan();
  const knownSetMut = useKnownSet();
  const discoverMut = useDiscoverEndpoints();
  const resetMut = useResetDevnet();

  const [dialog, setDialog] = useState<Dialog | null>(null);
  const [manageOpen, setManageOpen] = useState(false);
  const [settingsOpen, setSettingsOpen] = useState(false);
  const [openDetails, setOpenDetails] = useState<Set<number>>(new Set());
  const [pendingChains, setPendingChains] = useState<number[]>([]);
  const [orphanErr, setOrphanErr] = useState<Record<string, string | null>>({});

  // Drop a pending placeholder once the server echoes the chain back.
  useEffect(() => {
    const present = new Set((gw.networks ?? []).map((n) => n.chainId));
    setPendingChains((prev) => prev.filter((cid) => !present.has(cid)));
  }, [gw.networks]);

  const targetMode = targetModeOf(targets, gw.placement.targetId);
  const placementHasDevnet = targets.some((t) => t.id === gw.placement.targetId && t.hasDevnet);
  const networks = mergePendingNetworks(gw.networks ?? [], pendingChains, presets, gw.config.ProjectID);
  const displayBusy = ops.busy ?? (resetMut.isPending ? "reset" : null);

  const verifyResult = verifyMut.data ?? gw.tls?.verification ?? null;
  const verifyErr = verifyMut.error ? message(verifyMut.error) : null;

  // --- config saves --------------------------------------------------------

  async function saveConfig(cfg: api.GatewayConfig, note?: string): Promise<boolean> {
    ops.setActionErr(null);
    try {
      await putCfg.mutateAsync({ gid, config: cfg });
      return true;
    } catch (e) {
      ops.setActionErr(`${note ? note + ": " : ""}${message(e)}`);
      return false;
    }
  }

  // --- lifecycle -----------------------------------------------------------

  function onAction(action: string) {
    if (action === "start" || action === "stop" || action === "restart") void ops.runAction(action);
    else if (action === "create" || action === "recreate") void ops.provision();
    else if (action === "wipe") setDialog({ kind: "wipe" });
  }

  async function onReprobe() {
    setReprobing(true);
    await caps.refetch(true);
    setReprobing(false);
  }
  const [reprobing, setReprobing] = useState(false);

  // --- chains --------------------------------------------------------------

  async function addChain(chainId: number) {
    if ((gw.networks ?? []).some((n) => n.chainId === chainId)) return;
    const ok = await saveConfig(pruneEmptyNetworks(storedConfig(gw)));
    if (ok) {
      setPendingChains((prev) => (prev.includes(chainId) ? prev : [...prev, chainId]));
      setOpenDetails((prev) => new Set(prev).add(chainId));
      setDialog({ kind: "add-endpoint", chainId });
    }
  }

  async function addDevnetChain(chainId: number) {
    if (!placementHasDevnet) {
      setDialog({ kind: "create-devnet-first", chainId });
      return;
    }
    await saveConfig(withDevnetUpstream(storedConfig(gw), chainId, gw.placement.targetId), "Adding the devnet");
  }

  function onPickChain(preset: NetworkPreset) {
    setDialog(null);
    if (preset.devnet) void addDevnetChain(preset.chainId);
    else void addChain(preset.chainId);
  }

  async function removeChain(chainId: number) {
    setDialog(null);
    await saveConfig(withoutChain(storedConfig(gw), chainId), "Removing the network");
  }

  // --- endpoints -----------------------------------------------------------

  async function addManagedUpstream(chainId: number, kind: api.UpstreamKind, targetId: string) {
    setDialog(null);
    await saveConfig(withManagedUpstream(storedConfig(gw), chainId, kind, targetId), "Adding the endpoint");
  }

  async function addExternalUpstreams(chainId: number, urls: string[], recentOnly = false) {
    setDialog(null);
    if (!urls.length) return;
    await saveConfig(
      withExternalUpstreams(storedConfig(gw), chainId, urls, recentOnly),
      urls.length === 1 ? "Adding the endpoint" : `Adding ${urls.length} endpoints`,
    );
  }

  async function removeEndpoint(chainId: number, upstreamId: string) {
    setDialog(null);
    await saveConfig(withoutEndpoint(storedConfig(gw), chainId, upstreamId), "Removing the endpoint");
  }

  async function openKnownSet(chainId: number) {
    setDialog({ kind: "known-set", chainId, set: null, error: null });
    try {
      const set = await knownSetMut.mutateAsync({ gid, chainId });
      setDialog((d) => (d?.kind === "known-set" && d.chainId === chainId ? { ...d, set } : d));
    } catch (e) {
      setDialog((d) => (d?.kind === "known-set" && d.chainId === chainId ? { ...d, error: message(e) } : d));
    }
  }

  async function openDiscover(chainId: number) {
    setDialog({ kind: "discover", chainId, result: null });
    try {
      const result = await discoverMut.mutateAsync(chainId);
      setDialog((d) => (d?.kind === "discover" && d.chainId === chainId ? { ...d, result } : d));
    } catch (e) {
      setDialog(null);
      ops.setActionErr(`Could not discover endpoints: ${message(e)}`);
    }
  }

  // --- settings ------------------------------------------------------------

  async function saveSettings(values: SettingsValues) {
    const cfg = storedConfig(gw);
    if (values.port !== null) cfg.Port = values.port;
    cfg.BindAddr = values.bind;
    cfg.MetricsOff = !values.metricsOn;
    cfg.TLS = {
      Enabled: values.tls.enabled,
      Hostname: values.tls.hostname,
      CertSource: values.tls.certSource,
      CertFile: values.tls.certFile,
      KeyFile: values.tls.keyFile,
      HTTPSPort: values.tls.httpsPort ?? 443,
      BindAddr: gw.config.TLS?.BindAddr ?? "",
      ImageRef: gw.config.TLS?.ImageRef ?? "",
    };
    const wasRunning = gw.status.State === "running";
    if (await saveConfig(cfg, "Saving settings")) {
      setSettingsOpen(false);
      if (wasRunning) {
        ops.note(
          "Saved. The running container still has the old port and bind — press “Re-create (apply config)” to put them into effect.",
        );
      }
    }
  }

  // --- forget / trust / reset ---------------------------------------------

  async function forget() {
    setDialog(null);
    try {
      await forgetMut.mutateAsync(gid);
    } catch (e) {
      ops.setActionErr(message(e));
    }
  }

  async function onReset() {
    if (dialog?.kind !== "reset-confirm") return;
    const { targetId } = dialog;
    setDialog(null);
    ops.setActionErr(null);
    try {
      const result = await resetMut.mutateAsync(targetId);
      setDialog({ kind: "reset-result", targetId, result });
    } catch (e) {
      ops.setActionErr(`Reset failed: ${message(e)}`);
    }
  }

  async function onWipe() {
    setDialog(null);
    ops.setActionErr(null);
    try {
      const result = await wipeMut.mutateAsync(gid);
      setDialog({ kind: "wipe-result", result });
    } catch (e) {
      ops.setActionErr(`Wipe failed: ${message(e)}`);
    }
  }

  function dismissOrphan(name: string) {
    setOrphanErr((e) => ({ ...e, [name]: null }));
    dismissMut.mutate(name, {
      onError: (err) => setOrphanErr((e) => ({ ...e, [name]: message(err) })),
    });
  }

  // --- dialog rendering ----------------------------------------------------

  function renderDialog() {
    if (!dialog) return null;
    switch (dialog.kind) {
      case "add-chain": {
        const present = new Set((gw.networks ?? []).map((n) => n.chainId));
        return (
          <AddChainDialog
            baseUrl={gw.baseUrl}
            targetId={gw.placement.targetId}
            placementHasDevnet={placementHasDevnet}
            available={presets.filter((p) => !present.has(p.chainId))}
            already={presets.filter((p) => present.has(p.chainId))}
            onPick={onPickChain}
            onCustom={() => setDialog({ kind: "custom-chain" })}
            onCancel={() => setDialog(null)}
          />
        );
      }
      case "custom-chain":
        return (
          <CustomChainDialog
            onAdd={(chainId) => {
              setDialog(null);
              void addChain(chainId);
            }}
            onCancel={() => setDialog(null)}
          />
        );
      case "add-endpoint": {
        const chainId = dialog.chainId;
        const net = (gw.networks ?? []).find((n) => n.chainId === chainId);
        const nvName = net?.name ?? `chain ${chainId}`;
        const taken = new Set(
          (net?.upstreams ?? []).filter((u) => u.kind !== "external").map((u) => `${u.kind}|${u.targetId ?? ""}`),
        );
        const offer = sources.filter((s) => s.chainId === chainId && !taken.has(`${s.kind}|${s.targetId}`));
        return (
          <AddEndpointDialog
            networkName={nvName}
            chainId={chainId}
            offer={offer}
            onSource={(kind, targetId) => void addManagedUpstream(chainId, kind, targetId)}
            onKnownSet={() => void openKnownSet(chainId)}
            onManual={() => setDialog({ kind: "manual-endpoint", chainId })}
            onCancel={() => setDialog(null)}
          />
        );
      }
      case "manual-endpoint": {
        const chainId = dialog.chainId;
        return (
          <ManualEndpointDialog
            onAdd={(url, recentOnly) => void addExternalUpstreams(chainId, [url], recentOnly)}
            onCancel={() => setDialog(null)}
          />
        );
      }
      case "known-set": {
        const chainId = dialog.chainId;
        return (
          <KnownSetDialog
            chainId={chainId}
            set={dialog.set}
            error={dialog.error}
            onAdd={(urls) => void addExternalUpstreams(chainId, urls)}
            onDiscover={() => void openDiscover(chainId)}
            onCancel={() => setDialog(null)}
          />
        );
      }
      case "discover": {
        const chainId = dialog.chainId;
        return (
          <DiscoverDialog
            chainId={chainId}
            result={dialog.result}
            onAdd={(urls) => void addExternalUpstreams(chainId, urls)}
            onCancel={() => setDialog(null)}
          />
        );
      }
      case "forget":
        return (
          <ConfirmDialog
            title={`Forget ${gw.label}`}
            body={`valve-node-app will forget this gateway's configuration. Its container "${gw.containerName}" on ${gw.placement.targetId} is NOT touched — if it is running it keeps running and keeps serving. Stop or wipe it first if you wanted it gone.`}
            confirmLabel="Forget it"
            danger
            onConfirm={() => void forget()}
            onCancel={() => setDialog(null)}
          />
        );
      case "remove-chain":
        return (
          <ConfirmDialog
            title={`Remove ${dialog.name}`}
            body="This gateway will stop serving this chain. Nothing on the other end is touched — the nodes and endpoints behind it keep running."
            confirmLabel="Remove network"
            danger
            onConfirm={() => void removeChain(dialog.chainId)}
            onCancel={() => setDialog(null)}
          />
        );
      case "remove-endpoint":
        return (
          <ConfirmDialog
            title="Remove this endpoint"
            body="The gateway stops routing to it. Whatever is on the other end — a node, a devnet, a public endpoint — is left completely alone."
            confirmLabel="Remove"
            danger
            onConfirm={() => void removeEndpoint(dialog.chainId, dialog.upstreamId)}
            onCancel={() => setDialog(null)}
          />
        );
      case "reset-confirm":
        return (
          <ConfirmDialog
            title="Reset this devnet"
            body={`The chain on ${dialog.targetId} starts again from genesis at block 0 — every block, transaction and account it has produced is discarded. The gateways in front of it are restarted so they stop advertising the old head.`}
            confirmLabel="Reset the chain"
            onConfirm={() => void onReset()}
            onCancel={() => setDialog(null)}
          />
        );
      case "create-devnet-first":
        return (
          <MessageDialog
            title="Create a devnet first"
            onClose={() => setDialog(null)}
            link={{ href: `#/services/${encodeURIComponent(gw.placement.targetId)}`, label: `Create a devnet on ${gw.placement.targetId}` }}
          >
            <p>
              There is no devnet on <code>{gw.placement.targetId}</code>, so adding chain {dialog.chainId} here would
              create a network with nothing behind it.
            </p>
            <p className="muted small">
              A devnet belongs to a machine — it is reth in --dev mode in a container on that box — so it is created on
              that machine's own screen. Come back here afterwards and this option will point the gateway straight at it.
            </p>
          </MessageDialog>
        );
      case "wipe":
        return (
          <WipeDialog gid={gid} wipeDiscards={gw.wipeDiscards} onConfirm={() => void onWipe()} onCancel={() => setDialog(null)} />
        );
      case "wipe-result":
        return <WipeResultDialog gid={gid} result={dialog.result} onClose={() => setDialog(null)} />;
      case "reset-result":
        return <ResetResultDialog targetId={dialog.targetId} result={dialog.result} onClose={() => setDialog(null)} />;
    }
  }

  return (
    <>
      {showMachine ? <h2 className="rpc-machine">{gw.placement.targetId}</h2> : null}
      <GatewayIdentity gw={gw} />
      <AttentionStrip lines={attentionLines(gw, verifyResult, ops.actionErr)} />
      {ops.activity.length > 0 ? (
        <div className="config-block">
          <p className="muted small">Provisioning on {gw.placement.targetId}</p>
          <pre className="step-log">{ops.activity.join("\n")}</pre>
        </div>
      ) : null}
      <NetworksPanel
        gw={gw}
        networks={networks}
        caps={caps.data}
        capsBusy={reprobing}
        traffic={traffic.data ?? null}
        trafficLoading={traffic.isLoading}
        targetMode={targetMode}
        trustBusy={trustMut.isPending}
        trustMessage={trustMut.data ?? null}
        busy={displayBusy}
        isDetailOpen={(chainId) => openDetails.has(chainId)}
        onToggleDetail={(chainId) =>
          setOpenDetails((prev) => {
            const next = new Set(prev);
            if (next.has(chainId)) next.delete(chainId);
            else next.add(chainId);
            return next;
          })
        }
        onAddEndpoint={(chainId) => setDialog({ kind: "add-endpoint", chainId })}
        onRemoveChain={(chainId) =>
          setDialog({ kind: "remove-chain", chainId, name: (gw.networks ?? []).find((n) => n.chainId === chainId)?.name ?? `chain ${chainId}` })
        }
        onRemoveEndpoint={(chainId, upstreamId) => setDialog({ kind: "remove-endpoint", chainId, upstreamId })}
        onResetDevnet={(chainId, upstreamId, targetId) => setDialog({ kind: "reset-confirm", chainId, upstreamId, targetId })}
        onAddChain={() => setDialog({ kind: "add-chain" })}
        onAddDevnet={() => void addDevnetChain(1337)}
        onReprobe={() => void onReprobe()}
        onTrust={() => trustMut.mutate(gid)}
      />
      <ManageSection
        gw={gw}
        open={manageOpen}
        onToggle={() => setManageOpen((o) => !o)}
        orphans={orphans}
        orphanErr={orphanErr}
        onDismissOrphan={dismissOrphan}
        busy={displayBusy}
        onAction={onAction}
        capsBusy={reprobing}
        onReprobe={() => void onReprobe()}
        settingsOpen={settingsOpen}
        onToggleSettings={() => setSettingsOpen((o) => !o)}
        onSaveSettings={(values) => void saveSettings(values)}
        onForget={() => setDialog({ kind: "forget" })}
        hostOS={hostOS}
        targetMode={targetMode}
        trustBusy={trustMut.isPending}
        trustResult={trustMut.data ?? null}
        onTrust={() => trustMut.mutate(gid)}
        verifying={verifyMut.isPending}
        verifyResult={verifyResult}
        verifyErr={verifyErr}
        onVerify={() => verifyMut.mutate(gid)}
      />
      {renderDialog()}
    </>
  );
}
