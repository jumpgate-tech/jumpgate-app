// The update notice that sits above every screen. It appears ONLY when notices
// are on (notifyEnabled), a newer release exists, and the operator has not
// dismissed it this session. It never installs anything — it states the fact,
// links to the release page, and can be dismissed. An operator who chose "don't
// prompt me" never sees it; they check from the Settings page instead.
//
// The copy says "Version X" rather than a brand name: this same tree ships as
// both valve-node-app and Jumpgate, so a hardcoded name would be wrong in one
// of them.
import { useState } from "react";
import { useUpdate } from "../hooks/update";

export function UpdateBanner() {
  const update = useUpdate();
  // Dismiss is session-only: it clears the notice until the next launch.
  const [dismissed, setDismissed] = useState(false);

  const data = update.data;
  if (!data || !data.notifyEnabled || !data.updateAvailable || dismissed) {
    return null;
  }

  return (
    <div className="banner update-banner" role="status">
      <span className="update-banner-text">
        Version <strong>{data.latest}</strong> is available — you&apos;re on {data.current}.
      </span>
      <span className="update-banner-actions">
        {data.releaseUrl && (
          <a className="btn btn-primary btn-tiny" href={data.releaseUrl} target="_blank" rel="noopener noreferrer">
            View release
          </a>
        )}
        <button className="btn btn-ghost btn-tiny" type="button" onClick={() => setDismissed(true)}>
          Dismiss
        </button>
      </span>
    </div>
  );
}
