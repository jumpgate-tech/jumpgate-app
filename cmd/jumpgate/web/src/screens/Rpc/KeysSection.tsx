// Customer API keys for a metered gateway.
//
// The section shows as little as it can: a list, one button, and nothing else
// until the operator asks. The one exception is the freshly minted key, which
// gets the loudest treatment on the screen — the store keeps only a hash, so if
// the operator navigates away without copying it, it is gone for good and the
// only remedy is issuing another.
import { useCallback, useEffect, useState } from "react";
import { ApiError, createKey, listKeys, revokeKey, type KeyView } from "../../api";
import { CopyButton } from "./CopyButton";

// A gateway that sells no keys has no key store, and the server says so with a
// 501. That is a configuration fact rather than a failure, so it reads as a
// quiet note instead of an error.
const NOT_CONFIGURED = 501;

export function KeysSection({ gid, open, onToggle }: {
  gid: string;
  open: boolean;
  onToggle: () => void;
}) {
  const [keys, setKeys] = useState<KeyView[] | null>(null);
  const [unavailable, setUnavailable] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [busy, setBusy] = useState(false);
  const [label, setLabel] = useState("");
  const [minted, setMinted] = useState<{ id: string; key: string } | null>(null);

  const refresh = useCallback(async () => {
    try {
      setKeys(await listKeys(gid));
      setUnavailable(false);
      setError(null);
    } catch (e) {
      if (e instanceof ApiError && e.status === NOT_CONFIGURED) {
        setUnavailable(true);
        setKeys([]);
        return;
      }
      setError(e instanceof Error ? e.message : String(e));
    }
  }, [gid]);

  useEffect(() => {
    if (open) void refresh();
  }, [open, refresh]);

  async function onCreate() {
    setBusy(true);
    setError(null);
    try {
      setMinted(await createKey(gid, label.trim()));
      setLabel("");
      await refresh();
    } catch (e) {
      setError(e instanceof Error ? e.message : String(e));
    } finally {
      setBusy(false);
    }
  }

  async function onRevoke(keyId: string) {
    setBusy(true);
    setError(null);
    try {
      await revokeKey(gid, keyId);
      await refresh();
    } catch (e) {
      setError(e instanceof Error ? e.message : String(e));
    } finally {
      setBusy(false);
    }
  }

  const active = (keys ?? []).filter((k) => !k.disabled).length;

  return (
    <section className={`card manage-section${open ? " open" : ""}`}>
      <button type="button" className="manage-head" aria-expanded={open} onClick={onToggle}>
        <span className="manage-title">API keys</span>
        {keys && !unavailable ? <span className="badge badge-neutral">{active}</span> : null}
      </button>

      {open ? (
        <div className="manage-body">
          {unavailable ? (
            <p className="cap">
              This gateway has no key store. Start it with a billing socket to sell metered access.
            </p>
          ) : (
            <>
              {minted ? (
                <div className="banner banner-warn keys-minted">
                  <div className="cap">Copy this key now. It is never shown again.</div>
                  <code className="num keys-secret">{minted.key}</code>
                  <CopyButton value={minted.key} label="Copy key" />
                  <button className="btn btn-ghost btn-tiny" onClick={() => setMinted(null)}>
                    Done
                  </button>
                </div>
              ) : null}

              {error ? <div className="banner banner-bad">{error}</div> : null}

              <div className="add-actions">
                <input
                  type="text"
                  placeholder="Label (optional)"
                  value={label}
                  disabled={busy}
                  onChange={(e) => setLabel(e.target.value)}
                />
                <button className="btn" disabled={busy} onClick={() => void onCreate()}>
                  {busy ? <span className="spinner" aria-label="working" /> : "Issue key"}
                </button>
              </div>

              {keys && keys.length > 0 ? (
                <table className="surface">
                  <thead>
                    <tr>
                      <th>ID</th>
                      <th>Label</th>
                      <th>Status</th>
                      <th />
                    </tr>
                  </thead>
                  <tbody>
                    {keys.map((k) => (
                      <tr key={k.id}>
                        <td className="num">{k.id}</td>
                        <td>{k.label || <span className="cap">—</span>}</td>
                        <td>
                          <span className={`badge ${k.disabled ? "badge-bad" : "badge-ok"}`}>
                            {k.disabled ? "revoked" : "active"}
                          </span>
                        </td>
                        <td>
                          {k.disabled ? null : (
                            <button
                              className="btn btn-danger btn-tiny"
                              disabled={busy}
                              onClick={() => void onRevoke(k.id)}
                            >
                              Revoke
                            </button>
                          )}
                        </td>
                      </tr>
                    ))}
                  </tbody>
                </table>
              ) : keys ? (
                <p className="cap">No keys yet. Issue one to sell metered access.</p>
              ) : null}
            </>
          )}
        </div>
      ) : null}
    </section>
  );
}
