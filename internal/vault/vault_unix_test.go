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
