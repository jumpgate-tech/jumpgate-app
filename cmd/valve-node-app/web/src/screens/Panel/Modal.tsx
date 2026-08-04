// A small modal shell — the React replacement for ui.ts's openModal for this
// screen. It portals to document.body (so it escapes .p-wrap and themes with
// the app's own :root tokens, exactly as the old settings sheet did), closes on
// Escape and on a backdrop click, and renders whatever dialog body the caller
// passes as children. The individual dialogs (confirm, text-input, picker,
// settings) live alongside it and compose this shell.
import { useEffect, type ReactNode } from "react";
import { createPortal } from "react-dom";

// bare skips the default .modal card chrome, for a dialog body that IS its own
// card (the add-network picker carries panel-language styling of its own and
// would otherwise be double-boxed).
export function Modal({ onClose, children, bare }: { onClose: () => void; children: ReactNode; bare?: boolean }) {
  useEffect(() => {
    const onKey = (e: KeyboardEvent) => {
      if (e.key === "Escape") onClose();
    };
    document.addEventListener("keydown", onKey);
    return () => document.removeEventListener("keydown", onKey);
  }, [onClose]);

  return createPortal(
    <div className="modal-overlay" onClick={onClose}>
      {bare ? (
        <div role="dialog" aria-modal="true" onClick={(e) => e.stopPropagation()}>
          {children}
        </div>
      ) : (
        <div className="modal" role="dialog" aria-modal="true" onClick={(e) => e.stopPropagation()}>
          {children}
        </div>
      )}
    </div>,
    document.body,
  );
}
