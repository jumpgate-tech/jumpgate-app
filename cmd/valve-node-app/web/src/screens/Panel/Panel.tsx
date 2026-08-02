// The Easy-Button panel — the first fully-React screen and the pattern every
// later screen follows. It is a single card that drills list → network →
// endpoint, drives a gateway through its whole lifecycle, and stands one up
// from nothing with one click. This container owns the view state, the
// lifecycle flows and the dialogs; the views and the power/health/capability
// components are small and presentational.
//
// It is the port of panel.ts. Three things that file hand-rolled are GONE here,
// which is the point of the migration: the healthSignature/lastHealthSig
// render-skip machinery (useGatewayHealth polls every 5s and React's reconciler
// keeps the dot nodes, so their CSS animations never restart), escapeHtml +
// data-action delegation (JSX + onClick), and the manual load()/poll()/render()
// bookkeeping (the Task 3 hooks + React Query).
import { useMemo, useRef, useState } from "react";
import { useQueryClient } from "@tanstack/react-query";
import * as api from "../../api";
import {
  masterState,
  withNetwork,
  withoutNetwork,
  withUpstream,
  withoutUpstream,
  endpointNameFromUrl,
} from "../../panelModel";
import { SETUP_CHAINS, internalTLSConfig } from "../../home";
import {
  useGateways,
  useGatewayHealth,
  useGatewayCapabilities,
  useGatewayAction,
  usePutGatewayConfig,
  useWipeGateway,
} from "../../hooks/gateway";
import { useEventStream } from "../../hooks/useEventStream";
import { Sprite } from "./icons";
import { primaryAction } from "./PowerBand";
import { ListView } from "./ListView";
import { NetworkView } from "./NetworkView";
import { EndpointView } from "./EndpointView";
import { SettingsSheet } from "./SettingsSheet";
import { ConfirmDialog, TextInputDialog, AddNetworkDialog, validateUrlScheme } from "./Dialogs";

// FINAL_STEP is the id every gateway setup plan ends on (mirrors rpc.ts). The
// setup stream is done when a step errors, or when this step reports done.
const FINAL_STEP = "run";

// DEVNET_CHAIN_ID is reth's --dev genesis id. catalog.Networks() excludes it
// (a private scratch chain, not a supported public network), so the add-network
// picker appends it itself, always last.
const DEVNET_CHAIN_ID = 1337;

type View =
  | { name: "list" }
  | { name: "network"; chainId: number }
  | { name: "endpoint"; chainId: number; upstreamId: string };

type Dialog =
  | { kind: "settings" }
  | { kind: "confirm-wipe" }
  | { kind: "add-network"; picks: { chainId: number; name: string }[] }
  | { kind: "add-endpoint"; chainId: number }
  | { kind: "rename"; chainId: number; upstreamId: string; current: string }
  | { kind: "edit-address"; chainId: number; upstreamId: string; current: string }
  | { kind: "confirm-remove-network"; chainId: number; name: string }
  | { kind: "confirm-remove-endpoint"; chainId: number; upstreamId: string; label: string; networkName: string };

export function Panel() {
  const qc = useQueryClient();
  const [view, setView] = useState<View>({ name: "list" });
  const [dialog, setDialog] = useState<Dialog | null>(null);
  const [busy, setBusyState] = useState<string | null>(null);
  const [actionErr, setActionErr] = useState<string | null>(null);
  const [setupLog, setSetupLog] = useState<string[]>([]);
  const [networkErr, setNetworkErr] = useState<string | null>(null);
  const [endpointErr, setEndpointErr] = useState<string | null>(null);
  // provisionTargetId non-null subscribes useEventStream to that machine's
  // setup stream; the stream handler clears it on finish, which unsubscribes.
  const [provisionTargetId, setProvisionTargetId] = useState<string | null>(null);

  // busyRef mirrors busy so the flow guards (`if (busyRef.current) return`) see
  // the latest value synchronously, without waiting for a re-render — several
  // flows are chained programmatically (provision after a config write), not
  // just fired from one click.
  const busyRef = useRef<string | null>(null);
  function setBusy(v: string | null) {
    busyRef.current = v;
    setBusyState(v);
  }
  // onDoneRef holds the callback provision() runs once the stream finishes
  // WITHOUT an error and the gateway list has refreshed. captureLogRef marks
  // the one-click setup's stream so its progress lines accumulate into setupLog
  // (provision() for an existing gateway does not narrate).
  const onDoneRef = useRef<(() => void) | undefined>(undefined);
  const captureLogRef = useRef(false);

  const gwQuery = useGateways();
  const gw = useMemo(() => primaryGateway(gwQuery.data ?? null), [gwQuery.data]);
  const gid = gw?.id;

  const health = useGatewayHealth(gid, !!gid);
  const capsEnabled = !!gid && (view.name === "network" || view.name === "endpoint");
  const caps = useGatewayCapabilities(gid, capsEnabled);

  const actionMut = useGatewayAction();
  const putCfgMut = usePutGatewayConfig();
  const wipeMut = useWipeGateway();

  // The setup stream: only ever one provision in flight, so one subscription.
  useEventStream(provisionTargetId, (ev) => {
    if (captureLogRef.current) {
      const line = ev.err ? `${ev.stepId}: ${ev.err}` : ev.line ? `${ev.stepId}: ${ev.line}` : `${ev.stepId}: done`;
      setSetupLog((prev) => [...prev, line]);
    }
    const finished = !!ev.err || (ev.stepId === FINAL_STEP && !!ev.done);
    if (!finished) return;
    setProvisionTargetId(null);
    setBusy(null);
    if (ev.err) setActionErr(`Provisioning failed: ${ev.err}`);
    if (captureLogRef.current) setSetupLog([]);
    captureLogRef.current = false;
    const done = onDoneRef.current;
    onDoneRef.current = undefined;
    // Reload the gateway list, then — only on a clean finish — run the
    // caller's follow-up (e.g. land on the new network, re-probe caps).
    void qc.invalidateQueries({ queryKey: ["gateways"] }).then(() => {
      if (!ev.err) done?.();
    });
    void qc.invalidateQueries({ queryKey: ["gwHealth", gid] });
  });

  // --- lifecycle: the two paths --------------------------------------------

  // runAction: start/stop/restart go straight to the container action endpoint
  // — no config write, no stream to follow (mirrors panel.ts runAction).
  async function runAction(id: string, kind: api.ContainerActionKind) {
    if (busyRef.current) return;
    setBusy(kind);
    setActionErr(null);
    try {
      await actionMut.mutateAsync({ gid: id, action: kind });
    } catch (e) {
      setActionErr(`${kind} failed: ${message(e)}`);
    }
    setBusy(null);
    void qc.invalidateQueries({ queryKey: ["gateways"] });
  }

  // provision: create/recreate run the setup plan and follow the placement
  // machine's setup stream (mirrors panel.ts provision). onDone runs once the
  // stream finishes clean and the list has refreshed.
  async function provision(id: string, onDone?: () => void) {
    if (busyRef.current) return;
    setBusy("create");
    setActionErr(null);
    onDoneRef.current = onDone;
    captureLogRef.current = false;
    let started: { targetId: string };
    try {
      started = await api.provisionGateway(id);
    } catch (e) {
      setActionErr(message(e));
      setBusy(null);
      return;
    }
    setProvisionTargetId(started.targetId);
  }

  // runSetup: the empty state's one-click. Stands up a whole gateway from
  // nothing, mirroring home.ts's setupEndpoint step for step, so the panel's
  // path never drifts from the eRPC screen's — it shares SETUP_CHAINS and
  // internalTLSConfig with home.ts rather than a second copy.
  async function runSetup() {
    if (busyRef.current) return;
    setBusy("setup");
    setActionErr(null);
    setSetupLog([]);
    const say = (line: string) => setSetupLog((prev) => [...prev, line]);
    const fail = (msg: string, h?: string) => {
      setBusy(null);
      setActionErr(h ? `${msg} — ${h}` : msg);
    };

    say("Preparing your endpoint…");

    try {
      const targets = await api.listTargets();
      if (!targets.some((t) => t.id === "local")) await api.addTarget({ id: "local", mode: "local" });
    } catch (e) {
      fail(`Could not register this machine: ${message(e)}`, hint(e));
      return;
    }

    try {
      const c = await api.getContainers("local");
      if (!c.docker.reachable) {
        fail(
          c.docker.detail || "A gateway runs as a container, and no Docker engine answered on this machine.",
          c.docker.hint || "Start Docker Desktop, OrbStack or colima, then try again.",
        );
        return;
      }
    } catch (e) {
      fail(`Could not check Docker on this machine: ${message(e)}`, hint(e));
      return;
    }

    say("Creating the gateway…");
    let id = "default";
    try {
      const created = await api.createGateway({
        id,
        placement: { targetId: "local", backend: "docker" },
        config: internalTLSConfig([]),
      });
      id = created.id;
    } catch (e) {
      fail(`Could not create the gateway: ${message(e)}`, hint(e));
      return;
    }

    say("Adding Ethereum and PulseChain endpoints…");
    const networks: api.GatewayNetwork[] = [];
    for (const { chainId } of SETUP_CHAINS) {
      try {
        const set = await api.knownSet(id, chainId);
        const urls = (set.endpoints ?? []).filter((e) => !e.alreadyAdded).map((e) => e.url);
        if (urls.length === 0) continue;
        networks.push({
          ChainID: chainId,
          Upstreams: urls.map((url, i) => ({
            ID: `public-${chainId}-${i + 1}`,
            Kind: "external",
            Endpoint: url,
            Local: false,
            RecentOnly: false,
          })),
        });
      } catch (e) {
        fail(`Could not read valve's set for chain ${chainId}: ${message(e)}`, hint(e));
        return;
      }
    }
    if (networks.length === 0) {
      fail("valve has no measured endpoints for Ethereum or PulseChain right now, so there was nothing to add.");
      return;
    }
    try {
      await api.putGatewayConfig(id, internalTLSConfig(networks));
    } catch (e) {
      fail(`Could not save the endpoints: ${message(e)}`, hint(e));
      return;
    }

    say("Starting the gateway… the first run pulls the eRPC and Caddy images.");
    let started: { targetId: string };
    try {
      started = await api.provisionGateway(id);
    } catch (e) {
      fail(`Could not start the gateway: ${message(e)}`, hint(e));
      return;
    }
    onDoneRef.current = undefined;
    captureLogRef.current = true;
    setProvisionTargetId(started.targetId);
  }

  // --- config changes: write, then re-provision ----------------------------

  async function submitAddNetwork(chainId: number) {
    setDialog(null);
    if (!gw || busyRef.current) return;
    setBusy("create");
    setActionErr(null);
    let urls: string[];
    try {
      const set = await api.knownSet(gw.id, chainId);
      urls = (set.endpoints ?? []).filter((e) => !e.alreadyAdded).map((e) => e.url);
    } catch (e) {
      setBusy(null);
      setActionErr(`Could not read valve's known set for chain ${chainId}: ${message(e)}`);
      return;
    }
    if (urls.length === 0) {
      setBusy(null);
      setActionErr(`valve has no measured endpoints for chain ${chainId} yet, so there was nothing to add.`);
      return;
    }
    const upstreams: api.GatewayUpstream[] = urls.map((url, i) => ({
      ID: `public-${chainId}-${i + 1}`,
      Kind: "external",
      Endpoint: url,
      Local: false,
      RecentOnly: false,
    }));
    try {
      await api.putGatewayConfig(gw.id, withNetwork(gw.config, chainId, upstreams));
    } catch (e) {
      setBusy(null);
      setActionErr(`Could not add the network: ${message(e)}`);
      return;
    }
    // provision() guards on busy — clear the "create" we set for the writes
    // above, or it would return immediately and never recreate the container.
    setBusy(null);
    const gwId = gw.id;
    await provision(gwId, () => {
      setView({ name: "network", chainId });
      void caps.refetch(true);
    });
  }

  async function openAddNetwork() {
    if (!gw || busyRef.current) return;
    setBusy("add-network");
    setActionErr(null);
    let networks: api.Network[];
    try {
      const catalog = await api.getCatalog();
      networks = catalog.networks ?? [];
    } catch (e) {
      setBusy(null);
      setActionErr(`Could not load the network catalog: ${message(e)}`);
      return;
    }
    setBusy(null);
    const present = new Set((gw.networks ?? []).map((n) => n.chainId));
    const picks = networks.filter((n) => !present.has(n.ChainID)).map((n) => ({ chainId: n.ChainID, name: n.Name }));
    if (!present.has(DEVNET_CHAIN_ID)) picks.push({ chainId: DEVNET_CHAIN_ID, name: "Devnet" });
    if (picks.length === 0) {
      setActionErr("Every network valve's catalog knows about is already configured on this gateway.");
      return;
    }
    setDialog({ kind: "add-network", picks });
  }

  async function submitAddEndpoint(chainId: number, url: string) {
    if (!gw) return;
    const up: api.GatewayUpstream = {
      ID: crypto.randomUUID(),
      Kind: "external",
      Endpoint: url,
      Local: false,
      RecentOnly: false,
      Name: endpointNameFromUrl(url),
    };
    try {
      await putCfgMut.mutateAsync({ gid: gw.id, config: withUpstream(gw.config, chainId, up) });
    } catch (e) {
      throw new Error(`Could not add the endpoint: ${message(e)}`);
    }
    const gwId = gw.id;
    setDialog(null);
    await provision(gwId, () => void caps.refetch(true));
  }

  async function submitRename(chainId: number, upstreamId: string, name: string) {
    if (!gw) return;
    const cfgUp = findConfigUpstream(gw, chainId, upstreamId);
    if (!cfgUp) {
      setDialog(null);
      return;
    }
    const nextUp: api.GatewayUpstream = { ...cfgUp, Name: name || undefined };
    try {
      await putCfgMut.mutateAsync({ gid: gw.id, config: withUpstream(gw.config, chainId, nextUp) });
    } catch (e) {
      throw new Error(`Could not rename the endpoint: ${message(e)}`);
    }
    const gwId = gw.id;
    setDialog(null);
    await provision(gwId);
  }

  async function submitEditAddress(chainId: number, upstreamId: string, url: string) {
    if (!gw) return;
    const cfgUp = findConfigUpstream(gw, chainId, upstreamId);
    if (!cfgUp) {
      setDialog(null);
      return;
    }
    const nextUp: api.GatewayUpstream = { ...cfgUp, Endpoint: url };
    try {
      await putCfgMut.mutateAsync({ gid: gw.id, config: withUpstream(gw.config, chainId, nextUp) });
    } catch (e) {
      throw new Error(`Could not save the address: ${message(e)}`);
    }
    const gwId = gw.id;
    setDialog(null);
    await provision(gwId);
  }

  async function removeNetwork(chainId: number) {
    setDialog(null);
    if (!gw) return;
    setNetworkErr(null);
    try {
      await putCfgMut.mutateAsync({ gid: gw.id, config: withoutNetwork(gw.config, chainId) });
    } catch (e) {
      setNetworkErr(`Could not remove the network: ${message(e)}`);
      return;
    }
    const gwId = gw.id;
    setView({ name: "list" });
    await provision(gwId);
  }

  async function removeEndpoint(chainId: number, upstreamId: string) {
    setDialog(null);
    if (!gw) return;
    setEndpointErr(null);
    try {
      await putCfgMut.mutateAsync({ gid: gw.id, config: withoutUpstream(gw.config, chainId, upstreamId) });
    } catch (e) {
      setEndpointErr(`Could not remove the endpoint: ${message(e)}`);
      return;
    }
    const gwId = gw.id;
    setView({ name: "network", chainId });
    await provision(gwId);
  }

  async function runWipe() {
    setDialog(null);
    if (!gw) return;
    setBusy("wipe");
    setActionErr(null);
    try {
      const result = await wipeMut.mutateAsync(gw.id);
      if (result.error) setActionErr(result.error);
    } catch (e) {
      setActionErr(`wipe failed: ${message(e)}`);
    }
    setBusy(null);
    void qc.invalidateQueries({ queryKey: ["gateways"] });
  }

  // --- navigation + the master button --------------------------------------

  function onPower() {
    if (!gw || busyRef.current) return;
    const m = masterState(gw);
    const pa = primaryAction(gw, m);
    if (pa === "stop" || pa === "start") void runAction(gw.id, pa);
    else if (pa === "create") void provision(gw.id);
  }

  function onChip(action: string) {
    if (!gw || busyRef.current) return;
    if (action === "start" || action === "stop" || action === "restart") void runAction(gw.id, action);
    else if (action === "create" || action === "recreate") void provision(gw.id);
  }

  function recheck() {
    void caps.refetch(true);
    void gwQuery.refetch();
    void health.refetch();
  }

  // --- render --------------------------------------------------------------

  const capsBusy = caps.isFetching;
  const capsErr = caps.error ? message(caps.error) : null;

  function body() {
    if (gwQuery.isError) {
      return (
        <div className="p-band" style={{ padding: 16, color: "var(--red)" }}>
          {message(gwQuery.error)}
        </div>
      );
    }
    if (gwQuery.isLoading) return null;
    if (gw && view.name === "network") {
      return (
        <NetworkView
          key={view.chainId}
          gw={gw}
          chainId={view.chainId}
          health={health.data}
          caps={caps.data}
          capsBusy={capsBusy}
          capsErr={capsErr}
          busy={busy}
          error={networkErr}
          onBack={() => setView({ name: "list" })}
          onOpenEndpoint={(chainId, upstreamId) => {
            setEndpointErr(null);
            setView({ name: "endpoint", chainId, upstreamId });
          }}
          onAddEndpoint={() => setDialog({ kind: "add-endpoint", chainId: view.chainId })}
          onRemoveNetwork={() => {
            const nv = gw.networks?.find((n) => n.chainId === view.chainId);
            setDialog({ kind: "confirm-remove-network", chainId: view.chainId, name: nv?.name ?? `chain ${view.chainId}` });
          }}
          onVerifyTls={() => api.verifyGatewayTls(gw.id)}
          onRecheck={recheck}
        />
      );
    }
    if (gw && view.name === "endpoint") {
      const chainId = view.chainId;
      const upstreamId = view.upstreamId;
      return (
        <EndpointView
          key={`${chainId}/${upstreamId}`}
          gw={gw}
          chainId={chainId}
          upstreamId={upstreamId}
          health={health.data}
          caps={caps.data}
          capsBusy={capsBusy}
          capsErr={capsErr}
          error={endpointErr}
          onBack={() => setView({ name: "network", chainId })}
          onRename={() => {
            const up = findUpstream(gw, chainId, upstreamId);
            if (up) setDialog({ kind: "rename", chainId, upstreamId, current: up.label });
          }}
          onEditAddress={() => {
            const up = findUpstream(gw, chainId, upstreamId);
            if (up && up.kind === "external") setDialog({ kind: "edit-address", chainId, upstreamId, current: up.endpoint });
          }}
          onRemove={() => {
            const nv = gw.networks?.find((n) => n.chainId === chainId);
            const up = findUpstream(gw, chainId, upstreamId);
            setDialog({
              kind: "confirm-remove-endpoint",
              chainId,
              upstreamId,
              label: up?.label ?? "this endpoint",
              networkName: nv?.name ?? `chain ${chainId}`,
            });
          }}
          onRecheck={recheck}
        />
      );
    }
    return (
      <ListView
        gw={gw}
        health={health.data}
        busy={busy}
        actionErr={actionErr}
        setupLog={setupLog}
        onSetup={() => void runSetup()}
        onPower={onPower}
        onChip={onChip}
        onOpenSettings={() => setDialog({ kind: "settings" })}
        onOpenNetwork={(chainId) => {
          setNetworkErr(null);
          setView({ name: "network", chainId });
        }}
        onAddNetwork={() => void openAddNetwork()}
      />
    );
  }

  return (
    <>
      <Sprite />
      <div className="p-wrap">
        <div className="p-panel">{body()}</div>
      </div>
      {renderDialog()}
    </>
  );

  function renderDialog() {
    if (!dialog) return null;
    switch (dialog.kind) {
      case "settings":
        return (
          <SettingsSheet
            actions={gw?.actions}
            busy={busy}
            onWipe={() => setDialog({ kind: "confirm-wipe" })}
            onClose={() => setDialog(null)}
          />
        );
      case "confirm-wipe":
        return gw ? (
          <ConfirmDialog
            title={`Wipe ${gw.label}`}
            body={`This destroys ${gw.wipeDiscards}. Every chain it fronts stops being served until it comes back. Nothing behind it — no node, no devnet, no public endpoint — is touched.`}
            confirmLabel="Wipe"
            danger
            onConfirm={() => void runWipe()}
            onCancel={() => setDialog(null)}
          />
        ) : null;
      case "add-network":
        return (
          <AddNetworkDialog picks={dialog.picks} onPick={(chainId) => void submitAddNetwork(chainId)} onCancel={() => setDialog(null)} />
        );
      case "add-endpoint": {
        const chainId = dialog.chainId;
        return (
          <TextInputDialog
            title="Add an endpoint by URL"
            hint="http://, https://, ws:// or wss://."
            label="Endpoint"
            initialValue=""
            placeholder="https://rpc.example.com"
            saveLabel="Add endpoint"
            validate={validateUrlScheme}
            onSave={(url) => submitAddEndpoint(chainId, url)}
            onCancel={() => setDialog(null)}
          />
        );
      }
      case "rename": {
        const { chainId, upstreamId } = dialog;
        return (
          <TextInputDialog
            title="Rename endpoint"
            hint="Clear it to fall back to the automatic name."
            label="Name"
            initialValue={dialog.current}
            saveLabel="Save"
            onSave={(name) => submitRename(chainId, upstreamId, name)}
            onCancel={() => setDialog(null)}
          />
        );
      }
      case "edit-address": {
        const { chainId, upstreamId } = dialog;
        return (
          <TextInputDialog
            title="Edit endpoint address"
            hint="http://, https://, ws:// or wss://."
            label="URL"
            initialValue={dialog.current}
            placeholder="https://rpc.example.com"
            saveLabel="Save"
            validate={validateUrlScheme}
            onSave={(url) => submitEditAddress(chainId, upstreamId, url)}
            onCancel={() => setDialog(null)}
          />
        );
      }
      case "confirm-remove-network": {
        const chainId = dialog.chainId;
        return (
          <ConfirmDialog
            title="Remove network"
            body={`Stop serving ${dialog.name}?`}
            confirmLabel="Remove"
            danger
            onConfirm={() => void removeNetwork(chainId)}
            onCancel={() => setDialog(null)}
          />
        );
      }
      case "confirm-remove-endpoint": {
        const { chainId, upstreamId } = dialog;
        return (
          <ConfirmDialog
            title="Remove endpoint"
            body={`Stop routing to ${dialog.label}? The gateway keeps balancing across whatever else remains on ${dialog.networkName}.`}
            confirmLabel="Remove"
            danger
            onConfirm={() => void removeEndpoint(chainId, upstreamId)}
            onCancel={() => setDialog(null)}
          />
        );
      }
    }
  }
}

// primaryGateway: the panel manages one gateway — the local one, else the first.
function primaryGateway(gws: api.GatewayView[] | null): api.GatewayView | null {
  if (!gws || gws.length === 0) return null;
  return gws.find((g) => g.placement.targetId === "local") ?? gws[0];
}

// findUpstream reads the resolved, displayable form (label/endpoint/kind) — what
// the endpoint-detail dialogs show.
function findUpstream(g: api.GatewayView, chainId: number, upstreamId: string): api.UpstreamView | undefined {
  return g.networks?.find((n) => n.chainId === chainId)?.upstreams?.find((u) => u.id === upstreamId);
}

// findConfigUpstream reads the STORED GatewayUpstream — the object every config
// write mutates (via withUpstream), never the server-resolved UpstreamView.
function findConfigUpstream(g: api.GatewayView, chainId: number, upstreamId: string): api.GatewayUpstream | undefined {
  return g.config.Networks?.find((n) => n.ChainID === chainId)?.Upstreams.find((u) => u.ID === upstreamId);
}

function message(e: unknown): string {
  return e instanceof Error ? e.message : String(e);
}

// hint surfaces the server's operator-facing hint verbatim (e.g. "start Docker
// Desktop / OrbStack / colima").
function hint(e: unknown): string | undefined {
  return e instanceof api.ApiError ? e.hint : undefined;
}
