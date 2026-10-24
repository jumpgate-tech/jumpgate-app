package tui

import "testing"

func TestLayoutForMainFloor(t *testing.T) {
	for w := 80; w <= 200; w++ {
		for h := 24; h <= 60; h++ {
			for _, d := range []drawerState{drawerAuto, drawerOpen, drawerMin} {
				l := layoutFor(w, h, d)
				if l.Main.W < 50 || l.Main.H < 8 {
					t.Fatalf("%dx%d %v: main %+v below floor", w, h, d, l.Main)
				}
				if l.Left.W+l.Main.W != w {
					t.Fatalf("%dx%d %v: left %d + main %d != %d", w, h, d, l.Left.W, l.Main.W, w)
				}
				if got := l.Header.H + l.Main.H + l.Drawer.H + l.Hints.H; got != h {
					t.Fatalf("%dx%d %v: rows %d != %d", w, h, d, got, h)
				}
				rs := []rect{l.Header, l.Left, l.Main, l.Drawer, l.Hints}
				for i := range rs {
					for j := i + 1; j < len(rs); j++ {
						if overlaps(rs[i], rs[j]) {
							t.Fatalf("%dx%d %v: rects %d %+v and %d %+v overlap", w, h, d, i, rs[i], j, rs[j])
						}
					}
				}
			}
		}
	}
}

func overlaps(a, b rect) bool {
	if a.W == 0 || a.H == 0 || b.W == 0 || b.H == 0 {
		return false
	}
	return a.X < b.X+b.W && b.X < a.X+a.W && a.Y < b.Y+b.H && b.Y < a.Y+a.H
}

func TestLayoutForBreakpoints(t *testing.T) {
	l := layoutFor(80, 24, drawerAuto)
	if l.Left.W != 0 || l.Main.Y != 1 || l.DrawerOpen || l.Drawer.H != 0 || l.Hints.H != 1 {
		t.Errorf("80x24 auto: %+v", l)
	}
	l = layoutFor(80, 24, drawerOpen)
	if !l.DrawerOpen || !l.DrawerCompact || l.Drawer.H != 1 {
		t.Errorf("80x24 open: %+v", l)
	}
	l = layoutFor(100, 30, drawerAuto)
	if l.Left.W != 24 || l.Main.Y != 1 || l.DrawerOpen {
		t.Errorf("100x30 auto: %+v", l)
	}
	l = layoutFor(120, 40, drawerAuto)
	if !l.DrawerOpen || l.DrawerCompact || l.Drawer.H != 5 {
		t.Errorf("120x40 auto: %+v", l)
	}
	l = layoutFor(120, 40, drawerMin)
	if l.Drawer.H != 0 || l.DrawerOpen {
		t.Errorf("120x40 min: %+v", l)
	}
}

func TestDrawerPrefRoundTrip(t *testing.T) {
	for _, s := range []string{"auto", "open", "min"} {
		if got := drawerFromPref(s).pref(); got != s {
			t.Errorf("round trip %q = %q", s, got)
		}
	}
	for _, s := range []string{"", "bogus"} {
		if got := drawerFromPref(s); got != drawerAuto {
			t.Errorf("drawerFromPref(%q) = %v, want auto", s, got)
		}
	}
}
