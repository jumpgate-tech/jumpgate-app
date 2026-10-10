package main

import (
	"encoding/json"
	"io"
	"net/http"
	"path/filepath"
	"strings"
	"testing"

	"github.com/valve-tech/jumpgate/internal/api"
	"github.com/valve-tech/jumpgate/internal/intent"
)

// intentServer records the path and body of the one intent the CLI sends
// and answers it with reply.
func intentServer(t *testing.T, reply string) (path, body *string) {
	t.Helper()
	var p, b string
	withServer(t, func(w http.ResponseWriter, r *http.Request) {
		raw, _ := io.ReadAll(r.Body)
		p, b = r.URL.Path, string(raw)
		w.Header().Set("Content-Type", "application/json")
		io.WriteString(w, reply)
	})
	return &p, &b
}

// `jumpgate logs HOST -n N` asks for exactly N lines per unit.
func TestLogsSendsTheLineCount(t *testing.T) {
	path, body := intentServer(t, `{"status":0,"result":{"lines":[]}}`)
	stdout, stderr := captureStdio(t)
	if code := cmdLogs([]string{"box", "-n", "50"}); code != 0 {
		t.Fatalf("exit %d, stderr %q", code, stderr())
	}
	var p intent.LogsReadPayload
	if *path != "/api/targets/box/intent/"+intent.KindLogsRead || json.Unmarshal([]byte(*body), &p) != nil || p.N != 50 {
		t.Fatalf("sent %s %s", *path, *body)
	}
	if !strings.Contains(stdout(), `"lines"`) {
		t.Fatalf("stdout %q", stdout())
	}
}

// `jumpgate service HOST SVC ACTION` sends the service and the action as
// typed; the agent, not the CLI, decides what is allowed.
func TestServiceSendsTheServiceAndAction(t *testing.T) {
	path, body := intentServer(t, `{"status":0,"result":{}}`)
	captureStdio(t)
	if code := cmdService([]string{"box", "beacon", "restart"}); code != 0 {
		t.Fatalf("exit %d", code)
	}
	var p intent.ServiceActionPayload
	if *path != "/api/targets/box/intent/"+intent.KindServiceAction || json.Unmarshal([]byte(*body), &p) != nil ||
		p != (intent.ServiceActionPayload{Service: "beacon", Action: "restart"}) {
		t.Fatalf("sent %s %s", *path, *body)
	}
}

// Malformed node commands are usage errors and never reach the server.
func TestNodeCommandsCheckTheirArguments(t *testing.T) {
	withServer(t, func(w http.ResponseWriter, r *http.Request) { t.Errorf("a malformed command sent %s", r.URL.Path) })
	captureStdio(t)
	for name, code := range map[string]int{
		"logs":           cmdLogs(nil),
		"logs -n x":      cmdLogs([]string{"box", "-n", "x"}),
		"service 2 args": cmdService([]string{"box", "beacon"}),
		"status no host": cmdIntent("status", intent.KindStatusRead)(nil),
		"status 2 hosts": cmdIntent("status", intent.KindStatusRead)([]string{"a", "b"}),
	} {
		if code != exitCode("usage") {
			t.Errorf("%s: exit %d", name, code)
		}
	}
}

// A clock-skew refusal shows both clocks, so the operator can see which one
// is wrong; a failure the box reports exits 1 with its message.
func TestPrintReplyShowsBothClocksAndFailures(t *testing.T) {
	var out, errw strings.Builder
	code := printReply(&out, &errw, "box", api.IntentReply{Rejection: &intent.Rejection{Code: intent.ReasonClockSkew, Message: "skew", AgentTime: 1_800_000_000}})
	if code != exitCode("refused") || !strings.Contains(errw.String(), "this machine: ") || !strings.Contains(errw.String(), "the box: 2027-01-15T08:00:00Z") {
		t.Fatalf("exit %d, stderr %q", code, errw.String())
	}
	errw.Reset()
	code = printReply(&out, &errw, "box", api.IntentReply{Failure: &intent.Failure{Message: "systemctl exited 5"}})
	if code != exitCode("failed") || !strings.Contains(errw.String(), "box ran it and it failed: systemctl exited 5") || out.Len() != 0 {
		t.Fatalf("exit %d, stderr %q, stdout %q", code, errw.String(), out.String())
	}
}

// status.read's reply carries the reference chain head beside the node's
// status, so both are printed.
func TestPrintReplyCarriesTheReferenceHead(t *testing.T) {
	var out, errw strings.Builder
	if code := printReply(&out, &errw, "box", api.IntentReply{Result: json.RawMessage(`{"head":100}`), RefHead: 105}); code != 0 {
		t.Fatalf("exit %d", code)
	}
	var got struct {
		Node    map[string]int `json:"node"`
		RefHead uint64         `json:"refHead"`
	}
	if err := json.Unmarshal([]byte(out.String()), &got); err != nil || got.Node["head"] != 100 || got.RefHead != 105 {
		t.Fatalf("stdout %q: %v", out.String(), err)
	}
}

// The server resolves a key path from its own working directory, so hosts
// add sends --key as an absolute path (~ expanded), and the jump host is
// dialled with the same key.
func TestHostsAddSendsAnAbsoluteKeyForBothHops(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("USERPROFILE", home)
	terminalStdin(t)
	var probe, target api.SSHView
	withServer(t, func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/hostkeys/probe":
			var in api.HostKeyProbeRequest
			json.NewDecoder(r.Body).Decode(&in)
			probe = in.SSH
			io.WriteString(w, `{"hops":[],"allConfirmed":true}`)
		case "/api/targets":
			var in api.AddTarget
			json.NewDecoder(r.Body).Decode(&in)
			if in.SSH != nil {
				target = *in.SSH
			}
			w.WriteHeader(http.StatusNoContent)
		case "/api/targets/box/pair":
			w.Header().Set("Content-Type", "text/event-stream")
			io.WriteString(w, "data: {\"done\":true,\"agent\":\"0xabc\"}\n\n")
		default:
			http.NotFound(w, r)
		}
	})
	_, stderr := captureStdio(t)
	if code := hostsAdd([]string{"box", "--ssh", "root@203.0.113.5:2200", "--key", "~/.ssh/id_box", "--jump", "ops@jump.example"}); code != 0 {
		t.Fatalf("exit %d, stderr %q", code, stderr())
	}
	wantKey := filepath.Join(home, ".ssh", "id_box")
	if target.KeyPath != wantKey || target.Host != "203.0.113.5" || target.Port != 2200 || target.User != "root" {
		t.Fatalf("target ssh %+v", target)
	}
	if target.Jump == nil || target.Jump.Host != "jump.example" || target.Jump.User != "ops" || target.Jump.KeyPath != wantKey {
		t.Fatalf("jump %+v", target.Jump)
	}
	if probe.Host != "203.0.113.5" || probe.Jump == nil {
		t.Fatalf("the probe was not for the same path: %+v", probe)
	}
}

// Every malformed hosts add is a usage error before any server is asked.
func TestHostsAddChecksItsArgumentsFirst(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("USERPROFILE", home)
	terminalStdin(t)
	withServer(t, func(w http.ResponseWriter, r *http.Request) { t.Errorf("a malformed hosts add sent %s", r.URL.Path) })
	captureStdio(t)
	for _, args := range [][]string{
		{},
		{"--ssh", "root@h"},
		{"box"},
		{"box", "--ssh", "root@h", "--local"},
		{"box", "--ssh", "not an address"},
		{"box", "--ssh", "root@h", "--jump", "::bad::"},
	} {
		if code := hostsAdd(args); code != exitCode("usage") {
			t.Errorf("hosts add %v: exit %d", args, code)
		}
	}
	for _, args := range [][]string{{}, {"bogus"}} {
		if code := cmdHosts(args); code != exitCode("usage") {
			t.Errorf("hosts %v: exit %d", args, code)
		}
	}
}
