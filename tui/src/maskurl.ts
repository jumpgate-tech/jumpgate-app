// maskURL: a port of internal/tui/maskurl.go. Userinfo, odd hosts, query
// values, fragments and key-like path segments become ***.

function leadingScheme(s: string): string {
  if (s.startsWith("//")) return "//";
  if (!/^[a-zA-Z]/.test(s)) return "";
  const m = /^[a-zA-Z][a-zA-Z0-9+.-]*:\/\//.exec(s);
  return m ? m[0] : "";
}

function plainAuthority(a: string): boolean {
  let host = a;
  let port = "";
  if (a.startsWith("[")) {
    const end = a.indexOf("]");
    if (end < 2 || !/^[0-9a-fA-F:.]+$/.test(a.slice(1, end))) return false;
    host = "";
    port = a.slice(end + 1);
  } else {
    const c = a.indexOf(":");
    if (c >= 0) (host = a.slice(0, c)), (port = a.slice(c));
  }
  if (!/^[a-zA-Z0-9.-]*$/.test(host)) return false;
  return port === "" || /^:[0-9]+$/.test(port);
}

function plainSegment(seg: string): boolean {
  if (seg === "") return true;
  if (!/^[a-z0-9]+$/.test(seg)) return false;
  const lower = (seg.match(/[a-z]/g) ?? []).length;
  const digits = seg.length - lower;
  if (digits === 0) return lower <= 8;
  if (lower === 0) return digits <= 10;
  return seg[0] === "v" && lower === 1 && digits <= 2;
}

function keyName(seg: string): boolean {
  if (seg.length > 12) return false;
  return ["key", "apikey", "token", "auth", "secret", "accesstoken"].includes(seg.replace(/[-_]/g, "").toLowerCase());
}

const plainKey = (k: string) => /^[a-zA-Z_-]{1,16}$/.test(k);

export function maskURL(s: string): string {
  const scheme = leadingScheme(s);
  const rest = s.slice(scheme.length);
  let end = rest.indexOf("/");
  if (end < 0) end = rest.length;
  let auth = rest.slice(0, end);
  let tail = rest.slice(end);
  if (!plainAuthority(auth)) auth = "***";
  let frag = "";
  const hi = tail.indexOf("#");
  if (hi >= 0) (tail = tail.slice(0, hi)), (frag = "#***");
  let query = "";
  const qi = tail.indexOf("?");
  if (qi >= 0) {
    const pairs = tail
      .slice(qi + 1)
      .split(/[&;]/)
      .filter(Boolean)
      .map((p) => {
        const k = p.indexOf("=");
        if (k < 0) return "***";
        const key = p.slice(0, k);
        return `${plainKey(key) ? key : "***"}=***`;
      });
    tail = tail.slice(0, qi);
    query = "?" + pairs.join("&");
  }
  const segs = tail.split("/");
  const out = segs.map((seg, i) => {
    const semi = seg.indexOf(";");
    let head = semi >= 0 ? seg.slice(0, semi) : seg;
    const params = semi >= 0 ? seg.slice(semi + 1) : "";
    if (!plainSegment(head) || (i > 0 && keyName(segs[i - 1]!))) head = "***";
    if (semi >= 0 && params !== "") head += ";***";
    return head;
  });
  return scheme + auth + out.join("/") + query + frag;
}

export const MAX_MASK_TEXT = 4096; // bytes of UTF-8, as in Go

const SPACE = " \t\n\r\v\f";
const isSpace = (c: string) => SPACE.includes(c);
const asciiOnly = (t: string) => /^[\x00-\x7f]*$/.test(t);

/**
 * maskText: a port of maskText in internal/tui/maskurl.go. It masks every
 * word that could carry a credential (URLs, user:pass, @ and % forms, non-
 * ASCII look-alikes) and the words around them, failing closed.
 */
export function maskText(s: string): string {
  const bytes = new TextEncoder().encode(s);
  const cut = bytes.length > MAX_MASK_TEXT;
  if (cut) {
    let n = MAX_MASK_TEXT;
    while (n > 0 && (bytes[n]! & 0xc0) === 0x80) n--;
    s = new TextDecoder().decode(bytes.subarray(0, n));
  }
  const parts: string[] = [];
  const words: number[] = [];
  let start = 0;
  for (let i = 0; i <= s.length; i++) {
    if (i === s.length || (i > start && isSpace(s[i]!) !== isSpace(s[start]!))) {
      if (i > start) {
        if (!isSpace(s[start]!)) words.push(parts.length);
        parts.push(s.slice(start, i));
      }
      start = i;
    }
  }
  const mask = new Array<boolean>(words.length).fill(false);
  let open = -1;
  words.forEach((p, w) => {
    if (p > 0 && !isSpace(parts[p]![0]!) && /[\n\r]/.test(parts[p - 1]!)) open = -1;
    const t = parts[p]!;
    const ascii = asciiOnly(t);
    mask[w] = mask[w] || !ascii || /[@:/%\\=?#&]/.test(t);
    const at = !ascii || /[@%]/.test(t);
    if (at) for (let k = Math.max(w - 2, 0); k < w; k++) mask[k] = true;
    if (at && open >= 0) {
      for (let k = open; k < w; k++) mask[k] = true;
      open = -1;
    }
    if (t.includes("://")) {
      if (w + 1 < words.length) mask[w + 1] = true;
      if (open < 0) open = w + 1;
    }
  });
  if (cut) {
    if (open >= 0) for (let k = open; k < words.length; k++) mask[k] = true;
    for (let k = Math.max(words.length - 2, 0); k < words.length; k++) mask[k] = true;
  }
  words.forEach((p, w) => {
    if (mask[w]) parts[p] = "***";
  });
  return parts.join("") + (cut ? "…" : "");
}
