package main

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/valve-tech/jumpgate/internal/api"
	"github.com/valve-tech/jumpgate/internal/apiclient"
	"github.com/valve-tech/jumpgate/internal/config"
	"github.com/valve-tech/jumpgate/internal/eip712"
	"github.com/valve-tech/jumpgate/internal/executor"
	"github.com/valve-tech/jumpgate/internal/signer"
)

// pairFromConfig is a per-user foreground pairing of t: facts from this
// user's config.json, then the privileged steps.
func pairFromConfig(out io.Writer, t config.Target) (string, int) {
	f, code := factsFromConfig(t)
	if code != 0 {
		return "", code
	}
	return pairLocalForeground(context.Background(), out, f)
}

// systemPairServer stands in for the jumpgate service: it saves targets,
// answers pairing-info for a service running as uid 991, and finishes every
// pairing at once. The returned pointer holds the last pair request body.
func systemPairServer(t *testing.T, controller eip712.Address) *string {
	t.Helper()
	var pairBody string
	withServer(t, func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodPost && r.URL.Path == "/api/targets":
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusCreated)
			_, _ = io.WriteString(w, `{}`)
		case r.URL.Path == "/api/pairing-info":
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(api.PairingInfo{Controller: controller.Hex(), TransportKey: "ssh-ed25519 AAAA jumpgate-controller", System: true, ServiceUID: 991})
		case strings.HasSuffix(r.URL.Path, "/pair"):
			b, _ := io.ReadAll(r.Body)
			pairBody = string(b)
			w.Header().Set("Content-Type", "text/event-stream")
			_, _ = io.WriteString(w, "data: {\"done\":true,\"agent\":\"0x0100000000000000000000000000000000000000\"}\n\n")
		default:
			http.NotFound(w, r)
		}
	})
	old := serverIsSystem
	serverIsSystem = func(*apiclient.Client) bool { return true }
	t.Cleanup(func() { serverIsSystem = old })
	return &pairBody
}

func TestSystemSelfPairEnrollsTheServiceUID(t *testing.T) {
	localSeams(t, "linux", 1000, true)
	k, _ := signer.GenerateKey()
	body := systemPairServer(t, k.Address())
	captureStdio(t)
	_, got, ran := stubForeground(t, nil, nil, eip712.Address{0x01})
	if code := hostsAdd([]string{"b", "--local"}); code != 0 {
		t.Fatalf("exit %d", code)
	}
	if !*ran || got.LocalUID != 991 || !got.Local || got.Controller != k.Address() {
		t.Fatalf("ran=%v options: Local=%v LocalUID=%d Controller=%s", *ran, got.Local, got.LocalUID, got.Controller.Hex())
	}
	if !strings.Contains(*body, `"installed":"0x0100000000000000000000000000000000000000"`) {
		t.Fatalf("pair body %q", *body)
	}
}

func TestSystemSelfPairAsRootSkipsSudo(t *testing.T) {
	localSeams(t, "linux", 0, false)
	k, _ := signer.GenerateKey()
	systemPairServer(t, k.Address())
	captureStdio(t)
	_, got, ran := stubForeground(t, nil, nil, eip712.Address{0x01})
	sudoNonInteractive = func(context.Context) error { t.Fatal("sudo was checked as root"); return nil }
	rec := &cmdRecorder{}
	oldL := newLocalExecutor
	newLocalExecutor = func() executor.Executor { return rec }
	t.Cleanup(func() { newLocalExecutor = oldL })
	if code := hostsAdd([]string{"b", "--local"}); code != 0 || !*ran {
		t.Fatalf("exit %d ran=%v", code, *ran)
	}
	_, _ = got.Exec.Run(context.Background(), "id -u", nil)
	if len(rec.cmds) != 1 || rec.cmds[0] != "id -u" {
		t.Fatalf("commands %v, want id -u run directly", rec.cmds)
	}
}

func TestHostsAddInstalledNeedsSSH(t *testing.T) {
	localSeams(t, "linux", 1000, true)
	if code := cmdHosts([]string{"add", "b", "--local", "--installed", "0x0100000000000000000000000000000000000000"}); code != exitCode("usage") {
		t.Fatalf("exit %d, want usage", code)
	}
}
