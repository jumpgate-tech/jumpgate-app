package main

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"slices"
	"strings"
	"sync"
	"testing"

	"github.com/valve-tech/jumpgate/internal/api"
	"github.com/valve-tech/jumpgate/internal/apiclient"
	"github.com/valve-tech/jumpgate/internal/apiclient/apiclienttest"
	"github.com/valve-tech/jumpgate/internal/eip712"
	"github.com/valve-tech/jumpgate/internal/signer"
)

// fakeService is the jumpgate service as setup server sees it. It logs
// every request but GETs into rec.cmds, so one log holds the root commands
// and the API calls in order.
type fakeService struct {
	mu         sync.Mutex
	controller string // GET /api/controller state
	vault      string // GET /api/vault state
	paired     bool
	initBody   api.SystemInitKey
	unlockPass []byte
	pairBody   string
}

func setupServerSeams(t *testing.T, euid int, f *fakeService) *privRecorder {
	t.Helper()
	rec := setupSeams(t, "linux", euid, true)
	k, _ := signer.GenerateKey()
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		f.mu.Lock()
		defer f.mu.Unlock()
		if r.Method != http.MethodGet {
			rec.mu.Lock()
			rec.cmds = append(rec.cmds, r.Method+" "+r.URL.Path)
			rec.mu.Unlock()
		}
		w.Header().Set("Content-Type", "application/json")
		switch {
		case r.URL.Path == "/api/controller":
			_ = json.NewEncoder(w).Encode(api.ControllerView{State: f.controller})
		case r.URL.Path == "/api/vault":
			_ = json.NewEncoder(w).Encode(api.VaultView{State: f.vault})
		case r.URL.Path == "/api/system/init-key":
			_ = json.NewDecoder(r.Body).Decode(&f.initBody)
			f.controller, f.vault = "locked", "locked"
			w.WriteHeader(http.StatusCreated)
			_, _ = io.WriteString(w, `{}`)
		case r.URL.Path == "/api/vault/unlock":
			var u api.UnlockRequest
			_ = json.NewDecoder(r.Body).Decode(&u)
			f.unlockPass, f.vault = u.Passphrase, "unlocked"
			_ = json.NewEncoder(w).Encode(api.VaultView{State: "unlocked"})
		case r.URL.Path == "/api/targets" && r.Method == http.MethodGet:
			tv := []api.TargetView{}
			if f.paired {
				tv = append(tv, api.TargetView{ID: "box", Mode: "local", Agent: &api.AgentView{Address: "0x01"}})
			}
			_ = json.NewEncoder(w).Encode(tv)
		case r.URL.Path == "/api/targets":
			w.WriteHeader(http.StatusCreated)
			_, _ = io.WriteString(w, `{}`)
		case r.URL.Path == "/api/pairing-info":
			_ = json.NewEncoder(w).Encode(api.PairingInfo{Controller: k.Address().Hex(), System: true, ServiceUID: 991})
		case strings.HasSuffix(r.URL.Path, "/pair"):
			b, _ := io.ReadAll(r.Body)
			f.pairBody = string(b)
			w.Header().Set("Content-Type", "text/event-stream")
			_, _ = io.WriteString(w, "data: {\"done\":true,\"agent\":\"0x0100000000000000000000000000000000000000\"}\n\n")
		default:
			http.NotFound(w, r)
		}
	}))
	t.Cleanup(ts.Close)
	c := apiclienttest.NewHTTP(t, ts.URL, "tok")
	oldC, oldG, oldRe := connectService, userInGroup, reexecSudo
	connectService = func(context.Context) (*apiclient.Client, error) { return c, nil }
	userInGroup = func(string, string) bool { return false }
	reexecSudo = func([]string) int { t.Fatal("re-executed through sudo"); return 1 }
	t.Cleanup(func() { connectService, userInGroup, reexecSudo = oldC, oldG, oldRe })
	t.Setenv("SUDO_USER", "alice")
	stubForeground(t, nil, nil, eip712.Address{0x01})
	return rec
}

func stdinIs(t *testing.T, s string) {
	t.Helper()
	old := stdinReader
	stdinReader = strings.NewReader(s)
	t.Cleanup(func() { stdinReader = old })
}

func TestSetupServerRunsTheStepsInOrder(t *testing.T) {
	f := &fakeService{controller: "missing", vault: "none"}
	rec := setupServerSeams(t, 0, f)
	stdinIs(t, "pw\n")
	captureStdio(t)
	if code := cmdSetup([]string{"server", "--name", "box", "--stdin"}); code != 0 {
		t.Fatalf("exit %d; log %v", code, rec.cmds)
	}
	want := []string{"systemctl enable --now jumpgate.service", "usermod -aG jumpgate-admin alice", "POST /api/system/init-key", "POST /api/vault/unlock", "POST /api/targets", "POST /api/targets/box/pair"}
	if !slices.Equal(rec.cmds, want) {
		t.Fatalf("log\n%q\nwant\n%q", rec.cmds, want)
	}
	if !strings.Contains(f.pairBody, `"installed"`) {
		t.Fatalf("pair body %q", f.pairBody)
	}
	if string(f.unlockPass) != "pw" || f.initBody.Store != "vault" {
		t.Fatalf("unlock %q init %+v", f.unlockPass, f.initBody.Store)
	}
}

func TestSetupServerReadsOneLineFromStdin(t *testing.T) {
	for _, in := range []string{"pw\r\n", "pw\n\n"} {
		f := &fakeService{controller: "missing", vault: "none"}
		setupServerSeams(t, 0, f)
		stdinIs(t, in)
		captureStdio(t)
		if code := cmdSetup([]string{"server", "--name", "box", "--stdin"}); code != 0 {
			t.Fatalf("%q: exit %d", in, code)
		}
		if string(f.initBody.Passphrase) != "pw" {
			t.Fatalf("%q: init-key passphrase %q, want pw", in, f.initBody.Passphrase)
		}
	}
}

func TestSetupServerRefusesPlainWithoutTheFlag(t *testing.T) {
	rec := setupServerSeams(t, 0, &fakeService{controller: "missing", vault: "none"})
	captureStdio(t)
	if code := cmdSetup([]string{"server", "--store", "file"}); code != exitCode("usage") {
		t.Fatalf("exit %d, want usage", code)
	}
	if len(rec.cmds) != 0 {
		t.Fatalf("did %v", rec.cmds)
	}
}

func TestSetupServerSkipsDoneSteps(t *testing.T) {
	f := &fakeService{controller: "ok", vault: "unlocked", paired: true}
	rec := setupServerSeams(t, 0, f)
	userInGroup = func(string, string) bool { return true }
	stdinIs(t, "")
	captureStdio(t)
	if code := cmdSetup([]string{"server", "--name", "box", "--stdin"}); code != 0 {
		t.Fatalf("exit %d", code)
	}
	if want := []string{"systemctl enable --now jumpgate.service"}; !slices.Equal(rec.cmds, want) {
		t.Fatalf("log %q, want %q", rec.cmds, want)
	}
}

func TestSetupServerPrintsTheCustodyWarning(t *testing.T) {
	setupServerSeams(t, 0, &fakeService{controller: "missing", vault: "none"})
	stdinIs(t, "pw\n")
	stdout, _ := captureStdio(t)
	if code := cmdSetup([]string{"server", "--name", "box", "--stdin"}); code != 0 {
		t.Fatalf("exit %d", code)
	}
	if !strings.Contains(stdout(), api.CustodyWarning) || !strings.Contains(stdout(), "log out and back in") {
		t.Fatalf("stdout %q", stdout())
	}
}

func TestSetupServerReexecsThroughSudo(t *testing.T) {
	rec := setupServerSeams(t, 1000, &fakeService{controller: "missing", vault: "none"})
	var got []string
	reexecSudo = func(args []string) int { got = args; return 0 }
	stdout, _ := captureStdio(t)
	if code := cmdSetup([]string{"server", "--stdin"}); code != 0 {
		t.Fatalf("exit %d", code)
	}
	if !slices.Equal(got, []string{"--stdin"}) || len(rec.cmds) != 0 {
		t.Fatalf("reexec %q, log %v", got, rec.cmds)
	}
	if !strings.Contains(stdout(), "through sudo") {
		t.Fatalf("stdout %q", stdout())
	}
}

func TestSetupServerNeedsTheUnit(t *testing.T) {
	rec := setupServerSeams(t, 0, &fakeService{controller: "missing", vault: "none"})
	pathExists = func(string) bool { return false }
	_, stderr := captureStdio(t)
	if code := cmdSetup([]string{"server"}); code != exitCode("failed") || len(rec.cmds) != 0 {
		t.Fatalf("exit %d log %v", code, rec.cmds)
	}
	if !strings.Contains(stderr(), "install.sh --controller") {
		t.Fatalf("stderr %q", stderr())
	}
}
