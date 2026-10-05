//go:build windows

package fsperm

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"time"
	"unsafe"

	"golang.org/x/sys/windows"
)

// accessMask is every right that lets a SID read the content, change it,
// delete it, or change who may: an allow ACE with any of these for a stranger
// means the file is not private.
const accessMask = windows.GENERIC_ALL | windows.GENERIC_READ | windows.GENERIC_WRITE |
	0x1 /* FILE_READ_DATA */ | 0x2 /* FILE_WRITE_DATA */ | 0x4 /* FILE_APPEND_DATA */ |
	0x40 /* FILE_DELETE_CHILD */ | windows.DELETE |
	windows.WRITE_DAC | windows.WRITE_OWNER

// The deny ACE types. A deny entry only takes access away, so it never makes
// a file less private.
var denyACETypes = map[byte]bool{
	windows.ACCESS_DENIED_ACE_TYPE: true,
	0x6:                            true, // ACCESS_DENIED_OBJECT_ACE_TYPE
	0xA:                            true, // ACCESS_DENIED_CALLBACK_ACE_TYPE
	0xC:                            true, // ACCESS_DENIED_CALLBACK_OBJECT_ACE_TYPE
}

// openForSecurity opens path itself, never what it points to: the flags make
// it work on directories (BACKUP_SEMANTICS) and on AF_UNIX sockets and links
// (OPEN_REPARSE_POINT), which a by-name call would fail on or follow.
func openForSecurity(path string, access uint32) (windows.Handle, error) {
	p, err := windows.UTF16PtrFromString(path)
	if err != nil {
		return 0, err
	}
	return windows.CreateFile(p, access,
		windows.FILE_SHARE_READ|windows.FILE_SHARE_WRITE|windows.FILE_SHARE_DELETE, nil,
		windows.OPEN_EXISTING, windows.FILE_FLAG_BACKUP_SEMANTICS|windows.FILE_FLAG_OPEN_REPARSE_POINT, 0)
}

func currentUser() (*windows.SID, error) {
	tu, err := windows.GetCurrentProcessToken().GetTokenUser()
	if err != nil {
		return nil, err
	}
	return tu.User.Sid.Copy()
}

// privateSD is the descriptor every private object gets: owned by the
// current user, with a protected DACL (no inherited entries) granting full
// access to that user and SYSTEM only. A directory's entries are inherited
// by what is created in it.
func privateSD(dir bool) (*windows.SECURITY_DESCRIPTOR, error) {
	user, err := currentUser()
	if err != nil {
		return nil, err
	}
	inherit := ""
	if dir {
		inherit = "OICI"
	}
	u := user.String()
	return windows.SecurityDescriptorFromString(fmt.Sprintf("O:%sD:P(A;%s;FA;;;%s)(A;%s;FA;;;SY)", u, inherit, u, inherit))
}

func makePrivate(path string) error {
	fi, err := os.Lstat(path)
	if err != nil {
		return err
	}
	if fi.Mode()&os.ModeSymlink != 0 {
		return fmt.Errorf("fsperm: %s is a symlink; refusing to change what it points to", path)
	}
	sd, err := privateSD(fi.IsDir())
	if err != nil {
		return err
	}
	dacl, _, err := sd.DACL()
	if err != nil {
		return err
	}
	owner, _, err := sd.Owner()
	if err != nil {
		return err
	}
	if err := setSecurity(path, windows.WRITE_DAC, windows.DACL_SECURITY_INFORMATION|windows.PROTECTED_DACL_SECURITY_INFORMATION, nil, dacl); err != nil {
		return err
	}
	// Take ownership second: the new DACL grants this user WRITE_OWNER, which
	// the old one may not have. A file already owned by SYSTEM or
	// Administrators is acceptable to checkHandle, so failing to take it over
	// is not an error.
	if err := setSecurity(path, windows.WRITE_OWNER, windows.OWNER_SECURITY_INFORMATION, owner, nil); err != nil {
		if ok, oerr := ownerTrusted(path); oerr == nil && ok {
			return nil
		}
		return fmt.Errorf("fsperm: take ownership of %s: %w", path, err)
	}
	return nil
}

// The append flag needs no access change on Windows; see below.
func createPrivate(path string, _ bool) (*os.File, error) {
	sd, err := privateSD(false)
	if err != nil {
		return nil, err
	}
	p, err := windows.UTF16PtrFromString(path)
	if err != nil {
		return nil, err
	}
	sa := &windows.SecurityAttributes{SecurityDescriptor: sd}
	sa.Length = uint32(unsafe.Sizeof(*sa))
	// CREATE_NEW alone follows a dangling symlink at path and creates its
	// target; FILE_FLAG_OPEN_REPARSE_POINT makes the link itself the file
	// that already exists, so CREATE_NEW fails on it as on anything else.
	// The descriptor is applied as the file is created; its DACL is
	// protected, so nothing is inherited from the directory. A new file is
	// empty, so plain write access appends as well as append access would.
	h, err := windows.CreateFile(p, windows.GENERIC_READ|windows.GENERIC_WRITE,
		windows.FILE_SHARE_READ|windows.FILE_SHARE_WRITE, sa, windows.CREATE_NEW,
		windows.FILE_ATTRIBUTE_NORMAL|windows.FILE_FLAG_OPEN_REPARSE_POINT, 0)
	if err != nil {
		return nil, &os.PathError{Op: "open", Path: path, Err: err}
	}
	return os.NewFile(uintptr(h), path), nil
}

func openAppendExisting(path string) (*os.File, error) {
	f, err := os.OpenFile(path, os.O_WRONLY|os.O_APPEND, 0)
	if err != nil {
		return nil, err
	}
	// The handle has no WRITE_DAC, so the file is restricted by name;
	// makePrivate refuses a link, which os.OpenFile would have followed.
	if err := makePrivate(path); err != nil {
		f.Close()
		return nil, err
	}
	return f, nil
}

func setSecurity(path string, access uint32, info windows.SECURITY_INFORMATION, owner *windows.SID, dacl *windows.ACL) error {
	h, err := openForSecurity(path, windows.READ_CONTROL|access)
	if err != nil {
		return fmt.Errorf("fsperm: open %s: %w", path, err)
	}
	defer windows.CloseHandle(h)
	return windows.SetSecurityInfo(h, windows.SE_FILE_OBJECT, info, owner, nil, dacl, nil)
}

// role is what a SID is being trusted for. authorised is the one place that
// decides (ruling P31); nothing else compares SIDs.
type role int

const (
	// ownerOfSecret owns a private file or directory.
	ownerOfSecret role = iota
	// trustee holds an allow entry on a private file or directory, or the
	// right to move or replace what is in a parent of one.
	trustee
	// ownerOfParent owns a directory above a private one.
	ownerOfParent
	// ownerOfLinkTarget owns the real directory behind a symlinked state
	// directory (P28): it must be this user's own.
	ownerOfLinkTarget
)

// trustedInstallerSID is NT SERVICE\TrustedInstaller, which owns system
// directories such as C:\Windows.
const trustedInstallerSID = "S-1-5-80-956008885-3418522649-1831038044-1853292631-2271478464"

// isAdminMember reports whether this process acts as a member of
// Administrators (an elevated administrator). A seam for tests.
var isAdminMember = func(admins *windows.SID) (bool, error) { return windows.Token(0).IsMember(admins) }

// authorised is ruling P31, Windows' counterpart of unix's "this euid or
// root": the user, SYSTEM and Administrators (who can take ownership of any
// file anyway) may own a secret, hold an entry on it, or rearrange a parent;
// a parent may also be owned by TrustedInstaller. The real directory behind a
// followed symlink must be the user's, or Administrators' only while this
// process is acting as one. MakePrivate itself grants only the user and
// SYSTEM.
func authorised(sid, user *windows.SID, r role) bool {
	if sid == nil || user == nil {
		return false
	}
	isUser := sid.Equals(user)
	admins := sid.IsWellKnown(windows.WinBuiltinAdministratorsSid)
	if r == ownerOfLinkTarget {
		if isUser {
			return true
		}
		if admins {
			member, err := isAdminMember(sid)
			return err == nil && member
		}
		return false
	}
	if r == ownerOfParent && sid.String() == trustedInstallerSID {
		return true
	}
	return isUser || admins || sid.IsWellKnown(windows.WinLocalSystemSid)
}

// readSecurity reads the owner and DACL through h, which needs READ_CONTROL.
// The SIDs and ACL point into sd, which the caller keeps while using them.
func readSecurity(path string, h windows.Handle) (owner *windows.SID, dacl *windows.ACL, err error) {
	sd, err := windows.GetSecurityInfo(h, windows.SE_FILE_OBJECT, windows.OWNER_SECURITY_INFORMATION|windows.DACL_SECURITY_INFORMATION)
	if err != nil {
		return nil, nil, fmt.Errorf("read the security of %s: %w", path, err)
	}
	if owner, _, err = sd.Owner(); err != nil {
		return nil, nil, fmt.Errorf("read the owner of %s: %w", path, err)
	}
	if dacl, _, err = sd.DACL(); err != nil {
		return nil, nil, fmt.Errorf("read the DACL of %s: %w", path, err)
	}
	return owner, dacl, nil
}

func readSecurityAt(path string) (owner *windows.SID, dacl *windows.ACL, err error) {
	h, err := openForSecurity(path, windows.READ_CONTROL)
	if err != nil {
		return nil, nil, fmt.Errorf("open %s: %w", path, err)
	}
	defer windows.CloseHandle(h)
	return readSecurity(path, h)
}

// allows calls fn with the SID and mask of every allow entry that applies to
// the object itself. Inherit-only entries do not, and deny entries only take
// access away. Any other entry type (callback, object, compound allows) is
// one these checks cannot reason about, and is returned through unknown.
func allows(dacl *windows.ACL, fn func(sid *windows.SID, mask uint32) error, unknown func(aceType byte) error) error {
	for i := uint32(0); i < uint32(dacl.AceCount); i++ {
		var ace *windows.ACCESS_ALLOWED_ACE
		if err := windows.GetAce(dacl, i, &ace); err != nil {
			return err
		}
		if ace.Header.AceFlags&windows.INHERIT_ONLY_ACE != 0 || denyACETypes[ace.Header.AceType] {
			continue
		}
		if ace.Header.AceType != windows.ACCESS_ALLOWED_ACE_TYPE {
			return unknown(ace.Header.AceType)
		}
		if err := fn((*windows.SID)(unsafe.Pointer(&ace.SidStart)), uint32(ace.Mask)); err != nil {
			return err
		}
	}
	return nil
}

// ownerTrusted reports whether path's owner may own a secret.
func ownerTrusted(path string) (bool, error) {
	owner, _, err := readSecurityAt(path)
	if err != nil {
		return false, err
	}
	user, err := currentUser()
	if err != nil {
		return false, err
	}
	return authorised(owner, user, ownerOfSecret), nil
}

// ownedByCurrentUser reports whether this user owns path, for following a
// symlinked state directory (P28).
func ownedByCurrentUser(path string) (bool, error) {
	owner, _, err := readSecurityAt(path)
	if err != nil {
		return false, err
	}
	user, err := currentUser()
	if err != nil {
		return false, err
	}
	return authorised(owner, user, ownerOfLinkTarget), nil
}

// moveRights on a directory let a SID rename or replace what is in it:
// FILE_DELETE_CHILD removes any entry, DELETE removes the directory itself,
// and full control or WRITE_DAC/WRITE_OWNER can grant either. Adding files
// or subdirectories alone cannot move an existing entry (C:\ grants
// Authenticated Users that much), so it is not counted.
const moveRights = 0x40 /* FILE_DELETE_CHILD */ | windows.DELETE | windows.GENERIC_ALL | windows.WRITE_DAC | windows.WRITE_OWNER

// checkAncestors refuses a path one of whose parents someone untrusted owns
// (they can rewrite its DACL at any time) or may rearrange. path itself is
// not checked; the caller makes it private.
func checkAncestors(path string) error {
	user, err := currentUser()
	if err != nil {
		return err
	}
	for p := filepath.Dir(path); ; p = filepath.Dir(p) {
		owner, dacl, err := readSecurityAt(p)
		if err != nil {
			return fmt.Errorf("its parent: %w", err)
		}
		if err := ancestorProblem(p, owner, dacl, user); err != nil {
			return err
		}
		if filepath.Dir(p) == p {
			return nil
		}
	}
}

// ancestorProblem is the decision for one parent p, given its owner and DACL.
func ancestorProblem(p string, owner *windows.SID, dacl *windows.ACL, user *windows.SID) error {
	if !authorised(owner, user, ownerOfParent) {
		return fmt.Errorf("its parent %s belongs to %s, who could move it and put their own in its place", p, accountName(owner))
	}
	if dacl == nil {
		return fmt.Errorf("its parent %s has no DACL, so anyone could move it", p)
	}
	return allows(dacl, func(sid *windows.SID, mask uint32) error {
		if mask&moveRights != 0 && !authorised(sid, user, trustee) {
			return fmt.Errorf("its parent %s lets %s move or replace what is in it", p, accountName(sid))
		}
		return nil
	}, func(aceType byte) error {
		return fmt.Errorf("its parent %s has an access entry of type %d that jumpgate cannot verify", p, aceType)
	})
}

func checkPrivate(path string) error {
	h, err := openForSecurity(path, windows.READ_CONTROL)
	if err != nil {
		return fmt.Errorf("fsperm: open %s: %w", path, err)
	}
	defer windows.CloseHandle(h)
	return checkHandle(path, h)
}

func checkPrivateFile(f *os.File) error {
	return checkHandle(f.Name(), windows.Handle(f.Fd()))
}

func checkHandle(path string, h windows.Handle) error {
	owner, dacl, err := readSecurity(path, h)
	if err != nil {
		return fmt.Errorf("fsperm: %w", err)
	}
	user, err := currentUser()
	if err != nil {
		return err
	}
	return secretProblem(path, owner, dacl, user)
}

// secretProblem is the decision for a secret, given its owner and DACL: the
// owner must be authorised, and so must every SID an applying allow entry
// gives read, write, delete or control rights to.
func secretProblem(path string, owner *windows.SID, dacl *windows.ACL, user *windows.SID) error {
	if !authorised(owner, user, ownerOfSecret) {
		return fmt.Errorf("%w: %s is owned by %s; delete it and let jumpgate recreate it", ErrNotPrivate, path, accountName(owner))
	}
	if dacl == nil {
		return fmt.Errorf("%w: %s has no DACL, so everyone has full access", ErrNotPrivate, path)
	}
	return allows(dacl, func(sid *windows.SID, mask uint32) error {
		if mask&accessMask != 0 && !authorised(sid, user, trustee) {
			return fmt.Errorf("%w: %s grants access to %s; remove that entry (Properties > Security) or delete the file and let jumpgate recreate it", ErrNotPrivate, path, accountName(sid))
		}
		return nil
	}, func(aceType byte) error {
		return fmt.Errorf("%w: %s has an access entry of type %d that jumpgate cannot verify; delete the file and let jumpgate recreate it", ErrNotPrivate, path, aceType)
	})
}

func accountName(sid *windows.SID) string {
	if sid == nil {
		return "nobody"
	}
	if acct, dom, _, err := sid.LookupAccount(""); err == nil {
		return dom + `\` + acct
	}
	return sid.String()
}

// moveFile is a seam for the retry test.
var moveFile = os.Rename

func rename(oldpath, newpath string) error {
	var err error
	for i := 0; i < 10; i++ {
		err = moveFile(oldpath, newpath)
		if err == nil || !(errors.Is(err, windows.ERROR_SHARING_VIOLATION) || errors.Is(err, windows.ERROR_ACCESS_DENIED)) {
			return err
		}
		time.Sleep(50 * time.Millisecond)
	}
	return err
}
