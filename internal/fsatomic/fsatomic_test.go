package fsatomic

import (
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/valve-tech/jumpgate/internal/testutil"
)

func TestWriteFileReplacesAtomicallyAt0600(t *testing.T) {
	testutil.RequireUnix(t) // directories are fsynced, which Windows cannot do
	path := filepath.Join(t.TempDir(), "sub", "f.json")
	for _, body := range []string{"one", "two"} {
		if err := WriteFile(path, []byte(body)); err != nil {
			t.Fatal(err)
		}
		b, err := os.ReadFile(path)
		if err != nil || string(b) != body {
			t.Fatalf("after writing %q: %q, %v", body, b, err)
		}
		if fi, err := os.Stat(path); err != nil || fi.Mode().Perm() != 0o600 {
			t.Fatalf("mode after writing %q: %v, %v", body, fi, err)
		}
	}
	if ents, _ := os.ReadDir(filepath.Dir(path)); len(ents) != 1 {
		t.Fatalf("temp files left behind: %v", ents)
	}
}

// If the directory fsync fails the rename is not known to be durable, so the
// caller must hear about it.
func TestWriteFileReportsAFailedDirectorySync(t *testing.T) {
	testutil.RequireUnix(t)
	orig := SyncDir
	SyncDir = func(string) error { return errors.New("injected dir sync failure") }
	defer func() { SyncDir = orig }()
	if err := WriteFile(filepath.Join(t.TempDir(), "f"), []byte("x")); err == nil {
		t.Fatal("WriteFile succeeded although the directory sync failed")
	}
}
