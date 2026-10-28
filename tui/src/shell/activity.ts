// The activity pane: the last 200 lines, each redacted and sanitized.
import { redact } from "../secret.ts";
import { sanitizeLine } from "../text.ts";

export const ACTIVITY_MAX = 200;

export class Activity {
  private lines: string[] = [];
  constructor(private readonly now: () => Date = () => new Date()) {}

  add(line: string): void {
    const t = this.now().toISOString().slice(11, 19);
    this.lines.push(`${t} ${sanitizeLine(redact(line))}`);
    if (this.lines.length > ACTIVITY_MAX) this.lines.splice(0, this.lines.length - ACTIVITY_MAX);
  }

  /** last returns up to n of the newest lines, oldest first. */
  last(n: number): string[] {
    return n <= 0 ? [] : this.lines.slice(-n);
  }

  all(): readonly string[] {
    return this.lines;
  }
}
