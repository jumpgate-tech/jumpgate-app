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
				if l.Left.W+l.Main.W+l.Right.W != w {
					t.Fatalf("%dx%d %v: left %d + main %d + right %d != %d", w, h, d, l.Left.W, l.Main.W, l.Right.W, w)
				}
				if got := l.Header.H + l.KPI.H + l.Main.H + l.Drawer.H + l.Hints.H; got != h {
					t.Fatalf("%dx%d %v: rows %d != %d", w, h, d, got, h)
				}
				rs := []rect{l.Header, l.KPI, l.Left, l.Main, l.Right, l.Drawer, l.Hints}
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
	if l.Left.W != 0 || l.Main.Y != 2 || l.DrawerOpen || l.Drawer.H != 0 || l.Hints.H != 1 {
		t.Errorf("80x24 auto: %+v", l)
	}
	l = layoutFor(80, 24, drawerOpen)
	if !l.DrawerOpen || !l.DrawerCompact || l.Drawer.H != 1 {
		t.Errorf("80x24 open: %+v", l)
	}
	l = layoutFor(100, 30, drawerAuto)
	if l.Left.W != 24 || l.Main.Y != 5 || l.DrawerOpen {
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

func TestLayoutForKPI(t *testing.T) {
	for _, c := range []struct{ w, h, kpiH, mainY int }{{80, 24, 1, 2}, {80, 29, 1, 2}, {100, 30, 4, 5}, {120, 40, 4, 5}} {
		l := layoutFor(c.w, c.h, drawerAuto)
		if l.KPI != (rect{0, 1, c.w, c.kpiH}) || l.Main.Y != c.mainY || l.Left.Y != map[bool]int{true: c.mainY, false: 0}[l.Left.W > 0] {
			t.Errorf("%dx%d: %+v", c.w, c.h, l)
		}
	}
}

func TestLayoutForRight(t *testing.T) {
	for _, c := range []struct{ w, h, rightW int }{{119, 40, 0}, {120, 40, 32}, {148, 40, 39}, {160, 40, 40}, {200, 60, 40}, {100, 30, 0}} {
		l := layoutFor(c.w, c.h, drawerAuto)
		if l.Right.W != c.rightW {
			t.Errorf("%dx%d: right %+v, want width %d", c.w, c.h, l.Right, c.rightW)
		}
		if c.rightW > 0 && (l.Right.X != c.w-c.rightW || l.Right.Y != l.Main.Y || l.Right.H != l.Main.H) {
			t.Errorf("%dx%d: right %+v main %+v", c.w, c.h, l.Right, l.Main)
		}
	}
	if l := layoutFor(120, 40, drawerAuto); l.Right != (rect{88, 5, 32, 29}) || l.Main.W != 64 {
		t.Errorf("120x40: right %+v main %+v", l.Right, l.Main)
	}
}
