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
