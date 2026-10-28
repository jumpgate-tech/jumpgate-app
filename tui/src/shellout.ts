// runJumpgate: actions the API lacks run the jumpgate CLI. In the foreground
// the renderer is suspended and the child owns the terminal (it reads its own
// passphrases); the argv is fixed by the caller and never holds a secret.
import { closeSync, openSync } from "node:fs";
import { isAbsolute } from "node:path";

type Env = Record<string, string | undefined>;

export interface Suspendable {
  suspend(): Promise<void> | void;
  resume(): Promise<void> | void;
}

export interface RunOpts {
  /** The absolute path of jumpgate, from resolveBins at startup. */
  bin?: string | null;
  renderer?: Suspendable;
  env?: Env;
  /** Start in the background, with stdout/stderr appended to log (0600). */
  detach?: { log: string; cwd?: string };
  /** After the child exits, wait for Enter before the dashboard redraws (it may have printed a one-time secret). */
  pause?: boolean;
  /** readEnter replaces the in-process wait for Enter (tests). */
  readEnter?: () => Promise<void>;
}

/** NOT_FOUND is the exit code when the program is missing or not absolute. */
export const NOT_FOUND = 127;

export async function runJumpgate(args: string[], o: RunOpts = {}): Promise<number> {
  if (!o.bin || !isAbsolute(o.bin)) return NOT_FOUND;
  return runArgv([o.bin, ...args], o);
}

/** readEnter prints a note and waits, in this process, for a line on stdin. */
export function readEnter(): Promise<void> {
  process.stdout.write("\nCopy what you need now, then press Enter to return to the dashboard. ");
  return new Promise((resolve) => {
    const on = (b: Buffer) => {
      if (!b.includes(0x0a) && !b.includes(0x0d)) return;
      process.stdin.off("data", on);
      process.stdin.pause();
      resolve();
    };
    process.stdin.on("data", on);
    process.stdin.resume();
  });
}

/** runArgv runs argv as runJumpgate does (an ssh argv the server built, for one). */
export async function runArgv(argv: string[], o: RunOpts = {}): Promise<number> {
  const env = o.env ?? process.env;
  if (!argv[0] || !isAbsolute(argv[0])) return NOT_FOUND;
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
    if (o.pause) await (o.readEnter ?? readEnter)();
    return code;
  } catch {
    return 127;
  } finally {
    await o.renderer?.resume();
  }
}
