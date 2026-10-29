// request(): HTTP over the server's unix socket. Errors become ApiError with
// the server's code (internal/api/errors.go). No message ever holds the token.

export interface Conn {
  socket: string;
  /** Bearer token for a per-user server; null on the service's admin socket. */
  token: string | null;
  /** True for the jumpgate service found through system.json. */
  system: boolean;
  httpAddr?: string;
  /** The version the discovery file recorded, if any. */
  version?: string;
}

export const BASE = "http://jumpgate";

export class ApiError extends Error {
  constructor(
    message: string,
    readonly status: number,
    readonly code: string,
    readonly hint?: string,
    readonly retryAfter?: number,
  ) {
    super(message);
    this.name = "ApiError";
  }
}

export function headersFor(conn: Conn, extra?: Record<string, string>): Record<string, string> {
  const h: Record<string, string> = { ...extra };
  if (conn.token) h.authorization = `Bearer ${conn.token}`;
  return h;
}

/** open sends a request and returns the raw response; non-2xx is an ApiError. */
export async function open(
  conn: Conn,
  method: string,
  path: string,
  body?: unknown,
  signal?: AbortSignal,
  extra?: Record<string, string>,
): Promise<Response> {
  let payload: string | Uint8Array | undefined;
  const hdr: Record<string, string> = { ...extra };
  if (body instanceof Uint8Array) {
    payload = body;
    hdr["content-type"] = "application/json";
  } else if (body !== undefined) {
    payload = JSON.stringify(body);
    hdr["content-type"] = "application/json";
  }
  let res: Response;
  try {
    res = await fetch(BASE + path, { method, body: payload, headers: headersFor(conn, hdr), unix: conn.socket, signal });
  } catch (e) {
    if (signal?.aborted) throw e;
    throw new ApiError(`cannot reach the jumpgate server at ${conn.socket}`, 0, "unreachable");
  }
  if (!res.ok) throw await toError(res);
  return res;
}

export async function toError(res: Response): Promise<ApiError> {
  let msg = res.statusText || `HTTP ${res.status}`;
  let code = "";
  let hint: string | undefined;
  try {
    const j = (await res.json()) as { error?: string; code?: string; hint?: string };
    if (typeof j.error === "string") msg = j.error;
    if (typeof j.code === "string") code = j.code;
    if (typeof j.hint === "string" && j.hint) hint = j.hint;
  } catch {
    /* not JSON */
  }
  const ra = Number(res.headers.get("retry-after"));
  return new ApiError(msg, res.status, code || `http_${res.status}`, hint, Number.isFinite(ra) && ra > 0 ? ra : undefined);
}

/** request sends one call and decodes the JSON reply (undefined when empty). */
export async function request<T = unknown>(
  conn: Conn,
  method: string,
  path: string,
  body?: unknown,
  signal?: AbortSignal,
): Promise<T> {
  const res = await open(conn, method, path, body, signal);
  const text = await res.text();
  if (!text) return undefined as T;
  return JSON.parse(text) as T;
}
