import type { DiagReport } from "../../api";
import { DiagCheckItemRow } from "./DiagCheckItemRow";

export function DiagCheckList({ report }: { report: DiagReport }) {
  return (
    <ul className="check-list">
      {report.items.map((item) => (
        <DiagCheckItemRow
          // Keying on the report's timestamp too remounts every row on a
          // fresh run, so each row's expand/open defaults are re-derived
          // from that run's failedId rather than carrying over stale state
          // from the previous report — see DiagCheckItemRow's comment.
          key={`${report.at}-${item.ID}`}
          item={item}
          failedHere={item.ID === report.failedId}
        />
      ))}
    </ul>
  );
}
