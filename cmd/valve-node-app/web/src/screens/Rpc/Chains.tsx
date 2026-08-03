// The chains — the whole reason to be on this screen. Wallet-first cards
// (name, evm:<id>, one-word health, the dialable URL with a prominent Copy, and
// the cert hint inline when the gateway uses its own CA); the operator detail
// (the derived verdict, the redundancy bar, the per-upstream probe table,
// +Endpoint/Remove) folds away behind "Details". All presentational: it reads
// rpcModel's derivations and calls back up for every action.
import type {
  EndpointCapabilities,
  GatewayCapabilities,
  GatewayTraffic,
  GatewayView,
  NetworkView,
  TrustCertResult,
  UpstreamView,
} from "../../api";
import { Badge } from "../../components/Badge";
import { CopyButton } from "./CopyButton";
import {
  CAP_ORDER,
  CAP_TAGS,
  DEVNET_CHAIN_ID,
  capTagClass,
  capTagTitle,
  chainLacksEntirely,
  chainVerdict,
  healthWord,
  internalCaPath,
  orderedNetworks,
  redundancy,
  shareCellModel,
  shortTime,
  statusOf,
  type Verdict,
} from "./rpcModel";

export interface ChainCallbacks {
  isDetailOpen: (chainId: number) => boolean;
  onToggleDetail: (chainId: number) => void;
  onAddEndpoint: (chainId: number) => void;
  onRemoveChain: (chainId: number) => void;
  onRemoveEndpoint: (chainId: number, upstreamId: string) => void;
  onResetDevnet: (chainId: number, upstreamId: string, targetId: string) => void;
  onAddChain: () => void;
  onAddDevnet: () => void;
  onReprobe: () => void;
  onTrust: () => void;
}

export interface ChainsProps extends ChainCallbacks {
  gw: GatewayView;
  networks: NetworkView[];
  caps: GatewayCapabilities | undefined;
  capsBusy: boolean;
  traffic: GatewayTraffic | null | undefined;
  trafficLoading: boolean;
  targetMode: string;
  trustBusy: boolean;
  trustMessage: TrustCertResult | null;
  busy: string | null;
}

// VerdictText turns the structured verdict into the sentence + <code> the card
// shows — the one place the verdict's html lived in the legacy file.
function VerdictText({ v }: { v: Verdict }) {
  switch (v.kind) {
    case "no-endpoint":
      return <>No endpoint yet, so there is nowhere for calls on this path to go.</>;
    case "not-serviceable":
      return <>No upstream here can be dialed, so every call on this path fails.</>;
    case "no-websocket": {
      const schemes = v.schemes ?? [];
      return (
        <>
          No WebSocket upstream, so <code>eth_subscribe</code> fails on this chain
          {schemes.length ? (
            <>
              {" — every upstream here is configured as "}
              {schemes.map((s, i) => (
                <span key={s}>
                  {i > 0 ? " or " : ""}
                  <code>{s}://</code>
                </span>
              ))}
              .
            </>
          ) : (
            "."
          )}
        </>
      );
    }
    case "single":
      return <>One endpoint, so this chain stops when it does.</>;
    case "no-local":
      return <>No node of your own serves this chain.</>;
    case "some-unusable":
      return (
        <>
          {v.broken} of these {v.total} endpoints {v.broken === 1 ? "is" : "are"} unusable, so{" "}
          {v.usable === 1 ? "only one can" : `only ${v.usable} can`} actually answer — the segments above count what
          is configured, not what is working.
        </>
      );
    case "ok":
      return (
        <>
          {v.total} endpoints, one of them yours, and WebSocket among them — this chain can lose any one and still
          answer.
        </>
      );
  }
}

function RedundancyBar({ count, tone, setSize }: { count: number; tone: Verdict["tone"]; setSize: number }) {
  const r = redundancy(count, setSize);
  return (
    <>
      <span className="segs" title={r.title}>
        {Array.from({ length: r.total }, (_, i) => (
          <span key={i} className={`seg${i < r.filled ? ` seg-on seg-${tone}` : ""}`} />
        ))}
      </span>
      <span className="segs-n">{r.label}</span>
    </>
  );
}

function CapCell({
  network,
  upstream,
  caps,
}: {
  network: NetworkView;
  upstream: UpstreamView;
  caps: GatewayCapabilities | undefined;
}) {
  const e = (caps?.endpoints ?? []).find((x) => x.chainId === network.chainId && x.upstream === upstream.id);
  if (!e) return <span className="muted small">{caps === undefined ? "probing…" : "—"}</span>;
  if (e.unprobeable) {
    return (
      <span className="caps-none" title={e.unprobeable}>
        not probeable from here
      </span>
    );
  }
  const chainEndpoints: (EndpointCapabilities | undefined)[] = (network.upstreams ?? []).map((u) =>
    (caps?.endpoints ?? []).find((x) => x.chainId === network.chainId && x.upstream === u.id),
  );
  return (
    <span className="caps">
      {CAP_ORDER.map((key) => {
        const status = statusOf(e, key) ?? "inconclusive";
        const missing = status === "unsupported" && chainLacksEntirely(chainEndpoints, key);
        return (
          <span key={key} className={capTagClass(status, missing)} title={capTagTitle(e, key)}>
            {CAP_TAGS[key] ?? key.toUpperCase()}
          </span>
        );
      })}
    </span>
  );
}

function ShareCell({
  network,
  upstream,
  traffic,
  trafficLoading,
}: {
  network: NetworkView;
  upstream: UpstreamView;
  traffic: GatewayTraffic | null | undefined;
  trafficLoading: boolean;
}) {
  const s = shareCellModel(traffic, trafficLoading, network.chainId, upstream.id, upstream.local);
  switch (s.kind) {
    case "reading":
      return <span className="muted small">reading…</span>;
    case "unreadable":
      return (
        <span className="muted small" title="The counters could not be read.">
          —
        </span>
      );
    case "off":
      return (
        <span className="muted small" title="This gateway's request counters are turned off in its settings.">
          counters off
        </span>
      );
    case "none":
      return <span className="muted small">no traffic yet</span>;
    case "bar":
      return (
        <span className="share" title={s.title}>
          <span className="bar">
            <span className={`fill${s.fill ? " " + s.fill : ""}`} style={{ width: `${s.pct}%` }} />
            <span className="tick" style={{ left: `${s.tickPct}%` }} />
          </span>
          <span className={`share-n${s.fill === "warn" ? " warn" : ""}`}>{s.pct}%</span>
          {s.unconfigured ? <Badge text="not in config" kind="warn" /> : null}
        </span>
      );
  }
}

function EndpointRow({
  network,
  upstream,
  caps,
  traffic,
  trafficLoading,
  busy,
  onRemoveEndpoint,
  onResetDevnet,
}: {
  network: NetworkView;
  upstream: UpstreamView;
  caps: GatewayCapabilities | undefined;
  traffic: GatewayTraffic | null | undefined;
  trafficLoading: boolean;
  busy: string | null;
  onRemoveEndpoint: (chainId: number, upstreamId: string) => void;
  onResetDevnet: (chainId: number, upstreamId: string, targetId: string) => void;
}) {
  const actions = upstream.actions ?? [];
  return (
    <li className={`up${upstream.problem ? " up-bad" : ""}`}>
      <div className="up-what">
        <span className={`dot dot-${upstream.problem ? "bad" : "ok"}`} />
        <span className="up-label">{upstream.label}</span>
        <EndpointStateBadge upstream={upstream} />
      </div>
      <code className="up-url">{upstream.endpoint || "—"}</code>
      <div className="up-caps">
        <CapCell network={network} upstream={upstream} caps={caps} />
      </div>
      <div className="up-share">
        <ShareCell network={network} upstream={upstream} traffic={traffic} trafficLoading={trafficLoading} />
      </div>
      <div className="up-acts">
        {actions.includes("reset") ? (
          <button
            className="btn btn-ghost btn-tiny"
            title="Throw this devnet's chain away and start again from genesis. It is a scratch chain — this is routine."
            disabled={!!busy}
            onClick={() => onResetDevnet(network.chainId, upstream.id, upstream.targetId ?? "")}
          >
            {busy === "reset" ? <span className="spinner" aria-label="working" /> : "Reset"}
          </button>
        ) : null}
        <button className="btn btn-ghost btn-tiny" onClick={() => onRemoveEndpoint(network.chainId, upstream.id)}>
          Remove
        </button>
      </div>
      {upstream.problem ? <div className="up-problem error small">{upstream.problem}</div> : null}
    </li>
  );
}

function EndpointStateBadge({ upstream }: { upstream: UpstreamView }) {
  if (upstream.problem) return <Badge text="unusable" kind="bad" />;
  if (upstream.recentOnly) return <Badge text="recent blocks" kind="warn" />;
  return upstream.local ? <Badge text="yours" kind="ok" /> : <Badge text="public" kind="neutral" />;
}

function ChainConnect({ gw, network, props }: { gw: GatewayView; network: NetworkView; props: ChainsProps }) {
  if (!network.url) {
    const notRunning = gw.status.State !== "running";
    return (
      <p className="chain-connect-none muted small">
        {notRunning
          ? "No URL yet — the gateway is not running, so nothing answers on this path. Start it under “Manage gateway”."
          : "Not serviceable — nothing on this chain can be dialed, so there is no URL to connect to. Open Details to add an endpoint."}
      </p>
    );
  }
  const caPath = internalCaPath(gw);
  return (
    <div className="chain-connect">
      <code className="endpoint-url">{network.url}</code>
      <CopyButton value={network.url} label="Copy URL" className="btn btn-tiny" />
      {caPath ? (
        <>
          <span className="chain-cert muted small">Your wallet must trust this gateway's certificate first —</span>
          {props.targetMode === "local" ? (
            <button
              className="btn btn-ghost btn-tiny"
              disabled={props.trustBusy}
              title="Install this gateway's root certificate into this machine's trust store, then reload your wallet."
              onClick={props.onTrust}
            >
              {props.trustBusy ? "Trusting…" : "Trust on this machine"}
            </button>
          ) : null}
          <CopyButton
            value={caPath}
            label="Copy cert path"
            title={`Copy the path to Caddy's root certificate. Install it on ${gw.placement.targetId} and in the trust store of any device that will call this URL, and the warning goes away.`}
          />
          {props.trustMessage ? (
            props.trustMessage.ok ? (
              <span className="chain-cert muted small">Trusted — reload your wallet or browser.</span>
            ) : (
              <span className="chain-cert muted small">
                {props.trustMessage.message}
                {/* On failure, surface the run-by-hand command (e.g. the darwin
                    sudo fallback) inline too — the message alone tells the
                    operator what to do but not the command to do it with. */}
                {props.trustMessage.ranCommand ? (
                  <>
                    {" "}
                    <code className="strip-cmd">{props.trustMessage.ranCommand}</code>
                    <CopyButton value={props.trustMessage.ranCommand} label="Copy command" />
                  </>
                ) : null}
              </span>
            )
          ) : null}
        </>
      ) : null}
    </div>
  );
}

function ChainDetail({ network, verdict, props }: { network: NetworkView; verdict: Verdict; props: ChainsProps }) {
  const ups = network.upstreams ?? [];
  return (
    <div className="chain-detail">
      <p className={`chain-verdict${verdict.why ? " chain-verdict-why" : ""}`} title={verdict.why}>
        <VerdictText v={verdict} />
      </p>
      <div className="chain-detail-bar">
        <RedundancyBar count={ups.length} tone={verdict.tone} setSize={network.knownSetSize} />
        <button className="btn btn-ghost btn-tiny" onClick={() => props.onAddEndpoint(network.chainId)}>
          + Endpoint
        </button>
        <button className="btn btn-ghost btn-tiny" onClick={() => props.onRemoveChain(network.chainId)}>
          Remove
        </button>
      </div>
      {ups.length > 0 ? (
        <ul className="ups">
          {ups.map((u) => (
            <EndpointRow
              key={u.id}
              network={network}
              upstream={u}
              caps={props.caps}
              traffic={props.traffic}
              trafficLoading={props.trafficLoading}
              busy={props.busy}
              onRemoveEndpoint={props.onRemoveEndpoint}
              onResetDevnet={props.onResetDevnet}
            />
          ))}
        </ul>
      ) : null}
      {(network.warnings ?? []).map((w, i) => (
        <p key={i} className="chain-note">
          {w}
        </p>
      ))}
    </div>
  );
}

function ChainRow({ gw, network, props }: { gw: GatewayView; network: NetworkView; props: ChainsProps }) {
  const v = chainVerdict(network);
  const isDevnet = network.chainId === DEVNET_CHAIN_ID;
  const open = props.isDetailOpen(network.chainId);
  return (
    <section className={`chain chain-${v.tone}${isDevnet ? " chain-devnet" : ""}`}>
      <div className="chain-head">
        <span className="chain-name">{network.name}</span>
        <code className="chain-key">evm:{network.chainId}</code>
        {isDevnet ? <span className="chain-tag">local test chain (devnet)</span> : null}
        <Badge text={healthWord(v.tone)} kind={v.tone} />
        <span className="chain-right">
          <button
            className="btn btn-ghost btn-tiny"
            aria-expanded={open}
            onClick={() => props.onToggleDetail(network.chainId)}
          >
            {open ? "Hide details" : "Details"}
          </button>
        </span>
      </div>
      <ChainConnect gw={gw} network={network} props={props} />
      {open ? <ChainDetail network={network} verdict={v} props={props} /> : null}
    </section>
  );
}

function DevnetOptIn({ hasDevnet, onAddDevnet }: { hasDevnet: boolean; onAddDevnet: () => void }) {
  if (hasDevnet) return null;
  return (
    <button
      className="btn btn-ghost btn-tiny"
      title={`Add a throwaway local test chain (evm:${DEVNET_CHAIN_ID}) fronted by this gateway. Optional — real chains only by default.`}
      onClick={onAddDevnet}
    >
      Add a local devnet
    </button>
  );
}

function TrafficFootnote({ traffic }: { traffic: GatewayTraffic | null | undefined }) {
  if (!traffic) return null;
  if (!traffic.enabled) {
    return (
      <p className="muted small">
        This gateway is not counting its requests, so there is no traffic share to show. Turn the counters on in
        Settings — they stay on the machine the gateway runs on and nothing is sent anywhere.
      </p>
    );
  }
  if (traffic.error) {
    return <p className="muted small">The request counters could not be read: {traffic.error}</p>;
  }
  return (
    <p className="muted small">
      Share is measured from the gateway's own counters since it started
      {traffic.since ? ` (${shortTime(traffic.since)})` : ""}. The tick is the share routing intends: on a chain where
      you run a node, yours carries it and the public endpoints are there for when it cannot; on a chain served only by
      public endpoints there is nothing to prefer, so the intent is an even split across all of them.
    </p>
  );
}

export function NetworksPanel(props: ChainsProps) {
  const { gw, networks, caps, capsBusy } = props;
  const nets = orderedNetworks(networks);
  const hasDevnet = nets.some((n) => n.chainId === DEVNET_CHAIN_ID);

  if (nets.length === 0) {
    return (
      <div className="card rpc-surface">
        <p className="muted small">
          No networks yet. eRPC refuses a configuration with none, so add one before creating the gateway.
        </p>
        <div className="card-actions">
          <button className="btn" onClick={props.onAddChain}>
            Add a network
          </button>
          <DevnetOptIn hasDevnet={hasDevnet} onAddDevnet={props.onAddDevnet} />
        </div>
      </div>
    );
  }

  const when = caps?.at ? `probed ${shortTime(caps.at)}` : "not probed yet";
  return (
    <div className="card rpc-surface">
      <div className="chains">
        {nets.map((n) => (
          <ChainRow key={n.chainId} gw={gw} network={n} props={props} />
        ))}
      </div>
      <div className="chains-foot">
        <button className="btn btn-ghost btn-tiny" onClick={props.onAddChain}>
          + Network
        </button>
        <DevnetOptIn hasDevnet={hasDevnet} onAddDevnet={props.onAddDevnet} />
        <span className="chains-foot-gap" />
        <span className="muted small">{when}</span>
        <button
          className="btn btn-ghost btn-tiny"
          title="Ask every endpoint what it can do, again. This opens real connections to them."
          disabled={capsBusy}
          onClick={props.onReprobe}
        >
          {capsBusy ? <span className="spinner" aria-label="probing" /> : "Re-probe"}
        </button>
      </div>
      <TrafficFootnote traffic={props.traffic} />
    </div>
  );
}
