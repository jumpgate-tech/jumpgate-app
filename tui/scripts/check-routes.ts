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

export function check(used: readonly string[], have: Set<string>) {
  const missing = used.filter((r) => !have.has(r));
  const unused = [...have].filter((r) => !used.includes(r)).sort();
  return { missing, unused };
}

if (import.meta.main) {
  const strict = Bun.argv.includes("--strict");
  const dir = join(import.meta.dir, "..", "..", "internal", "server");
  const have = goRoutes(dir);
  if (have.size === 0) {
    console.error(`check-routes: no HandleFunc patterns found under ${dir}`);
    process.exit(1);
  }
  const { missing, unused } = check(ROUTES, have);
  for (const r of missing) console.error(`check-routes: ${r} is not a Go route`);
  if (unused.length) console.warn(`check-routes: ${unused.length} Go routes are not used by the dashboard yet`);
  if (strict) for (const r of unused) console.error(`  unused: ${r}`);
  process.exit(missing.length || (strict && unused.length) ? 1 : 0);
}
