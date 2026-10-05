package fsperm_test

import (
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/valve-tech/jumpgate/internal/fsperm"
	"github.com/valve-tech/jumpgate/internal/testutil"
)

func TestMkdirPrivateCreatesAnOwnerOnlyDir(t *testing.T) {
	d := filepath.Join(t.TempDir(), "a", "b")
	if err := fsperm.MkdirPrivate(d); err != nil {
		t.Fatal(err)
	}
	if err := fsperm.CheckPrivate(d); err != nil {
		t.Fatalf("CheckPrivate after MkdirPrivate: %v", err)
	}
	testutil.AssertPrivate(t, d)
}

// M-6: a ~/.jumpgate an older release (or a umask) left open must be
// tightened, not trusted because it already exists.
func TestMkdirPrivateTightensAnExistingDir(t *testing.T) {
	d := filepath.Join(t.TempDir(), "loose")
	if err := os.Mkdir(d, 0o755); err != nil {
		t.Fatal(err)
	}
	testutil.Loosen(t, d)
	if err := fsperm.CheckPrivate(d); !errors.Is(err, fsperm.ErrNotPrivate) {
		t.Fatalf("CheckPrivate on a loosened dir = %v, want ErrNotPrivate", err)
	}
	if err := fsperm.MkdirPrivate(d); err != nil {
		t.Fatal(err)
	}
	if err := fsperm.CheckPrivate(d); err != nil {
		t.Fatalf("still not private after MkdirPrivate: %v", err)
	}
}

func TestWriteFilePrivateWritesAnOwnerOnlyFileAtomically(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, "server.json")
	if err := os.WriteFile(p, []byte("old"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := fsperm.WriteFilePrivate(p, []byte("new")); err != nil {
		t.Fatal(err)
	}
	b, err := os.ReadFile(p)
	if err != nil || string(b) != "new" {
		t.Fatalf("content = %q, %v; want new", b, err)
	}
	if err := fsperm.CheckPrivate(p); err != nil {
		t.Fatalf("CheckPrivate: %v", err)
	}
	entries, _ := os.ReadDir(dir)
	if len(entries) != 1 {
		t.Fatalf("temp files left behind: %v", entries)
	}
}

func TestCheckPrivateRefusesALoosenedFile(t *testing.T) {
	p := filepath.Join(t.TempDir(), "k")
	if err := fsperm.WriteFilePrivate(p, []byte("x")); err != nil {
		t.Fatal(err)
	}
	testutil.Loosen(t, p)
	err := fsperm.CheckPrivate(p)
	if !errors.Is(err, fsperm.ErrNotPrivate) {
		t.Fatalf("CheckPrivate = %v, want ErrNotPrivate", err)
	}
	f, ferr := os.Open(p)
	if ferr != nil {
		t.Fatal(ferr)
	}
	defer f.Close()
	if err := fsperm.CheckPrivateFile(f); !errors.Is(err, fsperm.ErrNotPrivate) {
		t.Fatalf("CheckPrivateFile = %v, want ErrNotPrivate", err)
	}
}

func TestMakePrivateFixesALoosenedFile(t *testing.T) {
	p := filepath.Join(t.TempDir(), "k")
	if err := os.WriteFile(p, []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	testutil.Loosen(t, p)
	if err := fsperm.MakePrivate(p); err != nil {
		t.Fatal(err)
	}
	if err := fsperm.CheckPrivate(p); err != nil {
		t.Fatalf("CheckPrivate after MakePrivate: %v", err)
	}
}

// M-4: MakePrivate restricts the link itself or nothing; it must never
// change what a planted link points to. On Windows creating a symlink needs
// a privilege (or developer mode) the runner may lack; then there is nothing
// to test.
func TestMakePrivateRefusesASymlink(t *testing.T) {
	dir := t.TempDir()
	target := filepath.Join(dir, "target")
	if err := os.WriteFile(target, []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	testutil.Loosen(t, target)
	link := filepath.Join(dir, "link")
	if err := os.Symlink(target, link); err != nil {
		t.Skipf("cannot create a symlink here: %v", err)
	}
	if err := fsperm.MakePrivate(link); err == nil {
		t.Fatal("MakePrivate accepted a symlink")
	}
	if err := fsperm.CheckPrivate(target); !errors.Is(err, fsperm.ErrNotPrivate) {
		t.Fatalf("the link's target was changed: CheckPrivate(target) = %v, want ErrNotPrivate", err)
	}
}

func TestRenameReplacesTheTarget(t *testing.T) {
	dir := t.TempDir()
	a, b := filepath.Join(dir, "a"), filepath.Join(dir, "b")
	if err := os.WriteFile(a, []byte("a"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(b, []byte("b"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := fsperm.Rename(a, b); err != nil {
		t.Fatal(err)
	}
	if got, err := os.ReadFile(b); err != nil || string(got) != "a" {
		t.Fatalf("b = %q, %v; want a", got, err)
	}
	if _, err := os.Lstat(a); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("a still exists after Rename: %v", err)
	}
}

// On unix CheckPrivate looks at the path itself, as on Windows: a link is
// not a private file whatever its target's mode. (On Windows a link has its
// own DACL, which CheckPrivate checks like any other.)
func TestCheckPrivateRefusesASymlinkOnUnix(t *testing.T) {
	testutil.RequireUnix(t)
	dir := t.TempDir()
	target := filepath.Join(dir, "target")
	if err := fsperm.WriteFilePrivate(target, []byte("x")); err != nil {
		t.Fatal(err)
	}
	link := filepath.Join(dir, "link")
	if err := os.Symlink(target, link); err != nil {
		t.Fatal(err)
	}
	if err := fsperm.CheckPrivate(link); !errors.Is(err, fsperm.ErrNotPrivate) {
		t.Fatalf("CheckPrivate(symlink) = %v, want ErrNotPrivate", err)
	}
}

func TestCreatePrivateMakesAPrivateFileAndNeverReplacesOne(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, "key")
	f, err := fsperm.CreatePrivate(p)
	if err != nil {
		t.Fatal(err)
	}
	if err := fsperm.CheckPrivateFile(f); err != nil {
		t.Fatalf("CheckPrivateFile on a new file: %v", err)
	}
	if _, err := f.WriteString("secret"); err != nil {
		t.Fatal(err)
	}
	f.Close()
	testutil.AssertPrivate(t, p)
	if _, err := fsperm.CreatePrivate(p); !errors.Is(err, fs.ErrExist) {
		t.Fatalf("CreatePrivate over an existing file = %v, want fs.ErrExist", err)
	}
	link := filepath.Join(dir, "link")
	if err := os.Symlink(filepath.Join(dir, "absent"), link); err != nil {
		t.Skipf("cannot create a symlink here: %v", err)
	}
	if f, err := fsperm.CreatePrivate(link); err == nil {
		f.Close()
		t.Fatal("CreatePrivate wrote through a dangling symlink")
	}
	if _, err := os.Lstat(filepath.Join(dir, "absent")); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("the symlink's target was created: %v", err)
	}
}

func TestCreateTempPrivateMakesDistinctPrivateFiles(t *testing.T) {
	dir := t.TempDir()
	a, err := fsperm.CreateTempPrivate(dir, ".x.tmp-*.json")
	if err != nil {
		t.Fatal(err)
	}
	defer a.Close()
	b, err := fsperm.CreateTempPrivate(dir, ".x.tmp-*.json")
	if err != nil {
		t.Fatal(err)
	}
	defer b.Close()
	if a.Name() == b.Name() {
		t.Fatalf("two temp files share the name %s", a.Name())
	}
	for _, f := range []*os.File{a, b} {
		base := filepath.Base(f.Name())
		if filepath.Dir(f.Name()) != dir || !strings.HasPrefix(base, ".x.tmp-") || !strings.HasSuffix(base, ".json") {
			t.Fatalf("temp file %s does not match the pattern in %s", f.Name(), dir)
		}
		if err := fsperm.CheckPrivateFile(f); err != nil {
			t.Fatalf("CheckPrivateFile(%s): %v", f.Name(), err)
		}
	}
}

func TestOpenAppendPrivateCreatesOrTightensAndAppends(t *testing.T) {
	p := filepath.Join(t.TempDir(), "known_hosts")
	f, err := fsperm.OpenAppendPrivate(p)
	if err != nil {
		t.Fatal(err)
	}
	if err := fsperm.CheckPrivateFile(f); err != nil {
		t.Fatalf("CheckPrivateFile on a new file: %v", err)
	}
	f.WriteString("a\n")
	f.Close()
	testutil.Loosen(t, p)
	f, err = fsperm.OpenAppendPrivate(p)
	if err != nil {
		t.Fatal(err)
	}
	f.WriteString("b\n")
	f.Close()
	if err := fsperm.CheckPrivate(p); err != nil {
		t.Fatalf("an existing loose file was not tightened: %v", err)
	}
	if got, err := os.ReadFile(p); err != nil || string(got) != "a\nb\n" {
		t.Fatalf("content = %q, %v; want both lines", got, err)
	}
}
