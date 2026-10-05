//go:build linux

package server

import (
	"net/http"
	"os"
	"testing"
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
