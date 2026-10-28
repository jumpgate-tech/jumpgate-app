//go:build !windows

package main

import "testing"

// systemd 252 hands a credential over as the service user's own 0400 file;
// systemd 255 makes it root:root 0440 and grants the service user read with
// an ACL. Both must unlock the vault; anything wider must not.
func TestSystemdCredentialOwnership(t *testing.T) {
	const svc = 996
	cases := []struct {
		name     string
		perm     uint32
		uid, gid uint32
		ok       bool
	}{
		{"systemd 252: the service's own 0400", 0o400, svc, svc, true},
		{"systemd 255: root:root 0440 with an ACL", 0o440, 0, 0, true},
		{"root:root 0400", 0o400, 0, 0, true},
		{"root-owned but group jumpgate-ctl readable", 0o440, 0, svc, false},
		{"root:root and world readable", 0o444, 0, 0, false},
		{"root:root and group writable", 0o460, 0, 0, false},
		{"root:root 0640", 0o640, 0, 0, true},
		{"the service's own 0600", 0o600, svc, svc, true},
		{"another user's 0400", 0o400, 1000, 1000, false},
		{"the service's own, group readable", 0o440, svc, svc, false},
	}
	for _, c := range cases {
		err := checkSystemdCredential(c.perm, c.uid, c.gid, svc)
		if (err == nil) != c.ok {
			t.Errorf("%s: err = %v, want ok=%v", c.name, err, c.ok)
		}
	}
}
