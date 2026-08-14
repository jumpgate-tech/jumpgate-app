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
import { SETUP_CHAINS, internalTLSConfig } from "../../lib/gatewaySetup";
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
import { ConfirmDialog, TextInputDialog, validateUrlScheme } from "./Dialogs";
import { AddNetworkDialog } from "./AddNetworkModal";
import { DEVNET_CHAIN_ID, withDevnetUpstream } from "../Rpc/rpcModel";

// FINAL_STEP is the id every gateway setup plan ends on (mirrors rpc.ts). The
// setup stream is done when a step errors, or when this step reports done.
const FINAL_STEP = "run";

type View =
  | { name: "list" }
  | { name: "network"; chainId: number }
  | { name: "endpoint"; chainId: number; upstreamId: string };

type Dialog =
  | { kind: "settings" }
  | { kind: "confirm-wipe" }
  | { kind: "add-network" }
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
    // Keep the setup log on an ERROR finish — the "Preparing… / Creating… /
    // Starting…" narration and streamed step lines are exactly the diagnostic
    // context the operator needs. Only a clean finish clears it.
    if (captureLogRef.current && !ev.err) setSetupLog([]);
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
  // ensureDockerReady makes sure the local Docker engine is up before we try to
  // provision the gateway container — so the power button never dead-ends on a
  // raw "docker not found". If Docker is stopped and the app can start it
  // (macOS), it launches Docker and waits, narrating via `say` when given.
  // Returns false (and sets actionErr) when Docker can't be made ready.
  async function ensureDockerReady(say?: (line: string) => void): Promise<boolean> {
    let st: api.DockerStatus;
    try {
      st = await api.getDocker();
    } catch (e) {
      setActionErr(`Could not check Docker: ${message(e)}`);
      return false;
    }
    if (st.running) return true;
    if (!st.present) {
      setActionErr(
        st.hint ??
          "Docker isn't installed on this machine. Install Docker Desktop or OrbStack, or point Jumpgate at a remote machine.",
      );
      return false;
    }
    if (!st.canStart) {
      setActionErr(st.hint ?? "Docker is installed but not running. Start the Docker engine, then try again.");
      return false;
    }
    say?.("Starting Docker…");
    try {
      await api.startDocker();
    } catch {
      // A failed launch call is not fatal — Docker may already be starting;
      // fall through to the poll and let it settle.
    }
    // Poll until the daemon answers. Docker Desktop can take a while to be
    // ready on a cold start, so give it a generous window (~100s).
    for (let i = 0; i < 40; i++) {
      await new Promise((r) => setTimeout(r, 2500));
      try {
        if ((await api.getDocker()).running) return true;
      } catch {
        // transient — keep waiting.
      }
    }
    setActionErr("Docker is taking a while to start. Give it a moment, then press Set up again.");
    return false;
  }

  async function provision(id: string, onDone?: () => void) {
    if (busyRef.current) return;
    setBusy("create");
    setActionErr(null);
    if (!(await ensureDockerReady())) {
      setBusy(null);
      return;
    }
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
  // nothing, so the panel's path never drifts from the eRPC screen's — it
  // shares SETUP_CHAINS and internalTLSConfig (lib/gatewaySetup.ts) rather
  // than a second copy.
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

    // Docker readiness: start it for the operator (macOS) and wait, rather than
    // failing with "go start Docker yourself". ensureDockerReady sets actionErr
    // and returns false when Docker can't be made ready.
    if (!(await ensureDockerReady(say))) {
      setBusy(null);
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
        fail(`Could not read the curated endpoint set for chain ${chainId}: ${message(e)}`, hint(e));
        return;
      }
    }
    if (networks.length === 0) {
      fail("There are no measured public endpoints for Ethereum or PulseChain right now, so there was nothing to add.");
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

    // Devnet is reth --dev on this machine, not a public chain — it is fronted
    // by a managed-devnet upstream, not by discovered public endpoints. The
    // devnet has to already be running on the placement (same rule the RPC
    // screen enforces); if it isn't, say so rather than provisioning a network
    // that can never come up.
    if (chainId === DEVNET_CHAIN_ID) {
      const targetId = gw.placement.targetId;
      let hasDevnet = false;
      try {
        const resp = await api.getGateways();
        hasDevnet = (resp.targets ?? []).some((t) => t.id === targetId && t.hasDevnet);
      } catch (e) {
        setBusy(null);
        setActionErr(`Could not check for a devnet on ${targetId}: ${message(e)}`);
        return;
      }
      if (!hasDevnet) {
        setBusy(null);
        setActionErr(`No devnet is running on ${targetId} yet — create one from Machines → ${targetId}, then add it here.`);
        return;
      }
      try {
        await api.putGatewayConfig(gw.id, withDevnetUpstream(gw.config, DEVNET_CHAIN_ID, targetId));
      } catch (e) {
        setBusy(null);
        setActionErr(`Could not add the devnet: ${message(e)}`);
        return;
      }
      setBusy(null);
      const gwId = gw.id;
      await provision(gwId, () => {
        setView({ name: "network", chainId: DEVNET_CHAIN_ID });
        void caps.refetch(true);
      });
      return;
    }

    // Endpoints to wire: valve's measured known set first — chains 1/369/943,
    // which already carry g4mm4, pulsechain.com and valve — else discover the
    // live public endpoints from chainlist. That fallback is what lets the
    // searchable picker front ANY chain, not just the three valve has curated,
    // and it wires every endpoint that answers so eRPC can route across them.
    let urls: string[] = [];
    try {
      const set = await api.knownSet(gw.id, chainId);
      urls = (set.endpoints ?? []).filter((e) => !e.alreadyAdded).map((e) => e.url);
    } catch {
      // A missing/failed known set is not fatal — fall through to discovery.
    }
    if (urls.length === 0) {
      try {
        const res = await api.discoverEndpoints(chainId);
        urls = (res.endpoints ?? [])
          .filter((e) => e.status === "live" || e.status === "unprobed")
          .map((e) => e.url);
      } catch (e) {
        setBusy(null);
        setActionErr(`Could not find endpoints for chain ${chainId}: ${message(e)}`);
        return;
      }
    }
    if (urls.length === 0) {
      setBusy(null);
      setActionErr(
        `No public endpoints answered for chain ${chainId} right now, so it wasn't added. Try again in a moment.`,
      );
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

  // The shared modal carries its own catalogue (viem curated + the full
  // chainlist) and filters out what's already fronted, so opening it is just
  // showing the dialog — no catalog fetch or pick-list building here anymore.
  function openAddNetwork() {
    if (!gw || busyRef.current) return;
    setActionErr(null);
    setDialog({ kind: "add-network" });
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
    if (!gw || busyRef.current) return;
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
    if (!gw || busyRef.current) return;
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
    if (!gw || busyRef.current) return;
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
    if (!gw || busyRef.current) return;
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
        <div className="p-band p-empty">
          <div className="p-emptytitle">Couldn&apos;t load your gateway</div>
          <div className="p-emptysub">{message(gwQuery.error)}</div>
          <button type="button" className="btn" style={{ marginTop: 12 }} onClick={() => void gwQuery.refetch()}>
            Retry
          </button>
        </div>
      );
    }
    if (gwQuery.isLoading) {
      return (
        <div className="p-band p-empty" aria-busy="true">
          <div className="p-emptysub">Loading…</div>
        </div>
      );
    }
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
          busy={busy}
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
        <div className="p-panel">
          {view.name !== "list" && (busy || actionErr) ? (
            <div className="p-band p-provision" role={actionErr && !busy ? "alert" : "status"}>
              {busy ? (
                <span className="p-emptysub">Applying changes…</span>
              ) : (
                <span className="p-provision-err">{actionErr}</span>
              )}
            </div>
          ) : null}
          {body()}
        </div>
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
          <AddNetworkDialog
            presentChainIds={(gw?.networks ?? []).map((n) => n.chainId)}
            extraPinned={[{ chainId: DEVNET_CHAIN_ID, name: "Devnet", testnet: true }]}
            onPick={(chainId) => void submitAddNetwork(chainId)}
            onCancel={() => setDialog(null)}
          />
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
