// The React port of verdict.ts's old renderVerdictLine (since removed as dead
// code) — one status line above the machine list, above everything else, so
// the operator never assembles "is anything wrong?" from the cards below.
// computeFleetVerdict itself stays in verdict.ts: it's pure and unit-tested
// on its own there.
import type * as api from "../../api";
import { Badge } from "../../components/Badge";
import { computeFleetVerdict } from "../../verdict";

export function FleetVerdict({ targets, catalog }: { targets: api.Target[]; catalog: api.Catalog }) {
  const v = computeFleetVerdict(targets, catalog);
  return (
    <div className={`verdict-line verdict-${v.level}`}>
      <Badge text={v.level === "ok" ? "OK" : "Attention"} kind={v.level === "ok" ? "ok" : "warn"} />
      <strong className="verdict-sentence">{v.sentence}</strong>
      {v.machines.length > 0 && (
        <span className="verdict-machines">
          {v.machines.map((id) => (
            <a key={id} href={`#/setup/${encodeURIComponent(id)}`}>
              {id}
            </a>
          ))}
        </span>
      )}
    </div>
  );
}
