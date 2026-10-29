// User text says "gate", never "agent". Every string literal and template
// piece in src/ is scanned; the only allowed hits are route paths
// (/agent/...), the install script's --agent flag and the `jumpgate agent`
// CLI command, which are names the user types, not prose.
import { expect, test } from "bun:test";
import { readdirSync, readFileSync, statSync } from "node:fs";
import { join } from "node:path";
import ts from "typescript";

const ALLOWED = [/\/agent\//g, /--agent\b/g, /jumpgate agent\b/g];

function files(dir: string): string[] {
  return readdirSync(dir).flatMap((f) => {
    const p = join(dir, f);
    return statSync(p).isDirectory() ? files(p) : p.endsWith(".ts") ? [p] : [];
  });
}

export function agentHits(src: string, name = "x.ts"): string[] {
  const sf = ts.createSourceFile(name, src, ts.ScriptTarget.Latest, true);
  const hits: string[] = [];
  const visit = (n: ts.Node) => {
    let text: string | undefined;
    if (ts.isStringLiteral(n) || ts.isNoSubstitutionTemplateLiteral(n)) text = n.text;
    else if (ts.isTemplateHead(n) || ts.isTemplateMiddle(n) || ts.isTemplateTail(n)) text = n.text;
    if (text !== undefined) {
      // a property name such as `agent:` in an object literal type or key is not user text
      const isKey = n.parent && (ts.isPropertyAssignment(n.parent) || ts.isPropertySignature(n.parent)) && (n.parent as ts.PropertyAssignment).name === n;
      let t = text;
      for (const re of ALLOWED) t = t.replace(re, "");
      if (!isKey && /\bagents?\b/i.test(t)) hits.push(`${name}:${sf.getLineAndCharacterOfPosition(n.getStart()).line + 1}: ${text}`);
    }
    ts.forEachChild(n, visit);
  };
  visit(sf);
  return hits;
}

test("no user text in src says agent", () => {
  const dir = join(import.meta.dir, "..", "src");
  expect(files(dir).flatMap((f) => agentHits(readFileSync(f, "utf8"), f.slice(dir.length + 1)))).toEqual([]);
});

test("the scan catches prose and template pieces, and allows routes and CLI names", () => {
  expect(agentHits('const a = "restart the agent";').length).toBe(1);
  expect(agentHits("const a = `the ${x} Agent is down`;").length).toBe(1);
  expect(agentHits('const r = "POST /api/targets/{id}/agent/upgrade";')).toEqual([]);
  expect(agentHits('const k = { agent: "gate" };')).toEqual([]);
  expect(agentHits('const c = "sh -s -- --agent";')).toEqual([]);
});
