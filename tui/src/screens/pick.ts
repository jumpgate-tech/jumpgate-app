// A plain list picker: up/down over items, ">" marks the selected one.
import type { Key } from "../shell/keys.ts";
import { sanitizeLine } from "../text.ts";

export class Pick {
  sel = 0;
  lines(items: string[]): string[] {
    if (!items.length) return [];
    this.sel = Math.min(this.sel, items.length - 1);
    return items.map((s, i) => `${i === this.sel ? ">" : " "} ${sanitizeLine(s)}`);
  }
  key(k: Key, n: number): boolean {
    if (!n) return false;
    if (k.name === "up") return (this.sel = Math.max(0, this.sel - 1)), true;
    if (k.name === "down") return (this.sel = Math.min(n - 1, this.sel + 1)), true;
    return false;
  }
  of<T>(items: T[]): T | undefined {
    return items[Math.min(this.sel, items.length - 1)];
  }
}
