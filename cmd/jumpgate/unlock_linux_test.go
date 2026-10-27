//go:build linux

package main

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/hex"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/valve-tech/jumpgate/internal/daemon"
	"github.com/valve-tech/jumpgate/internal/testutil"
)

// descendants lists every process whose parent chain reaches root.
func descendants(root int) []int {
	ents, _ := os.ReadDir("/proc")
	parent := map[int]int{}
	for _, e := range ents {
		pid, err := strconv.Atoi(e.Name())
		if err != nil {
			continue
		}
		b, err := os.ReadFile(filepath.Join("/proc", e.Name(), "status"))
		if err != nil {
			continue
		}
		for _, l := range strings.Split(string(b), "\n") {
			if v, ok := strings.CutPrefix(l, "PPid:"); ok {
				parent[pid], _ = strconv.Atoi(strings.TrimSpace(v))
			}
		}
	}
	var out []int
	for pid := range parent {
		for p, n := pid, 0; p > 1 && n < 64; p, n = parent[p], n+1 {
			if p == root {
				out = append(out, pid)
				break
			}
		}
	}
	return out
}

// sampleArgvEnv reports whether secret is in the argv or environment of a
// process below this test.
func sampleArgvEnv(secret []byte) (pid int, where string, found bool) {
	for _, p := range descendants(os.Getpid()) {
		for _, f := range []string{"cmdline", "environ"} {
			b, err := os.ReadFile(filepath.Join("/proc", strconv.Itoa(p), f))
			if err == nil && bytes.Contains(b, secret) {
				return p, f, true
			}
		}
	}
	return 0, "", false
}

func TestUnlockPassphraseNeverInArgvOrEnviron(t *testing.T) {
	if testing.Short() {
		t.Skip("builds and runs the binary")
	}
	testutil.Home(t)
	raw := make([]byte, 12)
	if _, err := rand.Read(raw); err != nil {
		t.Fatal(err)
	}
	pass := hex.EncodeToString(raw) // 24 characters
	captureStdio(t)
	useFakeTTY(t, pass, pass)
	if code := keysInit([]string{"--store", "vault"}); code != 0 {
		t.Fatalf("keys init exit %d", code)
	}

	bin := filepath.Join(t.TempDir(), "jumpgate")
	if out, err := exec.Command("go", "build", "-o", bin, ".").CombinedOutput(); err != nil {
		t.Fatalf("go build: %v\n%s", err, out)
	}
	serve := exec.Command(bin, "serve", "--bind", "127.0.0.1:0")
	var serveOut bytes.Buffer
	serve.Stdout, serve.Stderr = &serveOut, &serveOut
	if err := serve.Start(); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = serve.Process.Kill(); _ = serve.Wait() })
	for deadline := time.Now().Add(20 * time.Second); ; {
		if _, ok, _ := daemon.Find(context.Background()); ok {
			break
		}
		if time.Now().After(deadline) {
			t.Fatalf("serve did not start:\n%s", serveOut.String())
		}
		time.Sleep(50 * time.Millisecond)
	}

	secret := []byte(pass)
	unlock := exec.Command(bin, "unlock", "--stdin")
	stdin, err := unlock.StdinPipe()
	if err != nil {
		t.Fatal(err)
	}
	var out bytes.Buffer
	unlock.Stdout, unlock.Stderr = &out, &out
	if err := unlock.Start(); err != nil {
		t.Fatal(err)
	}
	done := make(chan struct{})
	var mu sync.Mutex
	var leaks []string
	go func() {
		for {
			if p, f, ok := sampleArgvEnv(secret); ok {
				mu.Lock()
				leaks = append(leaks, strconv.Itoa(p)+"/"+f)
				mu.Unlock()
			}
			select {
			case <-done:
				return
			case <-time.After(time.Millisecond):
			}
		}
	}()
	if _, err := stdin.Write(append(secret, '\n')); err != nil {
		t.Fatal(err)
	}
	stdin.Close()
	werr := unlock.Wait()
	close(done)
	if werr != nil || !strings.Contains(out.String(), "unlocked") {
		t.Fatalf("unlock: %v\n%s\nserve:\n%s", werr, out.String(), serveOut.String())
	}
	if p, f, ok := sampleArgvEnv(secret); ok {
		leaks = append(leaks, strconv.Itoa(p)+"/"+f)
	}
	mu.Lock()
	defer mu.Unlock()
	if len(leaks) > 0 {
		t.Fatalf("the passphrase appeared in /proc/%v", leaks)
	}
}
