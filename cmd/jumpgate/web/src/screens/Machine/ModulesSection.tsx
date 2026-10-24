// Machine page → Modules section (spec D11): node modules on this node's
// execution client, then box modules ("Services on this box"), each one
// generic card from modulesModel.moduleCards. Whether a module applies comes
// from /api/catalog's Applies rows only.
//
// A toggle goes through PUT /api/targets/{id}/modules/{module}, which runs
// setup; the run's progress is shown inline from the setup stream. Turning a
// module on first opens a dialog: the D4 restart line for a node module, or
// the box module's own fields (boxFields) for a box module.
import { useEffect, useRef, useState } from "react";
import { useQuery, useQueryClient } from "@tanstack/react-query";
import * as api from "../../api";
import { useCatalog, useTargets } from "../../hooks/target";
import { useGateways } from "../../hooks/gateway";
import { useMonitorStream } from "../../hooks/dashboard";
import { Badge } from "../../components/Badge";
import { Modal } from "../Panel/Modal";
import { copyToClipboard } from "../../ui";
import {
  BOX_FIELD_DEFAULTS,
  boxFields,
  moduleCards,
  POST_WARNING,
  PUBLIC_OPTIONS,
  publicGatewayRefs,
  publicOptions,
  type ModuleCard,
} from "./modulesModel";

// boxName is how the operator knows the box: its SSH host, or "this machine"
// for the local target (the same words the Machine header uses).
function boxName(t: api.Target): string {
  return t.mode === "ssh" && t.ssh ? t.ssh.Host : "this machine";
}

function formatError(err: unknown): string {
  return err instanceof Error ? err.message : String(err);
}

// docsUrl is the module's docs step on jumpgate.app (docs-system spec §3:
// /docs/<pathway>/<step>; module steps live in the "modules" pathway).
export function docsUrl(step: string): string {
  return `https://jumpgate.app/docs/modules/${encodeURIComponent(step)}`;
}

type Confirm = { card: ModuleCard; place: "node" | "box" } | null;

// BOOTNODE_MODULES are the box modules whose identity other nodes can dial.
const BOOTNODE_MODULES = new Set(["el-bootnode", "cl-bootnode"]);

// identityKind names a record for the copy button: an enode or an ENR.
function identityKind(identity: string): string {
  return identity.startsWith("enr:") ? "ENR" : "enode";
}

// fleetNodesOn are the other targets running a node on chainId.
export function fleetNodesOn(targets: api.Target[], chainId: number, selfId: string): api.Target[] {
  return targets.filter((t) => t.id !== selfId && t.wire && t.wire.ChainID === chainId);
}

// withBootnode is a node's wire with entry appended (once) to Bootnodes.
export function withBootnode(wire: api.WireConfig, entry: string): api.WireConfig {
  const list = wire.Bootnodes ?? [];
  return { ...wire, Bootnodes: list.includes(entry) ? list : [...list, entry] };
}

interface Run {
  module: string;
  lines: string[];
  done: boolean;
  err: string | null;
}

export function ModulesSection({ targetId }: { targetId: string }) {
  const qc = useQueryClient();
  const targetsQuery = useTargets();
  const catalogQuery = useCatalog();
  const gatewaysQuery = useGateways();
  const target = targetsQuery.data?.find((t) => t.id === targetId);
  const wire = target?.wire;
  const { snapshot } = useMonitorStream(targetId, !!wire);
  const modulesQuery = useQuery({
    queryKey: ["modules", targetId],
    queryFn: () => api.listModules(targetId),
    refetchInterval: 5000,
    enabled: !!target,
  });

  const [confirm, setConfirm] = useState<Confirm>(null);
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [note, setNote] = useState<string | null>(null);
  const [run, setRun] = useState<Run | null>(null);
  const [useBoot, setUseBoot] = useState<{ card: ModuleCard; chainId: number } | null>(null);
  const [restartedAt, setRestartedAt] = useState<number | undefined>(undefined);
  const streamRef = useRef<(() => void) | null>(null);

  useEffect(() => () => streamRef.current?.(), []);

  if (targetsQuery.isLoading || catalogQuery.isLoading) return <p className="muted">Loading…</p>;
  const loadErr = targetsQuery.error ?? catalogQuery.error;
  if (loadErr) return <p className="error">Failed to load modules: {formatError(loadErr)}</p>;
  if (!target || !catalogQuery.data) return null;

  const catalog = catalogQuery.data;
  const statuses = modulesQuery.data?.modules ?? [];
  const gateStale = !!modulesQuery.data?.gateStale;
  const nodeEnabled = wire?.Modules ?? [];
  const boxEnabled = target.modules ?? [];

  const nodeCards = wire
    ? moduleCards(catalog, "node", {
        chainId: wire.ChainID,
        client: wire.ExecID,
        enabled: nodeEnabled.map((m) => m.id),
        statuses,
        restartedAt,
        gateStale,
        snapshot,
        box: boxName(target),
      }).filter((c) => c.state !== "hidden")
    : [];
  const boxCards = moduleCards(catalog, "box", {
    chainId: wire?.ChainID ?? 0,
    enabled: boxEnabled.map((m) => m.id),
    statuses,
    gateStale,
  }).filter((c) => c.state !== "hidden");

  const publicGateways = publicGatewayRefs(gatewaysQuery.data ?? [], targetId);
  const publicOptionsFor = (id: string) => {
    const m = catalog.modules?.find((x) => x.id === id);
    return m ? publicOptions(m) : [];
  };

  function followRun(module: string): void {
    streamRef.current?.();
    setRun({ module, lines: [], done: false, err: null });
    streamRef.current = api.streamSetup(targetId, (ev) => {
      setRun((r) => {
        if (!r) return r;
        const next = { ...r, lines: ev.line ? [...r.lines, ev.line].slice(-5) : r.lines };
        if (ev.err) next.err = ev.err;
        if (ev.done && ev.stepId === "") next.done = true;
        return next;
      });
      if (ev.done || ev.err) {
        void qc.invalidateQueries({ queryKey: ["targets"] });
        void qc.invalidateQueries({ queryKey: ["modules", targetId] });
      }
    });
  }

  async function submit(module: string, body: api.SetModuleRequest, restartsExec: boolean): Promise<boolean> {
    setBusy(true);
    setError(null);
    setNote(null);
    try {
      const res = await api.setModule(targetId, module, body);
      setConfirm(null);
      if (res.status === "started") {
        if (restartsExec) setRestartedAt(Date.now());
        followRun(module);
      } else if (res.status === "saved-gateways-failed") {
        setNote(
          `Saved. These gateways did not re-provision: ${(res.failed ?? []).map((f) => `${f.id} (${f.error})`).join(", ")}`,
        );
      } else {
        setNote("Saved.");
      }
      void qc.invalidateQueries({ queryKey: ["targets"] });
      return true;
    } catch (err) {
      setError(formatError(err));
      return false;
    } finally {
      setBusy(false);
    }
  }

  function toggle(card: ModuleCard, place: "node" | "box"): void {
    if (card.state === "on") {
      void submit(card.id, place === "box" ? { enabled: false, chainId: boxChain(card.id) } : { enabled: false }, place === "node");
      return;
    }
    setConfirm({ card, place });
  }

  function boxChain(id: string): number | undefined {
    return boxEnabled.find((m) => m.id === id)?.chainId ?? wire?.ChainID;
  }

  if (nodeCards.length === 0 && boxCards.length === 0) {
    return <p className="muted">No modules apply to this machine yet.</p>;
  }

  return (
    <div className="modules-section">
      {error && <p className="error">{error}</p>}
      {note && <p className="muted">{note}</p>}
      {run && (
        <div className="module-run">
          <p className="small">
            {run.err ? (
              <span className="error">{run.err}</span>
            ) : run.done ? (
              "Done."
            ) : (
              `Applying ${run.module}…`
            )}
          </p>
          {run.lines.length > 0 && <pre className="small">{run.lines.join("\n")}</pre>}
        </div>
      )}
      {nodeCards.map((c) => {
        const options = publicOptionsFor(c.id);
        return (
        <ModuleCardView key={c.id} card={c} busy={busy} onToggle={() => toggle(c, "node")}>
          {c.state === "on" && publicGateways.length > 0 && options.length > 0 && (
            <PublicChoice
              moduleId={c.id}
              options={options}
              value={nodeEnabled.find((m) => m.id === c.id)?.public ?? ""}
              disabled={busy}
              onChange={(v) => submit(c.id, { enabled: true, public: v }, false)}
            />
          )}
        </ModuleCardView>
        );
      })}
      {boxCards.length > 0 && (
        <>
          <h3>Services on this box</h3>
          {boxCards.map((c) => {
            const chainId = boxChain(c.id);
            const chainName = catalog.networks?.find((n) => n.ChainID === chainId)?.Name ?? `chain ${chainId}`;
            const offer = c.state === "on" && !!c.identity && BOOTNODE_MODULES.has(c.id) && chainId !== undefined;
            return (
              <ModuleCardView key={c.id} card={c} busy={busy} onToggle={() => toggle(c, "box")}>
                {offer && (
                  <button
                    type="button"
                    className="btn btn-ghost small"
                    disabled={busy}
                    onClick={() => setUseBoot({ card: c, chainId: chainId! })}
                  >
                    Use on my other {chainName} nodes
                  </button>
                )}
              </ModuleCardView>
            );
          })}
        </>
      )}
      {useBoot && (
        <UseBootnodeDialog
          entry={useBoot.card.identity!}
          nodes={fleetNodesOn(targetsQuery.data ?? [], useBoot.chainId, targetId)}
          onClose={() => {
            setUseBoot(null);
            void qc.invalidateQueries({ queryKey: ["targets"] });
          }}
        />
      )}
      {confirm && (
        <ConfirmDialog
          card={confirm.card}
          place={confirm.place}
          busy={busy}
          error={error}
          onCancel={() => setConfirm(null)}
          onConfirm={(fields) => {
            if (confirm.place === "node") {
              void submit(confirm.card.id, { enabled: true, public: "" }, true);
            } else {
              const body: api.SetModuleRequest = { enabled: true, chainId: wire?.ChainID };
              if (body.chainId === undefined) delete body.chainId;
              if (fields.port) body.port = fields.port;
              if (fields.extIp) body.extIp = fields.extIp;
              void submit(confirm.card.id, body, false);
            }
          }}
        />
      )}
    </div>
  );
}

export function ModuleCardView({
  card,
  busy,
  onToggle,
  children,
}: {
  card: ModuleCard;
  busy: boolean;
  onToggle: () => void;
  children?: React.ReactNode;
}) {
  const on = card.state === "on";
  return (
    <div className="card module-card" data-module={card.id}>
      <div className="module-card-head">
        <strong>{card.name}</strong> {card.beta && <Badge text="beta" kind="warn" />}{" "}
        <Badge text={on ? "on" : card.state === "disabled" ? "unavailable" : "off"} kind={on ? "ok" : "neutral"} />
        <button
          type="button"
          className="btn btn-ghost module-toggle"
          disabled={busy || (card.state === "disabled" && !on)}
          aria-label={`${on ? "Turn off" : "Turn on"} ${card.name}`}
          onClick={onToggle}
        >
          {on ? "Turn off" : "Turn on"}
        </button>
      </div>
      <p className="muted small">{card.summary}</p>
      {card.reason && <p className="small">{card.reason}</p>}
      {card.status && <p className="small module-status">{card.status}</p>}
      {card.identity && <IdentityLine identity={card.identity} />}
      {card.note && <p className="small warn module-note">{card.note}</p>}
      {card.resources && <p className="muted small">{card.resources}</p>}
      <p className="small">
        {card.siteUrl && (
          <>
            <a href={card.siteUrl} target="_blank" rel="noreferrer">
              Open {card.siteUrl.replace(/^https?:\/\//, "")}
            </a>{" "}
            ·{" "}
          </>
        )}
        <a href={docsUrl(card.docsStep)} target="_blank" rel="noreferrer">
          Docs
        </a>
      </p>
      {children}
    </div>
  );
}

// PublicChoice is the Public radio (spec D11). Its options are the ones the
// catalog says the module offers; the posting warning shows while "Public
// reads and posting" is the selected option. The selection is held locally
// so it reflects the click before the saved wire is re-read, and goes back
// to the saved value when the change fails.
function PublicChoice({
  moduleId,
  options,
  value,
  disabled,
  onChange,
}: {
  moduleId: string;
  options: typeof PUBLIC_OPTIONS;
  value: api.ModulePublic;
  disabled: boolean;
  onChange: (v: api.ModulePublic) => Promise<boolean>;
}) {
  const [selected, setSelected] = useState<api.ModulePublic>(value);
  useEffect(() => setSelected(value), [value]);
  return (
    <fieldset className="module-public">
      {options.map((o) => (
        <label key={o.value || "local"}>
          <input
            type="radio"
            name={`module-public-${moduleId}`}
            checked={selected === o.value}
            disabled={disabled}
            onChange={() => {
              setSelected(o.value);
              void onChange(o.value).then((ok) => {
                if (!ok) setSelected(value);
              });
            }}
          />{" "}
          {o.label}
        </label>
      ))}
      {selected === "post" && <p className="small warn">{POST_WARNING}</p>}
    </fieldset>
  );
}

function ConfirmDialog({
  card,
  place,
  busy,
  error,
  onCancel,
  onConfirm,
}: {
  card: ModuleCard;
  place: "node" | "box";
  busy: boolean;
  error: string | null;
  onCancel: () => void;
  onConfirm: (fields: { port?: number; extIp?: string }) => void;
}) {
  const fields = place === "box" ? boxFields(card.id) : [];
  const [port, setPort] = useState("");
  const [extIp, setExtIp] = useState("");
  const parsedPort = port.trim() === "" ? undefined : Number(port);
  const portBad = parsedPort !== undefined && !(Number.isInteger(parsedPort) && parsedPort > 0 && parsedPort < 65536);
  return (
    <Modal onClose={onCancel} label={`Turn on ${card.name}`}>
      <h3>Turn on {card.name}</h3>
      {place === "node" && card.restartLine && <p>{card.restartLine}</p>}
      {fields.includes("port") && (
        <label className="field">
          <span>Port</span>
          <input
            aria-label="Port"
            value={port}
            placeholder={String(BOX_FIELD_DEFAULTS[card.id]?.port ?? "")}
            onChange={(e) => setPort(e.target.value)}
          />
        </label>
      )}
      {portBad && <p className="error small">Port must be 1–65535.</p>}
      {fields.includes("extIp") && (
        <label className="field">
          <span>External IP</span>
          <input
            aria-label="External IP"
            value={extIp}
            placeholder="the public IP peers dial"
            onChange={(e) => setExtIp(e.target.value.trim())}
          />
        </label>
      )}
      {error && <p className="error">{error}</p>}
      <div className="wizard-actions">
        <button className="btn btn-ghost" type="button" onClick={onCancel}>
          Cancel
        </button>
        <button
          className="btn btn-primary"
          type="button"
          disabled={busy || portBad}
          onClick={() => onConfirm({ port: parsedPort, extIp: extIp || undefined })}
        >
          Turn on
        </button>
      </div>
    </Modal>
  );
}

// IdentityLine shows a running bootnode's enode/ENR with a copy button.
function IdentityLine({ identity }: { identity: string }) {
  const [copied, setCopied] = useState<string | null>(null);
  const kind = identityKind(identity);
  return (
    <p className="small module-identity-line">
      <code className="module-identity">{identity}</code>{" "}
      <button
        type="button"
        className="btn btn-ghost small"
        aria-label={`Copy ${kind}`}
        onClick={() => {
          void copyToClipboard(identity).then((ok) => setCopied(ok ? "Copied" : "Copy failed; select the text instead"));
        }}
      >
        {copied ?? "Copy"}
      </button>
    </p>
  );
}

// UseBootnodeDialog adds a running bootnode's identity to each of the
// fleet's other nodes on its chain: each node's saved wire, with Bootnodes
// appended, goes through POST /api/targets/{id}/setup, which re-renders the
// units and restarts the one that changed. The dialog says so per node.
function UseBootnodeDialog({
  entry,
  nodes,
  onClose,
}: {
  entry: string;
  nodes: api.Target[];
  onClose: () => void;
}) {
  const el = !entry.startsWith("enr:");
  const [results, setResults] = useState<Record<string, string>>({});
  const [running, setRunning] = useState(false);
  const clientOf = (t: api.Target) => (el ? t.wire!.ExecID : t.wire!.BeaconID);
  const has = (t: api.Target) => (t.wire!.Bootnodes ?? []).includes(entry);
  const todo = nodes.filter((t) => !has(t));

  async function apply(): Promise<void> {
    setRunning(true);
    for (const t of todo) {
      const name = boxName(t);
      try {
        await api.startSetup(t.id, withBootnode(t.wire!, entry));
        setResults((r) => ({ ...r, [t.id]: `${name}: setup started, ${clientOf(t)} restarts.` }));
      } catch (err) {
        setResults((r) => ({ ...r, [t.id]: `${name}: ${formatError(err)}` }));
      }
    }
    setRunning(false);
  }

  const done = Object.keys(results).length > 0;
  return (
    <Modal onClose={onClose} label="Use this bootnode">
      <h3>Use this bootnode</h3>
      {nodes.length === 0 ? (
        <p className="muted">No other nodes on this chain.</p>
      ) : (
        <ul className="small">
          {nodes.map((t) => (
            <li key={t.id}>
              {results[t.id] ??
                (has(t) ? `${boxName(t)}: already uses this bootnode.` : `Restarts ${clientOf(t)} on ${boxName(t)}.`)}
            </li>
          ))}
        </ul>
      )}
      <div className="wizard-actions">
        <button className="btn btn-ghost" type="button" onClick={onClose}>
          {done ? "Close" : "Cancel"}
        </button>
        {!done && (
          <button
            className="btn btn-primary"
            type="button"
            disabled={running || todo.length === 0}
            onClick={() => void apply()}
          >
            Use bootnode
          </button>
        )}
      </div>
    </Modal>
  );
}
