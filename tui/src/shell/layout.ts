// layoutFor: the one layout, as pure rects. A status strip on top, a nav
// column on the left, main, an activity pane on the right (from 100 columns),
// a progress drawer and a hints line at the bottom. Below 80x24 the window is
// too small to draw.
import type { DrawerPref } from "../api/types.ts";

export interface Rect {
  x: number;
  y: number;
  w: number;
  h: number;
}

export interface Layout {
  tooSmall: false;
  status: Rect;
  nav: Rect;
  main: Rect;
  activity: Rect | null;
  drawer: Rect;
  /** outside is a borderless full-width strip above the drawer, for a line meant to be copied. */
  outside: Rect | null;
  hints: Rect;
  /** The drawer pref, resolved: auto opens at 34+ rows. */
  drawerOpen: boolean;
  /** Boxed: a tall window draws the drawer with a border. */
  drawerBoxed: boolean;
}

export const MIN_W = 80;
export const MIN_H = 24;
export const ACTIVITY_MIN_W = 100;
export const TALL_H = 34;
const NAV_W = 16;
const ACTIVITY_W = 30;
const DRAWER_MAX = 6;
const DRAWER_COMPACT = 2;

export function layoutFor(w: number, h: number, drawer: DrawerPref, lines = 1, outsideRows = 0): Layout | { tooSmall: true } {
  if (w < MIN_W || h < MIN_H) return { tooSmall: true };
  const drawerOpen = drawer === "open" || (drawer === "auto" && h >= TALL_H);
  const drawerBoxed = drawerOpen && h >= TALL_H;
  let drawerH = 0;
  if (drawerOpen) drawerH = drawerBoxed ? 2 + Math.min(Math.max(lines, 1), DRAWER_MAX) : DRAWER_COMPACT;
  const bodyY = 1;
  // The outside strip never takes the body below half the window.
  const outH = Math.max(0, Math.min(outsideRows, Math.floor((h - 2 - drawerH) / 2)));
  const bodyH = h - 1 - 1 - drawerH - outH;
  const actW = w >= ACTIVITY_MIN_W ? ACTIVITY_W : 0;
  return {
    tooSmall: false,
    status: { x: 0, y: 0, w, h: 1 },
    nav: { x: 0, y: bodyY, w: NAV_W, h: bodyH },
    main: { x: NAV_W, y: bodyY, w: w - NAV_W - actW, h: bodyH },
    activity: actW ? { x: w - actW, y: bodyY, w: actW, h: bodyH } : null,
    outside: outH ? { x: 0, y: bodyY + bodyH, w, h: outH } : null,
    drawer: { x: 0, y: bodyY + bodyH + outH, w, h: drawerH },
    hints: { x: 0, y: h - 1, w, h: 1 },
    drawerOpen,
    drawerBoxed,
  };
}
