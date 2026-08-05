// The endpoint detail screen: one upstream's own dialable address (editable
// only for "external" upstreams — a managed node/devnet derives its endpoint on
// every read, so an edit there would silently do nothing), its own probe
// verdict, and health (with a chain-head lag line only when the gateway's
// analytics actually scored this exact upstream). headLag reads from the same
// 5s health poll the dots use — one scrape, both folds — rather than panel.ts's
// separate lazy epHealth fetch of the identical endpoint.
import { useState } from "react";
import type * as api from "../../api";
import { endpointSlowRate } from "../../panelModel";
import { copyToClipboard, fmtInt } from "../../ui";
import { Icon } from "./icons";
import { HealthDot } from "./HealthDot";
import { CapsBand } from "./CapabilityMeter";
import { singleCapabilities } from "./capsUtil";

export function EndpointView({
  gw,
  chainId,
  upstreamId,
  health,
  caps,
  capsBusy,
  capsErr,
  busy,
  error,
  onBack,
  onRename,
  onEditAddress,
  onRemove,
  onRecheck,
}: {
  gw: api.GatewayView;
  chainId: number;
  upstreamId: string;
  health: api.GatewayAnalytics | undefined;
  caps: api.GatewayCapabilities | undefined;
  capsBusy: boolean;
  capsErr: string | null;
  // busy is the in-flight lifecycle action (see Panel). While it is set, the
  // config-mutation controls (rename, edit address, remove) are disabled — a
  // provision can hold busy for minutes, and a config write landing mid-stream
  // would store a change provision() then no-ops on, diverging stored config
  // from the live erpc.yaml. Mirrors panel.ts's `if (busy) return` guards.
  busy: string | null;
  error: string | null;
  onBack: () => void;
  onRename: () => void;
  onEditAddress: () => void;
  onRemove: () => void;
  onRecheck: () => void;
}) {
  const [copyFlash, setCopyFlash] = useState(false);

  const nv = gw.networks?.find((n) => n.chainId === chainId);
  const up = nv?.upstreams?.find((u) => u.id === upstreamId);
  if (!nv || !up) {
    return (
      <>
        <div className="p-band p-dhead">
          <button type="button" className="p-back" aria-label="Back" onClick={onBack}>
            <Icon name="chevL" />
          </button>
          <span className="p-dtitle">
            <span className="p-nmtxt">Endpoint</span>
          </span>
        </div>
        <div className="p-band" style={{ padding: 16, color: "var(--dim)" }}>
          This endpoint is no longer configured.
        </div>
      </>
    );
  }

  const running = gw.status.State === "running";
  const na = health?.networks?.find((n) => n.chainId === chainId);
  const upRate = na ? endpointSlowRate(na, upstreamId) : undefined;
  const editable = up.kind === "external";

  async function copy(url: string) {
    if (!url) return;
    if (await copyToClipboard(url)) {
      setCopyFlash(true);
      window.setTimeout(() => setCopyFlash(false), 1200);
    }
  }

  const capStatuses = singleCapabilities(caps, chainId, upstreamId);
  const healthWord = !running ? "Stopped" : up.problem ? up.problem : "Healthy";
  const healthEntry = (health?.endpoints ?? []).find((e) => e.chainId === chainId && e.upstream === upstreamId);
  const showLag = healthEntry && healthEntry.scored && healthEntry.headLag > 0;

  return (
    <>
      <div className="p-band p-dhead">
        <button type="button" className="p-back" aria-label="Back" onClick={onBack}>
          <Icon name="chevL" />
        </button>
        <span className="p-dtitle">
          <HealthDot running={running} serviceable={!up.problem} slowRate={upRate} />{" "}
          <span className="p-nmtxt">{up.label}</span>{" "}
          <button
            type="button"
            className={`p-pen${busy ? " p-disabled" : ""}`}
            aria-label="Rename endpoint"
            disabled={!!busy}
            onClick={onRename}
          >
            <Icon name="pencil" />
          </button>
        </span>
      </div>

      <div className="p-band">
        <div className="p-lblrow">
          <span className="p-seclbl">Address</span>
          <span className="p-acts">
            <button
              type="button"
              className={`p-ic ${copyFlash ? "green" : "accent"}`}
              title="Copy the endpoint URL"
              aria-label="Copy the endpoint URL"
              onClick={() => void copy(up.endpoint)}
            >
              <Icon name="copy" />
            </button>
          </span>
        </div>
        {editable ? (
          <button
            type="button"
            className={`p-gwurl${!busy ? " p-gwurl-edit" : ""}`}
            title={!busy ? "Click to edit the address" : undefined}
            aria-label={`Edit address: ${up.endpoint || "—"}`}
            disabled={!!busy}
            onClick={onEditAddress}
          >
            {up.endpoint || "—"}
          </button>
        ) : (
          <div className="p-gwurl">{up.endpoint || "—"}</div>
        )}
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
            <button
              type="button"
              className="p-ic dim"
              title="Re-check capabilities and reload"
              aria-label="Re-check capabilities and reload"
              onClick={onRecheck}
            >
              <Icon name="refresh" />
            </button>
          </span>
        </div>
        <div className="p-srow">
          <span className="p-k">Health</span>
          <span className="p-v">
            <HealthDot running={running} serviceable={!up.problem} slowRate={upRate} /> {healthWord}
          </span>
        </div>
        {showLag ? (
          <div className="p-srow">
            <span className="p-k">Chain head</span>
            <span className="p-v">
              behind {fmtInt(healthEntry.headLag)} block{healthEntry.headLag === 1 ? "" : "s"}
            </span>
          </div>
        ) : null}
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
        onClick={onRemove}
      >
        <Icon name="trash" /> Remove endpoint
      </button>
    </>
  );
}
