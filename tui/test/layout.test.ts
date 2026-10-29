import { expect, test } from "bun:test";
import { layoutFor, type Layout } from "../src/shell/layout.ts";

const ok = (l: ReturnType<typeof layoutFor>): Layout => {
  if (l.tooSmall) throw new Error("too small");
  return l;
};

test("below 80x24 the layout is too small", () => {
  expect(layoutFor(79, 24, "open").tooSmall).toBe(true);
  expect(layoutFor(80, 23, "open").tooSmall).toBe(true);
  expect(layoutFor(80, 24, "open").tooSmall).toBe(false);
});

test("at 80x24 with the drawer open, main keeps 8+ rows and the drawer is 2 lines", () => {
  const l = ok(layoutFor(80, 24, "open", 5));
  expect(l.drawer.h).toBe(2);
  expect(l.main.h).toBeGreaterThanOrEqual(8);
  expect(l.drawerOpen).toBe(true);
});

test("below 100 columns there is no activity pane", () => {
  expect(ok(layoutFor(99, 30, "min")).activity).toBeNull();
  expect(ok(layoutFor(100, 30, "min")).activity).not.toBeNull();
});

test("auto resolves to min below 34 rows and to open at 34+", () => {
  expect(ok(layoutFor(120, 33, "auto")).drawerOpen).toBe(false);
  expect(ok(layoutFor(120, 34, "auto")).drawerOpen).toBe(true);
});

test("a tall open drawer is boxed with up to 6 lines", () => {
  expect(ok(layoutFor(120, 40, "open", 10)).drawer.h).toBe(8);
  expect(ok(layoutFor(120, 40, "open", 1)).drawer.h).toBe(3);
});

test("the regions tile the window with no overlap", () => {
  for (const [w, h, d] of [[80, 24, "open"], [100, 30, "min"], [160, 50, "open"]] as const) {
    const l = ok(layoutFor(w, h, d, 3));
    const rows = l.status.h + l.main.h + l.drawer.h + l.hints.h;
    expect(rows).toBe(h);
    expect(l.nav.w + l.main.w + (l.activity?.w ?? 0)).toBe(w);
    expect(l.nav.h).toBe(l.main.h);
  }
});
