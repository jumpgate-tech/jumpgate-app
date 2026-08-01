package catalog

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// A realistic versions.json matching the shape one.valve.city serves for
// evm/369 (verified live 2026-07-31): three entries, the FIRST spelling the
// range as `version_range` and the rest as `reth_version_range`, out of
// generated_at order so newest-wins is actually exercised.
const versions369 = `{
  "chain_id": 369,
  "available_versions": [
    {
      "arch": "evm",
      "chain_id": 369,
      "client": "reth",
      "version_range": "2.2.x",
      "manifest_url": "https://one.valve.city/snapshot/evm/369/reth/2.2/1779991009/manifest.json",
      "block": 26645565,
      "timestamp": 1779991009,
      "size_bytes": 1205812068352,
      "generated_at": 1779991009
    },
    {
      "reth_version_range": "2.3.x",
      "manifest_url": "https://one.valve.city/snapshot/evm/369/reth/2.3/1785121890/manifest.json",
      "block": 27132853,
      "timestamp": 1785171299,
      "size_bytes": 1223705475087,
      "generated_at": 1785171299
    },
    {
      "reth_version_range": "2.2.x",
      "manifest_url": "https://one.valve.city/snapshot/evm/369/reth/2.2/1782684425/manifest.json",
      "block": 26902110,
      "timestamp": 1782725284,
      "size_bytes": 1215529321326,
      "generated_at": 1782725284
    }
  ]
}`

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
		ChainID: 369,
		DataDir: "/var/lib/valve-node-app/369",
	}
	manifestURL := "https://one.valve.city/snapshot/vk_abc123/evm/369/reth/2.3/1785121890/manifest.json"
	cmd, err := RethDownloadCommand(w, manifestURL)
	if err != nil {
		t.Fatalf("RethDownloadCommand: %v", err)
	}

	args, _ := runThroughShell(t, cmd)
	want := []string{
		"download",
		"--chain", "pulsechain",
		"--datadir", "/var/lib/valve-node-app/369",
		"--manifest-url", manifestURL,
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

// The snapshot key is typed by the operator into the wizard and, after
// InjectSnapshotKey, lands inside the manifest URL this app hands to `sh -c`.
// A key with a quote in it must come out the far side as one argument with the
// quote in it — not as a second command.
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
			manifestURL, err := InjectSnapshotKey("https://one.valve.city/snapshot/evm/369/reth/2.3/1/manifest.json", key)
			if err != nil {
				t.Fatalf("InjectSnapshotKey: %v", err)
			}
			cmd, err := RethDownloadCommand(WireConfig{ChainID: 369, DataDir: "/data"}, manifestURL)
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
			if args[6] != manifestURL {
				t.Errorf("manifest url: got %q, want %q", args[6], manifestURL)
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
		cmd, err := RethDownloadCommand(WireConfig{ChainID: chainID, DataDir: "/data"}, "https://one.valve.city/snapshot/vk_x/evm/1/reth/2.3/1/manifest.json")
		if err == nil {
			t.Errorf("chain %d: got command %q, want an error — there is no snapshot for a chain reth cannot name", chainID, cmd)
		}
		if cmd != "" {
			t.Errorf("chain %d: got a command alongside the error: %q", chainID, cmd)
		}
	}
}

// A resolved manifest URL is the whole point of the new flow — refusing an
// empty one stops the step running `reth download` with no --manifest-url
// value, which is how the old code 404'd.
func TestRethDownloadCommand_RefusesAnEmptyManifestURL(t *testing.T) {
	cmd, err := RethDownloadCommand(WireConfig{ChainID: 369, DataDir: "/data"}, "  ")
	if err == nil {
		t.Errorf("got command %q, want an error for an empty manifest URL", cmd)
	}
}

// Every chain the catalog says supports snapshots must produce a command;
// otherwise the wizard offers a restore that fails at the last step.
func TestRethDownloadCommand_CoversEveryChainWithARethName(t *testing.T) {
	for chainID, name := range rethChainName {
		cmd, err := RethDownloadCommand(WireConfig{ChainID: chainID, DataDir: "/data"}, "https://one.valve.city/snapshot/vk_x/evm/1/reth/2.3/1/manifest.json")
		if err != nil {
			t.Errorf("chain %d (%s): %v", chainID, name, err)
			continue
		}
		if !strings.Contains(cmd, "--chain '"+name+"'") {
			t.Errorf("chain %d: command does not carry reth's own name %q: %s", chainID, name, cmd)
		}
	}
}

// ---------------------------------------------------------------------
// SnapshotVersionsURL
// ---------------------------------------------------------------------

// The discovery URL is a gateway contract, not an internal detail: it is the
// path one.valve.city serves keyless, and a typo here is a 404 the operator
// sees as "snapshots are broken".
func TestSnapshotVersionsURL_MatchesTheGatewayPath(t *testing.T) {
	got := SnapshotVersionsURL(369)
	want := "https://one.valve.city/snapshot/evm/369/reth/versions.json"
	if got != want {
		t.Fatalf("got %q, want %q", got, want)
	}
}

// ---------------------------------------------------------------------
// ParseRethMajorMinor
// ---------------------------------------------------------------------

func TestParseRethMajorMinor(t *testing.T) {
	for _, tc := range []struct {
		out, want string
	}{
		{"reth Version: 2.2.0-pulse", "2.2"},
		{"Reth Version: 2.2.0-pulse\nCommit SHA: deadbeef", "2.2"},
		{"reth 1.1.0", "1.1"},
		{"reth Version: 2.3.4", "2.3"},
	} {
		got, err := ParseRethMajorMinor(tc.out)
		if err != nil {
			t.Errorf("ParseRethMajorMinor(%q): %v", tc.out, err)
			continue
		}
		if got != tc.want {
			t.Errorf("ParseRethMajorMinor(%q) = %q, want %q", tc.out, got, tc.want)
		}
	}
}

func TestParseRethMajorMinor_ErrorsWhenNoVersionPresent(t *testing.T) {
	if got, err := ParseRethMajorMinor("command not found: reth\n"); err == nil {
		t.Errorf("got %q, want an error when the output carries no version", got)
	}
}

// ---------------------------------------------------------------------
// SelectSnapshotManifest
// ---------------------------------------------------------------------

// The reth on the box is 2.2, and TWO 2.2 snapshots exist — the newer by
// generated_at must win, and the 2.3 entry must be ignored, even though the
// 2.3 entry sits between them in file order and the first 2.2 entry uses the
// odd `version_range` spelling.
func TestSelectSnapshotManifest_NewestMatchingVersionWins(t *testing.T) {
	got, err := SelectSnapshotManifest([]byte(versions369), "2.2", 369)
	if err != nil {
		t.Fatalf("SelectSnapshotManifest: %v", err)
	}
	want := "https://one.valve.city/snapshot/evm/369/reth/2.2/1782684425/manifest.json"
	if got != want {
		t.Fatalf("got %q, want %q (the newer 2.2 by generated_at)", got, want)
	}
}

// The 2.3 entry uses the `reth_version_range` spelling; a reth on 2.3 must
// still resolve it — proving both key spellings are honoured for selection,
// not just tolerated by the decoder.
func TestSelectSnapshotManifest_HonoursBothRangeSpellings(t *testing.T) {
	got, err := SelectSnapshotManifest([]byte(versions369), "2.3", 369)
	if err != nil {
		t.Fatalf("SelectSnapshotManifest: %v", err)
	}
	want := "https://one.valve.city/snapshot/evm/369/reth/2.3/1785121890/manifest.json"
	if got != want {
		t.Fatalf("got %q, want %q", got, want)
	}
	// And the first entry alone (the `version_range` spelling) resolves for 2.2.
	if _, err := SelectSnapshotManifest([]byte(versions369), "2.2", 369); err != nil {
		t.Fatalf("the version_range spelling did not resolve: %v", err)
	}
}

// A reth on a minor line Valve has not cut a snapshot for must fail loudly,
// naming the chain, the version, and the ranges that DO exist — not silently
// pick a mismatched snapshot.
func TestSelectSnapshotManifest_ErrorsWhenNoRangeMatches(t *testing.T) {
	_, err := SelectSnapshotManifest([]byte(versions369), "9.9", 369)
	if err == nil {
		t.Fatal("got nil, want an error for a version with no matching snapshot")
	}
	for _, want := range []string{"369", "9.9", "2.2.x", "2.3.x"} {
		if !strings.Contains(err.Error(), want) {
			t.Errorf("error %q does not carry %q", err, want)
		}
	}
}

// Mainnet (evm/1) publishes no snapshot; in practice curl -f 404s before this
// is reached, but an empty available_versions must also fail clearly rather
// than return an empty URL that 404s hours later.
func TestSelectSnapshotManifest_ErrorsOnEmptyAvailableVersions(t *testing.T) {
	_, err := SelectSnapshotManifest([]byte(`{"chain_id":1,"available_versions":[]}`), "2.2", 1)
	if err == nil {
		t.Fatal("got nil, want an error when no snapshots are published")
	}
	if !strings.Contains(err.Error(), "1") {
		t.Errorf("error %q does not name the chain", err)
	}
}

func TestSelectSnapshotManifest_ErrorsOnMalformedJSON(t *testing.T) {
	if _, err := SelectSnapshotManifest([]byte("not json at all"), "2.2", 369); err == nil {
		t.Fatal("got nil, want an error for malformed versions.json")
	}
}

// ---------------------------------------------------------------------
// InjectSnapshotKey
// ---------------------------------------------------------------------

func TestInjectSnapshotKey_InsertsTheKeyIntoThePath(t *testing.T) {
	got, err := InjectSnapshotKey("https://one.valve.city/snapshot/evm/369/reth/2.3/1785121890/manifest.json", "vk_abc123")
	if err != nil {
		t.Fatalf("InjectSnapshotKey: %v", err)
	}
	want := "https://one.valve.city/snapshot/vk_abc123/evm/369/reth/2.3/1785121890/manifest.json"
	if got != want {
		t.Fatalf("got %q, want %q", got, want)
	}
}

func TestInjectSnapshotKey_RejectsAURLWithoutTheExpectedShape(t *testing.T) {
	for _, bad := range []string{
		"https://one.valve.city/snapshot/vk_x/evm/369/reth/manifest.json", // already keyed — no keyless marker
		"https://example.com/some/other/path.json",
		"",
	} {
		if got, err := InjectSnapshotKey(bad, "vk_abc123"); err == nil {
			t.Errorf("InjectSnapshotKey(%q) = %q, want an error for a URL without the /snapshot/evm/ shape", bad, got)
		}
	}
}

func TestInjectSnapshotKey_RejectsAnEmptyKey(t *testing.T) {
	if _, err := InjectSnapshotKey("https://one.valve.city/snapshot/evm/369/reth/manifest.json", "  "); err == nil {
		t.Fatal("got nil, want an error when the key is empty")
	}
}
