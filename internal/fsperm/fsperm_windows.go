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

// accessMask is every right that lets a SID read the content, change it, or
// change who may: an allow ACE with any of these for a stranger means the
// file is not private.
const accessMask = windows.GENERIC_ALL | windows.GENERIC_READ | windows.GENERIC_WRITE |
	0x1 /* FILE_READ_DATA */ | 0x2 /* FILE_WRITE_DATA */ | 0x4 /* FILE_APPEND_DATA */ |
	windows.WRITE_DAC | windows.WRITE_OWNER

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

func makePrivate(path string) error {
	fi, err := os.Lstat(path)
	if err != nil {
		return err
	}
	if fi.Mode()&os.ModeSymlink != 0 {
		return fmt.Errorf("fsperm: %s is a symlink; refusing to change what it points to", path)
	}
	user, err := currentUser()
	if err != nil {
		return err
	}
	system, err := windows.CreateWellKnownSid(windows.WinLocalSystemSid)
	if err != nil {
		return err
	}
	inherit := uint32(windows.NO_INHERITANCE)
	if fi.IsDir() {
		inherit = windows.SUB_CONTAINERS_AND_OBJECTS_INHERIT
	}
	grant := func(sid *windows.SID, kind windows.TRUSTEE_TYPE) windows.EXPLICIT_ACCESS {
		return windows.EXPLICIT_ACCESS{
			AccessPermissions: windows.GENERIC_ALL,
			AccessMode:        windows.GRANT_ACCESS,
			Inheritance:       inherit,
			Trustee: windows.TRUSTEE{
				TrusteeForm:  windows.TRUSTEE_IS_SID,
				TrusteeType:  kind,
				TrusteeValue: windows.TrusteeValueFromSID(sid),
			},
		}
	}
	acl, err := windows.ACLFromEntries([]windows.EXPLICIT_ACCESS{
		grant(user, windows.TRUSTEE_IS_USER),
		grant(system, windows.TRUSTEE_IS_WELL_KNOWN_GROUP),
	}, nil)
	if err != nil {
		return err
	}
	h, err := openForSecurity(path, windows.READ_CONTROL|windows.WRITE_DAC)
	if err != nil {
		return fmt.Errorf("fsperm: open %s: %w", path, err)
	}
	defer windows.CloseHandle(h)
	// PROTECTED drops inherited ACEs: a profile's inherited grants are
	// exactly what this must not depend on.
	return windows.SetSecurityInfo(h, windows.SE_FILE_OBJECT,
		windows.DACL_SECURITY_INFORMATION|windows.PROTECTED_DACL_SECURITY_INFORMATION, nil, nil, acl, nil)
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

// checkHandle accepts allow ACEs for the owner, SYSTEM and Administrators
// only (spec D16). Administrators can take ownership of any file anyway, and
// default profile ACLs include them. Inherit-only ACEs do not apply to the
// object itself and are skipped.
func checkHandle(path string, h windows.Handle) error {
	sd, err := windows.GetSecurityInfo(h, windows.SE_FILE_OBJECT, windows.DACL_SECURITY_INFORMATION)
	if err != nil {
		return fmt.Errorf("fsperm: read the DACL of %s: %w", path, err)
	}
	dacl, _, err := sd.DACL()
	if err != nil {
		return fmt.Errorf("fsperm: read the DACL of %s: %w", path, err)
	}
	if dacl == nil {
		return fmt.Errorf("%w: %s has no DACL, so everyone has full access", ErrNotPrivate, path)
	}
	user, err := currentUser()
	if err != nil {
		return err
	}
	for i := uint32(0); i < uint32(dacl.AceCount); i++ {
		var ace *windows.ACCESS_ALLOWED_ACE
		if err := windows.GetAce(dacl, i, &ace); err != nil {
			return err
		}
		if ace.Header.AceType != windows.ACCESS_ALLOWED_ACE_TYPE || ace.Header.AceFlags&windows.INHERIT_ONLY_ACE != 0 {
			continue
		}
		if uint32(ace.Mask)&accessMask == 0 {
			continue
		}
		sid := (*windows.SID)(unsafe.Pointer(&ace.SidStart))
		if sid.Equals(user) || sid.IsWellKnown(windows.WinLocalSystemSid) || sid.IsWellKnown(windows.WinBuiltinAdministratorsSid) {
			continue
		}
		who := sid.String()
		if acct, dom, _, lerr := sid.LookupAccount(""); lerr == nil {
			who = dom + `\` + acct
		}
		return fmt.Errorf("%w: %s grants access to %s; remove that entry (Properties > Security) or delete the file and let jumpgate recreate it", ErrNotPrivate, path, who)
	}
	return nil
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
