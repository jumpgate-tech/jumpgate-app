package tui

import (
	"testing"
	"time"
)

func TestFormatBytes(t *testing.T) {
	for _, c := range []struct {
		b     uint64
		units string
		want  string
	}{
		{0, "GB", "0 B"}, {999, "GB", "999 B"}, {1_200_000_000_000, "GB", "1.20 TB"}, {12_300_000_000, "GB", "12.3 GB"},
		{512_000_000_000, "GB", "512 GB"}, {1 << 40, "GiB", "1.00 TiB"}, {1_200_000_000_000, "GiB", "1.09 TiB"},
	} {
		if got := formatBytes(c.b, c.units); got != c.want {
			t.Errorf("formatBytes(%d, %s) = %q, want %q", c.b, c.units, got, c.want)
		}
	}
}

func TestFormatAgeAndCount(t *testing.T) {
	for d, want := range map[time.Duration]string{0: "0s", 12 * time.Second: "12s", 3 * time.Minute: "3m", 5 * time.Hour: "5h", 72 * time.Hour: "3d", -time.Second: "0s"} {
		if got := formatAge(d); got != want {
			t.Errorf("formatAge(%v) = %q, want %q", d, got, want)
		}
	}
	if formatCount(21345678) != "21,345,678" || formatCount(7) != "7" {
		t.Error("formatCount")
	}
}

func TestBarAndSparkline(t *testing.T) {
	g := asciiGlyphs
	if got := bar(50, 75, 100, 8, g); got != "####..|." {
		t.Errorf("bar = %q", got)
	}
	if got := bar(0, 0, 0, 4, g); got != "...." {
		t.Errorf("empty bar = %q", got)
	}
	if got := sparkline([]uint64{1, 2, 3, 4, 5}, g); got != "_.-=#" {
		t.Errorf("sparkline = %q", got)
	}
	if got := sparkline([]uint64{7, 7}, g); got != "__" {
		t.Errorf("flat sparkline = %q", got)
	}
}

func TestFitPadsAndCuts(t *testing.T) {
	got := fit("abcdefghij\nxy", 5, 3, "~")
	if got != "abcd~\nxy   \n     " {
		t.Errorf("fit = %q", got)
	}
}
