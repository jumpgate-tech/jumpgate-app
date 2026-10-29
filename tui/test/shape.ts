// assertShape checks a fixture against a sample typed as the TS interface:
// every key the sample has must be in the fixture with the same primitive
// type. Fixture keys the sample lacks are ignored (the TUI does not read
// them). Arrays compare their first elements; a null in the fixture matches
// an object sample (Go nil pointers).
export function shapeErrors(fixture: unknown, sample: unknown, path = "$"): string[] {
  const kind = (v: unknown) => (v === null ? "null" : Array.isArray(v) ? "array" : typeof v);
  const fk = kind(fixture);
  const sk = kind(sample);
  if (sk === "null" || fk === "null") return [];
  if (fk !== sk) return [`${path}: fixture is ${fk}, TS sample is ${sk}`];
  if (sk === "array") {
    const f = fixture as unknown[];
    const s = sample as unknown[];
    return f.length && s.length ? shapeErrors(f[0], s[0], `${path}[0]`) : [];
  }
  if (sk !== "object") return [];
  const errs: string[] = [];
  const f = fixture as Record<string, unknown>;
  for (const [k, v] of Object.entries(sample as Record<string, unknown>)) {
    if (!(k in f)) errs.push(`${path}.${k}: missing in the Go fixture`);
    else errs.push(...shapeErrors(f[k], v, `${path}.${k}`));
  }
  return errs;
}
