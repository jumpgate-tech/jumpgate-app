// shq quotes s for a POSIX shell: a plain word stays bare, anything else is
// single-quoted with each ' written as '"'"'. Every value from the server that
// goes into a printed command passes through it.
export function shq(s: string): string {
  if (/^[A-Za-z0-9._@%+=:,/-]+$/.test(s)) return s;
  return `'${s.replaceAll("'", `'"'"'`)}'`;
}
