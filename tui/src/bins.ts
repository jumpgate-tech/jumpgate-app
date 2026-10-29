// resolveBin: find a program the TUI runs (jumpgate, ssh) once, at startup, as
// an absolute path. Order: $<envVar> (must be absolute), a sibling of the
// running executable, then PATH with its relative entries dropped. A relative
// or empty result is refused, so a planted ./ssh in the working directory is
// never run.
import { accessSync, constants, statSync } from "node:fs";
import { delimiter, dirname, isAbsolute, join } from "node:path";

type Env = Record<string, string | undefined>;

export interface ResolveOpts {
  env?: Env;
  execPath?: string;
  platform?: string;
}

function executable(p: string): boolean {
  try {
    return statSync(p).isFile() && (accessSync(p, constants.X_OK), true);
  } catch {
    return false;
  }
}

export function resolveBin(name: string, envVar: string, o: ResolveOpts = {}): string | null {
  const env = o.env ?? process.env;
  const exe = (o.platform ?? process.platform) === "win32" ? `${name}.exe` : name;
  const fromEnv = env[envVar];
  if (fromEnv !== undefined && fromEnv !== "") return isAbsolute(fromEnv) && executable(fromEnv) ? fromEnv : null;
  const sibling = join(dirname(o.execPath ?? process.execPath), exe);
  if (isAbsolute(sibling) && executable(sibling)) return sibling;
  return onPath(exe, env);
}

/** onPath finds exe on PATH with its relative entries dropped; absolute or null. */
export function onPath(exe: string, env: Env = process.env): string | null {
  const path = (env.PATH ?? "").split(delimiter).filter((d) => d !== "" && isAbsolute(d)).join(delimiter);
  if (!path) return null;
  const found = Bun.which(exe, { PATH: path });
  return found && isAbsolute(found) && executable(found) ? found : null;
}

export interface Bins {
  jumpgate: string | null;
  ssh: string | null;
}

export const resolveBins = (o: ResolveOpts = {}): Bins => ({
  jumpgate: resolveBin("jumpgate", "JUMPGATE_BIN", o),
  ssh: resolveBin("ssh", "JUMPGATE_SSH", o),
});
