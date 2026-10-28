package main

import (
	"errors"
	"strings"
	"testing"

	"github.com/valve-tech/jumpgate/internal/config"
)

func TestServeSystemIsRefusedOffLinux(t *testing.T) {
	old := hostGOOS
	hostGOOS = "darwin"
	t.Cleanup(func() { hostGOOS = old })
	stderr := captureStderr(t)
	if got := cmdServe([]string{"--system"}); got != exitCode("failed") {
		t.Fatalf("cmdServe --system off Linux = %d, want failed", got)
	}
	if out := stderr(); !strings.Contains(out, "Linux") {
		t.Fatalf("stderr = %q, want a mention of Linux", out)
	}
}

func TestServeSystemExitsSeventyFiveOnRestart(t *testing.T) {
	if got := systemExitCode(true, nil); got != 75 {
		t.Fatalf("restart: %d, want 75", got)
	}
	if got := systemExitCode(false, nil); got != 0 {
		t.Fatalf("clean: %d, want 0", got)
	}
	if got := systemExitCode(false, errors.New("x")); got != 1 {
		t.Fatalf("error: %d, want 1", got)
	}
}

func TestServeSystemWarnsAboutAPlainKey(t *testing.T) {
	w := systemStartWarnings(config.Config{Controller: &config.Controller{KeyStore: "file"}})
	if len(w) != 1 || !strings.Contains(w[0], "stored unencrypted") {
		t.Fatalf("warnings = %q, want one about secrets stored unencrypted", w)
	}
}
