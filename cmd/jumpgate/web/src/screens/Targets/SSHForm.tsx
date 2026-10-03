// The "Add server over SSH" form. Port of targets.ts's sshFormMarkup(). It's
// revealed on demand (the "Add a server" button toggles it) rather than
// always shown, so the empty state stays a clean set of guiding actions.
// Controlled inputs replace the legacy version's read-the-DOM-at-submit-time
// approach; mounting is what used to need an explicit focus() call after
// toggling the form open, so the autofocus effect below fires once per mount
// (i.e. every time the form is shown), matching that behavior.
import { useEffect, useRef, useState } from "react";

export interface SSHFormValues {
  host: string;
  user: string;
  keyPath: string;
  port: string;
  id: string;
}

export function SSHForm({
  submitting,
  onSubmit,
}: {
  submitting: boolean;
  onSubmit: (values: SSHFormValues) => void;
}) {
  const [host, setHost] = useState("");
  const [user, setUser] = useState("");
  const [keyPath, setKeyPath] = useState("");
  const [port, setPort] = useState("");
  const [id, setId] = useState("");
  const hostRef = useRef<HTMLInputElement>(null);

  useEffect(() => {
    hostRef.current?.focus();
  }, []);

  return (
    <form className="card" onSubmit={(e) => e.preventDefault()}>
      <h3>Add server over SSH</h3>
      <label>
        Host
        <input
          ref={hostRef}
          type="text"
          placeholder="203.0.113.10"
          autoComplete="off"
          value={host}
          onChange={(e) => setHost(e.target.value)}
        />
      </label>
      <label>
        User
        <input
          type="text"
          placeholder="root"
          autoComplete="off"
          value={user}
          onChange={(e) => setUser(e.target.value)}
        />
      </label>
      <label>
        Private key path
        <input
          type="text"
          placeholder="/home/me/.ssh/id_ed25519"
          autoComplete="off"
          value={keyPath}
          onChange={(e) => setKeyPath(e.target.value)}
        />
      </label>
      <label>
        Port <span className="muted">(optional, default 22)</span>
        <input
          type="text"
          inputMode="numeric"
          placeholder="22"
          autoComplete="off"
          value={port}
          onChange={(e) => setPort(e.target.value)}
        />
      </label>
      <label>
        Target name <span className="muted">(optional, defaults to the host)</span>
        <input
          type="text"
          placeholder="my-node"
          autoComplete="off"
          value={id}
          onChange={(e) => setId(e.target.value)}
        />
      </label>
      <p className="muted small">
        The key never leaves this machine — only its path is stored, and the connection is dialed
        immediately so the host key can be pinned (trust-on-first-use) before it&apos;s saved.
      </p>
      <button
        className="btn"
        type="button"
        disabled={submitting}
        onClick={() => onSubmit({ host, user, keyPath, port, id })}
      >
        {submitting ? "Connecting…" : "Add server"}
      </button>
    </form>
  );
}
