// One diagnostics-ladder item: a collapsed head (status badge, title, detail)
// that expands to "why this matters" and, when the check has one, a
// copy-paste "Fix" command. Mirrors diag.ts's checkItemHtml + its
// toggle/copy data-actions.
//
// The ladder stops at its first failure, so that item ("failedHere") starts
// pre-expanded with both detail panels already open and its badge reading
// "failed here" instead of its raw status — exactly like diag.ts's
// `failedHere` branch. DiagCheckList keys each row on `${report.at}-${id}`,
// so a fresh run remounts every row and re-derives this default straight
// from the new report, matching diag.ts's full re-render on every run().
import { useEffect, useRef, useState } from "react";
import type { CheckItem } from "../../api";
import { copyToClipboard } from "../../ui";
import { Badge } from "../../components/Badge";
import { checkKind } from "../Security/securityModel";

const COPY_RESET_MS = 1500;

export function DiagCheckItemRow({ item, failedHere }: { item: CheckItem; failedHere: boolean }) {
  const [expanded, setExpanded] = useState(failedHere);
  const [copyLabel, setCopyLabel] = useState("Copy");
  const resetTimer = useRef<ReturnType<typeof setTimeout> | null>(null);

  // Clears the pending "revert to Copy" timer on unmount, replacing
  // diag.ts's `disposed` guard around the same setTimeout.
  useEffect(() => {
    return () => {
      if (resetTimer.current) clearTimeout(resetTimer.current);
    };
  }, []);

  async function handleCopy(): Promise<void> {
    const ok = await copyToClipboard(item.Fix);
    setCopyLabel(ok ? "Copied!" : "Copy failed");
    if (resetTimer.current) clearTimeout(resetTimer.current);
    resetTimer.current = setTimeout(() => setCopyLabel("Copy"), COPY_RESET_MS);
  }

  return (
    <li className={`check-item${expanded ? " expanded" : ""}`}>
      <button className="check-head" type="button" onClick={() => setExpanded((e) => !e)}>
        <Badge text={failedHere ? "failed here" : item.Status} kind={checkKind(item.Status)} />
        <strong>{item.Title}</strong>
        <span className="muted small check-detail-inline">{item.Detail}</span>
      </button>
      <div className="check-body">
        <details open={failedHere}>
          <summary>Why this matters</summary>
          <p className="muted small">{item.Why}</p>
        </details>
        {item.Fix && (
          <details open={failedHere}>
            <summary>Suggested fix</summary>
            <pre className="fix-block">{item.Fix}</pre>
            <button className="btn btn-ghost" type="button" onClick={() => void handleCopy()}>
              {copyLabel}
            </button>
          </details>
        )}
      </div>
    </li>
  );
}
