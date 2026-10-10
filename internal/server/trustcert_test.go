package server

import (
	"context"
	"errors"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/valve-tech/jumpgate/internal/config"
	"github.com/valve-tech/jumpgate/internal/executor"
)

// trustBox is an SSH gateway machine answering uname with goos, id -u with
// uid, and the install with install (or failing it with installErr). Its
// exported root is cert.
type trustBox struct {
	*trustExecutor
	installErr error
}

func (b *trustBox) Run(ctx context.Context, cmd string, o *executor.RunOpts) (executor.Result, error) {
	if b.installErr != nil && (strings.Contains(cmd, "update-ca-certificates") || strings.Contains(cmd, "osascript")) {
		b.mu.Lock()
		b.calls = append(b.calls, cmd)
		b.mu.Unlock()
		return executor.Result{}, b.installErr
	}
	return b.trustExecutor.Run(ctx, cmd, o)
}

func trustServer(t *testing.T, cert []byte, goos, uid string, install executor.Result, installErr error) (*apiTestServer, *trustBox) {
	t.Helper()
	var box *trustBox
	a := newAPITestServerWithExecutor(t, func(config.Target) (executor.Executor, error) {
		if box == nil {
			ex := &trustExecutor{scriptedExecutor: fleetExecutor("true|0|img|sha256:abc\n"), cert: cert}
			ex.script("uname", executor.Result{Stdout: goos + "\n"})
			ex.script("id -u", executor.Result{Stdout: uid + "\n"})
			ex.script("verify-cert", executor.Result{ExitCode: 1}) // not trusted yet
			ex.script("update-ca-certificates", install)
			ex.script("osascript", install)
			box = &trustBox{trustExecutor: ex, installErr: installErr}
		}
		return box, nil
	})
	addSSHTarget(t, a, "box")
	addGateway(t, a, "default", "box", internalTLSGateway(4100))
	return a, box
}

// Installing a root authority needs root. A Linux box reached as a user is
// handed the command to run with sudo; jumpgate never runs it there, nor
// prompts for a password it cannot supply.
func TestGatewayTrustCert_AsAUserHandsBackTheCommand(t *testing.T) {
	root := issueLeaf(t, "root", time.Now().Add(-time.Hour), time.Now().Add(24*time.Hour))
	a, box := trustServer(t, root, "Linux", "1000", executor.Result{}, nil)
	res := a.do(t, "POST", "/api/gateways/default/trust-cert", nil)
	body := decode[trustCertResult](t, res)
	if res.StatusCode != http.StatusOK || body.OK || !strings.Contains(body.RanCommand, "update-ca-certificates") || !strings.Contains(body.Message, "needs root") {
		t.Fatalf("got %d %+v", res.StatusCode, body)
	}
	if box.ran("update-ca-certificates") {
		t.Fatal("the install ran without root")
	}
}

// An install that fails says so, with the box's own words and the command
// to run by hand: never a success the browser would then contradict.
func TestGatewayTrustCert_ReportsAFailedInstall(t *testing.T) {
	root := issueLeaf(t, "root", time.Now().Add(-time.Hour), time.Now().Add(24*time.Hour))
	a, _ := trustServer(t, root, "Linux", "0", executor.Result{ExitCode: 1, Stderr: "no CA trust tool found"}, nil)
	body := decode[trustCertResult](t, a.do(t, "POST", "/api/gateways/default/trust-cert", nil))
	if body.OK || !strings.Contains(body.Message, "exited 1") || !strings.Contains(body.Message, "no CA trust tool found") || !strings.HasPrefix(body.RanCommand, "sudo sh -c ") {
		t.Fatalf("result %+v", body)
	}
}

// On macOS a failed osascript is the missing GUI session, and the message
// says to run the command in a Terminal.
func TestGatewayTrustCert_ExplainsAMacWithoutAGUISession(t *testing.T) {
	root := issueLeaf(t, "root", time.Now().Add(-time.Hour), time.Now().Add(24*time.Hour))
	a, _ := trustServer(t, root, "Darwin", "501", executor.Result{ExitCode: 1, Stderr: "no user interaction was possible"}, nil)
	body := decode[trustCertResult](t, a.do(t, "POST", "/api/gateways/default/trust-cert", nil))
	if body.OK || !strings.Contains(body.Message, "GUI login session") || !strings.Contains(body.RanCommand, "security add-trusted-cert") {
		t.Fatalf("result %+v", body)
	}
}

// An install that could not even be sent is a failure with the manual
// command, not a 500.
func TestGatewayTrustCert_ReportsAnInstallThatCouldNotRun(t *testing.T) {
	root := issueLeaf(t, "root", time.Now().Add(-time.Hour), time.Now().Add(24*time.Hour))
	a, _ := trustServer(t, root, "Linux", "0", executor.Result{}, errors.New("session closed"))
	res := a.do(t, "POST", "/api/gateways/default/trust-cert", nil)
	body := decode[trustCertResult](t, res)
	if res.StatusCode != http.StatusOK || body.OK || !strings.Contains(body.Message, "could not be run") || !strings.Contains(body.Message, "session closed") {
		t.Fatalf("got %d %+v", res.StatusCode, body)
	}
}

// SECURITY: only bytes that are a certificate are ever installed as a root.
// Whatever else sits at the exported path is refused before any command.
func TestGatewayTrustCert_RefusesAFileThatIsNotACertificate(t *testing.T) {
	a, box := trustServer(t, []byte("#!/bin/sh\nrm -rf /\n"), "Linux", "0", executor.Result{}, nil)
	res := a.do(t, "POST", "/api/gateways/default/trust-cert", nil)
	res.Body.Close()
	if res.StatusCode != http.StatusBadRequest {
		t.Fatalf("got %d, want 400", res.StatusCode)
	}
	if box.ran("update-ca-certificates") || box.ran("uname") {
		t.Fatal("a non-certificate went on to the install")
	}
}
