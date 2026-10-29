// Jobs and decisions (G, C): the inbox; Enter shows a job's steps, its
// question and live output. a approves (the chosen option, 1-9 to pick),
// x refuses (the abort option), A adopts; each after a confirm.
import * as api from "../api/client.ts";
import type { JobDetail, JobOutputLine, JobView } from "../api/types.ts";
import type { App, Screen } from "../shell/app.ts";
import type { Key } from "../shell/keys.ts";
import { maskText } from "../maskurl.ts";
import { isJobOutput } from "../api/guards.ts";
import { sanitizeLine } from "../text.ts";

const OUT_MAX = 500;

export function inboxLines(jobs: JobView[], sel: number, stale: boolean): string[] {
  if (!jobs.length) return ["No jobs."];
  const p = stale ? "~" : "";
  return jobs.map((j, i) => {
    const mark = j.state === "needs-decision" ? "!" : " ";
    return `${i === sel ? ">" : " "}${mark} ${sanitizeLine(`${j.target}/${j.id}`).padEnd(20)} ${p}${sanitizeLine(j.state).padEnd(15)} step ${j.step + 1}/${j.steps.length}  ${sanitizeLine(j.title)}`;
  });
}

export function detailLines(j: JobView, opt: number): string[] {
  const out = [`${sanitizeLine(j.title)} (${sanitizeLine(j.kind)})`, `state ${sanitizeLine(j.state)}  lease ${j.lease.mine ? "this controller" : sanitizeLine(j.lease.label ?? "n/a")}${j.lease.fresh ? "" : " (stale)"}`, ""];
  for (const s of j.steps) out.push(`[${sanitizeLine(s.state).padEnd(7)}] ${sanitizeLine(s.title)}`);
  if (j.error) out.push("", `FAIL ${maskText(sanitizeLine(j.error))}`);
  const q = j.question;
  if (q) {
    out.push("", `? ${sanitizeLine(q.prompt)}`);
    q.options.forEach((o, i) => out.push(`${i === opt ? ">" : " "} ${i + 1} ${sanitizeLine(o.label)}${o.detail ? ` - ${sanitizeLine(o.detail)}` : ""}`));
  }
  return out;
}

export class JobsScreen implements Screen {
  readonly id = "jobs";
  readonly title = "Jobs";
  private sel = 0;
  private open: { target: string; id: string } | null = null;
  private detail?: JobDetail;
  private output: JobOutputLine[] = [];
  private opt = 0;
  private ac: AbortController | null = null;
  /** outBack is how far the output is scrolled up from the newest line. */
  outBack = 0;

  constructor(private readonly app: App) {}

  leave(): void {
    this.close();
  }

  private close(): void {
    this.ac?.abort();
    this.ac = null;
    this.open = null;
    this.detail = undefined;
    this.output = [];
    this.outBack = 0;
  }

  crumbs(): string[] {
    return this.open ? [`${this.open.target}/${this.open.id}`] : [];
  }

  private jobs(): JobView[] {
    return this.app.state.inbox?.jobs ?? [];
  }

  private current(): JobView | undefined {
    if (!this.open) return undefined;
    const live = this.jobs().find((j) => j.target === this.open!.target && j.id === this.open!.id);
    return live ?? this.detail?.job;
  }

  private show(j: JobView): void {
    this.close();
    this.open = { target: j.target, id: j.id };
    this.opt = 0;
    const ac = new AbortController();
    this.ac = ac;
    api.jobDetail(this.app.conn, j.target, j.id).then(
      (d) => {
        if (this.ac === ac) (this.detail = d), this.app.render();
      },
      (e) => this.app.fail(`job ${j.target}/${j.id}`, e),
    );
    void (async () => {
      for await (const m of api.watchJobOutput(this.app.conn, j.target, j.id, ac.signal, this.app.opts.streamOpts)) {
        if (m.kind !== "event" || m.event !== "message" || !isJobOutput(m.data)) continue;
        this.output.push(...m.data);
        if (this.output.length > OUT_MAX) this.output.splice(0, this.output.length - OUT_MAX);
        this.app.render();
      }
    })();
  }

  view(_w: number, h: number): string[] {
    if (!this.app.state.conn) return ["Not connected."];
    const inbox = this.app.state.inbox;
    if (!inbox) return ["Waiting for jobs..."];
    const j = this.current();
    if (!j) {
      const errs = inbox.errors.map((e) => `FAIL ${sanitizeLine(e.target)}: ${maskText(sanitizeLine(e.error.error))}`);
      return [...inboxLines(this.jobs(), this.sel, this.app.state.inboxStale), ...(errs.length ? ["", ...errs] : [])];
    }
    const top = detailLines(j, this.opt);
    const room = Math.max(1, h - top.length - 2);
    this.outBack = Math.min(this.outBack, Math.max(0, this.output.length - room));
    const end = this.output.length - this.outBack;
    const out = this.output.slice(Math.max(0, end - room), end).map((o) => `${sanitizeLine(o.unit)} ${sanitizeLine(o.line)}`);
    return [...top, "", "output:", ...(out.length ? out : ["(none yet)"])];
  }

  key(k: Key): boolean {
    const j = this.current();
    if (!j) {
      const jobs = this.jobs();
      if (!jobs.length) return false;
      if (k.name === "up") return (this.sel = Math.max(0, this.sel - 1)), true;
      if (k.name === "down") return (this.sel = Math.min(jobs.length - 1, this.sel + 1)), true;
      if (k.name === "return") return this.show(jobs[Math.min(this.sel, jobs.length - 1)]!), true;
      return false;
    }
    if (k.name === "escape") return this.close(), true;
    if (k.name === "up") return (this.outBack += 1), true;
    if (k.name === "down") return (this.outBack = Math.max(0, this.outBack - 1)), true;
    const n = Number(k.name);
    if (j.question && n >= 1 && n <= j.question.options.length) return (this.opt = n - 1), true;
    if (k.seq === "A") return void this.adopt(j), true;
    if (k.seq === "a" && j.question) return void this.decide(j, j.question.options[this.opt]?.id), true;
    if (k.seq === "x" && j.question) {
      const abort = j.question.options.find((o) => o.id === "abort");
      if (!abort) return this.app.log("this question has no refuse option; pick one with 1-9 and press a"), true;
      return void this.decide(j, abort.id), true;
    }
    return false;
  }

  hints(): string {
    return this.open ? "1-9 pick  a approve  x refuse  A adopt  up/down scroll output  esc back" : "up/down select  enter open";
  }

  private async decide(j: JobView, option: string | undefined): Promise<void> {
    const q = j.question;
    const o = q?.options.find((x) => x.id === option);
    if (!q || !o) return;
    if (!(await this.app.confirmTyped(`${q.prompt}\nAnswer: ${o.label}${o.detail ? ` (${o.detail})` : ""}`, o.id))) return;
    try {
      await api.decide(this.app.conn, j.target, j.id, { decision: q.id, option: o.id });
      this.app.log(`answered ${j.target}/${j.id}: ${o.label}`);
    } catch (e) {
      this.app.fail(`answer ${j.target}/${j.id}`, e);
    }
  }

  private async adopt(j: JobView): Promise<void> {
    if (!(await this.app.confirmTyped(`Adopt ${j.target}/${j.id}? This controller takes over driving it.`, j.id))) return;
    try {
      await api.adopt(this.app.conn, j.target, j.id);
      this.app.log(`adopted ${j.target}/${j.id}`);
    } catch (e) {
      this.app.fail(`adopt ${j.target}/${j.id}`, e);
    }
  }
}
