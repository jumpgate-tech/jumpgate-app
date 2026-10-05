//go:build windows

package fsperm

import (
	"errors"
	"net"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"golang.org/x/sys/windows"
)

// M-5: an indexer or antivirus holding the target open makes MoveFileEx fail
// with a sharing violation for a moment. Rename must wait it out.
func TestRenameRetriesASharingViolation(t *testing.T) {
	dir := t.TempDir()
	a, b := filepath.Join(dir, "a"), filepath.Join(dir, "b")
	if err := os.WriteFile(a, []byte("a"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(b, []byte("b"), 0o600); err != nil {
		t.Fatal(err)
	}
	calls := 0
	old := moveFile
	moveFile = func(from, to string) error {
		calls++
		if calls < 3 {
			return &os.LinkError{Op: "rename", Old: from, New: to, Err: windows.ERROR_SHARING_VIOLATION}
		}
		return old(from, to)
	}
	t.Cleanup(func() { moveFile = old })
	if err := Rename(a, b); err != nil {
		t.Fatalf("Rename = %v after %d calls", err, calls)
	}
	if calls != 3 {
		t.Fatalf("moveFile called %d times, want 3", calls)
	}
	if got, err := os.ReadFile(b); err != nil || string(got) != "a" {
		t.Fatalf("b = %q, %v; want a", got, err)
	}
}

// The server socket is an AF_UNIX reparse point; MakePrivate must set its
// DACL without following it (D2).
func TestMakePrivateOnAUnixSocket(t *testing.T) {
	dir, err := os.MkdirTemp("", "jg")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(dir)
	sock := filepath.Join(dir, "s.sock")
	ln, err := netListenUnix(sock)
	if err != nil {
		t.Fatal(err)
	}
	defer ln.Close()
	if err := MakePrivate(sock); err != nil {
		t.Fatalf("MakePrivate(socket): %v", err)
	}
	if err := CheckPrivate(sock); err != nil {
		t.Fatalf("CheckPrivate(socket): %v", err)
	}
}

// netListenUnix is kept in the test so production code has no net import.
func netListenUnix(path string) (interface{ Close() error }, error) {
	return net.Listen("unix", path)
}

// grantEveryone adds an allow ACE for Everyone to path's DACL. fsperm's own
// tests cannot use testutil.Loosen: testutil imports fsperm.
func grantEveryone(t *testing.T, path string, mask windows.ACCESS_MASK, inherit uint32) {
	t.Helper()
	sd, err := windows.GetNamedSecurityInfo(path, windows.SE_FILE_OBJECT, windows.DACL_SECURITY_INFORMATION)
	if err != nil {
		t.Fatal(err)
	}
	old, _, err := sd.DACL()
	if err != nil {
		t.Fatal(err)
	}
	everyone, err := windows.CreateWellKnownSid(windows.WinWorldSid)
	if err != nil {
		t.Fatal(err)
	}
	acl, err := windows.ACLFromEntries([]windows.EXPLICIT_ACCESS{{
		AccessPermissions: mask,
		AccessMode:        windows.GRANT_ACCESS,
		Inheritance:       inherit,
		Trustee: windows.TRUSTEE{
			TrusteeForm:  windows.TRUSTEE_IS_SID,
			TrusteeType:  windows.TRUSTEE_IS_WELL_KNOWN_GROUP,
			TrusteeValue: windows.TrusteeValueFromSID(everyone),
		},
	}}, old)
	if err != nil {
		t.Fatal(err)
	}
	if err := windows.SetNamedSecurityInfo(path, windows.SE_FILE_OBJECT,
		windows.DACL_SECURITY_INFORMATION|windows.UNPROTECTED_DACL_SECURITY_INFORMATION, nil, nil, acl, nil); err != nil {
		t.Fatal(err)
	}
}

// looseDir is a directory whose DACL hands Everyone read access to
// everything created in it, the way a shared or redirected folder might.
func looseDir(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	grantEveryone(t, dir, windows.GENERIC_READ, windows.SUB_CONTAINERS_AND_OBJECTS_INHERIT)
	return dir
}

// A grant inherited from the parent counts as much as an explicit one.
func TestCheckPrivateSeesAnInheritedGrant(t *testing.T) {
	p := filepath.Join(looseDir(t), "plain")
	if err := os.WriteFile(p, []byte("x"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := CheckPrivate(p); !errors.Is(err, ErrNotPrivate) {
		t.Fatalf("CheckPrivate on a file inheriting Everyone:R = %v, want ErrNotPrivate", err)
	}
}

// A NULL DACL is the opposite of private: it grants everyone everything.
func TestCheckPrivateRefusesANullDACL(t *testing.T) {
	p := filepath.Join(t.TempDir(), "k")
	if err := WriteFilePrivate(p, []byte("x")); err != nil {
		t.Fatal(err)
	}
	h, err := openForSecurity(p, windows.READ_CONTROL|windows.WRITE_DAC)
	if err != nil {
		t.Fatal(err)
	}
	err = windows.SetSecurityInfo(h, windows.SE_FILE_OBJECT,
		windows.DACL_SECURITY_INFORMATION|windows.PROTECTED_DACL_SECURITY_INFORMATION, nil, nil, nil, nil)
	windows.CloseHandle(h)
	if err != nil {
		t.Fatal(err)
	}
	sd, err := windows.GetNamedSecurityInfo(p, windows.SE_FILE_OBJECT, windows.DACL_SECURITY_INFORMATION)
	if err != nil {
		t.Fatal(err)
	}
	if dacl, _, err := sd.DACL(); err != nil || dacl != nil {
		t.Fatalf("setup: the DACL is %v, %v; want a NULL DACL", dacl, err)
	}
	if err := CheckPrivate(p); !errors.Is(err, ErrNotPrivate) {
		t.Fatalf("CheckPrivate on a NULL DACL = %v, want ErrNotPrivate", err)
	}
}

// Delete access lets another user remove a secret and plant their own.
func TestCheckPrivateRefusesADeleteGrant(t *testing.T) {
	p := filepath.Join(t.TempDir(), "k")
	if err := WriteFilePrivate(p, []byte("x")); err != nil {
		t.Fatal(err)
	}
	grantEveryone(t, p, windows.DELETE, windows.NO_INHERITANCE)
	if err := CheckPrivate(p); !errors.Is(err, ErrNotPrivate) {
		t.Fatalf("CheckPrivate with Everyone:DELETE = %v, want ErrNotPrivate", err)
	}
}

func TestMakePrivateMakesTheUserTheOwner(t *testing.T) {
	p := filepath.Join(t.TempDir(), "k")
	if err := os.WriteFile(p, []byte("x"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := MakePrivate(p); err != nil {
		t.Fatal(err)
	}
	sd, err := windows.GetNamedSecurityInfo(p, windows.SE_FILE_OBJECT, windows.OWNER_SECURITY_INFORMATION)
	if err != nil {
		t.Fatal(err)
	}
	owner, _, err := sd.Owner()
	if err != nil {
		t.Fatal(err)
	}
	user, err := currentUser()
	if err != nil {
		t.Fatal(err)
	}
	if !owner.Equals(user) {
		t.Fatalf("owner = %s, want the current user %s", accountName(owner), accountName(user))
	}
}

// A secret created in a directory that hands Everyone read access is private
// from the moment it exists, before anything is written to it: there is no
// window in which another user can open a handle and keep it.
func TestSecretsArePrivateFromCreationUnderALooseParent(t *testing.T) {
	dir := looseDir(t)
	check := func(what string, f *os.File, err error) {
		t.Helper()
		if err != nil {
			t.Fatalf("%s: %v", what, err)
		}
		defer f.Close()
		if err := checkPrivateFile(f); err != nil {
			t.Fatalf("%s is not private at creation: %v", what, err)
		}
	}
	f, err := CreatePrivate(filepath.Join(dir, "key"))
	check("CreatePrivate", f, err)
	f, err = CreateTempPrivate(dir, ".t-*")
	check("CreateTempPrivate", f, err)
	f, err = OpenAppendPrivate(filepath.Join(dir, "log"))
	check("OpenAppendPrivate", f, err)
	p := filepath.Join(dir, "server.json")
	if err := WriteFilePrivate(p, []byte("{}")); err != nil {
		t.Fatal(err)
	}
	if err := CheckPrivate(p); err != nil {
		t.Fatalf("WriteFilePrivate under a loose parent: %v", err)
	}
}

// P28 on Windows: a symlinked directory whose parent lets Everyone delete
// what is in it could be swapped for another user's, so it is refused.
func TestMkdirPrivateRefusesASymlinkIntoAParentOthersCanEmpty(t *testing.T) {
	base := t.TempDir()
	shared := filepath.Join(base, "shared")
	if err := os.Mkdir(shared, 0o700); err != nil {
		t.Fatal(err)
	}
	grantEveryone(t, shared, 0x40 /* FILE_DELETE_CHILD */, windows.NO_INHERITANCE)
	real := filepath.Join(shared, "real")
	if err := os.Mkdir(real, 0o700); err != nil {
		t.Fatal(err)
	}
	link := filepath.Join(base, ".jumpgate")
	if err := os.Symlink(real, link); err != nil {
		t.Skipf("cannot create a symlink here: %v", err)
	}
	err := MkdirPrivate(link)
	if err == nil || !strings.Contains(err.Error(), "move or replace") {
		t.Fatalf("MkdirPrivate(link into a parent Everyone can empty) = %v, want a refusal", err)
	}
}
