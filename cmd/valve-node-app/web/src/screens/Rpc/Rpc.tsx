// #/rpc — the operator-grade eRPC/gateway management screen, converted from
// rpc.ts. A machine runs one gateway, and that gateway fronts as many chains as
// you list; each chain leads with the URL a wallet dials and whether it is
// healthy, with the operator detail one click away.
//
// This container owns only the FLEET-level concerns: the gateway list, the
// controller's OS (for the manual trust command), adding a gateway, and any
// leftover container on a machine with no surviving gateway. Everything scoped
// to one gateway lives in <GatewayCard>, which fetches its own traffic and
// capabilities — a hook per card rather than a loop here.
import { useState } from "react";
import { useQueryClient } from "@tanstack/react-query";
import * as api from "../../api";
import { useGatewaysFull, useCreateGateway, useDismissOrphan } from "../../hooks/rpc";
import { useHost } from "../../hooks/target";
import { Footer } from "../../components/Footer";
import { GatewayCard } from "./GatewayCard";
import { OrphanBanner } from "./OrphanBanner";
import { AddGatewayDialog, MessageDialog } from "./RpcDialogs";
import { canAddGatewayOn } from "./rpcModel";

type Dialog = { kind: "add-gw" } | { kind: "no-machines" } | { kind: "all-have-gateways" };

function message(e: unknown): string {
  return e instanceof Error ? e.message : String(e);
}

export function Rpc() {
  const qc = useQueryClient();
  const gwQuery = useGatewaysFull();
  const host = useHost();
  const createMut = useCreateGateway();
  const dismissMut = useDismissOrphan();

  const [dialog, setDialog] = useState<Dialog | null>(null);
  const [createErr, setCreateErr] = useState<string | null>(null);
  const [orphanErr, setOrphanErr] = useState<Record<string, string | null>>({});

  const hostOS = host.data?.os ?? "";

  function refresh() {
    void qc.invalidateQueries({ queryKey: ["gateways"] });
    void qc.invalidateQueries({ queryKey: ["gwTraffic"] });
    void qc.invalidateQueries({ queryKey: ["gwCaps"] });
  }

  function openAddGateway() {
    const allTargets = gwQuery.data?.targets ?? [];
    const gateways = gwQuery.data?.gateways ?? [];
    setCreateErr(null);
    if (allTargets.length === 0) {
      setDialog({ kind: "no-machines" });
      return;
    }
    if (!allTargets.some((t) => canAddGatewayOn(t.id, gateways))) {
      setDialog({ kind: "all-have-gateways" });
      return;
    }
    setDialog({ kind: "add-gw" });
  }

  async function createGateway(values: { id: string; targetId: string; port: number }) {
    if (!values.id) {
      setCreateErr("Give it a name — it becomes the container's name, which is how it is found again.");
      return;
    }
    if (!values.targetId) {
      setCreateErr("Pick the machine it runs on.");
      return;
    }
    try {
      await createMut.mutateAsync({
        id: values.id,
        placement: { targetId: values.targetId, backend: "docker" },
        config: { ProjectID: "main", BindAddr: "127.0.0.1", Port: values.port, Networks: [] },
      });
    } catch (e) {
      setCreateErr(message(e));
      return;
    }
    setDialog(null);
  }

  function dismissOrphan(name: string) {
    setOrphanErr((e) => ({ ...e, [name]: null }));
    dismissMut.mutate(name, { onError: (err) => setOrphanErr((e) => ({ ...e, [name]: message(err) })) });
  }

  return (
    <>
      <div className="page-head">
        <h1>RPC</h1>
        <button className="btn btn-ghost" onClick={refresh}>
          Refresh
        </button>
      </div>
      <p className="muted">
        A machine runs one gateway, and that gateway fronts as many chains as you list. Each chain below leads with the
        URL you point a wallet or dApp at and whether it is healthy. The operator detail — every upstream, its
        capabilities and share — is one click away under each chain's “Details”.
      </p>
      {body()}
      <Footer />
      {renderDialog()}
    </>
  );

  function body() {
    if (gwQuery.isError) {
      return <p className="error">Could not read the gateways: {message(gwQuery.error)}</p>;
    }
    if (!gwQuery.data) {
      return <p className="muted">Loading…</p>;
    }
    const gateways = gwQuery.data.gateways ?? [];
    const targets = gwQuery.data.targets ?? [];
    const sources = gwQuery.data.sources ?? [];
    const presets = gwQuery.data.presets ?? [];
    const orphans = gwQuery.data.orphans ?? [];
    const many = gateways.length > 1;
    const withGateway = new Set(gateways.map((g) => g.placement.targetId));
    const looseOrphans = orphans.filter((o) => !withGateway.has(o.targetId));
    const canAdd = targets.some((t) => canAddGatewayOn(t.id, gateways));

    return (
      <>
        {gateways.map((gw) => (
          <GatewayCard
            key={gw.id}
            gw={gw}
            targets={targets}
            sources={sources}
            presets={presets}
            orphans={orphans.filter((o) => o.targetId === gw.placement.targetId)}
            showMachine={many}
            hostOS={hostOS}
          />
        ))}
        {gateways.length === 0 ? <EmptyState targets={targets} /> : null}
        {looseOrphans.map((o) => (
          <OrphanBanner key={o.containerName} orphan={o} error={orphanErr[o.containerName]} onDismiss={dismissOrphan} />
        ))}
        {canAdd ? (
          <div className="card-actions rpc-add-gateway">
            <button className={`btn${gateways.length ? " btn-ghost" : ""}`} onClick={openAddGateway}>
              Add a gateway{gateways.length ? " on another machine" : ""}
            </button>
          </div>
        ) : null}
      </>
    );
  }

  function renderDialog() {
    if (!dialog) return null;
    if (dialog.kind === "no-machines") {
      return (
        <MessageDialog title="No machines yet" onClose={() => setDialog(null)} link={{ href: "#/targets", label: "Go to Machines" }}>
          <p>A gateway is a container, so it has to run somewhere. Add a machine first.</p>
        </MessageDialog>
      );
    }
    if (dialog.kind === "all-have-gateways") {
      return (
        <MessageDialog title="Every machine already has a gateway" onClose={() => setDialog(null)}>
          <p className="muted small">This machine already runs a gateway. Add chains to it rather than creating a second one.</p>
        </MessageDialog>
      );
    }
    const gateways = gwQuery.data?.gateways ?? [];
    const allTargets = gwQuery.data?.targets ?? [];
    const existing = new Set(gateways.map((g) => g.id));
    const addable = allTargets.filter((t) => canAddGatewayOn(t.id, gateways)).map((t) => ({ id: t.id, mode: t.mode }));
    return (
      <AddGatewayDialog
        targets={addable}
        suggestedName={existing.has("default") ? "" : "default"}
        error={createErr}
        onCreate={(values) => void createGateway(values)}
        onCancel={() => setDialog(null)}
      />
    );
  }
}

function EmptyState({ targets }: { targets: api.TargetSummary[] }) {
  if (targets.length === 0) {
    return (
      <div className="card empty-state">
        <p className="muted">
          No machines yet. A gateway is a container, so it has to run somewhere — add a machine on{" "}
          <a href="#/targets">Machines</a> first.
        </p>
      </div>
    );
  }
  return (
    <div className="card empty-state">
      <p className="muted">
        No gateway yet. A gateway is one eRPC instance fronting however many chains you list; it addresses a chain by URL
        path, so a single port serves all of them — and the same path serves WebSocket.
      </p>
    </div>
  );
}
