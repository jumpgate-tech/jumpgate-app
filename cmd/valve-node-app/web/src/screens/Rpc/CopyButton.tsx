// The React copy-to-clipboard button — the replacement for the legacy
// data-action="copy" delegation plus copyButton()'s "Copied!"/"Copy failed"
// flash. It keeps ui.ts's copyToClipboard (a pure helper the playbook says may
// stay), and reverts its own label after 1.5s exactly as the old one did.
import { useEffect, useRef, useState } from "react";
import { copyToClipboard } from "../../ui";

export function CopyButton({
  value,
  label = "Copy",
  className = "btn btn-ghost btn-tiny",
  title,
}: {
  value: string;
  label?: string;
  className?: string;
  title?: string;
}) {
  const [flash, setFlash] = useState<string | null>(null);
  const timer = useRef<ReturnType<typeof setTimeout> | null>(null);
  useEffect(() => () => { if (timer.current) clearTimeout(timer.current); }, []);

  async function onClick() {
    const ok = await copyToClipboard(value);
    setFlash(ok ? "Copied!" : "Copy failed");
    if (timer.current) clearTimeout(timer.current);
    timer.current = setTimeout(() => setFlash(null), 1500);
  }

  return (
    <button type="button" className={className} title={title ?? `Copy ${value}`} onClick={() => void onClick()}>
      {flash ?? label}
    </button>
  );
}
