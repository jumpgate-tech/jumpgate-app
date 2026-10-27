package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"testing"

	"github.com/valve-tech/jumpgate/internal/api"
)

// withStdin makes --stdin read s.
func withStdin(t *testing.T, s string) {
	t.Helper()
	old := stdinReader
	stdinReader = strings.NewReader(s)
	t.Cleanup(func() { stdinReader = old })
}

func TestUnlockSendsThePassphraseOnce(t *testing.T) {
	var seen [][]byte
	withServer(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/vault/unlock" || r.Method != http.MethodPost {
			http.Error(w, "unexpected "+r.URL.Path, http.StatusNotFound)
			return
		}
		var req api.UnlockRequest
		_ = json.NewDecoder(r.Body).Decode(&req)
		seen = append(seen, req.Passphrase)
		fmt.Fprint(w, `{"state":"unlocked","slots":["passphrase"]}`)
	})
	withStdin(t, "pw\n")
	stdout, stderr := captureStdio(t)
	if code := cmdUnlock([]string{"--stdin"}); code != 0 {
		t.Fatalf("exit %d: %s", code, stderr())
	}
	if len(seen) != 1 || !bytes.Equal(seen[0], []byte("pw")) {
		t.Fatalf("server saw %q", seen)
	}
	if !strings.Contains(stdout(), "unlocked") {
		t.Fatalf("stdout %q", stdout())
	}
}

func TestUnlockWrongPassphraseExits1(t *testing.T) {
	withServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusForbidden)
		fmt.Fprint(w, `{"error":"wrong passphrase","code":"wrong_passphrase"}`)
	})
	withStdin(t, "nope\n")
	_, stderr := captureStdio(t)
	if code := cmdUnlock([]string{"--stdin"}); code != 1 {
		t.Fatalf("exit %d: %s", code, stderr())
	}
	if !strings.Contains(stderr(), "wrong passphrase") {
		t.Fatalf("stderr %q", stderr())
	}
}

func TestUnlockExitClasses(t *testing.T) {
	for code, want := range map[string]int{"vault_corrupt": 4, "no_vault": 2} {
		withServer(t, func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusInternalServerError)
			fmt.Fprintf(w, `{"error":"x","code":%q}`, code)
		})
		withStdin(t, "pw\n")
		captureStdio(t)
		if got := cmdUnlock([]string{"--stdin"}); got != want {
			t.Errorf("%s: exit %d, want %d", code, got, want)
		}
	}
}

func TestUnlockRefusesArgv(t *testing.T) {
	called := false
	withServer(t, func(w http.ResponseWriter, r *http.Request) { called = true })
	_, stderr := captureStdio(t)
	if code := cmdUnlock([]string{"hunter2"}); code != 2 {
		t.Fatalf("exit %d", code)
	}
	if !strings.Contains(stderr(), "the passphrase is never taken from the command line; use --stdin or a terminal") {
		t.Fatalf("stderr %q", stderr())
	}
	if strings.Contains(stderr(), "hunter2") || called {
		t.Fatalf("echoed the argument or sent a request: %q", stderr())
	}
}

func TestUnlockRefusesAnEmptyPassphraseBeforeSending(t *testing.T) {
	called := false
	withServer(t, func(w http.ResponseWriter, r *http.Request) { called = true })
	withStdin(t, "\n")
	captureStdio(t)
	if code := cmdUnlock([]string{"--stdin"}); code != 1 || called {
		t.Fatalf("exit %d, called %v", code, called)
	}
}

func TestLockLocks(t *testing.T) {
	var path string
	withServer(t, func(w http.ResponseWriter, r *http.Request) {
		path = r.Method + " " + r.URL.Path
		fmt.Fprint(w, `{"state":"locked","slots":["passphrase"]}`)
	})
	stdout, _ := captureStdio(t)
	if code := cmdLock(nil); code != 0 || path != "POST /api/vault/lock" || !strings.Contains(stdout(), "locked") {
		t.Fatalf("exit %d, path %q, stdout %q", code, path, stdout())
	}
}

func TestHomeShowsLocked(t *testing.T) {
	fakeHome(t, true)
	termHome.vault = func(context.Context) (api.VaultView, error) { return api.VaultView{State: "locked"}, nil }
	var out bytes.Buffer
	printOverview(context.Background(), &out)
	if !strings.Contains(out.String(), "  vault:    locked: jobs wait for unlock; run `jumpgate unlock`\n") {
		t.Fatalf("overview:\n%s", out.String())
	}
}
