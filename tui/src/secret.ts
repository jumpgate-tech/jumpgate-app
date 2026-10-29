// Secret handling. A secret is a Uint8Array that is zeroed once its request
// settles; redact() runs on every message before it reaches the screen.

/** withSecret runs fn with buf and zeroes buf afterwards, whatever happens. */
export async function withSecret<T>(buf: Uint8Array, fn: (b: Uint8Array) => Promise<T>): Promise<T> {
  try {
    return await fn(buf);
  } finally {
    buf.fill(0);
  }
}

const RULES: [RegExp, string][] = [
  [/\b(Bearer)\s+[A-Za-z0-9._~+/=-]+/gi, "$1 ***"],
  [/([?&;](?:token|code|key|apikey|api_key|secret|password)=)[^&\s#]*/gi, "$1***"],
  [/(\b[a-z][a-z0-9+.-]*:\/\/)[^\s/@]+@/gi, "$1***@"],
  [/\b(?:0x)?[0-9a-fA-F]{64,}\b/g, "***"],
];

/** redact strips tokens, query secrets and URL userinfo from s. */
export function redact(s: string): string {
  for (const [re, to] of RULES) s = s.replace(re, to);
  return s;
}
