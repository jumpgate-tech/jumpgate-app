// The update notice that sits above every screen. It appears ONLY when a newer
// release exists and the operator has not skipped or dismissed it. It never
// installs anything — it states the fact, links to the release page, and lets
// the operator skip the version or dismiss it for this session.
//
// The copy says "Version X" rather than a brand name: this same tree ships as
// both valve-node-app and Jumpgate, so a hardcoded name would be wrong in one
// of them.
import { useState } from "react";
import { useUpdate, useSkipUpdate } from "../hooks/update";

export function UpdateBanner() {
  const update = useUpdate();
  const skip = useSkipUpdate();
  // Dismiss is session-only: it clears the notice until the next launch,
  // without recording a skip. Skipping is the durable choice; dismissing is
  // "not now".
  const [dismissed, setDismissed] = useState(false);

  const data = update.data;
  if (!data || !data.updateAvailable || dismissed) {
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
        <button
          className="btn btn-ghost btn-tiny"
          type="button"
          disabled={skip.isPending}
          onClick={() => skip.mutate(data.latest)}
        >
          Skip this version
        </button>
        <button className="btn btn-ghost btn-tiny" type="button" onClick={() => setDismissed(true)}>
          Dismiss
        </button>
      </span>
    </div>
  );
}
