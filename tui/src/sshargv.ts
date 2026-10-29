// checkSSHArgv: an exact port of api.CheckSSHArgv (internal/api/sshargv.go).
// The server builds the ssh argv; the TUI runs only the shape below, with the
// local ssh by name, never the server's argv[0]:
//   ssh -p PORT [-i KEY] -o StrictHostKeyChecking=yes -o UserKnownHostsFile=...
//       -o GlobalKnownHostsFile=none -o ProxyJump=none -o ProxyCommand=none
//       [-l USER] -- HOST

export const SSH_OPT_STRICT = "StrictHostKeyChecking=yes";
export const SSH_OPT_KNOWN_HOSTS_PREFIX = "UserKnownHostsFile=";
export const SSH_OPT_GLOBAL_KNOWN_HOSTS = "GlobalKnownHostsFile=none";
export const SSH_OPT_PROXY_JUMP = "ProxyJump=none";
export const SSH_OPT_PROXY_COMMAND = "ProxyCommand=none";

const base = (p: string) => p.split(/[\\/]/).pop() ?? p;

function checkSSHPath(s: string): string | null {
  if (s === "" || s[0] === "-" || s[0] === "~") return "empty, or starts with '-' or '~'";
  for (const ch of s) {
    const c = ch.codePointAt(0)!;
    if (c < 0x20 || c === 0x7f || ch === "%" || ch === "$") return "has a control character, '%' or '$'";
  }
  return null;
}

function checkKnownHostsPath(p: string): string | null {
  if (/['"]/.test(p)) return "the path has a quote or an apostrophe, which ssh would read as quoting";
  for (let i = 0; i < p.length; i++) {
    if (p[i] !== "\\") continue;
    if (i === p.length - 1 || p[i + 1] === "\\" || p[i + 1] === " ")
      return "the path has a double backslash, a backslash before a space, or a trailing backslash, which ssh would read as an escape";
  }
  return null;
}

function checkKnownHostsList(v: string): string | null {
  if (v === "") return "no known_hosts file";
  while (v !== "") {
    let tok: string;
    if (v[0] === '"') {
      const end = v.slice(1).indexOf('"');
      if (end < 0) return "known_hosts: unterminated quote";
      tok = v.slice(1, 1 + end);
      v = v.slice(end + 2);
      if (tok.includes("\t") || tok.endsWith("\\")) return "known_hosts: tab or escaped quote in a quoted path";
    } else {
      let end = v.indexOf(" ");
      if (end < 0) end = v.length;
      tok = v.slice(0, end);
      v = v.slice(end);
      if (/[\t"]/.test(tok)) return "known_hosts: whitespace or quote in a path";
    }
    if (v !== "") {
      if (v[0] !== " " || v.length === 1) return "known_hosts: malformed list";
      v = v.slice(1);
    }
    const e = checkSSHPath(tok) ?? checkKnownHostsPath(tok);
    if (e) return `known_hosts: ${e}`;
  }
  return null;
}

function plainName(s: string, host: boolean): boolean {
  if (s === "" || s[0] === "-") return false;
  for (const ch of s) {
    if (/[a-zA-Z0-9._-]/.test(ch)) continue;
    if (ch === ":" && host) continue;
    return false;
  }
  return true;
}

/** checkSSHArgv returns null for an argv of the one allowed shape, else why not. */
export function checkSSHArgv(argv: string[]): string | null {
  for (const a of argv) if (/[\n\r\0]/.test(a)) return "an argument has a line break or NUL";
  if (argv.length === 0) return "the ssh command is empty";
  const b = base(argv[0]!);
  if (b !== "ssh" && b !== "ssh.exe") return "the program is not the system ssh";
  let i = 1;
  const peek = () => (i < argv.length ? argv[i]! : "");
  const flag = (want: string) => {
    if (peek() !== want || i >= argv.length) throw `expected ${want}`;
    i++;
  };
  const next = (what: string) => {
    if (i >= argv.length) throw `the ssh command ends before ${what}`;
    return argv[i++]!;
  };
  const option = (want: string) => {
    flag("-o");
    if (next("an option") !== want) throw `expected the option ${want}`;
  };
  try {
    flag("-p");
    const port = next("the port");
    const n = Number(port);
    if (!/^[0-9]+$/.test(port) || n < 1 || n > 65535 || String(n) !== port) return "the port is not a plain number";
    if (peek() === "-i") {
      i++;
      const path = next("the key path");
      if (checkSSHPath(path) || /["']/.test(path)) return "key path: not a plain path";
    }
    option(SSH_OPT_STRICT);
    flag("-o");
    const kh = next("the known_hosts option");
    if (!kh.startsWith(SSH_OPT_KNOWN_HOSTS_PREFIX)) return "the UserKnownHostsFile option is missing";
    const e = checkKnownHostsList(kh.slice(SSH_OPT_KNOWN_HOSTS_PREFIX.length));
    if (e) return e;
    for (const o of [SSH_OPT_GLOBAL_KNOWN_HOSTS, SSH_OPT_PROXY_JUMP, SSH_OPT_PROXY_COMMAND]) option(o);
    if (peek() === "-l") {
      i++;
      const u = next("the user");
      if (!plainName(u, false) || u.length > 64) return "the user is not a plain name";
    }
    flag("--");
    const host = next("the destination");
    if (!plainName(host, true) || host.length > 253) return "the destination is not a plain host name";
    if (i !== argv.length) return "the ssh command has elements after the destination";
  } catch (err) {
    return String(err);
  }
  return null;
}

/** localSSH is the argv the TUI runs: the local ssh, resolved at startup, with the checked arguments. */
export const localSSH = (argv: string[], ssh: string) => [ssh, ...argv.slice(1)];
