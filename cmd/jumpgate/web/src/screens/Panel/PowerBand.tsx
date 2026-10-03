// The master power band: one round button performing the primary lifecycle
// transition (stop when running, else start, else create) plus a row of small
// chips for every OTHER action the server listed — so the panel can cycle a
// gateway through every state (start/stop/restart/recreate) rather than
// offering only one. wipe is deliberately excluded here: it now lives behind
// the gear's settings sheet.
import type * as api from "../../api";
import type { MasterState } from "../../panelModel";
import { Icon, type IconName } from "./icons";

// primaryAction picks the transition the big button performs — the same rule
// masterState's callers apply. It only ever returns an action the server
// actually listed, so the button never offers an impossible transition.
// Exported because Panel's power handler dispatches off the same choice.
export function primaryAction(gw: api.GatewayView, m: MasterState): string | null {
  if (m.tone === "blocked") return null;
  if (gw.status.State === "running" && m.actions.includes("stop")) return "stop";
  if (m.actions.includes("start")) return "start";
  if (m.actions.includes("create")) return "create";
  return null;
}

const ACTION_LABEL: Record<string, string> = {
  start: "Start",
  stop: "Stop",
  restart: "Restart",
  create: "Create",
  recreate: "Recreate",
  wipe: "Wipe",
};
const ACTION_ICON: Record<string, IconName> = { restart: "refresh", recreate: "refresh", wipe: "trash" };

export function PowerBand({
  gw,
  master,
  busy,
  actionErr,
  onPower,
  onChip,
}: {
  gw: api.GatewayView;
  master: MasterState;
  busy: string | null;
  actionErr: string | null;
  onPower: () => void;
  onChip: (action: string) => void;
}) {
  const subText = master.tone === "blocked" ? master.blocked ?? "" : master.sub;
  const primary = primaryAction(gw, master);
  // Every action the server listed minus the one the button performs and wipe
  // (behind the gear now) — the remaining safe transitions.
  const chips = (gw.actions ?? []).filter((a) => a !== primary && a !== "wipe");

  return (
    <>
      <button type="button" className={`p-power${busy ? " busy" : ""}`} disabled={!!busy} onClick={onPower}>
        <div className={`p-pbtn ${master.tone}`}>
          <Icon name="power" />
        </div>
        <div className="p-pmeta">
          <div className="p-pl">{master.label}</div>
          <div className="p-ps" style={master.tone === "blocked" ? { color: "var(--red)" } : undefined}>
            {subText}
          </div>
          {master.tone === "blocked" && gw.hint ? <div className="p-ps">{gw.hint}</div> : null}
          {actionErr ? (
            <div className="p-ps" role="alert" style={{ color: "var(--red)" }}>
              {actionErr}
            </div>
          ) : null}
        </div>
      </button>
      {chips.length > 0 ? (
        <div className="p-chips">
          {chips.map((a) => (
            <button
              key={a}
              type="button"
              className="p-chip"
              disabled={!!busy}
              onClick={() => onChip(a)}
            >
              {ACTION_ICON[a] ? <Icon name={ACTION_ICON[a]} /> : null}
              {ACTION_LABEL[a] ?? a}
            </button>
          ))}
        </div>
      ) : null}
    </>
  );
}
