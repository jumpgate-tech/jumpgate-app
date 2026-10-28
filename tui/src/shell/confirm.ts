// Modals: confirm (y/n), askSecret (masked, bytes only) and showOnce.
import { clip, sanitizeLine } from "../text.ts";
import { printable, type Key } from "./keys.ts";

export interface Modal {
  title: string;
  lines(w: number): string[];
  key(k: Key): void;
  /** True while a text entry has focus: global keys such as "." and "q" type instead. */
  readonly takesText: boolean;
}

export function confirmModal(text: string, done: (ok: boolean) => void): Modal {
  return {
    title: "confirm",
    takesText: false,
    lines: (w) => [...sanitizeLine(text).match(new RegExp(`.{1,${Math.max(10, w)}}`, "g"))!, "", "[y] yes   [n] no"],
    key(k) {
      if (k.name === "y" || k.name === "return") done(true);
      else if (k.name === "n" || k.name === "escape" || k.name === "q") done(false);
    },
  };
}

/**
 * SecretModal reads a passphrase as UTF-8 bytes, never as a string. Each
 * character is drawn as "*". On Enter it hands over a copy and zeroes its own
 * buffer; on Escape it zeroes and hands over null.
 */
export class SecretModal implements Modal {
  readonly title = "secret";
  readonly takesText = true;
  private buf = new Uint8Array(64);
  private len = 0;
  private sizes: number[] = [];
  private closed = false;

  constructor(
    private readonly prompt: string,
    private readonly done: (v: Uint8Array | null) => void,
  ) {}

  /** value is what an input would show: always "". */
  get value(): string {
    return "";
  }
  get chars(): number {
    return this.sizes.length;
  }

  private append(bytes: Uint8Array) {
    if (this.len + bytes.length > this.buf.length) {
      const next = new Uint8Array(Math.max(this.buf.length * 2, this.len + bytes.length));
      next.set(this.buf.subarray(0, this.len));
      this.buf.fill(0);
      this.buf = next;
    }
    this.buf.set(bytes, this.len);
    this.len += bytes.length;
    bytes.fill(0);
  }

  private wipe() {
    this.buf.fill(0);
    this.len = 0;
    this.sizes = [];
  }

  lines(): string[] {
    return [`${this.prompt}: ${"*".repeat(this.chars)}`, "", "[enter] send   [esc] cancel"];
  }

  key(k: Key) {
    if (this.closed) return;
    if (k.name === "return") {
      const out = this.buf.slice(0, this.len);
      this.wipe();
      this.closed = true;
      this.done(out);
    } else if (k.name === "escape" || (k.ctrl && k.name === "c")) {
      this.wipe();
      this.closed = true;
      this.done(null);
    } else if (k.name === "backspace") {
      const n = this.sizes.pop();
      if (n) {
        this.len -= n;
        this.buf.fill(0, this.len, this.len + n);
      }
    } else {
      const t = printable(k);
      if (!t) return;
      for (const ch of t) {
        const b = new TextEncoder().encode(ch);
        this.sizes.push(b.length);
        this.append(b);
      }
    }
  }
}

/** ShowOnce draws a one-time secret until any key, then drops it. */
export class ShowOnce implements Modal {
  readonly takesText = false;
  private value: string | null;
  constructor(
    readonly title: string,
    value: string,
    private readonly done: () => void,
  ) {
    this.value = value;
  }
  lines(w: number): string[] {
    const v = this.value ?? "";
    return [...sanitizeLine(v).match(new RegExp(`.{1,${Math.max(10, w)}}`, "g")) ?? [""], "", "Copy it now; it is shown once. [any key] close"].map((l) =>
      clip(l, w),
    );
  }
  key() {
    this.value = null;
    this.done();
  }
}

export function helpModal(lines: string[], done: () => void): Modal {
  return { title: "keys", takesText: false, lines: () => lines, key: () => done() };
}
