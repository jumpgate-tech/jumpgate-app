// cmd/jumpgate/cli_ssh_test.go
package main

import (
	"fmt"
	"net/http"
	"strings"
	"testing"

	"github.com/valve-tech/jumpgate/internal/config"
	"github.com/valve-tech/jumpgate/internal/executor"
)

func TestSSHRunsTheServersCommand(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("USERPROFILE", home)
	if _, err := config.Update(func(c *config.Config) error {
		c.Targets = []config.Target{{ID: "box", Mode: "ssh", SSH: &executor.SSHConfig{Host: "10.0.0.5", User: "root"}}}
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	cliServer(t)
	var got []string
	old := runInteractive
	runInteractive = func(argv []string) int { got = argv; return 0 }
	t.Cleanup(func() { runInteractive = old })
	if code := cmdSSH([]string{"box"}); code != 0 {
		t.Fatalf("exit %d", code)
	}
	if !strings.Contains(strings.Join(got, " "), "-l root -- 10.0.0.5") {
		t.Fatalf("argv %q", got)
	}
	if code := cmdSSH(nil); code != 2 {
		t.Fatalf("no host: exit %d", code)
	}
}

func TestSSHRefusesAProgramThatIsNotSSH(t *testing.T) {
	withServer(t, func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, `{"argv":["/bin/sh","-c","id"],"display":""}`)
	})
	ran := false
	old := runInteractive
	runInteractive = func([]string) int { ran = true; return 0 }
	t.Cleanup(func() { runInteractive = old })
	if code := cmdSSH([]string{"box"}); code == 0 || ran {
		t.Fatalf("exit %d, ran %v", code, ran)
	}
}

func TestSSHUsesTheSystemBinaryName(t *testing.T) {
	withServer(t, func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, `{"argv":["ssh.exe","-l","u","--","h"],"display":""}`)
	})
	var got []string
	old := runInteractive
	runInteractive = func(argv []string) int { got = argv; return 0 }
	t.Cleanup(func() { runInteractive = old })
	if code := cmdSSH([]string{"box"}); code != 0 || got[0] != sshBinary() {
		t.Fatalf("exit %d argv %q", code, got)
	}
}

func TestRunInteractiveOnlyRunsTheSystemSSH(t *testing.T) {
	for _, argv := range [][]string{nil, {"sh", "-c", "id"}, {"/usr/bin/ssh"}} {
		if code := runInteractive(argv); code == 0 {
			t.Errorf("%q ran", argv)
		}
	}
	// With no ssh on PATH the failure is a plain refusal, not a fallback.
	t.Setenv("PATH", t.TempDir())
	if code := runInteractive([]string{sshBinary(), "-V"}); code == 0 {
		t.Fatal("ran without ssh on PATH")
	}
}

func TestSSHServerErrorsUseTheRegistry(t *testing.T) {
	withServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusConflict)
		fmt.Fprint(w, `{"error":"this target has no SSH address","code":"no_ssh"}`)
	})
	_, stderr := captureStdio(t)
	if code := cmdSSH([]string{"here"}); code != 1 {
		t.Fatalf("exit %d", code)
	}
	if !strings.Contains(stderr(), "it is this machine") {
		t.Fatalf("no registry hint: %q", stderr())
	}
}

func TestSSHRefusesAJumpTargetWithTheRegistryHint(t *testing.T) {
	withServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusUnprocessableEntity)
		fmt.Fprint(w, `{"error":"jump host","code":"ssh_jump_unsupported"}`)
	})
	ran := false
	old := runInteractive
	runInteractive = func([]string) int { ran = true; return 0 }
	t.Cleanup(func() { runInteractive = old })
	_, stderr := captureStdio(t)
	if code := cmdSSH([]string{"far"}); code != 1 || ran {
		t.Fatalf("exit %d ran %v", code, ran)
	}
	if !strings.Contains(stderr(), "use the TUI") {
		t.Fatalf("no hint: %q", stderr())
	}
}

func TestSSHRefusesAHostileArgvAndRunsNothing(t *testing.T) {
	for _, body := range []string{
		`{"argv":["ssh","-o","ProxyCommand=nc evil 1","--","h"],"display":""}`,
		`{"argv":["ssh","-J","evil","--","h"],"display":""}`,
		`{"argv":["ssh","-o","LocalCommand=id","--","h"],"display":""}`,
		`{"argv":["ssh","-o","StrictHostKeyChecking=no","--","h"],"display":""}`,
		`{"argv":["ssh","h"],"display":""}`,
		`{"argv":["/tmp/x/ssh","--","h"],"display":""}`,
	} {
		body := body
		withServer(t, func(w http.ResponseWriter, r *http.Request) { fmt.Fprint(w, body) })
		ran := false
		old := runInteractive
		runInteractive = func([]string) int { ran = true; return 0 }
		if code := cmdSSH([]string{"box"}); code == 0 || ran {
			t.Errorf("%s: exit %d, ran %v", body, code, ran)
		}
		runInteractive = old
	}
}
