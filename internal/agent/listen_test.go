//go:build linux || darwin

package agent

import (
	"bytes"
	"context"
	"encoding/json"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"testing"

	"github.com/valve-tech/jumpgate/internal/intent"
	"github.com/valve-tech/jumpgate/internal/testutil"
)

func shortDir(t *testing.T) string {
	t.Helper()
	return testutil.ShortTempDir(t)
}

func unixClient(sock string) *http.Client {
	return &http.Client{Transport: &http.Transport{DialContext: func(ctx context.Context, _, _ string) (net.Conn, error) {
		return (&net.Dialer{}).DialContext(ctx, "unix", sock)
	}}}
}

func TestServeAnswersOverTheSocket(t *testing.T) {
	r := newRig(t, true)
	sock := filepath.Join(shortDir(t), "agent.sock")
	ln, err := Listen(sock, -1, 0o660)
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go Serve(ctx, r.a, ln)

	fi, _ := os.Stat(sock)
	if fi.Mode().Perm() != 0o660 {
		t.Errorf("socket mode %o, want 660", fi.Mode().Perm())
	}

	body, _ := json.Marshal(intent.Envelope{})
	res, err := unixClient(sock).Post("http://agent/v1/intent", "application/json", bytes.NewReader(body))
	if err != nil {
		t.Fatal(err)
	}
	defer res.Body.Close()
	var out intent.ReceiptEnvelope
	if err := json.NewDecoder(res.Body).Decode(&out); err != nil || out.Receipt.Status != intent.StatusRejected {
		t.Fatalf("status %d, %v", out.Receipt.Status, err)
	}

	big := bytes.Repeat([]byte("a"), maxBody+1)
	res2, err := unixClient(sock).Post("http://agent/v1/intent", "application/json", bytes.NewReader(big))
	if err != nil {
		t.Fatal(err)
	}
	res2.Body.Close()
	if res2.StatusCode != http.StatusRequestEntityTooLarge {
		t.Errorf("oversized body: %d, want 413", res2.StatusCode)
	}
}

// Listen replaces a stale socket but never deletes anything else.
func TestListenRefusesToDeleteANonSocket(t *testing.T) {
	path := filepath.Join(shortDir(t), "agent.sock")
	if err := os.WriteFile(path, []byte("precious"), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := Listen(path, -1, 0o660); err == nil {
		t.Fatal("Listen removed a regular file")
	}
}

// A socket left by a previous run is replaced.
func TestListenReplacesAStaleSocket(t *testing.T) {
	path := filepath.Join(shortDir(t), "agent.sock")
	old, err := net.Listen("unix", path)
	if err != nil {
		t.Fatal(err)
	}
	// Keep the file: a closed listener would unlink it.
	old.(*net.UnixListener).SetUnlinkOnClose(false)
	old.Close()
	ln, err := Listen(path, -1, 0o660)
	if err != nil {
		t.Fatalf("stale socket not replaced: %v", err)
	}
	ln.Close()
}

func TestPeerGateAllowsOwnUIDAndEnrolledUIDsOnly(t *testing.T) {
	p := Policy{LocalUIDs: []int{4242}}
	if !peerAllowed(p, 0, nil, -1) {
		t.Error("root was refused")
	}
	if !peerAllowed(p, os.Getuid(), nil, -1) {
		t.Error("the agent's own uid was refused")
	}
	if !peerAllowed(p, 4242, nil, -1) {
		t.Error("an enrolled local uid was refused")
	}
	if peerAllowed(p, 777, nil, -1) {
		t.Error("an unknown uid was allowed")
	}
	if !peerAllowed(p, 777, []int{55}, 55) {
		t.Error("a member of the jumpgate group was refused")
	}
}

// R23: the socket is world-connectable only when a local uid is enrolled.
// connect(2) needs write permission on the socket file, so an enrolled uid
// outside group jumpgate could never reach the Accept-time peer gate at 0660.
// Without local uids the kernel's group check stays as defence in depth.
func TestSocketModeFollowsLocalUIDs(t *testing.T) {
	if m := SocketMode(Policy{}); m != 0o660 {
		t.Errorf("no local uids: mode %o, want 660", m)
	}
	if m := SocketMode(Policy{LocalUIDs: []int{1000}}); m != 0o666 {
		t.Errorf("local uid enrolled: mode %o, want 666", m)
	}
}

func TestListenAppliesTheGivenMode(t *testing.T) {
	sock := filepath.Join(shortDir(t), "agent.sock")
	ln, err := Listen(sock, -1, 0o666)
	if err != nil {
		t.Fatal(err)
	}
	defer ln.Close()
	fi, err := os.Stat(sock)
	if err != nil || fi.Mode().Perm() != 0o666 {
		t.Fatalf("socket mode %v, %v; want 666", fi.Mode().Perm(), err)
	}
}

// ApplySocketMode re-chmods a live socket after enroll; it leaves a missing
// socket alone and never touches a file that is not a socket.
func TestApplySocketMode(t *testing.T) {
	dir := shortDir(t)
	sock := filepath.Join(dir, "agent.sock")
	ln, err := Listen(sock, -1, 0o660)
	if err != nil {
		t.Fatal(err)
	}
	defer ln.Close()
	if err := ApplySocketMode(sock, Policy{LocalUIDs: []int{1000}}); err != nil {
		t.Fatal(err)
	}
	if fi, _ := os.Stat(sock); fi.Mode().Perm() != 0o666 {
		t.Errorf("socket mode %o after enroll, want 666", fi.Mode().Perm())
	}
	if err := ApplySocketMode(filepath.Join(dir, "absent.sock"), Policy{LocalUIDs: []int{1}}); err != nil {
		t.Errorf("missing socket: %v, want nil", err)
	}
	plain := filepath.Join(dir, "plain")
	if err := os.WriteFile(plain, nil, 0o600); err != nil {
		t.Fatal(err)
	}
	if err := ApplySocketMode(plain, Policy{LocalUIDs: []int{1}}); err == nil {
		t.Error("chmodded a regular file")
	}
	if fi, _ := os.Stat(plain); fi.Mode().Perm() != 0o600 {
		t.Errorf("regular file mode changed to %o", fi.Mode().Perm())
	}
}
