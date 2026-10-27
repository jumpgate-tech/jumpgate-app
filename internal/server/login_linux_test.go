//go:build linux

package server

import (
	"net/http"
	"os"
	"testing"

	"golang.org/x/sys/unix"

	"github.com/valve-tech/jumpgate/internal/vault"
)

// The real /proc lookup finds this test's own client socket and its uid.
func TestProcPeerUIDFindsTheClient(t *testing.T) {
	s, ts, _ := loginServer(t)
	got := make(chan int, 1)
	inner := s.peerUID
	s.peerUID = func(r *http.Request) (int, peerVerdict) {
		uid, v := inner(r)
		if v != peerFound {
			uid = -2
		}
		got <- uid
		return uid, v
	}
	login(t, ts, s.NewLoginCode())
	if uid := <-got; uid != os.Getuid() {
		t.Fatalf("peer uid %d, want %d", uid, os.Getuid())
	}
}

// Review Focus 5: PR_SET_DUMPABLE=0 makes /proc/self/* root-owned, and the
// login peer check reads /proc/net/tcp; it must still find the client.
func TestLoginPeerLookupWorksWhenNotDumpable(t *testing.T) {
	if err := vault.DisableCoreDumps(); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = unix.Prctl(unix.PR_SET_DUMPABLE, 1, 0, 0, 0) })
	s, ts, _ := loginServer(t)
	got := make(chan int, 1)
	inner := s.peerUID
	s.peerUID = func(r *http.Request) (int, peerVerdict) {
		uid, v := inner(r)
		if v != peerFound {
			uid = -2
		}
		got <- uid
		return uid, v
	}
	login(t, ts, s.NewLoginCode())
	if uid := <-got; uid != os.Getuid() {
		t.Fatalf("peer uid %d, want %d", uid, os.Getuid())
	}
}
