package main

import (
	"context"
	"crypto/ed25519"
	"crypto/rand"
	"encoding/json"
	"io"
	"net/http"
	"path/filepath"
	"strings"
	"sync"
	"testing"

	"golang.org/x/crypto/ssh"

	"github.com/valve-tech/jumpgate/internal/api"
	"github.com/valve-tech/jumpgate/internal/apiclient"
	"github.com/valve-tech/jumpgate/internal/bootstrap"
	"github.com/valve-tech/jumpgate/internal/eip712"
	"github.com/valve-tech/jumpgate/internal/executor"
	"github.com/valve-tech/jumpgate/internal/signer"
	"github.com/valve-tech/jumpgate/internal/testutil"
)

const servedTransportKey = "ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAIOjumpgate jumpgate-controller"

// pushWorld is a system server stand-in for a push from the CLI, with the
// seams the push uses. Every request is recorded.
type pushWorld struct {
	controller eip712.Address
	mu         sync.Mutex
	bodies     map[string][]string // "METHOD path" -> bodies
	recorded   []api.RecordedHostKey
	dialled    *executor.SSHConfig
	box        *cmdRecorder
	opts       *bootstrap.Options
	ran        *bool
}

func (p *pushWorld) seen(key string) []string {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.bodies[key]
}

func newPushWorld(t *testing.T) *pushWorld {
	t.Helper()
	testutil.Home(t)
	k, _ := signer.GenerateKey()
	p := &pushWorld{controller: k.Address(), bodies: map[string][]string{}, box: &cmdRecorder{},
		recorded: []api.RecordedHostKey{{Fingerprint: "SHA256:box", KeyType: "ssh-ed25519", Store: api.HostKeyStoreJumpgate}}}
	withServer(t, func(w http.ResponseWriter, r *http.Request) {
		b, _ := io.ReadAll(r.Body)
		p.mu.Lock()
		key := r.Method + " " + r.URL.Path
		p.bodies[key] = append(p.bodies[key], string(b))
		recorded := p.recorded
		p.mu.Unlock()
		w.Header().Set("Content-Type", "application/json")
		switch key {
		case "POST /api/hostkeys/probe":
			_ = json.NewEncoder(w).Encode(api.HostKeyProbe{Hops: []api.HostKeyHop{{HostPort: "box.example:22", State: api.HostKeyConfirmed, Fingerprint: "SHA256:box"}}, AllConfirmed: true})
		case "POST /api/targets":
			w.WriteHeader(http.StatusCreated)
			_, _ = io.WriteString(w, `{}`)
		case "GET /api/pairing-info":
			_ = json.NewEncoder(w).Encode(api.PairingInfo{Controller: p.controller.Hex(), TransportKey: servedTransportKey, System: true, ServiceUID: 991})
		case "GET /api/hostkeys":
			_ = json.NewEncoder(w).Encode(api.RecordedHostKeys{HostPort: r.URL.Query().Get("hostPort"), Keys: recorded})
		case "POST /api/targets/b/pair":
			w.Header().Set("Content-Type", "text/event-stream")
			_, _ = io.WriteString(w, "data: {\"done\":true,\"agent\":\"0x0200000000000000000000000000000000000000\"}\n\n")
		default:
			http.NotFound(w, r)
		}
	})
	oldSys, oldTerm, oldSSH := serverIsSystem, stdinIsTerminal, newSSHExecutor
	serverIsSystem = func(*apiclient.Client) bool { return true }
	stdinIsTerminal = func() bool { return true }
	newSSHExecutor = func(_ context.Context, c executor.SSHConfig) (executor.Executor, error) {
		p.dialled = &c
		return p.box, nil
	}
	t.Cleanup(func() { serverIsSystem, stdinIsTerminal, newSSHExecutor = oldSys, oldTerm, oldSSH })
	_, p.opts, p.ran = stubForeground(t, nil, nil, eip712.Address{0x02})
	return p
}

func TestSystemPushRunsBootstrapInTheCLI(t *testing.T) {
	p := newPushWorld(t)
	captureStdio(t)
	if code := hostsAdd([]string{"b", "--ssh", "root@box.example"}); code != 0 {
		t.Fatalf("exit %d", code)
	}
	if !*p.ran || p.opts.Local || p.opts.TransportKey != servedTransportKey || p.opts.Controller != p.controller {
		t.Fatalf("ran=%v options: Local=%v TransportKey=%q Controller=%s", *p.ran, p.opts.Local, p.opts.TransportKey, p.opts.Controller.Hex())
	}
	pairs := p.seen("POST /api/targets/b/pair")
	if len(pairs) != 1 || !strings.Contains(pairs[0], `"installed":"0x0200000000000000000000000000000000000000"`) {
		t.Fatalf("pair bodies %q", pairs)
	}
}

func TestSystemPushNeverSendsTheKeyPath(t *testing.T) {
	p := newPushWorld(t)
	captureStdio(t)
	if code := hostsAdd([]string{"b", "--ssh", "alice@box.example", "--key", "/home/alice/.ssh/id"}); code != 0 {
		t.Fatalf("exit %d", code)
	}
	adds := p.seen("POST /api/targets")
	if len(adds) != 1 {
		t.Fatalf("adds %q", adds)
	}
	var tg api.AddTarget
	if err := json.Unmarshal([]byte(adds[0]), &tg); err != nil || tg.SSH == nil {
		t.Fatalf("add body %q: %v", adds[0], err)
	}
	for h := tg.SSH; h != nil; h = h.Jump {
		if h.KeyPath != "" {
			t.Fatalf("a hop's keyPath was sent: %q", adds[0])
		}
	}
	for _, b := range p.seen("POST /api/hostkeys/probe") {
		if strings.Contains(b, "/home/alice") {
			t.Fatalf("the probe carried the key path: %q", b)
		}
	}
	wantKey, _ := filepath.Abs("/home/alice/.ssh/id") // C:\home\alice\.ssh\id on Windows
	if p.dialled == nil || p.dialled.KeyPath != wantKey || p.dialled.HostKey == nil {
		t.Fatalf("dialled %+v", p.dialled)
	}
	// The operator is not root on the box: bootstrap goes through sudo.
	_, _ = p.opts.Exec.Run(context.Background(), "id -u", nil)
	if len(p.box.cmds) != 1 || !strings.HasPrefix(p.box.cmds[0], "sudo -n ") {
		t.Fatalf("commands %v, want one run through sudo -n", p.box.cmds)
	}
}

func TestSystemPushTrustsOnlyRecordedHostKeys(t *testing.T) {
	p := newPushWorld(t)
	p.recorded = nil
	_, stderr := captureStdio(t)
	if code := hostsAdd([]string{"b", "--ssh", "root@box.example"}); code == 0 {
		t.Fatal("pushed to a host with no confirmed key")
	}
	if p.dialled != nil || *p.ran {
		t.Fatal("dialled a host with no confirmed key")
	}
	if !strings.Contains(stderr(), "confirm") {
		t.Fatalf("stderr %q", stderr())
	}
}

func TestSystemPushHostKeyCallbackAcceptsOnlyRecordedKeys(t *testing.T) {
	pub, _, _ := ed25519.GenerateKey(rand.Reader)
	key, err := ssh.NewPublicKey(pub)
	if err != nil {
		t.Fatal(err)
	}
	cb := recordedHostKeyCallback("box.example:22", []api.RecordedHostKey{{Fingerprint: "SHA256:nope"}})
	if err := cb("box.example:22", nil, key); err == nil {
		t.Fatal("accepted a key that is not on record")
	}
	cb = recordedHostKeyCallback("box.example:22", []api.RecordedHostKey{{Fingerprint: executor.Fingerprint(key)}})
	if err := cb("box.example:22", nil, key); err != nil {
		t.Fatalf("refused the recorded key: %v", err)
	}
	if err := cb("other.example:22", nil, key); err == nil {
		t.Fatal("accepted the key for another host")
	}
}

func TestSystemPushPrintsTheCustodyWarning(t *testing.T) {
	newPushWorld(t)
	stdout, _ := captureStdio(t)
	if code := hostsAdd([]string{"b", "--ssh", "root@box.example"}); code != 0 {
		t.Fatalf("exit %d", code)
	}
	if n := strings.Count(stdout(), api.CustodyWarning); n != 1 {
		t.Fatalf("custody warning printed %d times:\n%s", n, stdout())
	}
}

func TestSystemPushRefusesAJumpHost(t *testing.T) {
	p := newPushWorld(t)
	_, stderr := captureStdio(t)
	if code := hostsAdd([]string{"b", "--ssh", "root@box.example", "--jump", "x@y"}); code == 0 {
		t.Fatal("pushed through a jump host")
	}
	if !strings.Contains(stderr(), string(api.CodeSSHJumpUnsupported)) {
		t.Fatalf("stderr %q", stderr())
	}
	if len(p.seen("POST /api/targets")) != 0 {
		t.Fatal("the target was sent")
	}
}
