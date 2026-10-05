package server

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"testing/fstest"

	"github.com/valve-tech/jumpgate/internal/catalog"
	"github.com/valve-tech/jumpgate/internal/config"
	"github.com/valve-tech/jumpgate/internal/executor"
)

// I-12: on a controller with no POSIX shell (Windows), "this host" features
// answer with a clear refusal and a hint, not a 500.
func TestLocalHostFeaturesAreRefusedClearly(t *testing.T) {
	s := New(Config{Token: NewSessionToken(), UI: fstest.MapFS{},
		NewExecutor: func(config.Target) (executor.Executor, error) {
			return nil, fmt.Errorf("%w, which windows does not provide", executor.ErrNoPOSIXShell)
		}})
	for name, call := range map[string]func(http.ResponseWriter) bool{
		"vpn": func(w http.ResponseWriter) bool {
			_, _, ok := s.vpnExecutor(w, config.Config{}, config.VPN{})
			return ok
		},
		"host": func(w http.ResponseWriter) bool { _, ok := s.hostExecutor(w, config.Config{}, ""); return ok },
	} {
		rec := httptest.NewRecorder()
		if call(rec) {
			t.Fatalf("%s: got an executor", name)
		}
		var e struct{ Code, Hint string }
		_ = json.NewDecoder(rec.Body).Decode(&e)
		if rec.Code != http.StatusConflict || e.Code != "local_unsupported" || e.Hint == "" {
			t.Fatalf("%s: %d %+v, want 409 local_unsupported with a hint", name, rec.Code, e)
		}
	}
}

// Any other executor failure keeps the status the route used before: the
// refusal is only for a missing shell.
func TestWriteExecutorErrorKeepsTheCallersStatus(t *testing.T) {
	rec := httptest.NewRecorder()
	writeExecutorError(rec, errors.New("dial tcp: no route to host"), http.StatusBadGateway)
	var e struct{ Code string }
	_ = json.NewDecoder(rec.Body).Decode(&e)
	if rec.Code != http.StatusBadGateway || e.Code != "" {
		t.Fatalf("%d %+v, want 502 with no code", rec.Code, e)
	}
}

// shellLess is "this computer" on Windows once Task 15 lands: the target can
// be added (Docker features need it), but no shell command can run.
type shellLess struct{ autoSucceedExecutor }

func (*shellLess) Run(context.Context, string, *executor.RunOpts) (executor.Result, error) {
	return executor.Result{}, fmt.Errorf("%w, which windows does not provide", executor.ErrNoPOSIXShell)
}

func (*shellLess) ShellError() error {
	return fmt.Errorf("%w, which windows does not provide", executor.ErrNoPOSIXShell)
}

// Every route that needs a shell refuses a shell-less target up front with
// local_unsupported, instead of failing halfway through with a raw error.
func TestShellRoutesRefuseAShellLessTarget(t *testing.T) {
	a := newAPITestServerWithExecutor(t, func(config.Target) (executor.Executor, error) { return &shellLess{}, nil })
	if res := a.do(t, "POST", "/api/targets", map[string]any{"id": "me", "mode": "local"}); res.StatusCode != http.StatusCreated {
		t.Fatalf("add local target: %d", res.StatusCode)
	}
	// Most routes need a completed setup; the setup route itself is refused,
	// so the wire is written to the config directly.
	if _, err := a.srv.updateConfig(func(c *config.Config) error {
		for i := range c.Targets {
			c.Targets[i].Wire = &catalog.WireConfig{ChainID: 369, ExecID: "reth", BeaconID: "lighthouse-pulse"}
		}
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	createTestVPN(t, a, "pvpn")
	for _, r := range []struct {
		method, path string
		body         any
	}{
		{"POST", "/api/targets/me/setup", catalog.WireConfig{ChainID: 369, ExecID: "reth", BeaconID: "lighthouse-pulse"}},
		{"POST", "/api/targets/me/services/execution/restart", nil},
		{"POST", "/api/targets/me/services/execution/clear", map[string]string{"confirm": "execution"}},
		{"GET", "/api/targets/me/du", nil},
		{"GET", "/api/targets/me/disk?path=/", nil},
		{"GET", "/api/targets/me/endpoints", nil},
		{"GET", "/api/targets/me/firewall", nil},
		{"GET", "/api/targets/me/diagnostics", nil},
		{"GET", "/api/targets/me/logs", nil},
		{"GET", "/api/targets/me/logs/stream", nil},
		{"GET", "/api/targets/me/monitor/stream", nil},
		{"POST", "/api/vpns/pvpn/up", nil},
		{"POST", "/api/vpns/pvpn/down", nil},
	} {
		res := a.do(t, r.method, r.path, r.body)
		var e struct{ Code, Hint string }
		_ = json.NewDecoder(res.Body).Decode(&e)
		res.Body.Close()
		if res.StatusCode != http.StatusConflict || e.Code != "local_unsupported" || e.Hint == "" {
			t.Errorf("%s %s: %d %+v, want 409 local_unsupported with a hint", r.method, r.path, res.StatusCode, e)
		}
	}
}

// An unreachable SSH host is still a 502, not a refusal: the guard only
// speaks for a missing shell.
func TestShellRoutesKeepBadGatewayForAnUnreachableHost(t *testing.T) {
	a := newAPITestServerWithExecutor(t, func(config.Target) (executor.Executor, error) {
		return nil, errors.New("dial tcp 203.0.113.9:22: i/o timeout")
	})
	if _, err := a.srv.updateConfig(func(c *config.Config) error {
		c.Targets = append(c.Targets, config.Target{ID: "far", Mode: "ssh",
			Wire: &catalog.WireConfig{ChainID: 369, ExecID: "reth", BeaconID: "lighthouse-pulse"}})
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	for _, p := range []string{"du", "endpoints", "firewall", "diagnostics", "logs"} {
		res := a.do(t, "GET", "/api/targets/far/"+p, nil)
		res.Body.Close()
		if res.StatusCode != http.StatusBadGateway {
			t.Errorf("GET %s: %d, want 502", p, res.StatusCode)
		}
	}
}
