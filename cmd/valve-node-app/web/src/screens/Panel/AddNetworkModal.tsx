// The shared "Add a network" picker — one component for the panel and the RPC
// screen. Search runs over the curated head (viem, chains.ts) plus the full
// chainlist catalogue (api.allChains); an empty query shows just the pinned
// suggestions. A numeric query that matches nothing offers "Add chain N", which
// is the custom path — a gateway can front any chain id, including ones this app
// cannot run a node for. Chains already fronted are excluded.
//
// This is the PRESENTATIONAL half: it takes the catalogue as data so it renders
// in the harness and unit tests without a backend. AddNetworkDialog (the
// container) wires api.allChains and the viem curated list to it.
import { useEffect, useMemo, useRef, useState } from "react";
import { useQuery } from "@tanstack/react-query";
import { allChains, type ChainSummary } from "../../api";
import { CURATED } from "../../chains";
import { Modal } from "./Modal";
import { Icon } from "./icons";

interface Row {
  chainId: number;
  name: string;
  testnet?: boolean;
  curated?: boolean;
}

// PinnedChain is a caller-supplied suggestion pinned alongside the curated head
// — used for entries the feed/viem don't carry, e.g. the RPC screen's local
// Devnet (which the parent routes to a managed devnet on pick).
export interface PinnedChain {
  chainId: number;
  name: string;
  testnet?: boolean;
}

const CAP = 40; // rows rendered before "keep typing to narrow"

// ChainLogo shows the chain's logo from gib.show (/image/{chainId}), falling
// back to a neutral glyph on 404 or offline — the slot is a fixed size either
// way so rows stay aligned whether or not an image loads. This is the one
// external fetch in the picker; an air-gapped box simply shows the placeholder.
function ChainLogo({ chainId }: { chainId: number }) {
  const [failed, setFailed] = useState(false);
  return (
    <span className="p-anm-logo" aria-hidden="true">
      {failed ? (
        <Icon name="globe" />
      ) : (
        <img
          src={`https://gib.show/image/${chainId}`}
          alt=""
          loading="lazy"
          onError={() => setFailed(true)}
        />
      )}
    </span>
  );
}

export function AddNetworkPicker({
  catalogue,
  loading,
  stale,
  extraPinned,
  presentChainIds,
  onPick,
  onCancel,
}: {
  catalogue: ChainSummary[];
  loading?: boolean;
  stale?: boolean;
  extraPinned?: PinnedChain[];
  presentChainIds: number[];
  onPick: (chainId: number) => void;
  onCancel: () => void;
}) {
  const [q, setQ] = useState("");
  const inputRef = useRef<HTMLInputElement>(null);
  useEffect(() => inputRef.current?.focus(), []);

  const present = useMemo(() => new Set(presentChainIds), [presentChainIds]);
  const query = q.trim().toLowerCase();

  const { rows, more, customId } = useMemo(() => {
    // The pinned head: viem's curated order, then any caller extras (Devnet),
    // deduped so an extra can't repeat a curated chain.
    const pinnedSeen = new Set<number>();
    const pinned: Row[] = [];
    for (const c of [...CURATED, ...(extraPinned ?? [])]) {
      if (pinnedSeen.has(c.chainId)) continue;
      pinnedSeen.add(c.chainId);
      pinned.push({ chainId: c.chainId, name: c.name, testnet: c.testnet, curated: true });
    }

    const seen = new Set<number>();
    const curated: Row[] = pinned.filter((c) => !present.has(c.chainId));
    curated.forEach((c) => seen.add(c.chainId));

    const rest: Row[] = [];
    for (const c of catalogue) {
      if (seen.has(c.chainId) || present.has(c.chainId)) continue;
      rest.push({ chainId: c.chainId, name: c.name });
    }

    let all: Row[];
    if (query) {
      const match = (r: Row) => r.name.toLowerCase().includes(query) || String(r.chainId).includes(query);
      all = [...curated, ...rest].filter(match);
    } else {
      all = curated; // empty query → just the pinned suggestions
    }

    const shown = all.slice(0, CAP);
    const moreCount = Math.max(0, all.length - CAP);

    // A numeric query that isn't already listed or present is a custom add.
    let custom: number | null = null;
    if (/^\d+$/.test(query)) {
      const n = Number.parseInt(query, 10);
      if (n > 0 && !present.has(n) && !shown.some((r) => r.chainId === n)) custom = n;
    }
    return { rows: shown, more: moreCount, customId: custom };
  }, [catalogue, present, query, extraPinned]);

  return (
    <div className="p-anm">
      <div className="p-anm-head">
        <h2 className="p-anm-title">Add a network</h2>
      </div>
      <div className="p-anm-searchwrap">
        <input
          ref={inputRef}
          className="p-anm-search"
          type="text"
          inputMode="text"
          autoComplete="off"
          spellCheck={false}
          placeholder="Search by name or chain id…"
          value={q}
          onChange={(e) => setQ(e.target.value)}
        />
      </div>
      <div className="p-anm-list">
        {!query ? <div className="p-anm-grouplbl">Suggested</div> : null}
        {rows.map((r) => (
          <button key={r.chainId} type="button" className="p-anm-row" onClick={() => onPick(r.chainId)}>
            <ChainLogo chainId={r.chainId} />
            <span className="p-anm-name">
              {r.name}
              {r.testnet ? <span className="p-anm-tag">testnet</span> : null}
            </span>
            <span className="p-anm-id">{r.chainId}</span>
            <span className="p-chev">
              <Icon name="chevR" />
            </span>
          </button>
        ))}
        {customId != null ? (
          <button type="button" className="p-anm-row p-anm-custom" onClick={() => onPick(customId)}>
            <ChainLogo chainId={customId} />
            <span className="p-anm-name">Add chain {customId}</span>
            <span className="p-anm-id">custom</span>
            <span className="p-chev">
              <Icon name="plus" />
            </span>
          </button>
        ) : null}
        {query && rows.length === 0 && customId == null ? (
          <div className="p-anm-empty">
            Nothing matches “{q}”. Type a chain id to front it directly.
          </div>
        ) : null}
        {more > 0 ? <div className="p-anm-foot">+{more} more — keep typing to narrow</div> : null}
        {loading ? <div className="p-anm-foot">loading the full list…</div> : null}
        {stale && !loading ? <div className="p-anm-foot">offline — showing a cached list</div> : null}
      </div>
      <div className="p-anm-actions">
        <button type="button" className="p-anm-cancel" onClick={onCancel}>
          Cancel
        </button>
      </div>
    </div>
  );
}

// AddNetworkDialog is the container both the panel and the RPC screen mount: it
// fetches the full chain catalogue (cached server-side, so per-open is cheap)
// and portals the picker over a backdrop. The parent decides what a pick means
// — the panel wires valve's known set, the RPC screen routes the local Devnet
// to a managed devnet — so this only hands back a chain id.
export function AddNetworkDialog({
  presentChainIds,
  extraPinned,
  onPick,
  onCancel,
}: {
  presentChainIds: number[];
  extraPinned?: PinnedChain[];
  onPick: (chainId: number) => void;
  onCancel: () => void;
}) {
  const { data, isLoading } = useQuery({
    queryKey: ["chainCatalogue"],
    queryFn: allChains,
    staleTime: 6 * 60 * 60 * 1000, // matches the server-side cache TTL
  });
  return (
    <Modal onClose={onCancel} bare>
      <AddNetworkPicker
        catalogue={data?.chains ?? []}
        loading={isLoading}
        stale={data?.stale}
        extraPinned={extraPinned}
        presentChainIds={presentChainIds}
        onPick={onPick}
        onCancel={onCancel}
      />
    </Modal>
  );
}
