// mouse: map a click on drawn text to what it means. Panes are plain text,
// so a "[u] unlock" button is found in the line itself.
import { key, type Key } from "./keys.ts";
import { isSelected } from "./style.ts";

/** Two clicks on one cell within this many ms are a double click. */
export const DOUBLE_MS = 400;

const NAMED: Record<string, string> = { esc: "escape", enter: "return", space: "space", any: "escape" };

/** buttonKey turns a button's label ("u", "esc", "any key") into a key, or null. */
export function buttonKey(label: string): Key | null {
  const w = label.trim().split(/\s+/)[0] ?? "";
  const named = NAMED[w.toLowerCase()];
  if (named) return key(named, { seq: named === "space" ? " " : "" });
  if ([...w].length !== 1) return null; // "1-3", "pass   ", "skew!": not a key
  const lower = w.toLowerCase();
  return key(lower, { seq: w, shift: w !== lower });
}

/**
 * buttonAt returns the key of the "[k] label" button under column col. A
 * button reaches from its "[" to the next button or a run of 3+ spaces.
 */
export function buttonAt(line: string, col: number): Key | null {
  const ms = [...line.matchAll(/\[([^\]\n]+)\]/g)];
  for (let i = 0; i < ms.length; i++) {
    const m = ms[i]!;
    const start = m.index!;
    const next = i + 1 < ms.length ? ms[i + 1]!.index! : line.length;
    const gap = line.slice(start + m[0].length, next).search(/ {3,}/);
    const end = gap < 0 ? next : start + m[0].length + gap;
    if (col >= start && col < end) return buttonKey(m[1]!);
  }
  return null;
}

/** inList: row is in the same run of list lines as the selected line sel. */
export function inList(lines: string[], sel: number, row: number): boolean {
  const isRow = (l: string | undefined) => l !== undefined && l.trim() !== "" && (isSelected(l) || l.startsWith(" "));
  if (!isRow(lines[row])) return false;
  const [a, b] = row < sel ? [row, sel] : [sel, row];
  for (let i = a; i <= b; i++) if (!isRow(lines[i])) return false;
  return true;
}

/** statusTarget: the screen a click on the status strip opens ("setup 3/6", "vault ..."). */
export function statusTarget(line: string, col: number): string | null {
  let x = 0;
  for (const part of line.split(" | ")) {
    if (col >= x && col < x + part.length) {
      const w = part.trim().split(" ")[0];
      if (w === "setup") return "setup";
      if (w === "vault") return "vault";
      if (w === "boxes") return "fleet";
      if (w === "!") return "jobs";
      return null;
    }
    x += part.length + 3;
  }
  return null;
}
