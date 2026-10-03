// The gear's settings sheet: the app-wide appearance control (System / Light /
// Dark, applied instantly and persisted via theme.ts) and — only when the
// server actually lists wipe as an available action — the destructive "Wipe
// gateway" that used to sit exposed in the power band's chip row. Picking a
// theme flips the whole app live behind the backdrop; the local pref state just
// moves the active pill.
import { useState } from "react";
import { Modal } from "./Modal";
import { Icon } from "./icons";
import { getThemePref, setThemePref, type ThemePref } from "../../theme";

const THEMES: ThemePref[] = ["system", "light", "dark"];
const cap = (s: string) => s[0].toUpperCase() + s.slice(1);

export function SettingsSheet({
  actions,
  busy,
  onWipe,
  onClose,
}: {
  actions: string[] | null | undefined;
  busy: string | null;
  onWipe: () => void;
  onClose: () => void;
}) {
  const [pref, setPref] = useState<ThemePref>(getThemePref());
  const canWipe = (actions ?? []).includes("wipe");

  function pick(p: ThemePref) {
    setThemePref(p);
    setPref(p);
  }

  return (
    <Modal onClose={onClose} label="Settings">
      <h2>Settings</h2>
      <div className="set-group">
        <div className="set-label">Appearance</div>
        <div className="theme-seg" role="group" aria-label="Appearance">
          {THEMES.map((p) => (
            <button
              key={p}
              type="button"
              className={`theme-opt${p === pref ? " active" : ""}`}
              onClick={() => pick(p)}
            >
              {cap(p)}
            </button>
          ))}
        </div>
      </div>
      {canWipe ? (
        <div className="set-group">
          <div className="set-label danger">Danger zone</div>
          <button className="btn btn-danger set-wipe" onClick={onWipe} disabled={!!busy}>
            <Icon name="trash" /> Wipe gateway
          </button>
          <p className="muted small">
            Destroys the gateway container and its stored config. Every chain it fronts stops being served
            until it comes back. Nothing behind it — no node, devnet or public endpoint — is touched.
          </p>
        </div>
      ) : null}
      <div className="modal-actions">
        <button className="btn btn-ghost" onClick={onClose}>
          Done
        </button>
      </div>
    </Modal>
  );
}
