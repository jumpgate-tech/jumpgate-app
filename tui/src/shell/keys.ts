export interface Key {
  /** "a", "return", "escape", "tab", "up", "backspace", "." ... */
  name: string;
  /** The raw characters, for text entry. */
  seq: string;
  ctrl: boolean;
  shift: boolean;
  meta: boolean;
}

export const key = (name: string, o: Partial<Key> = {}): Key => ({
  name,
  seq: o.seq ?? (name.length === 1 ? name : ""),
  ctrl: false,
  shift: false,
  meta: false,
  ...o,
});

/** printable is the text a key types, or "" for a control key. */
export function printable(k: Key): string {
  if (k.ctrl || k.meta) return "";
  if (k.name === "space") return " ";
  const s = k.seq;
  if (!s || /[\x00-\x1f\x7f]/.test(s)) return "";
  return s;
}
