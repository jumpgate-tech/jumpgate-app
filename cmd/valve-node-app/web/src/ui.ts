// Small shared pure formatting/utility helpers used across screens. The DOM
// rendering side of this module (shell, modal, dropdown, badge/dot/footer
// string templates) was retired once every screen finished its React
// migration — those are now components (see src/components and each
// screens/*/*.tsx) rather than string-template helpers.

export const LEARN_ROOT = "https://learn.valve.city/rpc";

// fmtInt formats a number with thousands separators for readability
// (block/slot numbers get large fast).
export function fmtInt(n: number): string {
  return Number.isFinite(n) ? n.toLocaleString("en-US") : "—";
}

export function fmtPct(n: number): string {
  return Number.isFinite(n) ? `${n.toFixed(1)}%` : "—";
}

// fmtDuration renders a duration given in seconds as a short human string
// ("~2h 14m", "~45s"). Returns "—" for non-finite or negative input.
export function fmtDuration(seconds: number): string {
  if (!Number.isFinite(seconds) || seconds < 0) return "—";
  if (seconds < 60) return `~${Math.round(seconds)}s`;
  const totalMinutes = Math.round(seconds / 60);
  const hours = Math.floor(totalMinutes / 60);
  const minutes = totalMinutes % 60;
  if (hours === 0) return `~${minutes}m`;
  if (hours < 48) return `~${hours}h ${minutes}m`;
  const days = Math.floor(hours / 24);
  const remHours = hours % 24;
  return `~${days}d ${remHours}h`;
}

const BYTE_UNITS = ["B", "KB", "MB", "GB", "TB", "PB"];

// fmtBytes renders a byte count as a human-readable size ("3.9 TB", "512 MB").
export function fmtBytes(n: number): string {
  if (!Number.isFinite(n) || n < 0) return "—";
  if (n === 0) return "0 B";
  let value = n;
  let unit = 0;
  while (value >= 1024 && unit < BYTE_UNITS.length - 1) {
    value /= 1024;
    unit++;
  }
  const digits = value < 10 ? 2 : value < 100 ? 1 : 0;
  return `${value.toFixed(digits)} ${BYTE_UNITS[unit]}`;
}

// copyToClipboard writes text to the clipboard, returning whether it
// succeeded (the Clipboard API can be unavailable — insecure context, denied
// permission — and callers should show a fallback message rather than throw).
export async function copyToClipboard(text: string): Promise<boolean> {
  try {
    await navigator.clipboard.writeText(text);
    return true;
  } catch {
    return false;
  }
}
