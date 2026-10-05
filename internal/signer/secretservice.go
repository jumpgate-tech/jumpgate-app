package signer

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// toolTimeout bounds every key-store tool run. It is long because macOS and
// 1Password may show a prompt a person has to answer; it exists so a locked
// keyring with nothing to prompt cannot hang server start forever (spec D17).
// A var so tests can shorten it.
var toolTimeout = 2 * time.Minute

// probeTimeout bounds the Secret Service liveness probe, which never prompts.
const probeTimeout = 3 * time.Second

// probeService is the service attribute the probe searches for. No jumpgate
// item uses it, so the probe never matches, and never prints, a stored key:
// an unlocked `secret-tool search` prints each matching item's secret.
const probeService = "jumpgate-probe"

// dbusSession reports whether a D-Bus session bus is reachable; the Secret
// Service lives on it. Over SSH there usually is none. A seam for tests.
var dbusSession = func() bool {
	if os.Getenv("DBUS_SESSION_BUS_ADDRESS") != "" {
		return true
	}
	if d := os.Getenv("XDG_RUNTIME_DIR"); d != "" {
		if _, err := os.Stat(filepath.Join(d, "bus")); err == nil {
			return true
		}
	}
	return false
}

// secretServiceAnswers runs a search that matches nothing. An answer means a
// Secret Service is there; an error printed on stderr, or no answer within
// probeTimeout, means it is not.
func secretServiceAnswers(ctx context.Context) bool {
	ctx, cancel := context.WithTimeout(ctx, probeTimeout)
	defer cancel()
	_, err := runCmd(ctx, "", "secret-tool", "search", "service", probeService)
	if err == nil {
		return true
	}
	// libsecret 0.20 exits 0 when nothing matches; an exit 1 with no output
	// at all is the same answer from builds that report an empty result.
	var ce *cmdError
	return errors.As(err, &ce) && ce.ExitCode == 1 && strings.TrimSpace(ce.Stderr) == "" && ctx.Err() == nil
}
