// The eRPC screen's dialogs — each composing Panel's <Modal> shell (portaled,
// Escape/backdrop close), the React replacement for ui.ts's openModal/closeModal
// dance. The generic confirm/text-input dialogs are reused from Panel/Dialogs;
// the ones here are the rpc-specific pickers and reports.
import { useEffect, useRef, useState } from "react";
import type {
  ChainlistResult,
  KnownSetResponse,
  UpstreamSource,
  WipeResult,
} from "../../api";
import { Modal } from "../Panel/Modal";
import { defaultSelection } from "./rpcModel";

// --- add a gateway -------------------------------------------------------

export function AddGatewayDialog({
  targets,
  suggestedName,
  error,
  onCreate,
  onCancel,
}: {
  targets: { id: string; mode: string }[];
  suggestedName: string;
  error: string | null;
  onCreate: (values: { id: string; targetId: string; port: number }) => void;
  onCancel: () => void;
}) {
  const [id, setId] = useState(suggestedName);
  const [targetId, setTargetId] = useState(targets[0]?.id ?? "");
  const [port, setPort] = useState("4000");
  const idRef = useRef<HTMLInputElement>(null);
  useEffect(() => idRef.current?.focus(), []);

  return (
    <Modal onClose={onCancel}>
      <h2>Add a gateway</h2>
      <p className="muted small">
        A gateway NAMES the machine it runs on; it does not belong to it. Its endpoints can be anywhere — this machine's
        devnet, a node on another box, a public endpoint.
      </p>
      <label>
        Name <span className="muted">— becomes its container name, so lower-case letters, digits, dot, dash or underscore</span>
        <input ref={idRef} type="text" autoComplete="off" spellCheck={false} value={id} placeholder="edge" onChange={(e) => setId(e.target.value)} />
      </label>
      <label>
        Runs on
        <select value={targetId} onChange={(e) => setTargetId(e.target.value)}>
          {targets.map((t) => (
            <option key={t.id} value={t.id}>
              {t.id} ({t.mode})
            </option>
          ))}
        </select>
      </label>
      <label>
        Listen port
        <input type="text" inputMode="numeric" value={port} autoComplete="off" onChange={(e) => setPort(e.target.value)} />
      </label>
      {error ? <p className="error small">{error}</p> : null}
      <div className="modal-actions">
        <button className="btn btn-ghost" onClick={onCancel}>
          Cancel
        </button>
        <button
          className="btn"
          onClick={() => {
            const p = Number.parseInt(port.trim(), 10);
            onCreate({ id: id.trim(), targetId, port: Number.isFinite(p) ? p : 4000 });
          }}
        >
          Create gateway
        </button>
      </div>
    </Modal>
  );
}

// A one-message modal — the "no machines yet" / "every machine already has a
// gateway" states, and the create-a-devnet-first nudge.
export function MessageDialog({
  title,
  children,
  onClose,
  link,
}: {
  title: string;
  children: React.ReactNode;
  onClose: () => void;
  link?: { href: string; label: string };
}) {
  return (
    <Modal onClose={onClose}>
      <h2>{title}</h2>
      {children}
      <div className="modal-actions">
        <button className="btn btn-ghost" onClick={onClose}>
          {link ? "Cancel" : "Close"}
        </button>
        {link ? (
          <a className="btn" href={link.href} onClick={onClose}>
            {link.label}
          </a>
        ) : null}
      </div>
    </Modal>
  );
}

// --- add an endpoint -----------------------------------------------------

export function AddEndpointDialog({
  networkName,
  chainId,
  offer,
  onSource,
  onKnownSet,
  onManual,
  onCancel,
}: {
  networkName: string;
  chainId: number;
  offer: UpstreamSource[];
  onSource: (kind: UpstreamSource["kind"], targetId: string) => void;
  onKnownSet: () => void;
  onManual: () => void;
  onCancel: () => void;
}) {
  return (
    <Modal onClose={onCancel}>
      <h2>Add an endpoint for {networkName}</h2>
      {offer.length ? (
        <>
          <p className="muted small">
            Machines you manage that serve this chain. These are stored as a reference, not a URL — move the node's port
            and the gateway follows it.
          </p>
          <ul className="plain-list rpc-picker">
            {offer.map((s) => (
              <li key={`${s.kind}|${s.targetId}`}>
                <button className="btn btn-ghost rpc-picker-option" onClick={() => onSource(s.kind, s.targetId)}>
                  <span>{s.label}</span>
                  <span className="muted small">{s.endpoint}</span>
                </button>
              </li>
            ))}
          </ul>
        </>
      ) : (
        <p className="muted small">No machine you manage serves chain {chainId}.</p>
      )}
      <div className="modal-actions modal-actions-stack">
        <button className="btn" onClick={onKnownSet}>
          Add valve's set…
        </button>
        <button className="btn btn-ghost" onClick={onManual}>
          Enter a URL by hand…
        </button>
      </div>
      <div className="modal-actions">
        <button className="btn btn-ghost" onClick={onCancel}>
          Cancel
        </button>
      </div>
    </Modal>
  );
}

export function ManualEndpointDialog({ onAdd, onCancel }: { onAdd: (url: string, recentOnly: boolean) => void; onCancel: () => void }) {
  const [url, setUrl] = useState("");
  const [recent, setRecent] = useState(false);
  const [err, setErr] = useState<string | null>(null);
  const ref = useRef<HTMLInputElement>(null);
  useEffect(() => ref.current?.focus(), []);
  function add() {
    const v = url.trim();
    if (!/^(https?|wss?):\/\//i.test(v)) {
      setErr("It needs a scheme eRPC can dial: http://, https://, ws:// or wss://.");
      return;
    }
    onAdd(v, recent);
  }
  return (
    <Modal onClose={onCancel}>
      <h2>Add an endpoint by URL</h2>
      <p className="muted small">
        http://, https://, ws:// or wss://. eRPC infers WebSocket from the scheme — there is no separate setting — and a
        ws upstream also serves ordinary calls.
      </p>
      <label>
        Endpoint
        <input ref={ref} type="text" autoComplete="off" spellCheck={false} placeholder="https://rpc.example.com" value={url} onChange={(e) => setUrl(e.target.value)} onKeyDown={(e) => { if (e.key === "Enter") add(); }} />
      </label>
      <label className="radio">
        <input type="checkbox" checked={recent} onChange={(e) => setRecent(e.target.checked)} />
        Recent blocks only <span className="muted">— tick for a pruned node that cannot answer historical state</span>
      </label>
      {err ? <p className="error small">{err}</p> : null}
      <div className="modal-actions">
        <button className="btn btn-ghost" onClick={onCancel}>
          Cancel
        </button>
        <button className="btn" onClick={add}>
          Add endpoint
        </button>
      </div>
    </Modal>
  );
}

// --- known set / discover ------------------------------------------------

export function KnownSetDialog({
  chainId,
  set,
  error,
  onAdd,
  onDiscover,
  onCancel,
}: {
  chainId: number;
  set: KnownSetResponse | null;
  error: string | null;
  onAdd: (urls: string[]) => void;
  onDiscover: () => void;
  onCancel: () => void;
}) {
  if (error) {
    return (
      <Modal onClose={onCancel}>
        <h2>Endpoints for chain {chainId}</h2>
        <p className="error small">Could not read the set: {error}</p>
        <div className="modal-actions">
          <button className="btn btn-ghost" onClick={onCancel}>
            Close
          </button>
        </div>
      </Modal>
    );
  }
  if (!set) {
    return (
      <Modal onClose={onCancel}>
        <h2>Endpoints for chain {chainId}</h2>
        <p>
          <span className="spinner" aria-label="working" /> reading valve's set…
        </p>
        <div className="modal-actions">
          <button className="btn btn-ghost" onClick={onCancel}>
            Cancel
          </button>
        </div>
      </Modal>
    );
  }
  const eps = set.endpoints ?? [];
  const toAdd = eps.filter((e) => !e.alreadyAdded).map((e) => e.url);
  const providers = new Set(eps.map((e) => e.provider)).size;
  return (
    <Modal onClose={onCancel}>
      <h2>Endpoints for chain {chainId}</h2>
      {eps.length ? (
        <>
          <p className="muted small">
            {providers} providers valve has measured, in the order the gateway should prefer them — {eps.length} entries,
            because a provider that serves both schemes appears twice: eRPC reads WebSocket off the scheme, so an{" "}
            <code>https://</code> upstream never answers <code>eth_subscribe</code> however well the host speaks it.
          </p>
          <ul className="plain-list">
            {eps.map((e) => (
              <li key={e.url}>
                <code>{e.url}</code> <span className="muted small">{e.provider}</span>{" "}
                {e.websocket ? <span className="t ws">websocket</span> : null}
                {e.archive ? <span className="t ar">archive</span> : null}
                {e.alreadyAdded ? <span className="t dup">already added</span> : null}
              </li>
            ))}
          </ul>
        </>
      ) : (
        <p className="muted small">valve has not measured a set for this chain yet — choose from the full list below.</p>
      )}
      {set.usingDefaultKey ? (
        <p className="muted small">
          valve's entries here are resolved with the key that ships with the app, so this works with no setup. To use an
          account of your own instead, put it in Settings under <code>VALVE_API_KEY</code>.
        </p>
      ) : (
        <p className="muted small">
          valve's entries here are resolved with your own <code>VALVE_API_KEY</code>.
        </p>
      )}
      <div className="modal-actions">
        <button className="btn btn-ghost" onClick={onCancel}>
          Cancel
        </button>
        <button className="btn btn-ghost" onClick={onDiscover}>
          Choose from the full list
        </button>
        <button className="btn" disabled={!toAdd.length} onClick={() => onAdd(toAdd)}>
          {toAdd.length ? `Add ${toAdd.length}` : "Nothing to add"}
        </button>
      </div>
    </Modal>
  );
}

export function DiscoverDialog({
  chainId,
  result,
  onAdd,
  onCancel,
}: {
  chainId: number;
  result: ChainlistResult | null;
  onAdd: (urls: string[]) => void;
  onCancel: () => void;
}) {
  const live = (result?.endpoints ?? []).filter((e) => e.status === "live" || e.status === "unprobed");
  const rejected = (result?.endpoints ?? []).filter((e) => e.status === "rejected");
  const [selected, setSelected] = useState<Set<string> | null>(null);
  // Initialise the tick set once, from the first result that arrives.
  useEffect(() => {
    if (result && selected === null) setSelected(defaultSelection(live));
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [result]);
  const sel = selected ?? new Set<string>();

  if (!result) {
    return (
      <Modal onClose={onCancel}>
        <h2>Public endpoints for chain {chainId}</h2>
        <p className="muted small">
          Reading chainid.network and asking each endpoint for its chain id. Only the ones that answer — with the right
          chain — are offered, because a feed lists plenty that no longer work.
        </p>
        <p>
          <span className="spinner" aria-label="working" /> probing…
        </p>
        <div className="modal-actions">
          <button className="btn btn-ghost" onClick={onCancel}>
            Cancel
          </button>
        </div>
      </Modal>
    );
  }

  function toggle(url: string) {
    setSelected((prev) => {
      const next = new Set(prev ?? sel);
      if (next.has(url)) next.delete(url);
      else next.add(url);
      return next;
    });
  }

  return (
    <Modal onClose={onCancel}>
      <h2>Public endpoints for chain {chainId}</h2>
      {result.source === "vendored" ? (
        <div className="banner banner-warn">
          chainid.network was unreachable, so this is the list valve-node-app ships with.
          {result.fetchError ? <div className="small">{result.fetchError}</div> : null}
        </div>
      ) : null}
      {live.length ? (
        <>
          <p className="muted small">
            {live.length} answered for this chain. The fastest are already ticked — more than one endpoint is what makes
            a chain survive an outage.
          </p>
          <ul className="plain-list rpc-picker">
            {live.map((e) => (
              <li key={e.url}>
                <label className="rpc-picker-option">
                  <input type="checkbox" checked={sel.has(e.url)} onChange={() => toggle(e.url)} />
                  <span>
                    <code>{e.url}</code>
                  </span>
                  <span className="muted small">
                    {e.status === "live" ? `answered in ${e.latencyMs ?? 0} ms` : "not probed (WebSocket)"}
                  </span>
                </label>
              </li>
            ))}
          </ul>
        </>
      ) : (
        <p className="error small">Nothing in the feed answered for chain {chainId} right now.</p>
      )}
      {rejected.length ? (
        <details className="rpc-rejected">
          <summary className="muted small">{rejected.length} were not offered — why</summary>
          <ul className="plain-list">
            {rejected.map((e) => (
              <li key={e.url} className="muted small">
                <code>{e.url}</code> — {e.reason ?? "rejected"}
              </li>
            ))}
          </ul>
        </details>
      ) : null}
      <div className="modal-actions">
        <button className="btn btn-ghost" onClick={onCancel}>
          Cancel
        </button>
        {live.length ? (
          <button className="btn" onClick={() => onAdd([...sel])}>
            Add selected
          </button>
        ) : null}
      </div>
    </Modal>
  );
}

// --- wipe / reset reports ------------------------------------------------

export function WipeDialog({ gid, wipeDiscards, onConfirm, onCancel }: { gid: string; wipeDiscards: string; onConfirm: () => void; onCancel: () => void }) {
  const [value, setValue] = useState("");
  const ref = useRef<HTMLInputElement>(null);
  useEffect(() => ref.current?.focus(), []);
  return (
    <Modal onClose={onCancel}>
      <h2>Wipe {gid}</h2>
      <p className="error">This destroys {wipeDiscards}</p>
      <p>
        Every chain it fronts stops being served until it comes back. Nothing behind it — no node, no devnet, no public
        endpoint — is touched.
      </p>
      <p>
        Type <code>{gid}</code> to confirm.
      </p>
      <input ref={ref} type="text" autoComplete="off" spellCheck={false} value={value} onChange={(e) => setValue(e.target.value)} />
      <div className="modal-actions">
        <button className="btn btn-ghost" onClick={onCancel}>
          Cancel
        </button>
        <button className="btn btn-danger" disabled={value.trim() !== gid} onClick={onConfirm}>
          Wipe {gid}
        </button>
      </div>
    </Modal>
  );
}

export function WipeResultDialog({ gid, result, onClose }: { gid: string; result: WipeResult; onClose: () => void }) {
  return (
    <Modal onClose={onClose}>
      <h2>{gid} wiped</h2>
      <ul className="plain-list">
        <li>{result.report.ContainerRemoved ? "Container removed." : "There was no container to remove."}</li>
        {result.report.Recreated ? <li>Container re-created from your saved configuration.</li> : null}
      </ul>
      {result.error ? <p className="error small">{result.error}</p> : null}
      <div className="modal-actions">
        <button className="btn" onClick={onClose}>
          Close
        </button>
      </div>
    </Modal>
  );
}

export function ResetResultDialog({ targetId, result, onClose }: { targetId: string; result: WipeResult; onClose: () => void }) {
  const cascaded = result.report.Cascaded ?? [];
  const skipped = result.report.CascadeSkipped ?? [];
  return (
    <Modal onClose={onClose}>
      <h2>Devnet on {targetId} reset</h2>
      <ul className="plain-list">
        <li>{result.report.ContainerRemoved ? "The old chain was removed." : "There was no devnet container to remove."}</li>
        {result.report.Recreated ? <li>A fresh chain was started from genesis.</li> : null}
      </ul>
      {cascaded.length ? (
        <p className="ok">
          Restarted in front of it: {cascaded.join(", ")} — the cached head was cleared, so each now reports this
          chain's real height rather than the one from before the reset.
        </p>
      ) : (
        <p className="muted small">Nothing needed restarting in front of it.</p>
      )}
      {skipped.length ? (
        <p className="muted small">Not restarted (they were not running, so they held no stale head): {skipped.join(", ")}.</p>
      ) : null}
      {result.error ? (
        <>
          <p className="error">
            The reset itself succeeded, but something in front of it could not be restarted — it is now serving a head
            this chain no longer has. Restart it by hand.
          </p>
          <p className="error small">{result.error}</p>
        </>
      ) : null}
      <div className="modal-actions">
        <button className="btn" onClick={onClose}>
          Close
        </button>
      </div>
    </Modal>
  );
}
