//go:build unix

package vault

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"syscall"
	"testing"

	"github.com/valve-tech/jumpgate/internal/testutil"
)

func TestOpenRefusesASymlink(t *testing.T) {
	testutil.RequireUnix(t)
	_, p := newVault(t)
	l := filepath.Join(t.TempDir(), "link")
	if err := os.Symlink(p, l); err != nil {
		t.Fatal(err)
	}
	_, err := Open(l)
	if !errors.Is(err, ErrUnsafeFile) || !strings.Contains(err.Error(), "link") {
		t.Fatalf("err = %v", err)
	}
}

func TestOpenRefusesGroupReadable(t *testing.T) {
	testutil.RequireUnix(t)
	_, p := newVault(t)
	if err := os.Chmod(p, 0o640); err != nil {
		t.Fatal(err)
	}
	if _, err := Open(p); !errors.Is(err, ErrUnsafeFile) {
		t.Fatalf("err = %v", err)
	}
	fi, _ := os.Stat(p)
	if fi.Mode().Perm() != 0o640 {
		t.Fatalf("mode = %o; the vault must never be tightened", fi.Mode().Perm())
	}
}

func TestOpenRefusesAFIFO(t *testing.T) {
	testutil.RequireUnix(t)
	p := filepath.Join(t.TempDir(), "vault")
	if err := mkfifo(p); err != nil {
		t.Skip(err)
	}
	if _, err := Open(p); !errors.Is(err, ErrUnsafeFile) {
		t.Fatalf("err = %v", err)
	}
}

func TestVaultIsPrivateFromCreation(t *testing.T) {
	testutil.RequireUnix(t)
	v, p := newVault(t)
	testutil.AssertPrivate(t, p)
	if err := v.Put("ai-key", []byte("k"), ""); err != nil {
		t.Fatal(err)
	}
	testutil.AssertPrivate(t, p)
}

// TestOpenRefusesAnotherOwner checks the owner rule. Changing a file's owner
// needs root, and root may open a service user's vault (S10), so the rule is
// tested as the decision it is.
func TestOpenRefusesAnotherOwner(t *testing.T) {
	testutil.RequireUnix(t)
	if err := ownerProblem("v", 65534, 501); !errors.Is(err, ErrUnsafeFile) {
		t.Fatalf("another owner: %v", err)
	}
	if err := ownerProblem("v", 501, 501); err != nil {
		t.Fatalf("own file: %v", err)
	}
	if err := ownerProblem("v", 990, 0); err != nil {
		t.Fatalf("root: %v", err)
	}
	if os.Geteuid() != 0 {
		return
	}
	_, p := newVault(t)
	if err := os.Chown(p, 65534, 65534); err != nil {
		t.Fatal(err)
	}
	if _, err := Open(p); err != nil {
		t.Fatalf("root could not open a service's vault: %v", err)
	}
}

func mkfifo(p string) error { return syscall.Mkfifo(p, 0o600) }

func TestSymlinkedLockIsRefused(t *testing.T) {
	testutil.RequireUnix(t)
	v, p := newVault(t)
	target := filepath.Join(t.TempDir(), "victim")
	if err := os.WriteFile(target, []byte("keep"), 0o600); err != nil {
		t.Fatal(err)
	}
	os.Remove(p + ".lock")
	if err := os.Symlink(target, p+".lock"); err != nil {
		t.Fatal(err)
	}
	if err := v.Put("ai-key", []byte("k"), ""); !errors.Is(err, ErrUnsafeFile) {
		t.Fatalf("Put = %v, want ErrUnsafeFile", err)
	}
	if b, _ := os.ReadFile(target); string(b) != "keep" {
		t.Fatal("the link target changed")
	}
	missing := filepath.Join(t.TempDir(), "missing")
	os.Remove(p + ".lock")
	if err := os.Symlink(missing, p+".lock"); err != nil {
		t.Fatal(err)
	}
	if err := v.Put("ai-key", []byte("k"), ""); !errors.Is(err, ErrUnsafeFile) {
		t.Fatalf("Put = %v", err)
	}
	if _, err := os.Lstat(missing); err == nil {
		t.Fatal("a file was created through the link")
	}
	q := filepath.Join(t.TempDir(), "vault")
	if err := os.Symlink(target, q+".lock"); err != nil {
		t.Fatal(err)
	}
	if _, err := Create(q, pass(), DefaultParams); !errors.Is(err, ErrUnsafeFile) {
		t.Fatalf("Create = %v", err)
	}
}

func TestSymlinkedVaultIsRefusedOnWrite(t *testing.T) {
	testutil.RequireUnix(t)
	v, p := newVault(t)
	real := p + ".real"
	if err := os.Rename(p, real); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(real, p); err != nil {
		t.Fatal(err)
	}
	if err := v.Put("ai-key", []byte("k"), ""); !errors.Is(err, ErrUnsafeFile) {
		t.Fatalf("Put = %v", err)
	}
	if !v.Locked() {
		t.Fatal("still unlocked")
	}
}
