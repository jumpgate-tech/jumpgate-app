// A small modal shell — the React replacement for ui.ts's openModal for this
// screen. It portals to document.body (so it escapes .p-wrap and themes with
// the app's own :root tokens, exactly as the old settings sheet did), closes on
// Escape and on a backdrop click, and renders whatever dialog body the caller
// passes as children. The individual dialogs (confirm, text-input, picker,
// settings) live alongside it and compose this shell.
//
// Accessibility: it moves focus into the dialog on open (so a keyboard/SR user
// lands inside, including on a destructive confirm), traps Tab within it, and
// restores focus to the previously-focused element on close. Pass `label` for
// an accessible name.
import { useEffect, useRef, type ReactNode } from "react";
import { createPortal } from "react-dom";

const FOCUSABLE =
  'a[href],button:not([disabled]),input:not([disabled]),select:not([disabled]),textarea:not([disabled]),[tabindex]:not([tabindex="-1"])';

// bare skips the default .modal card chrome, for a dialog body that IS its own
// card (the add-network picker carries panel-language styling of its own and
// would otherwise be double-boxed).
export function Modal({
  onClose,
  children,
  bare,
  label,
}: {
  onClose: () => void;
  children: ReactNode;
  bare?: boolean;
  label?: string;
}) {
  const dialogRef = useRef<HTMLDivElement>(null);

  useEffect(() => {
    const dialog = dialogRef.current;
    const prevFocus = document.activeElement as HTMLElement | null;
    const focusables = () => (dialog ? Array.from(dialog.querySelectorAll<HTMLElement>(FOCUSABLE)) : []);

    // Land focus inside the dialog on open (the first control, else the dialog).
    (focusables()[0] ?? dialog)?.focus();

    const onKey = (e: KeyboardEvent) => {
      if (e.key === "Escape") {
        onClose();
        return;
      }
      if (e.key !== "Tab") return;
      const els = focusables();
      if (els.length === 0) {
        e.preventDefault();
        return;
      }
      const first = els[0];
      const last = els[els.length - 1];
      if (e.shiftKey && document.activeElement === first) {
        e.preventDefault();
        last.focus();
      } else if (!e.shiftKey && document.activeElement === last) {
        e.preventDefault();
        first.focus();
      }
    };
    document.addEventListener("keydown", onKey);
    return () => {
      document.removeEventListener("keydown", onKey);
      // Restore focus to whatever opened the dialog, so keyboard context isn't lost.
      prevFocus?.focus?.();
    };
  }, [onClose]);

  return createPortal(
    <div className="modal-overlay" onClick={onClose}>
      <div
        ref={dialogRef}
        className={bare ? undefined : "modal"}
        role="dialog"
        aria-modal="true"
        aria-label={label}
        tabIndex={-1}
        onClick={(e) => e.stopPropagation()}
      >
        {children}
      </div>
    </div>,
    document.body,
  );
}
