// Pure derivations for the Security screen, kept apart from the components so
// they're testable without rendering.
import type { CheckItem } from "../../api";
import type { BadgeKind } from "../../components/Badge";

// checkKind maps a CheckItem's Status to the badge's color, mirroring
// security.ts's (and diag.ts's identical) inline ternary.
export function checkKind(status: CheckItem["Status"]): BadgeKind {
  return status === "pass" ? "ok" : status === "fail" ? "bad" : status === "warn" ? "warn" : "neutral";
}
