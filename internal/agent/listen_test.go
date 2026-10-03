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
)

func shortDir(t *testing.T) string {
	t.Helper()
	d, err := os.MkdirTemp("/tmp", "jga")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { os.RemoveAll(d) })
	return d
}

func unixClient(sock string) *http.Client {
	return &http.Client{Transport: &http.Transport{DialContext: func(ctx context.Context, _, _ string) (net.Conn, error) {
		return (&net.Dialer{}).DialContext(ctx, "unix", sock)
	}}}
}

func TestServeAnswersOverTheSocket(t *testing.T) {
	r := newRig(t, true)
	sock := filepath.Join(shortDir(t), "agent.sock")
	ln, err := Listen(sock, -1)
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
	if _, err := Listen(path, -1); err == nil {
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
	ln, err := Listen(path, -1)
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
