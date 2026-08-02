// The panel's landing view: one row per chain the gateway fronts, under the
// master power band — or, when no gateway exists yet, the one-click "Set up my
// endpoint" hero. The network rows' capability meters stay dim on purpose: the
// real probe (real sockets against real endpoints) is never run for a whole
// gateway's chains just to paint the list; it fires lazily when a network's own
// detail screen opens.
import type * as api from "../../api";
import { capabilityCells, masterState, networkSlowRate } from "../../panelModel";
import { Icon } from "./icons";
import { HealthDot } from "./HealthDot";
import { CapabilityDots } from "./CapabilityMeter";
import { PowerBand } from "./PowerBand";

function PanelHeader({ summary, onOpenSettings }: { summary?: string; onOpenSettings: () => void }) {
  return (
    <div className="p-band p-phead">
      <span className="p-brand">
        <span className="p-bd"></span> Valve
      </span>
      <span className="p-hright">
        {summary ? <span className="p-sum">{summary}</span> : null}
        <button type="button" className="p-gear" title="Settings" aria-label="Settings" onClick={onOpenSettings}>
          <Icon name="gear" />
        </button>
      </span>
    </div>
  );
}

function NetworkRow({
  gw,
  nv,
  divider,
  health,
  onOpen,
}: {
  gw: api.GatewayView;
  nv: api.NetworkView;
  divider: boolean;
  health: api.GatewayAnalytics | undefined;
  onOpen: (chainId: number) => void;
}) {
  const na = health?.networks?.find((n) => n.chainId === nv.chainId);
  const slowRate = na ? networkSlowRate(na) : undefined;
  return (
    <div className={`p-row${divider ? " p-rowdiv" : ""}`} onClick={() => onOpen(nv.chainId)}>
      <span className="p-lead">
        <HealthDot running={gw.status.State === "running"} serviceable={nv.serviceable} slowRate={slowRate} />
      </span>
      <span className="p-nm">{nv.name}</span>
      <CapabilityDots cells={capabilityCells({})} />
      <span className="p-chev">
        <Icon name="chevR" />
      </span>
    </div>
  );
}

export function ListView({
  gw,
  health,
  busy,
  actionErr,
  setupLog,
  onSetup,
  onPower,
  onChip,
  onOpenSettings,
  onOpenNetwork,
  onAddNetwork,
}: {
  gw: api.GatewayView | null;
  health: api.GatewayAnalytics | undefined;
  busy: string | null;
  actionErr: string | null;
  setupLog: string[];
  onSetup: () => void;
  onPower: () => void;
  onChip: (action: string) => void;
  onOpenSettings: () => void;
  onOpenNetwork: (chainId: number) => void;
  onAddNetwork: () => void;
}) {
  // Empty state: no gateway exists yet on this fleet — one centered hero whose
  // dimmed power button IS the one-click setup action. Nothing here navigates.
  if (gw === null) {
    const running = busy === "setup";
    return (
      <>
        <PanelHeader onOpenSettings={onOpenSettings} />
        <div className="p-band p-empty">
          <button type="button" className="p-emptybtn" disabled={running} onClick={onSetup}>
            <div className={`p-pbtn off big${running ? " busy" : ""}`}>
              <Icon name="power" />
            </div>
          </button>
          <div className="p-emptytitle">Set up my endpoint</div>
          <div className="p-emptysub">
            One click gets you a managed RPC endpoint for Ethereum and PulseChain — no node required.
          </div>
          {actionErr ? <div className="p-emptyerr">{actionErr}</div> : null}
          {setupLog.length ? (
            <div className="p-setup-log" aria-live="polite">
              {setupLog.map((l, i) => (
                <div key={i}>{l}</div>
              ))}
            </div>
          ) : null}
        </div>
      </>
    );
  }

  const m = masterState(gw);
  return (
    <>
      <PanelHeader summary={m.sub} onOpenSettings={onOpenSettings} />
      <div className="p-band">
        <PowerBand gw={gw} master={m} busy={busy} actionErr={actionErr} onPower={onPower} onChip={onChip} />
      </div>
      <div className="p-band">
        <div className="p-lblrow">
          <span className="p-seclbl">Networks</span>
        </div>
        {(gw.networks ?? []).map((nv, i) => (
          <NetworkRow key={nv.chainId} gw={gw} nv={nv} divider={i > 0} health={health} onOpen={onOpenNetwork} />
        ))}
        <div
          className={`p-row p-rowdiv addr${busy ? " p-disabled" : ""}`}
          aria-disabled={busy ? true : undefined}
          onClick={busy ? undefined : onAddNetwork}
        >
          <span className="p-lead">
            <Icon name="plus" />
          </span>
          <span className="p-nm">Add a network</span>
        </div>
      </div>
    </>
  );
}
