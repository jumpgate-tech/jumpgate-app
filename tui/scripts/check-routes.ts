// check-routes: every ROUTES entry must match a HandleFunc pattern in
// internal/server/*.go (test files skipped). Go routes the dashboard does not
// use are printed as a warning, or fail with --strict.
import { readdirSync, readFileSync } from "node:fs";
import { join } from "node:path";
import { ROUTES } from "../src/api/routes.ts";

export function goRoutes(serverDir: string): Set<string> {
  const out = new Set<string>();
  for (const f of readdirSync(serverDir)) {
    if (!f.endsWith(".go") || f.endsWith("_test.go")) continue;
    const src = readFileSync(join(serverDir, f), "utf8");
    for (const m of src.matchAll(/HandleFunc\("([A-Z]+ [^"]+)"/g)) out.add(m[1]!);
  }
  return out;
}

export function readAllowlist(path: string): Map<string, string> {
  const out = new Map<string, string>();
  for (const raw of readFileSync(path, "utf8").split("\n")) {
    const line = raw.trim();
    if (!line || line.startsWith("#")) continue;
    const m = /^([A-Z]+ \S+)\s+(.+)$/.exec(line);
    if (!m) throw new Error(`route-allowlist: no reason on ${JSON.stringify(line)}`);
    out.set(m[1]!, m[2]!);
  }
  return out;
}

export function check(used: readonly string[], have: Set<string>, allow: Map<string, string> = new Map()) {
  const missing = used.filter((r) => !have.has(r));
  const unused = [...have].filter((r) => !used.includes(r) && !allow.has(r)).sort();
  /** stale: an allowlisted route that is gone from Go or now used by the client. */
  const stale = [...allow.keys()].filter((r) => !have.has(r) || used.includes(r));
  return { missing, unused, stale };
}

if (import.meta.main) {
  const strict = Bun.argv.includes("--strict");
  const dir = join(import.meta.dir, "..", "..", "internal", "server");
  const have = goRoutes(dir);
  if (have.size === 0) {
    console.error(`check-routes: no HandleFunc patterns found under ${dir}`);
    process.exit(1);
  }
  const { missing, unused, stale } = check(ROUTES, have, readAllowlist(join(import.meta.dir, "route-allowlist.txt")));
  for (const r of missing) console.error(`check-routes: ${r} is not a Go route`);
  for (const r of stale) console.error(`check-routes: allowlisted ${r} is gone from Go or used by the client; drop it from the allowlist`);
  if (unused.length) console.warn(`check-routes: ${unused.length} Go routes are not used by the dashboard yet`);
  if (strict) for (const r of unused) console.error(`  unused: ${r}`);
  process.exit(missing.length || stale.length || (strict && unused.length) ? 1 : 0);
}
