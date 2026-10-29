// clipboard: copy a line of public text (the invite command) for the user.
// OSC 52 asks the terminal to set its clipboard, which works over ssh; a local
// pbcopy, wl-copy or xclip, found by absolute path, is used as well when there.
// Never call this with a secret: OSC 52 goes through every terminal layer.
import { isAbsolute } from "node:path";
import { onPath } from "./bins.ts";

type Env = Record<string, string | undefined>;

/** osc52 is the escape that sets the terminal's clipboard to text. */
export const osc52 = (text: string) => `\x1b]52;c;${Buffer.from(text, "utf8").toString("base64")}\x07`;

/** clipboardArgv is the local copy program for this platform, by absolute path, or null. */
export function clipboardArgv(env: Env = process.env, platform: string = process.platform): string[] | null {
  const tries: string[][] =
    platform === "darwin"
      ? [["pbcopy"]]
      : [
          ...(env.WAYLAND_DISPLAY ? [["wl-copy"]] : []),
          ...(env.DISPLAY ? [["xclip", "-selection", "clipboard"]] : []),
        ];
  for (const [name, ...rest] of tries) {
    const bin = onPath(name!, env);
    if (bin && isAbsolute(bin)) return [bin, ...rest];
  }
  return null;
}

export interface Clipboard {
  /** osc writes the escape to the terminal. */
  osc(seq: string): void;
  /** argv is the local copy program, or null. */
  argv: string[] | null;
  /** run feeds input to argv and returns its exit code. */
  run(argv: string[], input: string): Promise<number>;
}

export async function runWithInput(argv: string[], input: string): Promise<number> {
  const p = Bun.spawn(argv, { stdin: "pipe", stdout: "ignore", stderr: "ignore" });
  p.stdin.write(input);
  await p.stdin.end();
  return p.exited;
}

export const systemClipboard = (): Clipboard => ({
  osc: (s) => void process.stdout.write(s),
  argv: clipboardArgv(),
  run: runWithInput,
});

/** copyText sets the clipboard both ways; it returns what it did, for the log. */
export async function copyText(text: string, c: Clipboard): Promise<string> {
  c.osc(osc52(text));
  if (!c.argv) return "copied (terminal clipboard, OSC 52)";
  try {
    const code = await c.run(c.argv, text);
    return code === 0 ? `copied (OSC 52 and ${c.argv[0]})` : `copied (OSC 52); ${c.argv[0]} exited ${code}`;
  } catch {
    return `copied (OSC 52); ${c.argv[0]} did not run`;
  }
}
