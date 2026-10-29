// discover(): find the running server the way internal/daemon.Find does. The
// service's system.json comes first (no token, ever); then the user's
// ~/.jumpgate/run/server.json, which must be 0600 and owned by this user.
import { readFileSync, statSync } from "node:fs";
import { homedir } from "node:os";
import { join } from "node:path";
import { request, type Conn } from "./http.ts";

export type DiscoverKind = "not_admin" | "not_running" | "insecure" | "bad_file";

export class DiscoverError extends Error {
  constructor(readonly kind: DiscoverKind, message: string) {
    super(message);
    this.name = "DiscoverError";
  }
}

export const NOT_ADMIN =
  "a jumpgate service runs on this machine, and you are not in jumpgate-admin; ask an administrator to run `sudo usermod -aG jumpgate-admin $USER`, then log in again";

type Env = Record<string, string | undefined>;

function errno(e: unknown): string | undefined {
  return (e as { code?: string })?.code;
}

async function healthy(conn: Conn): Promise<boolean> {
  try {
    await request(conn, "GET", "/api/health");
    return true;
  } catch {
    return false;
  }
}

/** jumpgateHome is ~/.jumpgate, or $JUMPGATE_HOME. */
export function jumpgateHome(env: Env = process.env): string {
  return env.JUMPGATE_HOME || join(env.HOME || homedir(), ".jumpgate");
}

export async function discover(env: Env = process.env, socketOverride?: string): Promise<Conn> {
  const sysPath = join(env.JUMPGATE_SYSTEM_DIR || "/run/jumpgate-ctl", "system.json");
  let sysRaw: string | undefined;
  try {
    sysRaw = readFileSync(sysPath, "utf8");
  } catch (e) {
    if (errno(e) === "EACCES" || errno(e) === "EPERM") throw new DiscoverError("not_admin", NOT_ADMIN);
    if (errno(e) !== "ENOENT" && errno(e) !== "ENOTDIR") throw new DiscoverError("bad_file", `cannot read ${sysPath}`);
  }
  if (sysRaw !== undefined) {
    let si: { socket?: string; version?: string; httpAddr?: string; uid?: number } = {};
    try {
      si = JSON.parse(sysRaw);
    } catch {
      /* treated as absent, as daemon.FindSystem does */
    }
    const st = statSync(sysPath);
    if (si.socket && (st.uid === 0 || st.uid === si.uid)) {
      const conn: Conn = { socket: socketOverride || si.socket, token: null, system: true, version: si.version, httpAddr: si.httpAddr };
      try {
        await request(conn, "GET", "/api/health");
        return conn;
      } catch {
        throw new DiscoverError("not_running", "the jumpgate service is installed but not answering; run `sudo systemctl start jumpgate`");
      }
    }
  }

  const path = join(jumpgateHome(env), "run", "server.json");
  let raw: string;
  try {
    raw = readFileSync(path, "utf8");
  } catch (e) {
    if (errno(e) === "ENOENT" || errno(e) === "ENOTDIR") throw new DiscoverError("not_running", "No jumpgate server is running.");
    throw new DiscoverError("bad_file", `cannot read ${path}`);
  }
  const st = statSync(path);
  if ((st.mode & 0o077) !== 0 || (process.getuid && st.uid !== process.getuid())) {
    throw new DiscoverError("insecure", `${path} must be mode 0600 and owned by you; refusing to use it`);
  }
  let info: { socket?: unknown; token?: unknown; version?: unknown; httpAddr?: unknown };
  try {
    info = JSON.parse(raw);
  } catch {
    throw new DiscoverError("bad_file", `${path} is not valid JSON`);
  }
  if (typeof info.socket !== "string" || typeof info.token !== "string" || !info.token) {
    throw new DiscoverError("bad_file", `${path} has no socket or token`);
  }
  const conn: Conn = {
    socket: socketOverride || info.socket,
    token: info.token,
    system: false,
    version: typeof info.version === "string" ? info.version : undefined,
    httpAddr: typeof info.httpAddr === "string" ? info.httpAddr : undefined,
  };
  if (!(await healthy(conn))) throw new DiscoverError("not_running", "No jumpgate server is running.");
  return conn;
}
