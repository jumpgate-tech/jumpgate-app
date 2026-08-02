// One machine card: id, where it lives (this machine / user@host), its
// status line, and Open/Remove actions. Port of targets.ts's targetCard().
//
// Everything you can do to a machine now lives on ONE page (#/machine/<id>):
// setup, dashboard, logs and the devnet as expandable sections. This card no
// longer sprouts a link per capability — it just opens the machine.
//
// The RPC gateway is deliberately NOT linked per-machine here: it fronts
// chains across every machine, so it has one top-level screen (#/rpc) rather
// than a copy under each box.
import type * as api from "../../api";
import { Badge } from "../../components/Badge";
import { targetStatus } from "./targetsModel";

export function TargetCard({
  target,
  catalog,
  canRunNode,
  hostOS,
  onRemove,
}: {
  target: api.Target;
  catalog: api.Catalog;
  canRunNode: boolean;
  hostOS: string;
  onRemove: (id: string) => void;
}) {
  const status = targetStatus(target, catalog, canRunNode, hostOS);
  const location =
    target.mode === "ssh" && target.ssh ? `${target.ssh.User}@${target.ssh.Host}` : "this machine";

  return (
    <div className="card">
      <h2>{target.id}</h2>
      <p className="muted">{location}</p>
      <p>
        {status.kind === "cant-run" && (
          <>
            <Badge text="can't run a node" kind="warn" /> <Badge text={status.hostOS || "not Linux"} kind="neutral" />
          </>
        )}
        {status.kind === "not-set-up" && <Badge text="not set up" kind="neutral" />}
        {status.kind === "wired" && (
          <>
            <Badge text={status.networkName} kind="ok" /> <Badge text={status.execId} kind="neutral" />{" "}
            <Badge text={status.beaconId} kind="neutral" />
            {status.archive && (
              <>
                {" "}
                <Badge text="archive" kind="warn" />
              </>
            )}
          </>
        )}
      </p>
      <div className="card-actions">
        <a className="btn" href={`#/machine/${encodeURIComponent(target.id)}`}>
          Open
        </a>
        <button className="btn btn-danger" onClick={() => onRemove(target.id)}>
          Remove
        </button>
      </div>
    </div>
  );
}
