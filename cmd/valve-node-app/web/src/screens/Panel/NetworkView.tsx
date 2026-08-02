// The network detail screen: the one dialable gateway URL for this chain (with
// a live HTTPS lock and a copy-to-clipboard flash), the endpoints balanced
// across it, the folded capability meter, health, and the remove action.
// Local state (the last live TLS check, the copy flash) is scoped to this
// component; Panel keys it by chainId, so switching networks resets it — the
// reset panel.ts did by hand on open-network.
import { useState } from "react";
import type * as api from "../../api";
import { networkSlowRate, endpointSlowRate } from "../../panelModel";
import { copyToClipboard } from "../../ui";
import { Icon } from "./icons";
import { HealthDot } from "./HealthDot";
import { CapsBand } from "./CapabilityMeter";
import { unionCapabilities } from "./capsUtil";

export function NetworkView({
  gw,
  chainId,
  health,
  caps,
  capsBusy,
  capsErr,
  busy,
  error,
  onBack,
  onOpenEndpoint,
  onAddEndpoint,
  onRemoveNetwork,
  onVerifyTls,
  onRecheck,
}: {
  gw: api.GatewayView;
  chainId: number;
  health: api.GatewayAnalytics | undefined;
  caps: api.GatewayCapabilities | undefined;
  capsBusy: boolean;
  capsErr: string | null;
  busy: string | null;
  error: string | null;
  onBack: () => void;
  onOpenEndpoint: (chainId: number, upstreamId: string) => void;
  onAddEndpoint: () => void;
  onRemoveNetwork: () => void;
  onVerifyTls: () => Promise<api.TlsVerification>;
  onRecheck: () => void;
}) {
  const [tls, setTls] = useState<api.TlsVerification | null>(null);
  const [tlsBusy, setTlsBusy] = useState(false);
  const [tlsErr, setTlsErr] = useState<string | null>(null);
  const [copyFlash, setCopyFlash] = useState(false);

  const nv = gw.networks?.find((n) => n.chainId === chainId);
  if (!nv) {
    return (
      <>
        <div className="p-band p-dhead">
          <span className="p-back" onClick={onBack}>
            <Icon name="chevL" />
          </span>
          <span className="p-dtitle">
            <span className="p-nmtxt">Chain {chainId}</span>
          </span>
        </div>
        <div className="p-band" style={{ padding: 16, color: "var(--dim)" }}>
          This network is no longer configured.
        </div>
      </>
    );
  }

  const running = gw.status.State === "running";
  const na = health?.networks?.find((n) => n.chainId === chainId);
  const networkRate = na ? networkSlowRate(na) : undefined;
  const ups = nv.upstreams ?? [];

  const tlsResult = tls ?? gw.tls.verification ?? null;
  const tlsOk = tlsResult?.ok === true;
  const lockTitle = tlsBusy
    ? "Verifying…"
    : tlsOk
      ? `Verified ${tlsResult ? new Date(tlsResult.at).toLocaleString() : ""}`
      : "Verify HTTPS now";

  async function verify() {
    if (tlsBusy) return;
    setTlsBusy(true);
    setTlsErr(null);
    try {
      setTls(await onVerifyTls());
    } catch (e) {
      setTlsErr(e instanceof Error ? e.message : String(e));
    }
    setTlsBusy(false);
  }

  async function copy(url: string) {
    if (!url) return;
    if (await copyToClipboard(url)) {
      setCopyFlash(true);
      window.setTimeout(() => setCopyFlash(false), 1200);
    }
  }

  const capStatuses = unionCapabilities(caps, chainId, ups.map((u) => u.id));
  const healthWord = !running ? "Stopped" : nv.serviceable ? "Healthy" : "Unserviceable";

  return (
    <>
      <div className="p-band p-dhead">
        <span className="p-back" onClick={onBack}>
          <Icon name="chevL" />
        </span>
        <span className="p-dtitle">
          <HealthDot running={running} serviceable={nv.serviceable} slowRate={networkRate} />{" "}
          <span className="p-nmtxt">{nv.name}</span>
        </span>
      </div>

      <div className="p-band">
        <div className="p-lblrow">
          <span className="p-seclbl">
            Gateway <span style={{ color: "var(--dim3)", letterSpacing: 0 }}> · balanced across all</span>
          </span>
          <span className="p-acts">
            <span className={`p-ic ${tlsOk ? "green" : "dim"}`} title={lockTitle} onClick={() => void verify()}>
              <Icon name="lock" />
            </span>
            <span
              className={`p-ic ${copyFlash ? "green" : "accent"}`}
              title="Copy the gateway URL"
              onClick={() => void copy(nv.url ?? "")}
            >
              <Icon name="copy" />
            </span>
          </span>
        </div>
        <div className="p-gwurl">{nv.url || "—"}</div>
        {tlsErr ? (
          <div className="p-ps" style={{ color: "var(--red)", padding: "0 var(--gut) 10px" }}>
            {tlsErr}
          </div>
        ) : null}
      </div>

      <div className="p-band">
        <div className="p-lblrow">
          <span className="p-seclbl">Endpoints · {ups.length}</span>
        </div>
        {ups.map((u, i) => {
          const upRate = na ? endpointSlowRate(na, u.id) : undefined;
          return (
            <div
              key={u.id}
              className={`p-row${i > 0 ? " p-rowdiv" : ""}`}
              onClick={() => onOpenEndpoint(chainId, u.id)}
            >
              <span className="p-lead">
                <HealthDot running={running} serviceable={!u.problem} slowRate={upRate} />
              </span>
              <span className="p-nm">{u.label}</span>
              <span className="p-chev">
                <Icon name="chevR" />
              </span>
            </div>
          );
        })}
        <div
          className={`p-row${ups.length > 0 ? " p-rowdiv" : ""} addr${busy ? " p-disabled" : ""}`}
          aria-disabled={busy ? true : undefined}
          onClick={busy ? undefined : onAddEndpoint}
        >
          <span className="p-lead">
            <Icon name="plus" />
          </span>
          <span className="p-nm">Add endpoint</span>
        </div>
      </div>

      <div className="p-band">
        <div className="p-lblrow">
          <span className="p-seclbl">Capabilities</span>
        </div>
        <CapsBand statuses={capStatuses} busy={capsBusy} err={capsErr} hasData={!!caps} />
      </div>

      <div className="p-band">
        <div className="p-lblrow">
          <span className="p-seclbl">Status</span>
          <span className="p-acts">
            <span className="p-ic dim" title="Re-check capabilities and reload" onClick={onRecheck}>
              <Icon name="refresh" />
            </span>
          </span>
        </div>
        <div className="p-srow">
          <span className="p-k">Health</span>
          <span className="p-v">
            <HealthDot running={running} serviceable={nv.serviceable} slowRate={networkRate} /> {healthWord}
          </span>
        </div>
      </div>

      {error ? (
        <div className="p-band" style={{ padding: "10px 16px", color: "var(--red)" }}>
          {error}
        </div>
      ) : null}

      <div className="p-band p-remove" onClick={onRemoveNetwork}>
        <Icon name="trash" /> Remove network
      </div>
    </>
  );
}
