// #/vpn — "Private access": one global page for standing up WireGuard access.
//
// Two ways in, both on the same page:
//   · Set up on a device — Jumpgate provisions a WireGuard SERVER on one of
//     your machines. The private key is minted on the host and never leaves
//     it; you then enroll phones/laptops as peers.
//   · Provide credentials — paste a provider's .conf and Jumpgate brings that
//     BYO overlay up and down for you.
//
// The screen consumes the shipped hooks/vpn.ts data layer only — it never
// touches api.ts directly (the type-only import below is just for request
// shapes), so every load/poll/invalidate is React Query's job, matching the
// rest of the migrated screens. Every input is a controlled useState value,
// the same rule Settings.tsx follows.
import { useState } from "react";
import type * as api from "../../api";
import { Footer } from "../../components/Footer";
import { Badge } from "../../components/Badge";
import {
  useVpns,
  useVpnServers,
  useVpnStatus,
  useVpnServerStatus,
  useSaveVpn,
  useDeleteVpn,
  useVpnAction,
  useProvisionVpnServer,
  useEnrollVpnDevice,
  useRevokeVpnDevice,
  useVpnServerAction,
  useDeleteVpnServer,
} from "../../hooks/vpn";
import { useTargets } from "../../hooks/target";

// --- small shared helpers --------------------------------------------------

// A Copy button that writes to the clipboard and flips its own label for a
// beat. navigator.clipboard is absent in some contexts (and in jsdom), so the
// write is guarded rather than assumed.
function CopyBtn({ text, label = "Copy" }: { text: string; label?: string }) {
  const [copied, setCopied] = useState(false);
  return (
    <button
      type="button"
      className="btn btn-ghost btn-tiny"
      onClick={() => {
        if (navigator.clipboard?.writeText) {
          navigator.clipboard.writeText(text).catch(() => {});
        }
        setCopied(true);
        setTimeout(() => setCopied(false), 1200);
      }}
    >
      {copied ? "Copied" : label}
    </button>
  );
}

// download offers `text` as a file the browser saves, without a server round
// trip — the enrolled device's config is emitted once and kept nowhere, so the
// Blob is the only copy the operator gets besides Copy.
function download(name: string, text: string) {
  const blob = new Blob([text], { type: "text/plain" });
  const url = URL.createObjectURL(blob);
  const a = document.createElement("a");
  a.href = url;
  a.download = `${name}.conf`;
  document.body.appendChild(a);
  a.click();
  a.remove();
  URL.revokeObjectURL(url);
}

// relTime renders a unix-seconds handshake time as "14s ago" / "2m ago" / …,
// or "never" for the zero value a peer that has never connected carries.
function relTime(unixSeconds: number): string {
  if (!unixSeconds) return "never";
  const secs = Math.max(0, Math.floor(Date.now() / 1000 - unixSeconds));
  if (secs < 60) return `${secs}s ago`;
  if (secs < 3600) return `${Math.floor(secs / 60)}m ago`;
  if (secs < 86400) return `${Math.floor(secs / 3600)}h ago`;
  return `${Math.floor(secs / 86400)}d ago`;
}

// fingerprint is the head of a public key with an ellipsis — enough to tell two
// peers apart without wrapping a full 44-char base64 key across the row.
function fingerprint(publicKey: string): string {
  return publicKey.length > 8 ? `${publicKey.slice(0, 8)}…` : publicKey;
}

// --- one provisioned server ------------------------------------------------

function ServerCard({ server, firewallHint }: { server: api.VpnServerView; firewallHint?: string }) {
  const statusQ = useVpnServerStatus(server.id, true);
  const enroll = useEnrollVpnDevice();
  const revoke = useRevokeVpnDevice();
  const action = useVpnServerAction();
  const del = useDeleteVpnServer();
  const setEndpoint = useProvisionVpnServer();

  const [deviceName, setDeviceName] = useState("");
  const [fullTunnel, setFullTunnel] = useState(false);
  const [endpointHost, setEndpointHost] = useState("");
  // handoff is the one-time config the last enroll returned — shown until the
  // operator dismisses it. Jumpgate keeps no copy, so this is the only place
  // it ever appears.
  const [handoff, setHandoff] = useState<api.VpnEnrollResult | null>(null);
  const [localHint, setLocalHint] = useState<string | undefined>(undefined);
  const [error, setError] = useState<string | null>(null);

  const endpointSet = server.endpoint !== "";
  const status = statusQ.data;
  const up = status?.up ?? false;
  const hint = localHint ?? firewallHint ?? `ufw allow ${server.listenPort}/udp`;
  const machine = server.targetId || "this machine";

  async function handleEnroll() {
    setError(null);
    try {
      const result = await enroll.mutateAsync({
        id: server.id,
        body: { name: deviceName.trim(), fullTunnel },
      });
      setHandoff(result);
      setDeviceName("");
    } catch (err) {
      setError(err instanceof Error ? err.message : String(err));
    }
  }

  async function handleSetEndpoint() {
    setError(null);
    try {
      // Provision is idempotent: the same id with an endpointHost updates the
      // endpoint on the existing server rather than standing up a new one.
      const res = await setEndpoint.mutateAsync({ id: server.id, endpointHost: endpointHost.trim() });
      setLocalHint(res.firewallHint);
      setEndpointHost("");
    } catch (err) {
      setError(err instanceof Error ? err.message : String(err));
    }
  }

  async function handleRevoke(peer: api.VpnPeerView) {
    if (!window.confirm(`Revoke ${peer.name}? Its config stops working on the next handshake.`)) return;
    setError(null);
    try {
      await revoke.mutateAsync({ id: server.id, publicKey: peer.publicKey });
    } catch (err) {
      setError(err instanceof Error ? err.message : String(err));
    }
  }

  // handleToggle disconnects an up server or reconnects a down one. A
  // disconnect is reversible — the conf, key and enrolled devices stay put — so
  // it needs no confirm; reconnecting brings the same identity back.
  async function handleToggle() {
    setError(null);
    try {
      await action.mutateAsync({ id: server.id, action: up ? "down" : "up" });
    } catch (err) {
      setError(err instanceof Error ? err.message : String(err));
    }
  }

  async function handleRemove() {
    if (
      !window.confirm(
        `Wipe server ${server.id}? This tears the interface down on ${machine} and deletes its conf and key ` +
          `from the host — every enrolled device loses access and cannot be reconnected. To pause it instead, ` +
          `use Disconnect.`,
      )
    )
      return;
    setError(null);
    try {
      await del.mutateAsync(server.id);
    } catch (err) {
      setError(err instanceof Error ? err.message : String(err));
    }
  }

  return (
    <div className="card">
      <div className="service-head">
        <h2>
          {server.interface} <span className="muted" style={{ fontWeight: 400 }}>on</span> {machine}
        </h2>
        <span className="head-badges">
          {status ? (
            <Badge text={status.up ? "up" : "down"} kind={status.up ? "ok" : "neutral"} />
          ) : (
            <Badge text="checking" kind="neutral" />
          )}
          {endpointSet ? null : <Badge text="endpoint not set" kind="warn" />}
          {status?.handshaked && <Badge text={`handshake ${relTime(status.lastHandshake)}`} kind="ok" />}
          <Badge text={`${server.peers.length} device${server.peers.length === 1 ? "" : "s"}`} kind="neutral" />
        </span>
      </div>

      <dl className="stat-list">
        <div>
          <dt>overlay address</dt>
          <dd>{server.address}</dd>
        </div>
        <div>
          <dt>listen port</dt>
          <dd>{server.listenPort}/udp</dd>
        </div>
        <div>
          <dt>endpoint</dt>
          <dd>{server.endpoint || "—"}</dd>
        </div>
        <div>
          <dt>interface</dt>
          <dd>{server.interface}</dd>
        </div>
      </dl>

      <label style={{ marginBottom: 0 }}>
        server public key
        <span className="keybox">
          <code>{server.publicKey}</code>
          <CopyBtn text={server.publicKey} />
        </span>
      </label>
      <p className="muted small" style={{ margin: "0.5rem 0 0" }}>
        The server&apos;s private key was minted on {machine} and never left it. Jumpgate never saw it.
      </p>

      <div className="strip">
        <div className="strip-line strip-warn">
          <span className="strip-text">
            Open the port on the machine&apos;s firewall — Jumpgate never changes your firewall for you.
          </span>
          <code className="strip-cmd">{hint}</code>
          <CopyBtn text={hint} />
        </div>
      </div>

      {!endpointSet && (
        <>
          <div className="strip">
            <div className="strip-line strip-warn">
              <span className="strip-text">
                Devices need a public address to dial. Set the host or domain that reaches this machine on
                port {server.listenPort}.
              </span>
            </div>
            <div className="strip-line strip-note">
              <span className="strip-text">
                Enrollment is disabled until the endpoint is set — a config minted now would have nowhere to
                dial.
              </span>
            </div>
          </div>
          <div className="endpoint-fix">
            <label>
              public host or domain
              <input
                type="text"
                placeholder="vpn.example.com or 203.0.113.7"
                value={endpointHost}
                onChange={(e) => setEndpointHost(e.target.value)}
              />
            </label>
            <button
              type="button"
              className="btn"
              disabled={setEndpoint.isPending || !endpointHost.trim()}
              onClick={() => void handleSetEndpoint()}
            >
              {setEndpoint.isPending ? "Setting…" : "Set endpoint"}
            </button>
          </div>
        </>
      )}

      {/* --- enrolled devices --- */}
      {server.peers.length > 0 && (
        <div style={{ overflowX: "auto" }}>
          <div className="peers-head">
            <span>device</span>
            <span>overlay ip</span>
            <span>public key</span>
            <span />
          </div>
          {server.peers.map((peer) => (
            <div className="peer-row" key={peer.publicKey}>
              <span className="peer-name">
                <span className="dot dot-neutral" />
                {peer.name}
              </span>
              <span className="peer-ip">{peer.allowedIp}</span>
              <span className="peer-key">
                <code>{fingerprint(peer.publicKey)}</code>
              </span>
              <span className="peer-acts">
                <button
                  type="button"
                  className="btn btn-danger btn-tiny"
                  disabled={revoke.isPending}
                  onClick={() => void handleRevoke(peer)}
                >
                  Revoke
                </button>
              </span>
            </div>
          ))}
        </div>
      )}

      {/* --- enroll a device --- */}
      <div className="strip" style={{ marginTop: "0.9rem" }}>
        <div className="strip-line" style={{ borderLeft: "3px solid transparent", display: "block" }}>
          <h3 style={{ margin: "0 0 0.4rem" }}>Enroll a device</h3>
          <label>
            device name
            <input
              type="text"
              placeholder="mikes-iphone"
              value={deviceName}
              onChange={(e) => setDeviceName(e.target.value)}
              disabled={!endpointSet}
            />
          </label>
          <label style={{ marginBottom: "0.25rem" }}>traffic</label>
          <label className="radio">
            <input
              type="radio"
              name={`route-${server.id}`}
              checked={!fullTunnel}
              onChange={() => setFullTunnel(false)}
              disabled={!endpointSet}
            />
            <span>
              Just reach this network
              <span className="radio-sub">
                Only the overlay goes through the tunnel. Everything else uses the device&apos;s normal
                connection.
              </span>
            </span>
          </label>
          <label className="radio">
            <input
              type="radio"
              name={`route-${server.id}`}
              checked={fullTunnel}
              onChange={() => setFullTunnel(true)}
              disabled={!endpointSet}
            />
            <span>
              Route all traffic
              <span className="radio-sub">The device sends everything through this server (0.0.0.0/0, ::/0).</span>
            </span>
          </label>
          <div className="card-actions">
            <button
              type="button"
              className="btn btn-primary"
              disabled={!endpointSet || enroll.isPending || !deviceName.trim()}
              onClick={() => void handleEnroll()}
            >
              {enroll.isPending ? "Enrolling…" : "Enroll"}
            </button>
          </div>
        </div>
      </div>

      {/* --- one-time handoff --- */}
      {handoff && (
        <div className="card" style={{ marginTop: "0.9rem" }}>
          <div className="service-head">
            <h2>{handoff.name}</h2>
            <span className="head-badges">
              <Badge text="enrolled" kind="ok" />
              <Badge text={handoff.allowedIp} kind="neutral" />
            </span>
          </div>
          <div className="banner banner-warn" style={{ margin: "0.5rem 0 0" }}>
            Shown once — save it now. The device&apos;s private key is in this config and Jumpgate does not keep
            a copy. Close this and it is gone for good; you would revoke and re-enroll.
          </div>
          <div className="handoff">
            {/*
              QR placeholder: a future mobile feature will render `handoff.config`
              as a scannable QR here so a phone can import it by camera. The
              config string below is the single source; when the QR lands it will
              encode exactly this text. No QR library is pulled in now.
            */}
            <div className="conf-wrap">
              <div className="conf-head">
                <span className="muted small">{handoff.name}.conf</span>
                <span>
                  <CopyBtn text={handoff.config} />
                  <button
                    type="button"
                    className="btn btn-ghost btn-tiny"
                    onClick={() => download(handoff.name, handoff.config)}
                  >
                    Download
                  </button>
                </span>
              </div>
              <pre className="conf-block">{handoff.config}</pre>
            </div>
          </div>
          <div className="card-actions">
            <button type="button" className="btn" onClick={() => setHandoff(null)}>
              I saved it — close
            </button>
          </div>
        </div>
      )}

      {error && <p className="error">{error}</p>}

      <div className="card-actions">
        {/*
          Two distinct teardowns. Disconnect is reversible — it drops the
          interface but keeps the conf, key and enrolled devices, so Reconnect
          brings the exact same server back. Wipe is terminal — it deletes the
          conf and key from the host, so enrolled devices can never reconnect.
        */}
        <button
          type="button"
          className="btn"
          disabled={action.isPending}
          onClick={() => void handleToggle()}
        >
          {action.isPending ? (up ? "Disconnecting…" : "Reconnecting…") : up ? "Disconnect" : "Reconnect"}
        </button>
        <button type="button" className="btn btn-danger" disabled={del.isPending} onClick={() => void handleRemove()}>
          {del.isPending ? "Wiping…" : "Wipe server"}
        </button>
      </div>
    </div>
  );
}

// --- one BYO overlay -------------------------------------------------------

function OverlayCard({ overlay }: { overlay: api.VpnView }) {
  const statusQ = useVpnStatus(overlay.id, true);
  const action = useVpnAction();
  const del = useDeleteVpn();
  const [error, setError] = useState<string | null>(null);

  const up = statusQ.data?.up ?? false;

  async function handleToggle() {
    setError(null);
    try {
      await action.mutateAsync({ id: overlay.id, action: up ? "down" : "up" });
    } catch (err) {
      setError(err instanceof Error ? err.message : String(err));
    }
  }

  async function handleRemove() {
    if (!window.confirm(`Remove overlay ${overlay.id}?`)) return;
    setError(null);
    try {
      await del.mutateAsync(overlay.id);
    } catch (err) {
      setError(err instanceof Error ? err.message : String(err));
    }
  }

  return (
    <div className="card">
      <div className="service-head">
        <h2>
          {overlay.id} <span className="muted" style={{ fontWeight: 400 }}>· provided credentials</span>
        </h2>
        <span className="head-badges">
          <Badge text={up ? "up" : "down"} kind={up ? "ok" : "neutral"} />
          <button
            type="button"
            className={`sw${up ? " on" : ""}`}
            role="switch"
            aria-checked={up}
            aria-label="Bring overlay up or down"
            disabled={action.isPending}
            onClick={() => void handleToggle()}
          />
        </span>
      </div>
      <dl className="stat-list">
        <div>
          <dt>endpoint</dt>
          <dd>{overlay.endpoints[0] ?? "—"}</dd>
        </div>
        <div>
          <dt>overlay address</dt>
          <dd>{overlay.overlay[0] ?? "—"}</dd>
        </div>
      </dl>
      <p className="muted small" style={{ margin: "0.5rem 0 0" }}>
        From the <code>.conf</code> you pasted. Jumpgate brings this tunnel up and down; the provider runs the
        server. Grades bind to it as <Badge text="private" kind="ok" /> while it is up.
      </p>
      {overlay.error && <p className="error">{overlay.error}</p>}
      {error && <p className="error">{error}</p>}
      <div className="card-actions">
        <button type="button" className="btn btn-danger" disabled={del.isPending} onClick={() => void handleRemove()}>
          Remove
        </button>
      </div>
    </div>
  );
}

// --- the screen ------------------------------------------------------------

export function PrivateAccess() {
  const serversQ = useVpnServers();
  const overlaysQ = useVpns();
  const targetsQ = useTargets();
  const provision = useProvisionVpnServer();
  const save = useSaveVpn();

  const [mode, setMode] = useState<"device" | "credentials" | null>(null);

  // provision form
  const [provId, setProvId] = useState("");
  const [provTargetId, setProvTargetId] = useState("");
  const [iface, setIface] = useState("jumpgate0");
  const [address, setAddress] = useState("10.9.0.1/24");
  const [listenPort, setListenPort] = useState("51820");
  const [endpointHost, setEndpointHost] = useState("");
  const [provError, setProvError] = useState<string | null>(null);
  // firewall hints keyed by server id — the hint rides only the provision
  // result, never the server list, so it is kept here to hand to the card.
  const [hints, setHints] = useState<Record<string, string>>({});

  // BYO form
  const [byoId, setByoId] = useState("");
  const [byoConfig, setByoConfig] = useState("");
  const [byoError, setByoError] = useState<string | null>(null);

  const targets = targetsQ.data ?? [];
  const servers = serversQ.data ?? [];
  const overlays = overlaysQ.data ?? [];

  async function handleProvision() {
    setProvError(null);
    const body: api.ProvisionVpnServerRequest = { id: provId.trim() };
    if (provTargetId) body.targetId = provTargetId;
    if (iface.trim()) body.interface = iface.trim();
    if (address.trim()) body.address = address.trim();
    const port = parseInt(listenPort, 10);
    if (!Number.isNaN(port)) body.listenPort = port;
    if (endpointHost.trim()) body.endpointHost = endpointHost.trim();
    try {
      const res = await provision.mutateAsync(body);
      setHints((h) => ({ ...h, [res.server.id]: res.firewallHint }));
      setProvId("");
      setEndpointHost("");
      setMode(null);
    } catch (err) {
      setProvError(err instanceof Error ? err.message : String(err));
    }
  }

  async function handleSaveByo() {
    setByoError(null);
    try {
      await save.mutateAsync({ id: byoId.trim(), config: byoConfig });
      setByoId("");
      setByoConfig("");
      setMode(null);
    } catch (err) {
      setByoError(err instanceof Error ? err.message : String(err));
    }
  }

  return (
    <>
      <h1>Private access</h1>
      <p className="muted small" style={{ maxWidth: "62ch" }}>
        A private tunnel in front of your endpoints. Only enrolled devices can reach what sits behind it — your
        gateways stay off the public internet.
      </p>

      {/* --- entry: the two modes --- */}
      <div className="section">
        <div className="section-head">
          <h2 style={{ margin: 0 }}>Add private access</h2>
        </div>
        <div className="mode-grid">
          <button
            type="button"
            className={`mode-card recommended${mode === "device" ? " selected" : ""}`}
            onClick={() => setMode(mode === "device" ? null : "device")}
          >
            <span className="mode-head">
              <strong>Set up on a device</strong>
              <Badge text="recommended" kind="ok" />
            </span>
            <p>
              Pick one of your machines and Jumpgate installs a WireGuard server on it. The key is minted on the
              host and never leaves it. Then enroll your phone and laptop onto it.
            </p>
            <span className="mode-foot">Your hardware, your keys →</span>
          </button>
          <button
            type="button"
            className={`mode-card${mode === "credentials" ? " selected" : ""}`}
            onClick={() => setMode(mode === "credentials" ? null : "credentials")}
          >
            <span className="mode-head">
              <strong>Provide credentials</strong>
              <Badge text="simplest" kind="neutral" />
            </span>
            <p>
              Paste a WireGuard <code>.conf</code> from a provider — Proton, Mullvad, IVPN, or your own mesh.
              Jumpgate brings the tunnel up and down for you.
            </p>
            <span className="mode-foot">Bring your own overlay →</span>
          </button>
        </div>

        {mode === "device" && (
          <div className="card">
            <h2>Set up on a device</h2>
            <label>
              name
              <input
                type="text"
                placeholder="wg0"
                value={provId}
                onChange={(e) => setProvId(e.target.value)}
              />
            </label>
            <label>
              machine
              <select value={provTargetId} onChange={(e) => setProvTargetId(e.target.value)}>
                <option value="">This machine (local)</option>
                {targets.map((t) => (
                  <option key={t.id} value={t.id}>
                    {t.id}
                  </option>
                ))}
              </select>
            </label>
            <details className="advanced">
              <summary>Advanced</summary>
              <label>
                interface
                <input type="text" value={iface} onChange={(e) => setIface(e.target.value)} />
              </label>
              <label>
                overlay address
                <input type="text" value={address} onChange={(e) => setAddress(e.target.value)} />
              </label>
              <label>
                listen port
                <input type="text" value={listenPort} onChange={(e) => setListenPort(e.target.value)} />
              </label>
              <label>
                public host or domain (endpoint)
                <input
                  type="text"
                  placeholder="vpn.example.com — can be set later"
                  value={endpointHost}
                  onChange={(e) => setEndpointHost(e.target.value)}
                />
              </label>
            </details>
            {provError && <p className="error">{provError}</p>}
            <div className="card-actions">
              <button
                type="button"
                className="btn btn-primary"
                disabled={provision.isPending || !provId.trim()}
                onClick={() => void handleProvision()}
              >
                {provision.isPending ? "Provisioning…" : "Provision server"}
              </button>
            </div>
          </div>
        )}

        {mode === "credentials" && (
          <div className="card">
            <h2>Provide credentials</h2>
            <label>
              name
              <input
                type="text"
                placeholder="proton-nl-42"
                value={byoId}
                onChange={(e) => setByoId(e.target.value)}
              />
            </label>
            <label>
              WireGuard .conf
              <textarea
                placeholder="[Interface]&#10;PrivateKey = …&#10;Address = …&#10;&#10;[Peer]&#10;…"
                value={byoConfig}
                onChange={(e) => setByoConfig(e.target.value)}
              />
            </label>
            {byoError && <p className="error">{byoError}</p>}
            <div className="card-actions">
              <button
                type="button"
                className="btn btn-primary"
                disabled={save.isPending || !byoId.trim() || !byoConfig.trim()}
                onClick={() => void handleSaveByo()}
              >
                {save.isPending ? "Adding…" : "Add overlay"}
              </button>
            </div>
          </div>
        )}
      </div>

      {/* --- provisioned servers --- */}
      <div className="section">
        <h2>Servers</h2>
        {serversQ.isLoading ? (
          <p className="muted">Loading…</p>
        ) : serversQ.isError ? (
          <p className="error">Failed to load servers: {String(serversQ.error)}</p>
        ) : servers.length === 0 ? (
          <div className="card empty-state">
            <p>No servers yet</p>
            <p className="muted small">Set one up on a device above to enroll your phone and laptop.</p>
          </div>
        ) : (
          <div className="card-grid card-grid-wide">
            {servers.map((s) => (
              <ServerCard key={s.id} server={s} firewallHint={hints[s.id]} />
            ))}
          </div>
        )}
      </div>

      {/* --- BYO overlays --- */}
      <div className="section">
        <h2>Bring-your-own overlays</h2>
        {overlaysQ.isLoading ? (
          <p className="muted">Loading…</p>
        ) : overlaysQ.isError ? (
          <p className="error">Failed to load overlays: {String(overlaysQ.error)}</p>
        ) : overlays.length === 0 ? (
          <div className="card empty-state">
            <p>No overlays yet</p>
            <p className="muted small">Paste a provider&apos;s .conf above to bring one up.</p>
          </div>
        ) : (
          <div className="card-grid card-grid-wide">
            {overlays.map((o) => (
              <OverlayCard key={o.id} overlay={o} />
            ))}
          </div>
        )}
      </div>

      <Footer />
    </>
  );
}
