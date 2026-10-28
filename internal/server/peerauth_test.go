//go:build !windows

package server

import (
	"context"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"syscall"
	"testing"
	"testing/fstest"
	"time"

	"github.com/valve-tech/jumpgate/internal/peercred"
	"github.com/valve-tech/jumpgate/internal/testutil"
)

func TestAdmitPeer(t *testing.T) {
	const self, admin = 999, 4242
	for _, tc := range []struct {
		c    peercred.Cred
		want bool
	}{
		{peercred.Cred{UID: 0}, true},
		{peercred.Cred{UID: self}, true},
		{peercred.Cred{UID: 4000, GIDs: []int{admin}}, true},
		{peercred.Cred{UID: 4000, GIDs: []int{9}}, false},
	} {
		if got := admitPeer(tc.c, self, admin); got != tc.want {
			t.Errorf("admitPeer(%+v) = %v, want %v", tc.c, got, tc.want)
		}
	}
}

// startSystemSocket serves s on a system socket and returns its path once it
// accepts.
func startSystemSocket(t *testing.T, s *Server) string {
	t.Helper()
	sock := filepath.Join(testutil.ShortTempDir(t), "s.sock")
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})
	go func() { _ = s.ServeUnixSystem(ctx, sock); close(done) }()
	t.Cleanup(func() { cancel(); <-done })
	for i := 0; i < 100; i++ {
		if c, err := net.Dial("unix", sock); err == nil {
			c.Close()
			return sock
		}
		time.Sleep(20 * time.Millisecond)
	}
	t.Fatal("system socket never came up")
	return ""
}

func unixClient(sock string) *http.Client {
	return &http.Client{Transport: &http.Transport{DialContext: func(ctx context.Context, _, _ string) (net.Conn, error) {
		return (&net.Dialer{}).DialContext(ctx, "unix", sock)
	}}}
}

func TestSystemSocketAdmitsByPeerNotToken(t *testing.T) {
	s := New(Config{System: true, AdminGID: os.Getgid(), Token: "tok", UI: fstest.MapFS{}})
	sock := startSystemSocket(t, s)
	res, err := unixClient(sock).Get("http://x/api/health")
	if err != nil {
		t.Fatal(err)
	}
	res.Body.Close()
	if res.StatusCode != http.StatusOK {
		t.Fatalf("health with no token over the system socket: %d, want 200", res.StatusCode)
	}
}

func TestSystemSocketModeAndGroup(t *testing.T) {
	s := New(Config{System: true, AdminGID: os.Getgid(), Token: "tok", UI: fstest.MapFS{}})
	sock := startSystemSocket(t, s)
	fi, err := os.Stat(sock)
	if err != nil {
		t.Fatal(err)
	}
	if m := fi.Mode() & 0o777; m != 0o660 {
		t.Errorf("mode = %o, want 660", m)
	}
	if gid := int(fi.Sys().(*syscall.Stat_t).Gid); gid != os.Getgid() {
		t.Errorf("gid = %d, want %d", gid, os.Getgid())
	}
}

func TestTCPListenerStillNeedsTheToken(t *testing.T) {
	s := New(Config{System: true, AdminGID: os.Getgid(), Token: "tok", UI: fstest.MapFS{}})
	ts := httptest.NewServer(s.Handler())
	defer ts.Close()
	res, err := http.Get(ts.URL + "/api/health")
	if err != nil {
		t.Fatal(err)
	}
	res.Body.Close()
	if res.StatusCode != http.StatusUnauthorized {
		t.Fatalf("TCP health without a token: %d, want 401", res.StatusCode)
	}
}

func TestLoginCodeMintRecordsTheSocketPeer(t *testing.T) {
	s := New(Config{System: true, AdminGID: os.Getgid(), Token: "tok", UI: fstest.MapFS{}})
	sock := startSystemSocket(t, s)
	res, err := unixClient(sock).Post("http://x/api/login-code", "application/json", nil)
	if err != nil {
		t.Fatal(err)
	}
	res.Body.Close()
	if res.StatusCode != http.StatusOK {
		t.Fatalf("mint over the system socket: %d", res.StatusCode)
	}
	s.codes.mu.Lock()
	defer s.codes.mu.Unlock()
	if len(s.codes.list) != 1 || s.codes.list[0].uid != os.Getuid() {
		t.Fatalf("codes = %+v, want one for uid %d", s.codes.list, os.Getuid())
	}
}
