// App: the one layout's renderables, the key router, the screens, and the
// live feeds (fleet, jobs) every screen reads from.
import { BoxRenderable, TextRenderable, type CliRenderer, type KeyEvent } from "@opentui/core";
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
import type { Key } from "./keys.ts";
import { layoutFor, MIN_H, MIN_W, type Rect } from "./layout.ts";
import { statusLine } from "./statusbar.ts";
import { alertLine, alerts } from "./alerts.ts";

export interface Screen {
  id: string;
  title: string;
  /** A sub-screen (node detail, job detail) highlights its parent in the nav. */
  parent?: string;
  view(w: number, h: number): string[];
  /** key returns true when it used the key. */
  key?(k: Key): boolean;
  hints?(): string;
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

const GLOBAL_KEYS = [
  "tab / shift-tab   next / previous screen",
  "up / down         move (the nav, when the screen has no list)",
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

  set(p: Partial<AppState>): void {
    Object.assign(this.state, p);
    if (p.fleet) {
      // New alerts reach the activity pane once each; one that clears may come back.
      const now = new Set(alerts(p.fleet, new Date()).map(alertLine));
      for (const a of now) if (!this.seenAlerts.has(a)) this.activity.add(`alert ${a}`);
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
    if (k.name === "tab") return this.moveNav(k.shift ? -1 : 1);
    if (this.current.key?.(k)) return this.render();
    if (k.name === "up") return this.moveNav(-1);
    if (k.name === "down") return this.moveNav(1);
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

  private place(name: keyof App["boxes"], r: Rect | null, lines: string[], title?: string): void {
    const { box, text } = this.boxes[name];
    if (!r || r.h <= 0 || r.w <= 0) {
      box.visible = false;
      return;
    }
    box.visible = true;
    box.left = r.x;
    box.top = r.y;
    box.width = r.w;
    box.height = r.h;
    if (title !== undefined) box.title = clip(title, Math.max(0, r.w - 4));
    const border = box.border ? 2 : 0;
    const iw = Math.max(0, r.w - border);
    const ih = Math.max(0, r.h - border);
    text.content = lines.slice(0, ih).map((l) => clip(sanitizeLine(l), iw)).join("\n");
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
    const cur = this.current.parent ?? this.current.id;
    this.place("nav", l.nav, this.nav.map((id) => `${id === cur ? ">" : " "} ${this.screens.get(id)!.title}`));
    const inner = { w: Math.max(0, l.main.w - 2), h: Math.max(0, l.main.h - 2) };
    if (this.modal) this.place("main", l.main, this.modal.lines(inner.w), this.modal.title);
    else this.place("main", l.main, this.current.view(inner.w, inner.h), this.current.title);
    this.place("activity", l.activity, this.activity.last(Math.max(0, (l.activity?.h ?? 2) - 2)));
    const db = this.boxes.drawer.box;
    db.border = l.drawerBoxed;
    this.place("drawer", l.drawer, lines, "jobs");
    const hint = this.modal ? "" : (this.current.hints?.() ?? "");
    const min = l.drawerOpen ? "" : minimizedLine(running, st.conn ? this.decisions() : undefined);
    this.place("hints", l.hints, [[min, this.flash || hint, "? keys  . drawer  q quit"].filter(Boolean).join("  ")]);
    this.renderer.requestRender();
  }
}
