// addOptions renders the ways to add a machine as options that each carry
// their own availability and the reason behind it. Port of targets.ts's
// addOptions().
//
// This used to be two bare buttons whose caveat was repeated in a paragraph,
// a tooltip, and a confirm() dialog — so the only place the constraint was
// stated up front was a modal that interrupted you to say what the screen
// already said, after you had committed to the action. Availability belongs
// on the option itself, visible before you choose.
import type { ReactNode } from "react";
import { Badge } from "../../components/Badge";

export function AddOptions({
  localViable,
  hasLocal,
  hostOS,
  showSSHForm,
  onToggleSSH,
  onAddLocal,
}: {
  localViable: boolean;
  hasLocal: boolean;
  hostOS: string;
  showSSHForm: boolean;
  onToggleSSH: () => void;
  onAddLocal: () => void;
}) {
  const ssh = (
    <div className="card" key="ssh">
      <h3>
        A server over SSH <Badge text="Available" kind="ok" />
      </h3>
      <p className="muted small">
        Run a node on a remote Linux server.
        {!localViable && " The only option that can finish setup from here."}
      </p>
      <div className="card-actions">
        <button className={`btn${localViable ? " btn-ghost" : ""}`} type="button" onClick={onToggleSSH}>
          {showSSHForm ? "Cancel" : "Add a server"}
        </button>
      </div>
    </div>
  );

  const local: ReactNode = localViable ? (
    <div className="card" key="local">
      <h3>
        This machine <Badge text="Available" kind="ok" />
      </h3>
      <p className="muted small">Run a node here, on the Linux host valve-node-app is running on.</p>
      <div className="card-actions">
        <button className="btn" type="button" onClick={onAddLocal}>
          Add this machine
        </button>
      </div>
    </div>
  ) : (
    <div className="card card-warn" key="local">
      <h3>
        This machine{hostOS ? ` (${hostOS})` : ""} <Badge text="Can't run a node" kind="warn" />
      </h3>
      <p className="muted small">
        Setup installs systemd units, uses apt, and needs root, so it only completes on a Linux
        host. valve-node-app runs here as your <strong>controller</strong>, driving nodes on other
        machines.
      </p>
      <div className="card-actions">
        <button className="btn btn-ghost" type="button" onClick={onAddLocal}>
          Add anyway — preview the wizard
        </button>
      </div>
    </div>
  );

  // This machine can only be added once, so stop offering it once it is in
  // the list — an option that can only fail is worse than no option.
  if (hasLocal) return <div className="card-grid card-grid-wide">{ssh}</div>;
  // Lead with whichever option can actually complete setup.
  return <div className="card-grid card-grid-wide">{localViable ? [local, ssh] : [ssh, local]}</div>;
}
