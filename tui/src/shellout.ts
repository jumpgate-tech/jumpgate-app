// runJumpgate: actions the API lacks run the jumpgate CLI. In the foreground
// the renderer is suspended and the child owns the terminal (it reads its own
// passphrases); the argv is fixed by the caller and never holds a secret.
import { closeSync, openSync } from "node:fs";

type Env = Record<string, string | undefined>;

export interface Suspendable {
  suspend(): Promise<void> | void;
  resume(): Promise<void> | void;
}

export interface RunOpts {
  renderer?: Suspendable;
  env?: Env;
  /** Start in the background, with stdout/stderr appended to log (0600). */
  detach?: { log: string; cwd?: string };
  /** After the child exits, wait for Enter before the dashboard redraws (it may have printed a one-time secret). */
  pause?: boolean;
}

export const jumpgateBin = (env: Env = process.env) => env.JUMPGATE_BIN || "jumpgate";

export async function runJumpgate(args: string[], o: RunOpts = {}): Promise<number> {
  const env = o.env ?? process.env;
  const argv = [jumpgateBin(env), ...args];
  if (o.detach) {
    const fd = openSync(o.detach.log, "a", 0o600);
    try {
      const p = Bun.spawn(argv, { stdio: ["ignore", fd, fd], cwd: o.detach.cwd, detached: true, env: { ...process.env, ...env } });
      p.unref();
      return 0;
    } catch {
      return 127;
    } finally {
      closeSync(fd);
    }
  }
  await o.renderer?.suspend();
  try {
    const p = Bun.spawn(argv, { stdio: ["inherit", "inherit", "inherit"], env: { ...process.env, ...env } });
    const code = await p.exited;
    if (o.pause) {
      const w = Bun.spawn(["sh", "-c", "printf '\\nCopy what you need now, then press Enter to return to the dashboard. '; read _"], { stdio: ["inherit", "inherit", "inherit"] });
      await w.exited;
    }
    return code;
  } catch {
    return 127;
  } finally {
    await o.renderer?.resume();
  }
}
