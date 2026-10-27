// The unlock prompt. The field is type="password" and is cleared as soon as
// the passphrase is sent, whatever the answer; the passphrase is never logged.
import { useState, type FormEvent } from "react";
import { ApiError } from "../api";
import { useUnlockVault } from "../hooks/vault";

export function UnlockDialog({ onClose }: { onClose: () => void }) {
  const [passphrase, setPassphrase] = useState("");
  const [error, setError] = useState<string | null>(null);
  const unlock = useUnlockVault();

  function submit(e: FormEvent) {
    e.preventDefault();
    if (!passphrase) return;
    const sent = passphrase;
    setPassphrase("");
    setError(null);
    unlock.mutate(sent, {
      onSuccess: () => onClose(),
      onError: (err) => {
        if (err instanceof ApiError && err.code === "wrong_passphrase") setError("Wrong passphrase");
        else setError(err instanceof Error ? err.message : "Unlock failed");
      },
    });
  }

  return (
    <div className="banner vault-dialog" role="dialog" aria-modal="true" aria-labelledby="vault-dialog-title">
      <form onSubmit={submit}>
        <strong id="vault-dialog-title">Unlock the vault</strong>{" "}
        <label htmlFor="vault-passphrase">Passphrase</label>{" "}
        <input
          id="vault-passphrase"
          type="password"
          autoComplete="current-password"
          autoFocus
          value={passphrase}
          onChange={(e) => setPassphrase(e.target.value)}
        />{" "}
        <button className="btn btn-primary btn-tiny" type="submit" disabled={unlock.isPending}>
          Unlock
        </button>{" "}
        <button className="btn btn-ghost btn-tiny" type="button" onClick={onClose}>
          Not now
        </button>
        {error && (
          <span className="vault-dialog-error" role="alert">
            {" "}
            {error}
          </span>
        )}
      </form>
    </div>
  );
}
