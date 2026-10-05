//go:build windows

package executor

import (
	"fmt"
	"path/filepath"
	"unsafe"

	"golang.org/x/sys/windows"
)

// Rights that let an account change a file or, on a directory, add to it or
// delete from it.
const writeMask = 0x2 | // FILE_WRITE_DATA / FILE_ADD_FILE
	0x4 | // FILE_APPEND_DATA / FILE_ADD_SUBDIRECTORY
	0x10 | // FILE_WRITE_EA
	0x40 | // FILE_DELETE_CHILD
	0x10000 | // DELETE
	0x40000 | // WRITE_DAC
	0x80000 | // WRITE_OWNER
	0x40000000 | // GENERIC_WRITE
	0x10000000 // GENERIC_ALL

// trustedInstallerSID is NT SERVICE\TrustedInstaller, which owns much of
// %ProgramData% on a stock install.
const trustedInstallerSID = "S-1-5-80-956008885-3418522649-1831038044-1853292631-2271478464"

func currentUserSID() (*windows.SID, error) {
	u, err := windows.GetCurrentProcessToken().GetTokenUser()
	if err != nil {
		return nil, err
	}
	return u.User.Sid, nil
}

// trustedSIDs are the accounts allowed to own or write a known_hosts file:
// this user, SYSTEM, Administrators and TrustedInstaller.
func trustedSIDs() ([]*windows.SID, error) {
	me, err := currentUserSID()
	if err != nil {
		return nil, err
	}
	sids := []*windows.SID{me}
	for _, t := range []windows.WELL_KNOWN_SID_TYPE{windows.WinLocalSystemSid, windows.WinBuiltinAdministratorsSid} {
		s, err := windows.CreateWellKnownSid(t)
		if err != nil {
			return nil, err
		}
		sids = append(sids, s)
	}
	if s, err := windows.StringToSid(trustedInstallerSID); err == nil {
		sids = append(sids, s)
	}
	return sids, nil
}

func sidIn(s *windows.SID, set []*windows.SID) bool {
	for _, t := range set {
		if windows.EqualSid(s, t) {
			return true
		}
	}
	return false
}

// checkKnownHostsFile accepts path only when it and its directory are owned by
// a trusted account and no other account holds a write right on them.
func checkKnownHostsFile(path string) error {
	trusted, err := trustedSIDs()
	if err != nil {
		return err
	}
	for _, p := range []string{path, filepath.Dir(path)} {
		if err := checkTrustedDACL(p, trusted); err != nil {
			return err
		}
	}
	return nil
}

func checkTrustedDACL(path string, trusted []*windows.SID) error {
	sd, err := windows.GetNamedSecurityInfo(path, windows.SE_FILE_OBJECT, windows.OWNER_SECURITY_INFORMATION|windows.DACL_SECURITY_INFORMATION)
	if err != nil {
		return err
	}
	owner, _, err := sd.Owner()
	if err != nil || owner == nil {
		return fmt.Errorf("cannot read the owner of %s: %v", path, err)
	}
	if !sidIn(owner, trusted) {
		return fmt.Errorf("%s is owned by %s, not a trusted account", path, owner)
	}
	dacl, _, err := sd.DACL()
	if err != nil || dacl == nil {
		return fmt.Errorf("%s has no access list (anyone may write it)", path)
	}
	for i := uint32(0); i < uint32(dacl.AceCount); i++ {
		var ace *windows.ACCESS_ALLOWED_ACE
		if err := windows.GetAce(dacl, i, &ace); err != nil {
			return err
		}
		if ace.Header.AceType != windows.ACCESS_ALLOWED_ACE_TYPE || ace.Header.AceFlags&windows.INHERIT_ONLY_ACE != 0 || ace.Mask&writeMask == 0 {
			continue
		}
		if sid := (*windows.SID)(unsafe.Pointer(&ace.SidStart)); !sidIn(sid, trusted) {
			return fmt.Errorf("%s is writable by %s", path, sid)
		}
	}
	return nil
}
