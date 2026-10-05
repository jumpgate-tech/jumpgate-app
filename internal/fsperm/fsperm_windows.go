//go:build windows

package fsperm

import (
	"errors"
	"fmt"
	"os"
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
	// CREATE_NEW fails on anything already at path, a link included. The
	// descriptor is applied as the file is created; its DACL is protected,
	// so nothing is inherited from the directory. A new file is empty, so
	// plain write access appends as well as append access would.
	h, err := windows.CreateFile(p, windows.GENERIC_READ|windows.GENERIC_WRITE,
		windows.FILE_SHARE_READ|windows.FILE_SHARE_WRITE, sa, windows.CREATE_NEW, windows.FILE_ATTRIBUTE_NORMAL, 0)
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

// ownerTrusted reports whether path's owner is one checkHandle accepts.
func ownerTrusted(path string) (bool, error) {
	h, err := openForSecurity(path, windows.READ_CONTROL)
	if err != nil {
		return false, err
	}
	defer windows.CloseHandle(h)
	sd, err := windows.GetSecurityInfo(h, windows.SE_FILE_OBJECT, windows.OWNER_SECURITY_INFORMATION)
	if err != nil {
		return false, err
	}
	owner, _, err := sd.Owner()
	if err != nil || owner == nil {
		return false, err
	}
	user, err := currentUser()
	if err != nil {
		return false, err
	}
	return trusted(owner, user), nil
}

// trusted is the set of SIDs a private file may grant access to or be owned
// by (spec D16): the user, SYSTEM, and Administrators, who can take
// ownership of any file anyway.
func trusted(sid, user *windows.SID) bool {
	return sid.Equals(user) || sid.IsWellKnown(windows.WinLocalSystemSid) || sid.IsWellKnown(windows.WinBuiltinAdministratorsSid)
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

// checkHandle requires a trusted owner and accepts allow ACEs for trusted
// SIDs only. Inherit-only ACEs do not apply to the object itself and are
// skipped; deny ACEs only take access away. Any other ACE type (callback,
// object, compound allows) is one this check cannot reason about, so it
// counts as not private.
func checkHandle(path string, h windows.Handle) error {
	sd, err := windows.GetSecurityInfo(h, windows.SE_FILE_OBJECT, windows.OWNER_SECURITY_INFORMATION|windows.DACL_SECURITY_INFORMATION)
	if err != nil {
		return fmt.Errorf("fsperm: read the security of %s: %w", path, err)
	}
	user, err := currentUser()
	if err != nil {
		return err
	}
	owner, _, err := sd.Owner()
	if err != nil {
		return fmt.Errorf("fsperm: read the owner of %s: %w", path, err)
	}
	if owner == nil || !trusted(owner, user) {
		return fmt.Errorf("%w: %s is owned by %s; delete it and let jumpgate recreate it", ErrNotPrivate, path, accountName(owner))
	}
	dacl, _, err := sd.DACL()
	if err != nil {
		return fmt.Errorf("fsperm: read the DACL of %s: %w", path, err)
	}
	if dacl == nil {
		return fmt.Errorf("%w: %s has no DACL, so everyone has full access", ErrNotPrivate, path)
	}
	for i := uint32(0); i < uint32(dacl.AceCount); i++ {
		var ace *windows.ACCESS_ALLOWED_ACE
		if err := windows.GetAce(dacl, i, &ace); err != nil {
			return err
		}
		if ace.Header.AceFlags&windows.INHERIT_ONLY_ACE != 0 || denyACETypes[ace.Header.AceType] {
			continue
		}
		if ace.Header.AceType != windows.ACCESS_ALLOWED_ACE_TYPE {
			return fmt.Errorf("%w: %s has an access entry of type %d that jumpgate cannot verify; delete the file and let jumpgate recreate it", ErrNotPrivate, path, ace.Header.AceType)
		}
		if uint32(ace.Mask)&accessMask == 0 {
			continue
		}
		sid := (*windows.SID)(unsafe.Pointer(&ace.SidStart))
		if trusted(sid, user) {
			continue
		}
		return fmt.Errorf("%w: %s grants access to %s; remove that entry (Properties > Security) or delete the file and let jumpgate recreate it", ErrNotPrivate, path, accountName(sid))
	}
	return nil
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
