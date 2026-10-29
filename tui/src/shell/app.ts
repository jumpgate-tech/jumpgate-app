// App: the one layout's renderables, the key router, the screens, and the
// live feeds (fleet, jobs) every screen reads from.
import { BoxRenderable, TextRenderable, type CliRenderer, type KeyEvent, type MouseEvent } from "@opentui/core";
import * as api from "../api/client.ts";
import type { ApiError, Conn } from "../api/http.ts";
import type { StreamOpts } from "../api/sse.ts";
import type { DrawerPref, Fleet, JobsInbox, RunningJobs, UIPrefs, VaultView } from "../api/types.ts";
import { maskText } from "../maskurl.ts";
import { redact } from "../secret.ts";
import { clip, sanitizeLine } from "../text.ts";
import { Activity } from "./activity.ts";
import { confirmModal, helpModal, SecretModal, ShowOnce, TextModal, TypedConfirm, type Modal } from "./confirm.ts";
import { runArgv, runJumpgate, type RunOpts } from "../shellout.ts";
import { isFleet, isJobsInbox } from "../api/guards.ts";
import type { Bins } from "../bins.ts";
import { drawerLines, minimizedLine } from "./drawer.ts";
import { layoutFor, MIN_H, MIN_W, type Layout, type Rect } from "./layout.ts";
import { statusLine } from "./statusbar.ts";
import { alertLine, alerts } from "./alerts.ts";
import { DIM_COLOR, FOCUS_COLOR, isSelected, styled } from "./style.ts";
import { buttonAt, DOUBLE_MS, inList, statusTarget } from "./mouse.ts";
import { key, type Key } from "./keys.ts";

export interface Screen {
  id: string;
  title: string;
  /** A sub-screen (node detail, job detail) highlights its parent in the nav. */
  parent?: string;
  view(w: number, h: number): string[];
  /** key returns true when it used the key. */
  key?(k: Key): boolean;
  hints?(): string;
  /** crumbs is where the user is inside the screen, after its nav title: ["alpha", "exec"]. */
  crumbs?(): string[];
  /** click handles a click on view() line `line`, column `col`; true when it used it. */
  click?(line: number, col: number, double: boolean): boolean;
  enter?(arg?: unknown): void;
  leave?(): void;
}

export interface AppState {
  conn: Conn | null;
  prefs?: UIPrefs;
  drawer: DrawerPref;
  serverVersion?: string;
  vault?: VaultView;
  fleet?: Fleet;
  fleetStale: boolean;
  inbox?: JobsInbox;
  inboxStale: boolean;
  running?: RunningJobs;
}

export interface AppOptions {
  myVersion: string;
  streamOpts?: StreamOpts;
  /** How often to re-read /api/jobs/running, in ms. */
  runningEveryMs?: number;
  /** bins are jumpgate and ssh, resolved to absolute paths at startup. */
  bins?: Bins;
  /** run replaces runJumpgate (tests record argv). */
  run?: (args: string[], o: RunOpts) => Promise<number>;
  /** runArgv replaces shellout.runArgv. */
  runArgv?: (argv: string[], o: RunOpts) => Promise<number>;
  /** Keys that arrive this soon after a modal opens were typed before it: they are dropped. */
  modalGuardMs?: number;
  /** platform replaces process.platform (Linux-only actions). */
  platform?: string;
}

export type Pane = "nav" | "main" | "activity";

/** SEP joins the breadcrumb in the main pane's title. */
export const SEP = " › ";

const GLOBAL_KEYS = [
  "tab / shift-tab   focus the next / previous pane (double border)",
  "up / down         move in the focused pane",
  "enter             open (in the nav: go to the screen)",
  "esc               back, or from the screen to the nav",
  "mouse             click a pane, a row or a [key] button; double-click opens",
  "                  the wheel scrolls; a typed confirm still needs typing",
  ".                 open or minimize the drawer",
  "?                 this help",
  "q / ctrl-c        quit",
];

export const toKey = (e: KeyEvent): Key => ({ name: e.name, seq: e.sequence, ctrl: e.ctrl, shift: e.shift, meta: e.meta });

export class App {
  readonly state: AppState = { conn: null, drawer: "auto", fleetStale: false, inboxStale: false };
  readonly activity = new Activity();
  modal: Modal | null = null;
  /** flash is the last failure, shown on the hints row until the next key. */
  flash = "";
  private screens = new Map<string, Screen>();
  private nav: string[] = [];
  current!: Screen;
  private feeds: AbortController | null = null;
  private timers: ReturnType<typeof setInterval>[] = [];
  private closed = false;
  private resolveDone!: () => void;
  readonly done = new Promise<void>((r) => (this.resolveDone = r));

  private boxes: Record<"status" | "nav" | "main" | "activity" | "drawer" | "hints", { box: BoxRenderable; text: TextRenderable }>;
  private small: TextRenderable;

  constructor(
    readonly renderer: CliRenderer,
    readonly opts: AppOptions,
  ) {
    const mk = (border: boolean, title?: string) => {
      const box = new BoxRenderable(renderer, { position: "absolute", left: 0, top: 0, width: 1, height: 1, border, title });
      const text = new TextRenderable(renderer, { content: "" });
      box.add(text);
      renderer.root.add(box);
      return { box, text };
    };
    this.boxes = {
      status: mk(false),
      nav: mk(true, "jumpgate"),
      main: mk(true),
      activity: mk(true, "activity"),
      drawer: mk(true, "jobs"),
      hints: mk(false),
    };
    this.small = new TextRenderable(renderer, { content: "", position: "absolute", left: 0, top: 0 });
    renderer.root.add(this.small);
    renderer.keyInput.on("keypress", (e: KeyEvent) => this.handleKey(toKey(e)));
    renderer.on("resize", () => this.render());
    this.wireMouse();
  }

  get conn(): Conn {
    if (!this.state.conn) throw new Error("not connected");
    return this.state.conn;
  }

  add(screen: Screen, inNav = true): void {
    this.screens.set(screen.id, screen);
    if (inNav) this.nav.push(screen.id);
    if (!this.current) this.current = screen;
  }

  screen<T extends Screen>(id: string): T {
    return this.screens.get(id) as T;
  }

  go(id: string, arg?: unknown): void {
    const next = this.screens.get(id);
    if (!next) return;
    if (next !== this.current) this.current?.leave?.();
    this.current = next;
    next.enter?.(arg);
    this.render();
  }

  log(line: string): void {
    this.activity.add(line);
    this.render();
  }

  /** fail logs an error, redacted, with its hint. */
  fail(what: string, e: unknown): void {
    const err = e as Partial<ApiError>;
    const msg = typeof err?.message === "string" ? err.message : String(e);
    const clean = (s: string) => maskText(sanitizeLine(redact(s)));
    const line = `FAIL ${what}: ${clean(msg)}${err?.hint ? ` (${clean(err.hint)})` : ""}`;
    this.flash = sanitizeLine(line);
    this.log(line);
  }

  private modalReadyAt = 0;

  private open(m: Modal): void {
    this.modal = m;
    // Flush: keys already queued (typed ahead of the modal) must not answer it.
    this.modalReadyAt = performance.now() + (this.opts.modalGuardMs ?? 150);
    this.render();
  }

  /** confirmTyped asks for want to be typed exactly; Enter alone never says yes. */
  confirmTyped(text: string, want: string): Promise<boolean> {
    return new Promise((resolve) =>
      this.open(
        new TypedConfirm(text, want, (ok) => {
          this.close();
          resolve(ok);
        }),
      ),
    );
  }
  private close(): void {
    this.modal = null;
    this.render();
  }

  confirm(text: string): Promise<boolean> {
    return new Promise((resolve) =>
      this.open(
        confirmModal(text, (ok) => {
          this.close();
          resolve(ok);
        }),
      ),
    );
  }

  askSecret(prompt: string): Promise<Uint8Array | null> {
    return new Promise((resolve) =>
      this.open(
        new SecretModal(prompt, (v) => {
          this.close();
          resolve(v);
        }),
      ),
    );
  }

  askText(prompt: string, initial = ""): Promise<string | null> {
    return new Promise((resolve) =>
      this.open(
        new TextModal(prompt, initial, (v) => {
          this.close();
          resolve(v);
        }),
      ),
    );
  }

  get platform(): string {
    return this.opts.platform ?? process.platform;
  }

  /**
   * run executes `jumpgate <args>` with the renderer suspended; the child owns
   * the terminal. args is fixed by the caller and never holds a secret.
   */
  /** runArgv runs an argv the server built (ssh), the terminal handed to it. */
  async runArgv(argv: string[], label: string): Promise<number> {
    const code = await (this.opts.runArgv ?? runArgv)(argv, { renderer: this.renderer });
    this.log(`${label}: exit ${code}${code === 0 ? "" : " FAIL"}`);
    return code;
  }

  async run(args: string[], o: Omit<RunOpts, "renderer"> = {}): Promise<number> {
    const code = await (this.opts.run ?? runJumpgate)(args, { ...o, bin: this.opts.bins?.jumpgate, renderer: this.renderer });
    if (code === 127 && !this.opts.bins?.jumpgate) this.log("FAIL jumpgate was not found: set JUMPGATE_BIN to its absolute path");
    this.log(`jumpgate ${args.join(" ")}: exit ${code}${code === 0 ? "" : " FAIL"}`);
    void this.refreshVault();
    return code;
  }

  showOnce(title: string, value: string): Promise<void> {
    return new Promise((resolve) =>
      this.open(
        new ShowOnce(title, value, () => {
          this.close();
          resolve();
        }),
      ),
    );
  }

  // ---- feeds ----

  /** attach connects the feeds to conn: version, vault, prefs, fleet, jobs. */
  attach(conn: Conn): void {
    this.detach();
    this.state.conn = conn;
    const ac = new AbortController();
    this.feeds = ac;
    const s = ac.signal;
    api.version(conn).then((v) => this.set({ serverVersion: v.version }), (e) => this.fail("version", e));
    this.refreshVault();
    api.prefs(conn).then(
      (p) => this.set({ prefs: p, drawer: p.drawer ?? "auto" }),
      (e) => this.fail("ui prefs", e),
    );
    this.refreshRunning();
    this.timers.push(setInterval(() => this.refreshRunning(), this.opts.runningEveryMs ?? 5000));
    void this.follow(api.watchFleet(conn, s, this.opts.streamOpts), "fleet", (f) => this.set({ fleet: f, fleetStale: false }), () => this.set({ fleetStale: true }), isFleet);
    void this.follow(api.watchJobs(conn, s, this.opts.streamOpts), "jobs", (i) => {
      this.set({ inbox: i, inboxStale: false });
      this.refreshRunning();
    }, () => this.set({ inboxStale: true }), isJobsInbox);
    this.render();
  }

  detach(): void {
    this.feeds?.abort();
    this.feeds = null;
    for (const t of this.timers) clearInterval(t);
    this.timers = [];
  }

  private async follow<T>(it: AsyncGenerator<api.SSEMsg<T>>, what: string, value: (v: T) => void, stale: () => void, ok: (v: unknown) => boolean) {
    try {
      for await (const m of it) {
        if (m.kind === "event") {
          if (m.event === "message") {
            if (ok(m.data)) value(m.data);
            else this.log(`${what}: dropped a frame that was not the expected shape`);
          }
          else if (m.event === "error") this.fail(what, { message: (m.data as { error?: string })?.error ?? "stream error" });
        } else if (m.state === "retrying" || m.state === "failed") {
          stale();
          if (m.state === "failed" && m.error) this.fail(`${what} stream`, m.error);
        }
      }
    } catch (e) {
      if (!this.closed) this.fail(`${what} stream`, e);
    }
  }

  refreshVault(): Promise<void> {
    if (!this.state.conn) return Promise.resolve();
    return api.vault(this.state.conn).then((v) => this.set({ vault: v }), (e) => this.fail("vault", e));
  }

  refreshRunning(): void {
    if (!this.state.conn) return;
    api.runningJobs(this.state.conn).then((r) => this.set({ running: r }), () => {});
  }

  private seenAlerts = new Set<string>();
  private alertsSeeded = false;

  set(p: Partial<AppState>): void {
    Object.assign(this.state, p);
    if (p.fleet) {
      // New alerts reach the activity pane once each; one that clears may come back.
      // The first frame's alerts are the Alerts screen's list: they are counted, not repeated.
      const now = new Set(alerts(p.fleet, new Date()).map(alertLine));
      if (!this.alertsSeeded) {
        this.alertsSeeded = true;
        if (now.size) this.activity.add(`${now.size} alert${now.size === 1 ? "" : "s"} open: see Alerts`);
      } else for (const a of now) if (!this.seenAlerts.has(a)) this.activity.add(`alert ${a}`);
      this.seenAlerts = now;
    }
    this.render();
  }

  decisions(): number | undefined {
    const i = this.state.inbox;
    return i ? i.jobs.filter((j) => j.state === "needs-decision").length : undefined;
  }

  // ---- keys ----

  handleKey(k: Key): void {
    if (this.closed) return;
    this.flash = "";
    if (k.ctrl && k.name === "c") {
      if (this.modal instanceof SecretModal) this.modal.key({ ...k, name: "escape" });
      return this.quit();
    }
    if (this.modal) {
      if (performance.now() < this.modalReadyAt) return;
      this.modal.key(k);
      this.render();
      return;
    }
    if (k.name === "q" && !k.ctrl && !k.meta) return this.quit();
    if (k.name === "?" || k.seq === "?") return this.open(helpModal([...GLOBAL_KEYS, "", this.current.hints?.() ?? ""], () => this.close()));
    if (k.name === "." || k.seq === ".") return this.toggleDrawer();
    if (k.name === "tab") return this.cycleFocus(k.shift ? -1 : 1);
    if (this.focus === "nav") {
      if (k.name === "up") return this.moveNav(-1);
      if (k.name === "down") return this.moveNav(1);
      if (k.name === "return" || k.name === "right") return this.setFocus("main");
    }
    if (this.focus === "activity") {
      const page = Math.max(1, (this.lastLayout?.activity?.h ?? 4) - 3);
      if (k.name === "up") return this.scrollActivity(1);
      if (k.name === "down") return this.scrollActivity(-1);
      if (k.name === "pageup") return this.scrollActivity(page);
      if (k.name === "pagedown") return this.scrollActivity(-page);
    }
    if (this.current.key?.(k)) {
      this.focus = "main";
      return this.render();
    }
    if (k.name === "escape" && this.focus === "main") return this.setFocus("nav");
  }

  // ---- mouse ----

  private lastDown = { x: -1, y: -1, t: 0 };

  /** wireMouse: a click focuses the pane under it and acts; the wheel scrolls it. */
  private wireMouse(): void {
    const b = this.boxes;
    const at = (name: keyof App["boxes"], e: MouseEvent) => {
      const s = this.shown[name];
      if (!s) return null;
      const inset = b[name].box.border ? 1 : 0;
      return { row: e.y - s.r.y - inset, col: e.x - s.r.x - inset, lines: s.lines };
    };
    const down = (name: keyof App["boxes"], fn: (p: { row: number; col: number; lines: string[] }, double: boolean) => void) => {
      b[name].box.onMouseDown = (e: MouseEvent) => {
        if (this.closed || e.button !== 0) return;
        const p = at(name, e);
        if (!p) return;
        const now = performance.now();
        const double = now - this.lastDown.t < DOUBLE_MS && this.lastDown.x === e.x && this.lastDown.y === e.y;
        this.lastDown = double ? { x: -1, y: -1, t: 0 } : { x: e.x, y: e.y, t: now };
        this.flash = "";
        fn(p, double);
      };
    };
    const wheel = (name: keyof App["boxes"], fn: (d: 1 | -1) => void) => {
      b[name].box.onMouseScroll = (e: MouseEvent) => {
        if (this.closed) return;
        const dir = e.scroll?.direction;
        if (dir === "up" || dir === "down") fn(dir === "up" ? -1 : 1);
      };
    };
    down("nav", ({ row }) => {
      if (this.modal) return;
      const id = this.nav[row];
      this.focus = "nav";
      if (id) this.go(id);
      else this.render();
    });
    wheel("nav", (d) => !this.modal && this.moveNav(d));
    down("main", ({ row, col, lines }, double) => this.clickMain(row, col, lines, double));
    wheel("main", (d) => {
      if (this.modal) return;
      this.focus = "main";
      this.current.key?.(key(d < 0 ? "up" : "down"));
      this.render();
    });
    down("activity", () => !this.modal && this.setFocus("activity"));
    wheel("activity", (d) => this.scrollActivity(-d * 3));
    down("drawer", () => !this.modal && this.toggleDrawer());
    down("hints", ({ col, lines }) => {
      // The folded drawer, "[1 running]", opens it.
      const m = /^\[\d+ running\]/.exec(lines[0] ?? "");
      if (!this.modal && m && col < m[0].length) this.toggleDrawer();
    });
    down("status", ({ col, lines }) => {
      if (this.modal) return;
      const id = statusTarget(lines[0] ?? "", col);
      if (id && this.screens.has(id)) this.go(id);
    });
  }

  /**
   * clickMain: in a modal, a click presses the button under it, but a typed
   * confirm only takes [esc] (a click never confirms it). On a screen, the
   * screen's own click, then a [key] button, then a list row: one click
   * selects it, a double click opens it.
   */
  private clickMain(row: number, col: number, lines: string[], double: boolean): void {
    const line = lines[row] ?? "";
    if (this.modal) {
      if (performance.now() < this.modalReadyAt) return;
      const k = buttonAt(line, col);
      if (!k) return;
      if (this.modal instanceof TypedConfirm && k.name !== "escape") return;
      this.modal.key(k);
      return this.render();
    }
    this.focus = "main";
    const s = this.current;
    if (s.click?.(row, col, double)) return this.render();
    const k = buttonAt(line, col);
    if (k) {
      s.key?.(k);
      return this.render();
    }
    const sel = lines.findIndex(isSelected);
    if (sel >= 0 && row < lines.length && inList(lines, sel, row)) {
      const d = row - sel;
      for (let i = 0; i < Math.abs(d); i++) s.key?.(key(d < 0 ? "up" : "down"));
      if (double) s.key?.(key("return"));
    }
    this.render();
  }

  /** focus is the pane that takes up/down/enter; tab cycles it. */
  focus: Pane = "main";
  /** actScroll is how many lines the activity pane is scrolled up from the newest. */
  actScroll = 0;
  private lastLayout?: Layout;

  setFocus(p: Pane): void {
    this.focus = p;
    this.render();
  }

  private panes(): Pane[] {
    return this.lastLayout?.activity ? ["nav", "main", "activity"] : ["nav", "main"];
  }

  private cycleFocus(d: number): void {
    const ps = this.panes();
    const i = Math.max(0, ps.indexOf(this.focus));
    this.setFocus(ps[(i + d + ps.length) % ps.length]!);
  }

  scrollActivity(d: number): void {
    const h = Math.max(0, (this.lastLayout?.activity?.h ?? 2) - 2);
    const max = Math.max(0, this.activity.all().length - h);
    this.actScroll = Math.min(max, Math.max(0, this.actScroll + d));
    this.render();
  }

  /** crumbs is the main pane's title: the nav screen, then where the user is inside it. */
  crumbs(): string[] {
    const s = this.current;
    const root = s.parent ? this.screens.get(s.parent)?.title ?? s.title : s.title;
    return [root, ...(s.crumbs?.() ?? (s.parent ? [s.title] : []))].map((c) => sanitizeLine(c));
  }

  /** hintLine is the focused pane's keys only, then the few global ones. */
  hintLine(): string {
    if (this.modal) return "";
    let mine = "";
    if (this.focus === "nav") mine = "up/down screen  enter go to it";
    else if (this.focus === "activity") mine = "up/down scroll";
    else mine = this.current.hints?.() ?? "";
    return [mine, "tab pane  ? keys  q quit"].filter(Boolean).join("  ");
  }

  private moveNav(d: number): void {
    const cur = this.current.parent ?? this.current.id;
    const i = Math.max(0, this.nav.indexOf(cur));
    const id = this.nav[(i + d + this.nav.length) % this.nav.length]!;
    this.go(id);
  }

  private toggleDrawer(): void {
    const l = layoutFor(this.renderer.width, this.renderer.height, this.state.drawer);
    const open = l.tooSmall ? this.state.drawer === "open" : l.drawerOpen;
    const next: DrawerPref = open ? "min" : "open";
    this.set({ drawer: next });
    const { conn, prefs } = this.state;
    if (!conn || !prefs) return;
    api.savePrefs(conn, { ...prefs, drawer: next }).then(
      (p) => (this.state.prefs = p),
      (e) => {
        this.fail("save the drawer setting", e);
        api.prefs(conn).then((p) => this.set({ prefs: p, drawer: p.drawer ?? "auto" }), () => {});
      },
    );
  }

  quit(): void {
    if (this.closed) return;
    this.closed = true;
    this.detach();
    this.current?.leave?.();
    this.renderer.destroy();
    this.resolveDone();
  }

  get isClosed(): boolean {
    return this.closed;
  }

  // ---- drawing ----

  /** shown is what each pane drew last, line by line: mouse clicks map onto it. */
  readonly shown: Partial<Record<keyof App["boxes"], { r: Rect; lines: string[] }>> = {};

  private place(name: keyof App["boxes"], r: Rect | null, lines: string[], title?: string, o: { focused?: boolean; marks?: boolean } = {}): void {
    const { box, text } = this.boxes[name];
    if (!r || r.h <= 0 || r.w <= 0) {
      box.visible = false;
      delete this.shown[name];
      return;
    }
    box.visible = true;
    box.left = r.x;
    box.top = r.y;
    box.width = r.w;
    box.height = r.h;
    if (title !== undefined) box.title = clip(title, Math.max(0, r.w - 4));
    if (box.border) {
      // The focused pane: a double border in a bright color; the rest single and dim.
      box.borderStyle = o.focused ? "double" : "single";
      box.borderColor = o.focused ? FOCUS_COLOR : DIM_COLOR;
    }
    const border = box.border ? 2 : 0;
    const iw = Math.max(0, r.w - border);
    const ih = Math.max(0, r.h - border);
    const out = lines.slice(0, ih).map((l) => clip(sanitizeLine(l), iw));
    this.shown[name] = { r, lines: out };
    text.content = o.marks ? styled(out, iw) : out.join("\n");
  }

  render(): void {
    if (this.closed) return;
    const w = this.renderer.width;
    const h = this.renderer.height;
    const running = this.state.running?.jobs;
    const lines = drawerLines(running);
    const l = layoutFor(w, h, this.state.drawer, lines.length);
    if (l.tooSmall) {
      for (const b of Object.values(this.boxes)) b.box.visible = false;
      this.small.visible = true;
      this.small.content = `window too small: ${w}x${h}, need ${MIN_W}x${MIN_H}`;
      this.renderer.requestRender();
      return;
    }
    this.small.visible = false;
    const st = this.state;
    this.place("status", l.status, [
      statusLine({
        connected: !!st.conn,
        serverVersion: st.serverVersion,
        myVersion: this.opts.myVersion,
        vaultState: st.vault?.state,
        fleet: st.fleet,
        fleetStale: st.fleetStale,
        decisions: st.conn ? this.decisions() : undefined,
      }),
    ]);
    this.lastLayout = l;
    if (this.modal || (this.focus === "activity" && !l.activity)) this.focus = "main";
    const f = this.focus;
    const cur = this.current.parent ?? this.current.id;
    this.place("nav", l.nav, this.nav.map((id) => `${id === cur ? ">" : " "} ${this.screens.get(id)!.title}`), "jumpgate", { focused: f === "nav", marks: true });
    const inner = { w: Math.max(0, l.main.w - 2), h: Math.max(0, l.main.h - 2) };
    if (this.modal) this.place("main", l.main, this.modal.lines(inner.w), this.modal.title, { focused: true });
    else this.place("main", l.main, this.current.view(inner.w, inner.h), this.crumbs().join(SEP), { focused: f === "main", marks: true });
    const ah = Math.max(0, (l.activity?.h ?? 2) - 2);
    this.actScroll = Math.min(this.actScroll, Math.max(0, this.activity.all().length - ah));
    const act = this.activity.window(ah, this.actScroll);
    this.place("activity", l.activity, act, this.actScroll ? `activity (${this.actScroll} newer below)` : "activity", { focused: f === "activity" });
    const db = this.boxes.drawer.box;
    db.border = l.drawerBoxed;
    this.place("drawer", l.drawer, lines, "jobs");
    const min = l.drawerOpen ? "" : minimizedLine(running);
    // The last failure is in the activity pane; without one, it flashes here.
    const flash = l.activity ? "" : this.flash;
    this.place("hints", l.hints, [[min, flash || this.hintLine()].filter(Boolean).join("  ")]);
    this.renderer.requestRender();
  }
}
