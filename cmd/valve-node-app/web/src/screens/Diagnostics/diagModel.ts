// Pure derivations for the Diagnostics screen, kept apart from the
// components so they're testable without rendering. checkKind (a check's
// Status -> the badge's color) lives in Security's securityModel.ts and is
// reused as-is here: diag.ts and security.ts computed it with the identical
// inline ternary.
import type { DiagReport } from "../../api";

// failedCheckTitle mirrors diag.ts's titleOf(): the title of the ladder item
// that stopped the run, falling back to the raw id if the report's items
// somehow don't contain it.
export function failedCheckTitle(report: DiagReport): string {
  return report.items.find((it) => it.ID === report.failedId)?.Title ?? report.failedId ?? "";
}
