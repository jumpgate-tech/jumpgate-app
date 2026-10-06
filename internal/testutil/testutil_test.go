package testutil

import (
	"os"
	"path/filepath"
	"runtime"
	"sync"
	"testing"

	"github.com/valve-tech/jumpgate/internal/fsperm"
)

// A unix socket path must fit in sun_path: 104 bytes on macOS, 108 on Linux
// and Windows. The run dir's socket is <dir>/.jumpgate/run/server.sock, so the
// base dir has to leave room for about 30 more bytes.
func TestShortTempDirLeavesRoomForASocket(t *testing.T) {
	d := ShortTempDir(t)
	if fi, err := os.Stat(d); err != nil || !fi.IsDir() {
		t.Fatalf("ShortTempDir = %q, not a directory: %v", d, err)
	}
	sock := filepath.Join(d, ".jumpgate", "run", "server.sock")
	if len(sock) > 100 {
		t.Fatalf("socket path %q is %d bytes; want at most 100", sock, len(sock))
	}
}

func TestHomeSetsHomeAndUserProfile(t *testing.T) {
	h := Home(t)
	if os.Getenv("HOME") != h || os.Getenv("USERPROFILE") != h {
		t.Fatalf("HOME=%q USERPROFILE=%q, want both %q", os.Getenv("HOME"), os.Getenv("USERPROFILE"), h)
	}
	got, err := os.UserHomeDir()
	if err != nil || got != h {
		t.Fatalf("os.UserHomeDir() = %q, %v; want %q", got, err, h)
	}
}

func TestAssertPrivateAcceptsAnOwnerOnlyFile(t *testing.T) {
	p := filepath.Join(t.TempDir(), "secret")
	if err := os.WriteFile(p, []byte("x"), 0o600); err != nil {
		t.Fatal(err)
	}
	// On Windows the new file inherits the temp dir's DACL; restrict it the
	// way every secret write does.
	if err := fsperm.MakePrivate(p); err != nil {
		t.Fatal(err)
	}
	AssertPrivate(t, p)
}

func TestRequirePOSIXShellSkipsOnWindows(t *testing.T) {
	if runtime.GOOS != "windows" {
		t.Skip("only meaningful on Windows")
	}
	ran := t.Run("inner", func(t *testing.T) {
		RequirePOSIXShell(t)
		t.Fatal("RequirePOSIXShell did not skip on Windows")
	})
	if !ran {
		t.Fatal("inner test failed instead of skipping")
	}
}

// failRecorder stands in for *testing.T so a test can observe a helper's
// failure without failing itself. Fatalf stops the calling goroutine, as the
// real one does.
type failRecorder struct {
	testing.TB
	failed bool
}

func (f *failRecorder) Helper() {}
func (f *failRecorder) Fatalf(string, ...any) {
	f.failed = true
	runtime.Goexit()
}
func (f *failRecorder) Fatal(...any) {
	f.failed = true
	runtime.Goexit()
}

func TestAssertPrivateRejectsAWorldReadableFile(t *testing.T) {
	p := filepath.Join(t.TempDir(), "open")
	if err := os.WriteFile(p, []byte("x"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := fsperm.MakePrivate(p); err != nil {
		t.Fatal(err)
	}
	Loosen(t, p)
	rec := &failRecorder{TB: t}
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		AssertPrivate(rec, p)
	}()
	wg.Wait()
	if !rec.failed {
		t.Fatal("AssertPrivate accepted a file other users can read")
	}
}
