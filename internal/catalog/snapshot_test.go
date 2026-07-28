package catalog

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// runThroughShell executes cmd the way the executor does — `sh -c` — with a
// stub `reth` first on PATH that writes each argument it received on its own
// line. It returns those arguments and the working directory the shell ran in,
// so a caller can look for side effects the command should not have had.
//
// Asserting on the argv a real shell produced, rather than on the command
// string, is the only way to test quoting: a string comparison passes on a
// command whose quoting is broken, because the string is exactly what the
// broken code produced.
func runThroughShell(t *testing.T, cmd string) ([]string, string) {
	t.Helper()
	dir := t.TempDir()
	stub := filepath.Join(dir, "reth")
	// printf %s\n on "$@" — one line per argument, no interpretation.
	script := "#!/bin/sh\nfor a in \"$@\"; do printf '%s\\n' \"$a\"; done\n"
	if err := os.WriteFile(stub, []byte(script), 0o755); err != nil {
		t.Fatalf("write stub reth: %v", err)
	}

	c := exec.Command("sh", "-c", cmd)
	// The stub dir goes FIRST, but the real PATH stays behind it. Handing the
	// shell a PATH with nothing but the stub on it would defeat the injection
	// tests silently: `touch` would not resolve, so an injected command could
	// never leave a trace even when the quoting was broken and the shell did
	// genuinely try to run it.
	c.Env = append(os.Environ(), "PATH="+dir+string(os.PathListSeparator)+os.Getenv("PATH"))
	c.Dir = dir
	out, err := c.Output()
	if err != nil {
		t.Fatalf("sh -c %q: %v", cmd, err)
	}
	return strings.Split(strings.TrimRight(string(out), "\n"), "\n"), dir
}

func TestRethDownloadCommand_ArgumentsSurviveTheShellIntact(t *testing.T) {
	w := WireConfig{
		ChainID:     369,
		DataDir:     "/var/lib/valve-node-app/369",
		SnapshotKey: "vk_abc123",
	}
	cmd, err := RethDownloadCommand(w)
	if err != nil {
		t.Fatalf("RethDownloadCommand: %v", err)
	}

	args, _ := runThroughShell(t, cmd)
	want := []string{
		"download",
		"--chain", "pulsechain",
		"--datadir", "/var/lib/valve-node-app/369",
		"--manifest-url", "https://one.valve.city/snapshot/vk_abc123/evm/369/reth/manifest.json",
	}
	if len(args) != len(want) {
		t.Fatalf("argv: got %q, want %q", args, want)
	}
	for i := range want {
		if args[i] != want[i] {
			t.Errorf("argv[%d]: got %q, want %q", i, args[i], want[i])
		}
	}
}

// The snapshot key is typed by the operator into the wizard and lands inside a
// string this app hands to `sh -c`. A key with a quote in it must come out the
// far side as one argument with the quote in it — not as a second command.
//
// The canary is a file: if the shell ever runs the injected `touch`, the file
// exists, and no amount of plausible-looking argv can hide that.
func TestRethDownloadCommand_AKeyCannotEscapeIntoASecondCommand(t *testing.T) {
	for _, key := range []string{
		`vk_x'; touch pwned; echo '`,
		`vk_x'$(touch pwned)'`,
		"vk_x`touch pwned`",
		`vk_x; touch pwned`,
		`vk_x $(touch pwned)`,
	} {
		t.Run(key, func(t *testing.T) {
			cmd, err := RethDownloadCommand(WireConfig{ChainID: 369, DataDir: "/data", SnapshotKey: key})
			if err != nil {
				t.Fatalf("RethDownloadCommand: %v", err)
			}
			args, dir := runThroughShell(t, cmd)

			if _, err := os.Stat(filepath.Join(dir, "pwned")); err == nil {
				t.Fatalf("the injected command ran: %q produced a file the command had no business creating", key)
			}
			// The --manifest-url value must be exactly one argument, with the
			// operator's key embedded verbatim. Splitting it means the shell
			// re-parsed it, which is the same bug seen from the other side.
			if len(args) != 7 {
				t.Fatalf("argv: got %d arguments %q, want 7 — the key was re-parsed by the shell", len(args), args)
			}
			if want := SnapshotManifestURL(369, key); args[6] != want {
				t.Errorf("manifest url: got %q, want %q", args[6], want)
			}
		})
	}
}

// The stub `reth` writes nothing to the canary, so this proves the canary is
// capable of being observed at all — a check that can never fail proves
// nothing about the quoting tests above.
func TestRethDownloadCommand_TheInjectionCanaryActuallyFires(t *testing.T) {
	dir := t.TempDir()
	c := exec.Command("sh", "-c", "touch pwned")
	c.Dir = dir
	if err := c.Run(); err != nil {
		t.Fatalf("sh -c touch: %v", err)
	}
	if _, err := os.Stat(filepath.Join(dir, "pwned")); err != nil {
		t.Fatalf("the canary file cannot be observed even when the command definitely ran: %v", err)
	}
}

// A chain with no reth --chain name has no snapshot to download. Returning an
// error rather than an empty --chain is what stops the caller running reth
// download with an empty --chain value, which reth accepts and points at
// mainnet.
func TestRethDownloadCommand_RefusesAChainRethCannotName(t *testing.T) {
	for _, chainID := range []int{DevnetChainID, 0, 1337, 999999} {
		cmd, err := RethDownloadCommand(WireConfig{ChainID: chainID, DataDir: "/data", SnapshotKey: "vk_x"})
		if err == nil {
			t.Errorf("chain %d: got command %q, want an error — there is no snapshot for a chain reth cannot name", chainID, cmd)
		}
		if cmd != "" {
			t.Errorf("chain %d: got a command alongside the error: %q", chainID, cmd)
		}
	}
}

// Every chain the catalog says supports snapshots must produce a command;
// otherwise the wizard offers a restore that fails at the last step.
func TestRethDownloadCommand_CoversEveryChainWithARethName(t *testing.T) {
	for chainID, name := range rethChainName {
		cmd, err := RethDownloadCommand(WireConfig{ChainID: chainID, DataDir: "/data", SnapshotKey: "vk_x"})
		if err != nil {
			t.Errorf("chain %d (%s): %v", chainID, name, err)
			continue
		}
		if !strings.Contains(cmd, "--chain '"+name+"'") {
			t.Errorf("chain %d: command does not carry reth's own name %q: %s", chainID, name, cmd)
		}
	}
}

// The manifest URL is a runbook contract, not an internal detail: it is the
// path one.valve.city serves, and a typo here is a 404 the operator sees as
// "snapshots are broken".
func TestSnapshotManifestURL_MatchesTheRunbookPath(t *testing.T) {
	got := SnapshotManifestURL(369, "vk_abc")
	want := "https://one.valve.city/snapshot/vk_abc/evm/369/reth/manifest.json"
	if got != want {
		t.Fatalf("got %q, want %q", got, want)
	}
}
