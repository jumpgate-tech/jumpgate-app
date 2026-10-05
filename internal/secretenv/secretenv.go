// Package secretenv keeps jumpgate's secrets out of the environment its child
// processes inherit. The server reads its tokens once and unsets them, and
// every place that launches a child (the local executor, the key-store
// helpers) scrubs the environment it passes on as a second line of defence.
package secretenv

import (
	"os"
	"strings"
)

// IsSecret reports whether an environment variable name holds a secret that
// a child must not inherit: any JUMPGATE_* name containing TOKEN, or any name
// containing _SECRET. Other tools' own credentials (OP_SESSION_*, an
// ssh-agent socket) are left alone, since the helpers that need them are the
// children being launched.
func IsSecret(name string) bool {
	upper := strings.ToUpper(name)
	if strings.HasPrefix(upper, "JUMPGATE_") && strings.Contains(upper, "TOKEN") {
		return true
	}
	return strings.Contains(upper, "_SECRET")
}

// Scrub returns a copy of env (KEY=VALUE entries) without the secrets.
func Scrub(env []string) []string {
	out := make([]string, 0, len(env))
	for _, kv := range env {
		name, _, _ := strings.Cut(kv, "=")
		if IsSecret(name) {
			continue
		}
		out = append(out, kv)
	}
	return out
}

// Environ is os.Environ() without the secrets.
func Environ() []string { return Scrub(os.Environ()) }

// Unset removes each named variable from this process's environment. The
// server calls it right after reading its tokens, so nothing it launches
// later can inherit them.
func Unset(names ...string) {
	for _, n := range names {
		_ = os.Unsetenv(n)
	}
}
