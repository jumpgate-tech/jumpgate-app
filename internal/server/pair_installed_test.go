package server

import (
	"context"
	"io"
	"net/http"
	"strings"
	"sync/atomic"
	"testing"

	gliderssh "github.com/gliderlabs/ssh"

	"github.com/valve-tech/jumpgate/internal/api"
	"github.com/valve-tech/jumpgate/internal/config"
	"github.com/valve-tech/jumpgate/internal/executor"
	"github.com/valve-tech/jumpgate/internal/signer"
)

// An agent installed by the CLI over SSH is only verified: the server opens
// no login session on the box.
func TestPairInstalledOverSSHRunsNoLoginSession(t *testing.T) {
	var sessions atomic.Int32
	d := startPairTestSSHD(t, func(s gliderssh.Session) { sessions.Add(1); _ = s.Exit(0) })
	ts, token := pairServer(t, d)
	confirmed, _ := config.ConfirmedHostsFile()
	if err := executor.RecordHostKey(confirmed, d.hostname(), d.hostKey); err != nil {
		t.Fatal(err)
	}
	res := postPair(t, ts, token, `{"installed":"0x0000000000000000000000000000000000000001"}`)
	body, _ := io.ReadAll(res.Body)
	if res.StatusCode != http.StatusOK {
		t.Fatalf("status %d: %s", res.StatusCode, body)
	}
	if !strings.Contains(string(body), `"step":"install","line":"done in the foreground by the CLI"`) {
		t.Fatalf("stream:\n%s", body)
	}
	if n := sessions.Load(); n != 0 {
		t.Fatalf("%d login sessions opened", n)
	}
}

// An installed agent that answers with another key is named as such: the
// operator typed or pasted the wrong address, or reached the wrong host.
func TestPairInstalledMismatchIsNamed(t *testing.T) {
	requireAgentPeer(t)
	var srv *Server
	pairedBoxWith(t, nopExec{}, pairedOpts{onServer: func(s *Server) { srv = s }})
	tg := pairedTarget(t)
	other, _ := signer.GenerateKey()
	ap := *tg.Agent
	ap.Address = other.Address().Hex()
	tg.Agent = &ap
	if _, ev := srv.verifyInstalled(context.Background(), tg); ev == nil || ev.Code != api.CodeInstalledMismatch || ev.Hint == "" {
		t.Fatalf("verify an installed agent answering with another key: %+v", ev)
	}
}

// The service refuses to log in to a box to pair it: the CLI does that.
func TestSystemModePairWithoutInstalledIsRefused(t *testing.T) {
	ts, token, _ := systemServer(t, nil)
	if _, err := config.Update(func(c *config.Config) error {
		c.Targets = append(c.Targets, config.Target{ID: "box", Mode: "ssh", SSH: &executor.SSHConfig{Host: "h", User: "root"}})
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	status, code, _ := readCode(t, postPair(t, ts, token, `{}`))
	if status != http.StatusConflict || code != string(api.CodeServiceNoLogin) {
		t.Fatalf("got %d %q, want 409 service_no_login", status, code)
	}
}
