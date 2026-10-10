package fsatomic

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
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

// A directory the process cannot write to must fail the write and leave the
// old file as it was: an atomic writer that half-succeeds is worse than none.
func TestWriteFileFailsInAnUnwritableDirectoryAndKeepsTheOldFile(t *testing.T) {
	testutil.RequireUnix(t)
	if os.Geteuid() == 0 {
		t.Skip("root writes through directory permissions")
	}
	dir := t.TempDir()
	path := filepath.Join(dir, "f")
	if err := WriteFile(path, []byte("old")); err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(dir, 0o500); err != nil {
		t.Fatal(err)
	}
	defer os.Chmod(dir, 0o700)
	if err := WriteFile(path, []byte("new")); err == nil {
		t.Fatal("WriteFile succeeded in a read-only directory")
	}
	if b, _ := os.ReadFile(path); string(b) != "old" {
		t.Fatalf("old file now holds %q", b)
	}
	if ents, _ := os.ReadDir(dir); len(ents) != 1 {
		t.Fatalf("temp files left behind: %v", ents)
	}
}

// A parent path that is a file cannot become a directory; the error must
// surface instead of a write somewhere else.
func TestWriteFileFailsWhenTheParentIsAFile(t *testing.T) {
	parent := filepath.Join(t.TempDir(), "file")
	if err := os.WriteFile(parent, []byte("x"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := WriteFile(filepath.Join(parent, "f"), []byte("y")); err == nil {
		t.Fatal("WriteFile succeeded under a regular file")
	}
}

// When the rename fails (here the target is a non-empty directory) the
// caller hears about it and the temp file is cleaned up, not left to pile up
// beside the target.
func TestWriteFileReportsAFailedRenameAndRemovesTheTempFile(t *testing.T) {
	testutil.RequireUnix(t)
	dir := t.TempDir()
	path := filepath.Join(dir, "target")
	if err := os.MkdirAll(filepath.Join(path, "child"), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := WriteFile(path, []byte("x")); err == nil {
		t.Fatal("WriteFile succeeded although the rename could not replace a directory")
	}
	ents, _ := os.ReadDir(dir)
	if len(ents) != 1 || ents[0].Name() != "target" {
		t.Fatalf("temp files left behind: %v", ents)
	}
}

// After a failed directory sync the data was renamed into place, but no temp
// file may remain either way.
func TestWriteFileLeavesNoTempFileWhenTheDirectorySyncFails(t *testing.T) {
	testutil.RequireUnix(t)
	orig := SyncDir
	SyncDir = func(string) error { return errors.New("injected dir sync failure") }
	defer func() { SyncDir = orig }()
	dir := t.TempDir()
	_ = WriteFile(filepath.Join(dir, "f"), []byte("x"))
	ents, _ := os.ReadDir(dir)
	if len(ents) != 1 || ents[0].Name() != "f" {
		t.Fatalf("directory holds %v, want only f", ents)
	}
}

// The real SyncDir names the directory it could not open, so an operator can
// tell which store failed.
func TestSyncDirReportsAMissingDirectoryByName(t *testing.T) {
	missing := filepath.Join(t.TempDir(), "gone")
	err := SyncDir(missing)
	if err == nil || !strings.Contains(err.Error(), missing) {
		t.Fatalf("SyncDir(missing) = %v", err)
	}
}
