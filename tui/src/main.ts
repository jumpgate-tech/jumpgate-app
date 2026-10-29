// jumpgate-tui: a terminal dashboard for the jumpgate server, over its API.
import pkg from "../package.json";

export const VERSION: string = pkg.version;

const HELP = `jumpgate-tui ${VERSION}

Usage: jumpgate-tui [options]

Options:
  --socket PATH   talk to the server on this unix socket
  --version       print the version and exit
  --help          print this help and exit

Keys inside the dashboard: ? lists them, q quits.
`;

export interface Args {
  socket?: string;
}

export function parseArgs(argv: string[]): Args | { exit: number; out?: string; err?: string } {
  const args: Args = {};
  for (let i = 0; i < argv.length; i++) {
    const a = argv[i];
    switch (a) {
      case "--version":
        return { exit: 0, out: `jumpgate-tui ${VERSION}\n` };
      case "--help":
      case "-h":
        return { exit: 0, out: HELP };
      case "--socket": {
        const v = argv[++i];
        if (!v) return { exit: 2, err: "--socket needs a path\n" };
        args.socket = v;
        break;
      }
      default:
        return { exit: 2, err: `unknown option ${a}\n${HELP}` };
    }
  }
  return args;
}

async function main(): Promise<number> {
  const parsed = parseArgs(Bun.argv.slice(2));
  if ("exit" in parsed) {
    if (parsed.out) process.stdout.write(parsed.out);
    if (parsed.err) process.stderr.write(parsed.err);
    return parsed.exit;
  }
  const { start } = await import("./shell/start.ts");
  return start(parsed);
}

if (import.meta.main) {
  process.exitCode = await main();
}
