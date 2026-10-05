//go:build !windows

package executor

import "testing"

func TestSocketOwnerOK(t *testing.T) {
	for _, c := range []struct {
		uid, euid uint32
		ok        bool
	}{
		{1000, 1000, true},
		{0, 1000, true},     // root-owned socket
		{1000, 0, true},     // root using the invoking user's agent (sudo)
		{1001, 1000, false}, // another user's socket
	} {
		if got := socketOwnerOK(c.uid, c.euid); got != c.ok {
			t.Errorf("socketOwnerOK(%d, %d) = %v", c.uid, c.euid, got)
		}
	}
}
