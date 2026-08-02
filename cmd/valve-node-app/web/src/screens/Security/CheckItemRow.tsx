// One firewall-checklist item: a collapsed head (status badge, title, detail)
// that expands to "why this matters" and, when the check has one, a
// copy-paste "Fix" command. Mirrors security.ts's checkItemHtml + its
// toggle/copy data-actions.
import { useEffect, useRef, useState } from "react";
import type { CheckItem } from "../../api";
import { copyToClipboard } from "../../ui";
import { Badge } from "../../components/Badge";
import { checkKind } from "./securityModel";

const COPY_RESET_MS = 1500;

export function CheckItemRow({ item }: { item: CheckItem }) {
  const [expanded, setExpanded] = useState(false);
  const [copyLabel, setCopyLabel] = useState("Copy");
  const resetTimer = useRef<ReturnType<typeof setTimeout> | null>(null);

  // Clears the pending "revert to Copy" timer on unmount, replacing
  // security.ts's `disposed` guard around the same setTimeout.
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
        <Badge text={item.Status} kind={checkKind(item.Status)} />
        <strong>{item.Title}</strong>
        <span className="muted small check-detail-inline">{item.Detail}</span>
      </button>
      <div className="check-body">
        <details>
          <summary>Why this matters</summary>
          <p className="muted small">{item.Why}</p>
        </details>
        {item.Fix && (
          <details open>
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
