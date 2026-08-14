package executor

import (
	"slices"
	"strings"
	"testing"
)

func pathOf(env []string) string {
	for _, kv := range env {
		if strings.HasPrefix(kv, "PATH=") {
			return strings.TrimPrefix(kv, "PATH=")
		}
	}
	return ""
}

func TestLocalEnvDarwinExtendsPath(t *testing.T) {
	in := []string{"HOME=/Users/x", "PATH=/usr/bin:/bin"}
	got := localEnv(in, "darwin", "/Users/x")

	path := pathOf(got)
	// The minimal Finder PATH must be preserved AND extended so docker resolves.
	if !strings.HasPrefix(path, "/usr/bin:/bin") {
		t.Errorf("existing PATH not preserved first: %q", path)
	}
	for _, want := range []string{"/opt/homebrew/bin", "/usr/local/bin", "/Users/x/.docker/bin", "/Users/x/.orbstack/bin"} {
		if !slices.Contains(strings.Split(path, ":"), want) {
			t.Errorf("PATH missing %q; got %q", want, path)
		}
	}
}

func TestLocalEnvOffDarwinUnchanged(t *testing.T) {
	in := []string{"PATH=/usr/bin:/bin"}
	got := localEnv(in, "linux", "/home/x")
	if pathOf(got) != "/usr/bin:/bin" {
		t.Errorf("PATH changed off darwin: %q", pathOf(got))
	}
}

func TestLocalEnvNoDuplicate(t *testing.T) {
	// A user whose PATH already carries /usr/local/bin must not get it twice.
	in := []string{"PATH=/usr/local/bin:/usr/bin"}
	got := pathOf(localEnv(in, "darwin", "/Users/x"))
	count := 0
	for _, d := range strings.Split(got, ":") {
		if d == "/usr/local/bin" {
			count++
		}
	}
	if count != 1 {
		t.Errorf("/usr/local/bin appears %d times, want 1: %q", count, got)
	}
}

func TestLocalEnvNoPathEntry(t *testing.T) {
	// An environment with no PATH at all gets one built from the extras.
	got := pathOf(localEnv([]string{"HOME=/Users/x"}, "darwin", "/Users/x"))
	if !strings.Contains(got, "/usr/local/bin") {
		t.Errorf("PATH not synthesized from extras: %q", got)
	}
}

func TestGuiPathDirsEmptyOffDarwin(t *testing.T) {
	if dirs := guiPathDirs("linux", "/home/x"); dirs != nil {
		t.Errorf("guiPathDirs(linux) = %v, want nil", dirs)
	}
}
