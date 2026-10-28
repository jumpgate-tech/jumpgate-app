// Connect (A): find the server; offer to start one; name the admin group.
import { join } from "node:path";
import { discover as realDiscover, DiscoverError, jumpgateHome } from "../api/discover.ts";
import type { Conn } from "../api/http.ts";
import type { App, Screen } from "../shell/app.ts";
import type { Key } from "../shell/keys.ts";
import { maskText } from "../maskurl.ts";
import { sanitizeLine, wrap } from "../text.ts";
import { runJumpgate, type RunOpts } from "../shellout.ts";

export interface ConnectDeps {
  discover?: () => Promise<Conn>;
  run?: (args: string[], o: RunOpts) => Promise<number>;
  pollMs?: number;
  timeoutMs?: number;
  home?: string;
  /** The screen to go to once connected. */
  next?: string;
}

type State =
  | { s: "finding" }
  | { s: "starting" }
  | { s: "error"; kind: DiscoverError["kind"] | "other"; msg: string }
  | { s: "connected" };

export class ConnectScreen implements Screen {
  readonly id = "connect";
  readonly title = "Connect";
  private st: State = { s: "finding" };
  private readonly d: Required<ConnectDeps>;

  constructor(
    private readonly app: App,
    deps: ConnectDeps = {},
  ) {
    this.d = {
      discover: deps.discover ?? (() => realDiscover()),
      run: deps.run ?? runJumpgate,
      pollMs: deps.pollMs ?? 250,
      timeoutMs: deps.timeoutMs ?? 10_000,
      home: deps.home ?? jumpgateHome(),
      next: deps.next ?? "fleet",
    };
  }

  enter(arg?: unknown): void {
    if ((arg as { stay?: boolean })?.stay || this.st.s === "starting") return;
    void this.find();
  }

  private async find(): Promise<boolean> {
    this.st = { s: "finding" };
    this.app.render();
    try {
      const conn = await this.d.discover();
      this.st = { s: "connected" };
      this.app.attach(conn);
      this.app.log(`connected to ${conn.system ? "the jumpgate service" : "your jumpgate server"}`);
      this.app.go(this.d.next);
      return true;
    } catch (e) {
      this.st =
        e instanceof DiscoverError ? { s: "error", kind: e.kind, msg: e.message } : { s: "error", kind: "other", msg: String((e as Error)?.message ?? e) };
      this.app.render();
      return false;
    }
  }

  private async start(): Promise<void> {
    this.st = { s: "starting" };
    this.app.log("starting jumpgate serve");
    const run = join(this.d.home, "run");
    const code = await this.d.run(["serve", "--no-open"], { bin: this.app.opts.bins?.jumpgate, detach: { log: join(run, "server.log"), cwd: run } });
    if (code !== 0) {
      this.st = { s: "error", kind: "not_running", msg: `jumpgate serve did not start (exit ${code})` };
      return this.app.render();
    }
    const until = Date.now() + this.d.timeoutMs;
    while (Date.now() < until) {
      await Bun.sleep(this.d.pollMs);
      try {
        await this.d.discover();
        this.st = { s: "finding" };
        await this.find();
        return;
      } catch {
        /* not up yet */
      }
    }
    this.st = { s: "error", kind: "not_running", msg: "the server did not answer within 10 s; see ~/.jumpgate/run/server.log" };
    this.app.render();
  }

  view(w: number): string[] {
    return this.lines().flatMap((l) => wrap(l, w));
  }

  private lines(): string[] {
    const st = this.st;
    switch (st.s) {
      case "finding":
        return ["Looking for the jumpgate server..."];
      case "starting":
        return ["Starting jumpgate serve..."];
      case "connected": {
        const sv = this.app.state.serverVersion ?? "n/a";
        const mine = this.app.opts.myVersion;
        const lines = [`Connected: server ${sv}, this dashboard ${mine}`];
        if (this.app.state.serverVersion !== undefined && sv !== mine) lines.push("", "[skew!] the server and this dashboard are different builds");
        return lines;
      }
      case "error":
        if (st.kind === "not_running") return [st.msg === "No jumpgate server is running." ? "No jumpgate server is running. [s] start it" : `${st.msg}  [s] start it`, "", "[r] look again"];
        return [st.kind === "other" ? maskText(sanitizeLine(st.msg)) : st.msg, "", "[r] look again"];
    }
  }

  key(k: Key): boolean {
    if (k.name === "r" && this.st.s !== "starting") {
      void this.find();
      return true;
    }
    if (k.name === "s" && this.st.s === "error" && this.st.kind === "not_running") {
      void this.start();
      return true;
    }
    return false;
  }

  hints(): string {
    return "r look again  s start the server";
  }
}
