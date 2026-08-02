// The live health dot. Its motion is CSS driven by the class panelModel's
// healthClass computes (stable / occasional / frequent / off) — stillness is
// health, faster motion is a worse slow-request rate. Rendering it as a plain
// <span> whose className changes (never a fresh node) is what lets React's
// reconciler keep the same DOM node across a 5s health poll, so the animation
// never restarts from frame zero — the exact stutter panel.ts fought with its
// healthSignature/lastHealthSig skip machinery, now deleted.
import { healthClass } from "../../panelModel";

export function HealthDot({
  running,
  serviceable,
  slowRate,
}: {
  running: boolean;
  serviceable: boolean;
  slowRate?: number;
}) {
  return <span className={`p-dot ${healthClass({ running, serviceable, slowRate })}`} />;
}
