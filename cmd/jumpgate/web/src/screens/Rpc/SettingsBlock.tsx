// The gateway's settings, edited in place: port, bind, the request-counter
// switch, and the HTTPS front (three cert sources: Caddy's own CA, a file on
// disk, and a public Let's Encrypt certificate for the operator's own domain).
// Controlled inputs replace the old querySelector-on-save read; the
// values are handed up on Save and the parent builds the config. The live
// "Verify HTTPS now" check sits here because that is where the question is
// asked.
import { useState } from "react";
import type { GatewayView, TlsVerification } from "../../api";
import { Badge, type BadgeKind } from "../../components/Badge";

export interface SettingsValues {
  port: number | null;
  bind: string;
  metricsOn: boolean;
  tls: {
    enabled: boolean;
    hostname: string;
    certSource: string;
    certFile: string;
    keyFile: string;
    acmeEmail: string;
    httpsPort: number | null;
  };
}

function verifyBadge(status: string): { text: string; kind: BadgeKind } {
  switch (status) {
    case "pass":
      return { text: "pass", kind: "ok" };
    case "fail":
      return { text: "fail", kind: "bad" };
    case "unavailable":
      return { text: "unavailable", kind: "warn" };
    default:
      return { text: "skipped", kind: "neutral" };
  }
}

function VerifyReport({ v }: { v: TlsVerification }) {
  return (
    <>
      <div className={`banner ${v.ok ? (v.subscriptionsOk ? "banner-ok" : "banner-warn") : "banner-bad"}`}>{v.summary}</div>
      <ul className="verify-list">
        {(v.assertions ?? []).map((a) => {
          const b = verifyBadge(a.status);
          return (
            <li key={a.id} className="small">
              <Badge text={b.text} kind={b.kind} />
              <strong>{a.title}</strong>
              <div className="muted">{a.detail}</div>
            </li>
          );
        })}
      </ul>
      <p className="muted small">
        Checked {new Date(v.at).toLocaleString()} against <code>{v.address}</code>
        {v.notAfter ? (
          <>
            {" · certificate valid until "}
            <code>{new Date(v.notAfter).toLocaleString()}</code> ({v.expiresIn ?? ""})
          </>
        ) : null}
      </p>
      {v.expiryWarning ? <div className="banner banner-warn">{v.expiryWarning}</div> : null}
    </>
  );
}

function VerifyPanel({
  enabled,
  verifying,
  verifyResult,
  verifyErr,
  onVerify,
}: {
  enabled: boolean;
  verifying: boolean;
  verifyResult: TlsVerification | null;
  verifyErr: string | null;
  onVerify: () => void;
}) {
  return (
    <>
      <hr />
      <div className="card-actions">
        <button
          className="btn btn-ghost"
          disabled={!enabled || verifying}
          title="Open a real connection to this front: handshake, certificate name, chain, an RPC call and a subscription."
          onClick={onVerify}
        >
          {verifying ? (
            <>
              <span className="spinner" aria-label="verifying" /> Verifying…
            </>
          ) : (
            "Verify HTTPS now"
          )}
        </button>
        {enabled ? null : (
          <span className="muted small">Turn HTTPS on and re-create the gateway — there is nothing to verify yet.</span>
        )}
      </div>
      {verifyErr ? <p className="error small">{verifyErr}</p> : null}
      {verifyResult ? <VerifyReport v={verifyResult} /> : null}
    </>
  );
}

export function SettingsBlock({
  gw,
  onSave,
  verifying,
  verifyResult,
  verifyErr,
  onVerify,
}: {
  gw: GatewayView;
  onSave: (values: SettingsValues) => void;
  verifying: boolean;
  verifyResult: TlsVerification | null;
  verifyErr: string | null;
  onVerify: () => void;
}) {
  const c = gw.config;
  const t = c.TLS ?? null;
  const suggested = gw.tls?.suggestedHostname ?? "";

  const [port, setPort] = useState(String(c.Port));
  const [bind, setBind] = useState(c.BindAddr);
  const [metricsOn, setMetricsOn] = useState(!c.MetricsOff);
  const [tlsOn, setTlsOn] = useState(t?.Enabled ?? false);
  const [hostname, setHostname] = useState(t?.Hostname ?? suggested);
  const [certSource, setCertSource] = useState(t?.CertSource || "internal");
  const [certFile, setCertFile] = useState(t?.CertFile ?? "");
  const [keyFile, setKeyFile] = useState(t?.KeyFile ?? "");
  const [acmeEmail, setAcmeEmail] = useState(t?.ACMEEmail ?? "");
  const [httpsPort, setHttpsPort] = useState(String(t?.HTTPSPort || 443));

  const isAcme = certSource === "acme";

  function save() {
    const p = Number.parseInt(port.trim(), 10);
    const hp = Number.parseInt(httpsPort.trim(), 10);
    onSave({
      port: Number.isFinite(p) ? p : null,
      bind: bind.trim(),
      metricsOn,
      tls: {
        enabled: tlsOn,
        hostname: hostname.trim(),
        certSource,
        certFile: certFile.trim(),
        keyFile: keyFile.trim(),
        acmeEmail: acmeEmail.trim(),
        httpsPort: Number.isFinite(hp) ? hp : null,
      },
    });
  }

  return (
    <div className="card config-block">
      <p className="muted small">Gateway settings — saved here, applied by “Re-create”.</p>
      <label>
        Listen port
        <input type="text" inputMode="numeric" value={port} autoComplete="off" onChange={(e) => setPort(e.target.value)} />
      </label>
      <label>
        Bind address <span className="muted">— 127.0.0.1 keeps it on that machine; 0.0.0.0 exposes it to your network</span>
        <input type="text" value={bind} autoComplete="off" spellCheck={false} onChange={(e) => setBind(e.target.value)} />
      </label>
      <p className="muted small">
        Requests are addressed by path: <code>/{c.ProjectID}/evm/&lt;chainId&gt;</code>. One port serves every network in
        the bar above, and the same path serves WebSocket with a <code>ws://</code> scheme.
      </p>
      <label className="check">
        <input type="checkbox" checked={metricsOn} onChange={(e) => setMetricsOn(e.target.checked)} />
        Count this gateway's own requests
      </label>
      <p className="muted small">
        The gateway counts which endpoints answer its requests, so this screen can show where your traffic is actually
        going. The counters stay on the machine the gateway runs on — they are served on loopback and nothing is sent
        anywhere. Turn this off and the share column goes blank.
      </p>
      <hr />
      <label className="check">
        <input type="checkbox" checked={tlsOn} onChange={(e) => setTlsOn(e.target.checked)} />
        Serve HTTPS (a Caddy container in front of eRPC)
      </label>
      <p className="muted small">
        A page served over <code>https://</code> cannot call an <code>http://</code> endpoint. Chrome and Firefox make an
        exception for <code>http://localhost</code>; Safari does not, and every browser blocks it for any other address —
        so a gateway on a LAN or Tailscale address is unusable from a browser dApp without this.
      </p>
      <label>
        {isAcme ? (
          <>
            Public domain <span className="muted">— your own name, pointed at this box</span>
          </>
        ) : (
          <>
            Hostname <span className="muted">— must resolve to this machine</span>
          </>
        )}
        <input
          type="text"
          value={hostname}
          placeholder={isAcme ? "rpc.your-company.com" : suggested || "gateway.example.com"}
          autoComplete="off"
          spellCheck={false}
          onChange={(e) => setHostname(e.target.value)}
        />
      </label>
      {!isAcme && suggested ? (
        <p className="muted small">
          The default is <code>{suggested}</code>. That whole domain's wildcard resolves to <code>127.0.0.1</code> from
          any network, so the name works on this machine with nothing to install and no hosts file to edit — and it is
          unique to this install, so two machines never serve different certificates for the same name.
        </p>
      ) : null}
      <label>
        HTTPS port
        <input type="text" inputMode="numeric" value={httpsPort} autoComplete="off" onChange={(e) => setHttpsPort(e.target.value)} />
      </label>
      <label>
        Certificate
        <select value={certSource} onChange={(e) => setCertSource(e.target.value)}>
          <option value="internal">Caddy's own authority — works offline, one trust-store install</option>
          <option value="files">A certificate file on this machine</option>
          <option value="acme">A public domain — a real certificate, trusted everywhere</option>
        </select>
      </label>
      {isAcme ? (
        <>
          <label>
            Contact email <span className="muted">— optional, for Let's Encrypt expiry notices</span>
            <input
              type="text"
              value={acmeEmail}
              placeholder="ops@your-company.com"
              autoComplete="off"
              spellCheck={false}
              onChange={(e) => setAcmeEmail(e.target.value)}
            />
          </label>
          <p className="muted small">
            Caddy gets a real certificate from Let's Encrypt for the public domain above. Three things must be true, and
            this app cannot check them for you: this box is reachable from the internet on port 80 and port 443, and a
            DNS record for the domain points at this box. Port 80 answers the certificate challenge.
          </p>
          <div className="banner banner-warn">
            This endpoint is OPEN. Anyone who learns the URL can use it for free. Keys and metering arrive in a later
            step — do not put this in front of paying customers yet.
          </div>
        </>
      ) : (
        <>
          <label>
            Certificate file <span className="muted">— path on that machine, used only for “a certificate file”</span>
            <input
              type="text"
              value={certFile}
              placeholder="/var/lib/valve-node-app/tls/cert.pem"
              autoComplete="off"
              spellCheck={false}
              onChange={(e) => setCertFile(e.target.value)}
            />
          </label>
          <label>
            Private key file
            <input
              type="text"
              value={keyFile}
              placeholder="/var/lib/valve-node-app/tls/key.pem"
              autoComplete="off"
              spellCheck={false}
              onChange={(e) => setKeyFile(e.target.value)}
            />
          </label>
          <p className="muted small">
            If that certificate is missing, unreadable, expired or does not cover the hostname, HTTPS stays on and falls
            back to Caddy's own authority — with the reason shown above. A dead endpoint is worse than a one-time browser
            warning, and certificate lifetimes are shrinking every year.
          </p>
        </>
      )}
      <VerifyPanel enabled={tlsOn} verifying={verifying} verifyResult={verifyResult} verifyErr={verifyErr} onVerify={onVerify} />
      <div className="card-actions">
        <button className="btn" onClick={save}>
          Save settings
        </button>
      </div>
    </div>
  );
}
