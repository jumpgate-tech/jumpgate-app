// The panel's dialogs, each composing the <Modal> shell. They replace the
// imperative openModal/closeModal/appendModalError dance panel.ts ran: a save
// that fails throws, and the dialog shows the reason inline (the old
// appendModalError) while staying open next to the input the operator can fix;
// a save that succeeds resolves, and the parent unmounts the dialog.
import { useEffect, useRef, useState } from "react";
import { Modal } from "./Modal";

const URL_SCHEME = /^(https?|wss?):\/\//i;
const SCHEME_HINT = "It needs a scheme eRPC can dial: http://, https://, ws:// or wss://.";

// validateUrlScheme is the shared guard for the add-endpoint and edit-address
// inputs — returns an error string, or null when the URL carries a scheme eRPC
// can dial.
export function validateUrlScheme(url: string): string | null {
  return URL_SCHEME.test(url) ? null : SCHEME_HINT;
}

// ConfirmDialog is the remove-network / remove-endpoint / wipe gate.
export function ConfirmDialog({
  title,
  body,
  confirmLabel,
  danger,
  onConfirm,
  onCancel,
}: {
  title: string;
  body: string;
  confirmLabel: string;
  danger?: boolean;
  onConfirm: () => void;
  onCancel: () => void;
}) {
  return (
    <Modal onClose={onCancel} label={title}>
      <h2>{title}</h2>
      <p>{body}</p>
      <div className="modal-actions">
        <button className="btn btn-ghost" onClick={onCancel}>
          Cancel
        </button>
        <button className={danger ? "btn btn-danger" : "btn"} onClick={onConfirm}>
          {confirmLabel}
        </button>
      </div>
    </Modal>
  );
}

// TextInputDialog backs rename-endpoint, edit-address and add-endpoint. onSave
// throws to surface an inline error and keep the dialog open; it resolves once
// the config write lands (the parent then closes this dialog and provisions).
export function TextInputDialog({
  title,
  hint,
  label,
  initialValue,
  placeholder,
  saveLabel,
  validate,
  onSave,
  onCancel,
}: {
  title: string;
  hint?: string;
  label: string;
  initialValue: string;
  placeholder?: string;
  saveLabel: string;
  validate?: (value: string) => string | null;
  onSave: (value: string) => Promise<void>;
  onCancel: () => void;
}) {
  const [value, setValue] = useState(initialValue);
  const [err, setErr] = useState<string | null>(null);
  const [saving, setSaving] = useState(false);
  const inputRef = useRef<HTMLInputElement>(null);

  useEffect(() => {
    inputRef.current?.focus();
    inputRef.current?.select();
  }, []);

  async function submit() {
    const v = value.trim();
    const verr = validate?.(v) ?? null;
    if (verr) {
      setErr(verr);
      return;
    }
    setSaving(true);
    setErr(null);
    try {
      await onSave(v);
    } catch (e) {
      setErr(e instanceof Error ? e.message : String(e));
      setSaving(false);
    }
  }

  return (
    <Modal onClose={onCancel} label={title}>
      <h2>{title}</h2>
      {hint ? <p className="muted small">{hint}</p> : null}
      <label>
        {label}
        <input
          ref={inputRef}
          type="text"
          autoComplete="off"
          spellCheck={false}
          value={value}
          placeholder={placeholder}
          disabled={saving}
          onChange={(e) => setValue(e.target.value)}
          onKeyDown={(e) => {
            if (e.key === "Enter") void submit();
          }}
        />
      </label>
      {err ? <p className="error small">{err}</p> : null}
      <div className="modal-actions">
        <button className="btn btn-ghost" onClick={onCancel} disabled={saving}>
          Cancel
        </button>
        <button className="btn" onClick={() => void submit()} disabled={saving}>
          {saving ? "Saving…" : saveLabel}
        </button>
      </div>
    </Modal>
  );
}

