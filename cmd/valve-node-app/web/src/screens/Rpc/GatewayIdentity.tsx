// The one quiet line that says WHICH gateway these chains belong to: its state,
// the machine, the image, whether HTTPS fronts it, and the base URL — kept
// discoverable but quiet, because a chain is addressed by PATH on top of it, so
// the per-chain URLs on the cards are the thing to copy, not this. Colour lives
// in exactly one place: the state dot and badge.
import type { GatewayView } from "../../api";
import { Badge, type BadgeKind } from "../../components/Badge";

function stateBadge(gw: GatewayView): { text: string; kind: BadgeKind } {
  switch (gw.status.State) {
    case "running":
      return { text: "running", kind: "ok" };
    case "created-but-stopped":
      return { text: "stopped", kind: "warn" };
    case "not-created":
      return { text: "not created", kind: "neutral" };
    default:
      return { text: "unknown", kind: "bad" };
  }
}

function dotKind(gw: GatewayView): "ok" | "bad" | "neutral" {
  if (gw.status.State === "running") return "ok";
  if (gw.status.State === "unknown") return "bad";
  return "neutral";
}

export function GatewayIdentity({ gw }: { gw: GatewayView }) {
  const running = gw.status.State === "running";
  const t = gw.tls;
  const sb = stateBadge(gw);
  return (
    <div className="rpc-ident">
      <span className={`dot dot-${dotKind(gw)}`} />
      <strong>{gw.label}</strong>
      <Badge text={sb.text} kind={sb.kind} />
      <span className="muted small">
        {"on "}
        <strong>{gw.placement.targetId}</strong>
        {gw.status.Image ? (
          <>
            {" · "}
            <code>{gw.status.Image}</code>
          </>
        ) : null}
        {" · "}
        {t?.enabled ? (
          <>
            {"HTTPS front "}
            <code>{t.containerName || "caddy"}</code>
          </>
        ) : (
          "no HTTPS front"
        )}
      </span>
      <span className="rpc-ident-base muted small">
        {running ? (
          <>
            {"base "}
            <code>{gw.baseUrl}</code>
          </>
        ) : (
          "not serving"
        )}
      </span>
    </div>
  );
}
