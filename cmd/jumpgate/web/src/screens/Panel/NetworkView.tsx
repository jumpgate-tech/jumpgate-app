// The network detail screen: the one dialable gateway URL for this chain (with
// a live HTTPS lock and a copy-to-clipboard flash), the endpoints balanced
// across it, the folded capability meter, health, and the remove action.
// Local state (the last live TLS check, the copy flash) is scoped to this
// component; Panel keys it by chainId, so switching networks resets it — the
// reset panel.ts did by hand on open-network.
import { useState } from "react";
import type * as api from "../../api";
import { networkSlowRate, endpointSlowRate, capabilityCells, shortEndpoint, healthWord } from "../../panelModel";
import { copyToClipboard } from "../../ui";
import { Icon } from "./icons";
import { HealthDot } from "./HealthDot";
import { CapsBand, CapabilityDots } from "./CapabilityMeter";
import { unionCapabilities, singleCapabilities, hasProbeFor } from "./capsUtil";

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
  const [walletFlash, setWalletFlash] = useState(false);

  const nv = gw.networks?.find((n) => n.chainId === chainId);
  if (!nv) {
    return (
      <>
        <div className="p-band p-dhead">
          <button type="button" className="p-back" aria-label="Back" onClick={onBack}>
            <Icon name="chevL" />
          </button>
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
  const lockTitle = !running
    ? "Start the gateway to verify HTTPS"
    : tlsBusy
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
      setTlsErr(`Couldn't verify HTTPS — ${e instanceof Error ? e.message : String(e)}`);
    }
    setTlsBusy(false);
  }

  async function copy(url: string, flash: (v: boolean) => void) {
    if (!url) return;
    if (await copyToClipboard(url)) {
      flash(true);
      window.setTimeout(() => flash(false), 1200);
    }
  }

  const capStatuses = unionCapabilities(caps, chainId, ups.map((u) => u.id));
  const healthLabel = healthWord({ running, serviceable: nv.serviceable, slowRate: networkRate });

  return (
    <>
      <div className="p-band p-dhead">
        <button type="button" className="p-back" aria-label="Back" onClick={onBack}>
          <Icon name="chevL" />
        </button>
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
            <button
              type="button"
              className={`p-ic ${tlsOk ? "green" : "dim"}`}
              title={lockTitle}
              aria-label={lockTitle}
              disabled={tlsBusy || !running}
              onClick={() => void verify()}
            >
              <Icon name="lock" />
            </button>
            <button
              type="button"
              className={`p-ic ${copyFlash ? "green" : "accent"}`}
              title="Copy the gateway URL"
              aria-label="Copy the gateway URL"
              onClick={() => void copy(nv.url ?? "", setCopyFlash)}
            >
              <Icon name="copy" />
            </button>
          </span>
        </div>
        <div className="p-gwurl">{nv.url || "—"}</div>
        {nv.url ? (
          <div className="p-ps" style={{ color: "var(--dim3)", padding: "0 var(--gut) 4px" }}>
            Share this, or use it from another machine. A wallet must trust this gateway's certificate first.
          </div>
        ) : null}
        {tlsErr ? (
          <div className="p-ps" role="alert" style={{ color: "var(--red)", padding: "0 var(--gut) 10px" }}>
            {tlsErr}
          </div>
        ) : null}
      </div>

      {nv.localUrl ? (
        <div className="p-band">
          <div className="p-lblrow">
            <span className="p-seclbl">
              Wallet on this machine
              <span style={{ color: "var(--dim3)", letterSpacing: 0 }}> · no certificate to trust</span>
            </span>
            <span className="p-acts">
              <button
                type="button"
                className={`p-ic ${walletFlash ? "green" : "accent"}`}
                title="Copy the localhost wallet URL"
                aria-label="Copy the localhost wallet URL"
                onClick={() => void copy(nv.localUrl ?? "", setWalletFlash)}
              >
                <Icon name="copy" />
              </button>
            </span>
          </div>
          <div className="p-gwurl">{nv.localUrl}</div>
          <div className="p-ps" style={{ color: "var(--dim3)", padding: "0 var(--gut) 4px" }}>
            Paste into a wallet on this Mac. It answers over plain http on loopback, so there is nothing to install.
          </div>
        </div>
      ) : null}

      <div className="p-band">
        <div className="p-lblrow">
          <span className="p-seclbl">Endpoints · {ups.length}</span>
        </div>
        {ups.map((u, i) => {
          const upRate = na ? endpointSlowRate(na, u.id) : undefined;
          const yours = u.local;
          return (
            <button
              type="button"
              key={u.id}
              className={`p-row p-eprow${i > 0 ? " p-rowdiv" : ""}`}
              onClick={() => onOpenEndpoint(chainId, u.id)}
            >
              <span className="p-lead">
                <HealthDot running={running} serviceable={!u.problem} slowRate={upRate} />
              </span>
              <span className="p-epname">
                <span className="p-eplabel">{u.label}</span>
                <span className="p-epurl">{shortEndpoint(u.endpoint)}</span>
              </span>
              <CapabilityDots cells={capabilityCells(singleCapabilities(caps, chainId, u.id))} />
              {yours ? (
                <span className="p-src" title="A node or devnet you run">
                  <Icon name="server" />
                </span>
              ) : null}
              <span className="p-chev">
                <Icon name="chevR" />
              </span>
            </button>
          );
        })}
        <button
          type="button"
          className={`p-row${ups.length > 0 ? " p-rowdiv" : ""} addr${busy ? " p-disabled" : ""}`}
          disabled={!!busy}
          onClick={onAddEndpoint}
        >
          <span className="p-lead">
            <Icon name="plus" />
          </span>
          <span className="p-nm">Add endpoint</span>
        </button>
      </div>

      <div className="p-band">
        <div className="p-lblrow">
          <span className="p-seclbl">
            Capabilities
            <span style={{ color: "var(--dim3)", letterSpacing: 0 }}> · combined across endpoints</span>
          </span>
        </div>
        <CapsBand
          statuses={capStatuses}
          busy={capsBusy}
          err={capsErr}
          hasData={!!caps}
          running={running}
          probed={hasProbeFor(caps, chainId)}
        />
      </div>

      <div className="p-band">
        <div className="p-lblrow">
          <span className="p-seclbl">Status</span>
          <span className="p-acts">
            <button
              type="button"
              className="p-ic dim"
              title={capsBusy ? "Re-checking…" : "Re-check capabilities and reload"}
              aria-label="Re-check capabilities and reload"
              disabled={capsBusy}
              onClick={onRecheck}
            >
              <Icon name="refresh" className={capsBusy ? "p-spinning" : undefined} />
            </button>
          </span>
        </div>
        <div className="p-srow">
          <span className="p-k">Health</span>
          <span className="p-v">
            <HealthDot running={running} serviceable={nv.serviceable} slowRate={networkRate} /> {healthLabel}
          </span>
        </div>
      </div>

      {error ? (
        <div className="p-band" role="alert" style={{ padding: "10px 16px", color: "var(--red)" }}>
          {error}
        </div>
      ) : null}

      <button
        type="button"
        className={`p-band p-remove${busy ? " p-disabled" : ""}`}
        disabled={!!busy}
        onClick={onRemoveNetwork}
      >
        <Icon name="trash" /> Remove network
      </button>
    </>
  );
}
