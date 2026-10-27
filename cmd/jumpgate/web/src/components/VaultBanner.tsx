// The vault notice above every screen. Locked: a banner with Unlock, and the
// unlock prompt opens by itself once per page load (the desktop prompt at
// launch); dismissing it does not reopen it on the next poll. Unlocked: a Lock
// button. Secrets still in config.json: a migrate notice.
import { useEffect, useRef, useState } from "react";
import { useLockVault, useVault } from "../hooks/vault";
import { UnlockDialog } from "./UnlockDialog";

export function VaultBanner() {
  const vault = useVault();
  const lock = useLockVault();
  const [open, setOpen] = useState(false);
  const prompted = useRef(false);

  const state = vault.data?.state;
  useEffect(() => {
    if (state === "locked" && !prompted.current) {
      prompted.current = true;
      setOpen(true);
    }
  }, [state]);

  const data = vault.data;
  if (!data) return null;
  const plain = (data.plain ?? []).length > 0;

  return (
    <>
      {data.state === "locked" && (
        <div className="banner vault-banner" role="status">
          <span>Locked: box actions and plans wait until you unlock.</span>{" "}
          <span className="update-banner-actions">
            <button className="btn btn-primary btn-tiny" type="button" onClick={() => setOpen(true)}>
              Unlock
            </button>
          </span>
        </div>
      )}
      {data.state === "unlocked" && (
        <div className="vault-lock">
          <button className="btn btn-ghost btn-tiny" type="button" disabled={lock.isPending} onClick={() => lock.mutate()}>
            Lock
          </button>
        </div>
      )}
      {plain && (
        <div className="banner vault-plain" role="status">
          Secrets are stored unencrypted. Run jumpgate vault migrate.
        </div>
      )}
      {open && data.state === "locked" && <UnlockDialog onClose={() => setOpen(false)} />}
    </>
  );
}
