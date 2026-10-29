// Text helpers for anything a server sends: strip terminal control sequences
// (a port of internal/termsafe), clip to a width, and show unknowns fail-closed.

const isCf = /\p{Cf}/u;

function skipCSI(s: string, i: number): number {
  while (i < s.length && s.charCodeAt(i) >= 0x20 && s.charCodeAt(i) <= 0x3f) i++;
  if (i < s.length && s.charCodeAt(i) >= 0x40 && s.charCodeAt(i) <= 0x7e) i++;
  return i;
}

function skipString(s: string, i: number): number {
  while (i < s.length) {
    const c = s.charCodeAt(i);
    if (c === 0x07 || c === 0x9c) return i + 1;
    if (c === 0x1b && s[i + 1] === "\\") return i + 2;
    i++;
  }
  return i;
}

function skipEscape(s: string, i: number): number {
  if (i >= s.length) return i;
  const c = s[i];
  if (c === "[") return skipCSI(s, i + 1);
  if (c === "]" || c === "P" || c === "X" || c === "^" || c === "_") return skipString(s, i + 1);
  while (i < s.length && s.charCodeAt(i) >= 0x20 && s.charCodeAt(i) <= 0x2f) i++;
  if (i < s.length && s.charCodeAt(i) >= 0x30 && s.charCodeAt(i) <= 0x7e) i++;
  return i;
}

/** sanitize drops escape sequences and control characters, keeping \t and \n. A non-string is coerced first. */
export function sanitize(v: unknown): string {
  const s = typeof v === "string" ? v : v === undefined || v === null ? "" : String(v);
  let out = "";
  for (let i = 0; i < s.length; ) {
    const cp = s.codePointAt(i)!;
    const size = cp > 0xffff ? 2 : 1;
    if (cp === 0x1b) {
      i = skipEscape(s, i + 1);
    } else if (cp === 0x9b) {
      i = skipCSI(s, i + 1);
    } else if (cp === 0x9d || cp === 0x90 || cp === 0x98 || cp === 0x9e || cp === 0x9f) {
      i = skipString(s, i + 1);
    } else if (cp === 0x09 || cp === 0x0a) {
      out += s[i];
      i++;
    } else if (cp < 0x20 || cp === 0x7f || (cp >= 0x80 && cp <= 0x9f) || isCf.test(String.fromCodePoint(cp))) {
      i += size;
    } else {
      out += s.slice(i, i + size);
      i += size;
    }
  }
  return out;
}

/** sanitizeLine is sanitize with line breaks and tabs turned into spaces. */
export const sanitizeLine = (s: unknown) => sanitize(s).replace(/[\n\t\r]/g, " ");

/** clip cuts s to w columns (one per code point), marking a cut with "…". */
export function clip(s: string, w: number): string {
  const cps = [...s];
  if (cps.length <= w) return s;
  if (w <= 0) return "";
  return cps.slice(0, w - 1).join("") + "…";
}

/** pad right-pads s to w columns, clipping when longer. */
export const pad = (s: string, w: number) => {
  const c = clip(s, w);
  return c + " ".repeat(Math.max(0, w - [...c].length));
};

/** unknown: n/a for a missing value; never 0. */
export const na = (v: unknown, fmt: (x: never) => string = String as never): string =>
  v === undefined || v === null ? "n/a" : fmt(v as never);

/** wrap breaks s into lines of at most w columns, at spaces where it can. */
export function wrap(s: string, w: number): string[] {
  if (w <= 0) return [s];
  const out: string[] = [];
  for (const para of s.split("\n")) {
    let line = "";
    for (const word of para.split(" ")) {
      let wd = word;
      while ([...wd].length > w) {
        if (line) out.push(line), (line = "");
        out.push([...wd].slice(0, w).join(""));
        wd = [...wd].slice(w).join("");
      }
      if (!line) line = wd;
      else if ([...line].length + 1 + [...wd].length <= w) line += " " + wd;
      else out.push(line), (line = wd);
    }
    out.push(line);
  }
  return out;
}

/** stepOf is "step N/M" for a job at 0-based step i of n; a finished job's i is n. */
export const stepOf = (i: number, n: number) => `step ${Math.min(i + 1, n)}/${n}`;
