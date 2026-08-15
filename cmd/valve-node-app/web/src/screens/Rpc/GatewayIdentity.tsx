// The one quiet line that says WHICH gateway these chains belong to: its state,
// the machine, the image, whether HTTPS fronts it, and the base URL — kept
// discoverable but quiet, because a chain is addressed by PATH on top of it, so
// the per-chain URLs on the cards are the thing to copy, not this. Colour lives
// in exactly one place: the state dot and badge.
//
// This is also the ONE place cert-trust lives. Every chain URL is just a path
// under the base URL, so trusting the gateway's certificate is a base-URL /
// host-level action, not a per-chain one — it used to be repeated under every
// chain row, which was both conceptually wrong and where the failure command
// truncated inline.
import type { GatewayView, TrustCertResult } from "../../api";
import { Badge, type BadgeKind } from "../../components/Badge";
import { CopyButton } from "./CopyButton";

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

// TrustResultLine mirrors ManageSection's — same strip-line/strip-warn/
// strip-text/strip-cmd markup — so an install failure (and, critically, the
// command to run by hand) renders the same way wherever trust surfaces. The
// command lives in its own <code> block rather than an inline nowrap span so
// a long sudo command wraps instead of truncating.
//
// A failure carries a "Try again" button because the common darwin case is
// fixable in place: a detached launch's automatic install has no GUI session to
// prompt in, the operator runs the sudo command by hand, and a retry then
// succeeds — the backend verifies the root is trusted and short-circuits rather
// than re-running the install. So the failure reads as a step to redo, not a
// dead end. The retry disables while one is in flight (trustBusy).
function TrustResultLine({
  r,
  trustBusy,
  onTrust,
}: {
  r: TrustCertResult;
  trustBusy: boolean;
  onTrust: () => void;
}) {
  if (r.ok) {
    return (
      <div className="strip-line strip-note">
        <span className="strip-text">Trusted — reload your wallet or browser.</span>
      </div>
    );
  }
  return (
    <div className="strip-line strip-warn">
      <span className="strip-text">{r.message}</span>
      {r.ranCommand ? (
        <>
          <code className="strip-cmd">{r.ranCommand}</code>
          <CopyButton value={r.ranCommand} label="Copy command" />
        </>
      ) : null}
      <button
        className="btn btn-ghost btn-tiny"
        disabled={trustBusy}
        title="Run the trust install again. If you just ran the command by hand, this confirms the certificate is trusted."
        onClick={onTrust}
      >
        {trustBusy ? "Trying…" : "Try again"}
      </button>
    </div>
  );
}

export function GatewayIdentity({
  gw,
  caPath,
  targetMode,
  trustBusy,
  trustResult,
  onTrust,
}: {
  gw: GatewayView;
  caPath: string | null;
  targetMode: string;
  trustBusy: boolean;
  trustResult: TrustCertResult | null;
  onTrust: () => void;
}) {
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
      {running && targetMode === "local" && caPath ? (
        <div className="rpc-ident-trust">
          <span className="muted small">Wallets must trust this gateway's certificate —</span>
          <button
            className="btn btn-ghost btn-tiny"
            disabled={trustBusy}
            title="Install this gateway's root certificate into this machine's trust store, then reload your wallet."
            onClick={onTrust}
          >
            {trustBusy ? "Trusting…" : "Trust on this machine"}
          </button>
          <CopyButton
            value={caPath}
            label="Copy cert path"
            title={`Copy the path to Caddy's root certificate. Install it on ${gw.placement.targetId} and in the trust store of any device that will call this URL, and the warning goes away.`}
          />
          {trustResult ? <TrustResultLine r={trustResult} trustBusy={trustBusy} onTrust={onTrust} /> : null}
        </div>
      ) : null}
    </div>
  );
}
