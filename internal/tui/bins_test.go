package tui

import (
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"slices"
	"strings"
	"testing"
)

func fakeExe(t *testing.T, dir, name, body string) string {
	t.Helper()
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	p := filepath.Join(dir, name)
	if err := os.WriteFile(p, []byte(body), 0o755); err != nil {
		t.Fatal(err)
	}
	return p
}

func envOf(kv map[string]string) func(string) string { return func(k string) string { return kv[k] } }

func TestResolveBin(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("unix modes")
	}
	root := t.TempDir()
	path := fakeExe(t, filepath.Join(root, "path"), "ssh", "#!/bin/sh\n")
	sib := fakeExe(t, filepath.Join(root, "app"), "ssh", "#!/bin/sh\n")
	self := filepath.Join(root, "app", "jumpgate")
	noSib := filepath.Join(root, "nothing", "jumpgate")
	pathDir := filepath.Join(root, "path")

	for _, c := range []struct {
		name string
		env  map[string]string
		self string
		want string
	}{
		{"absolute env wins", map[string]string{"JUMPGATE_SSH": path, "PATH": ""}, self, path},
		{"relative env refused", map[string]string{"JUMPGATE_SSH": "bin/ssh", "PATH": pathDir}, noSib, ""},
		{"sibling of the executable", map[string]string{"PATH": pathDir}, self, sib},
		{"then PATH", map[string]string{"PATH": pathDir}, noSib, path},
		{"relative PATH entries skipped", map[string]string{"PATH": ".:path:" + filepath.Join(root, "none")}, noSib, ""},
	} {
		if got := resolveBin("ssh", "JUMPGATE_SSH", c.self, envOf(c.env), "linux"); got != c.want {
			t.Errorf("%s: %q, want %q", c.name, got, c.want)
		}
	}

	// A planted ./ssh in the working directory is never found.
	wd, _ := os.Getwd()
	t.Cleanup(func() { _ = os.Chdir(wd) })
	if err := os.Chdir(pathDir); err != nil {
		t.Fatal(err)
	}
	if got := onPath("ssh", envOf(map[string]string{"PATH": "."}), "linux"); got != "" {
		t.Errorf("found %q through a relative PATH entry", got)
	}
}

func TestResolveSelf(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("unix modes")
	}
	jg := fakeExe(t, t.TempDir(), "jumpgate", "#!/bin/sh\n")
	if got := resolveSelf("/opt/jg/jumpgate", envOf(nil), "linux"); got != "/opt/jg/jumpgate" {
		t.Errorf("self: %q", got)
	}
	if got := resolveSelf("jumpgate", envOf(nil), "linux"); got != "" {
		t.Errorf("a relative self was kept: %q", got)
	}
	if got := resolveSelf("/opt/jg/jumpgate", envOf(map[string]string{"JUMPGATE_BIN": jg}), "linux"); got != jg {
		t.Errorf("JUMPGATE_BIN: %q", got)
	}
	if got := resolveSelf("/opt/jg/jumpgate", envOf(map[string]string{"JUMPGATE_BIN": "./jumpgate"}), "linux"); got != "" {
		t.Errorf("a relative JUMPGATE_BIN was used: %q", got)
	}
}

func TestClipboardArgv(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("unix modes")
	}
	d := t.TempDir()
	pb := fakeExe(t, d, "pbcopy", "#!/bin/sh\n")
	wl := fakeExe(t, d, "wl-copy", "#!/bin/sh\n")
	xc := fakeExe(t, d, "xclip", "#!/bin/sh\n")
	for _, c := range []struct {
		env  map[string]string
		goos string
		want []string
	}{
		{map[string]string{"PATH": d}, "darwin", []string{pb}},
		{map[string]string{"PATH": d, "WAYLAND_DISPLAY": "w", "DISPLAY": ":0"}, "linux", []string{wl}},
		{map[string]string{"PATH": d, "DISPLAY": ":0"}, "linux", []string{xc, "-selection", "clipboard"}},
		{map[string]string{"PATH": d}, "linux", nil}, // no display: OSC 52 only
		{map[string]string{"PATH": ".:relative"}, "darwin", nil},
	} {
		if got := clipboardArgv(envOf(c.env), c.goos); !slices.Equal(got, c.want) {
			t.Errorf("%v %s: %v, want %v", c.env, c.goos, got, c.want)
		}
	}
}

func TestRunCopy(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("sh")
	}
	d := t.TempDir()
	out := filepath.Join(d, "out")
	ok := fakeExe(t, d, "pbcopy", "#!/bin/sh\ncat > '"+out+"'\n")
	if got := runCopy(execCmd(ok), ok, "hello"); got != "copied (OSC 52 and "+ok+")" {
		t.Errorf("ok: %q", got)
	}
	if b, _ := os.ReadFile(out); string(b) != "hello" {
		t.Errorf("fed %q", b)
	}
	bad := fakeExe(t, d, "bad", "#!/bin/sh\nexit 3\n")
	if got := runCopy(execCmd(bad), bad, "x"); !strings.HasSuffix(got, bad+" exited 3") {
		t.Errorf("bad: %q", got)
	}
	if got := runCopy(execCmd(filepath.Join(d, "missing")), "missing", "x"); got != "copied (OSC 52); missing did not run" {
		t.Errorf("missing: %q", got)
	}
}

func execCmd(p string) *exec.Cmd { return exec.Command(p) }
