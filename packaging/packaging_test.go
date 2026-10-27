// Package packaging holds tests for the release packaging files; it has no
// code of its own.
package packaging

import (
	"os"
	"os/exec"
	"strings"
	"testing"
)

func read(t *testing.T, path string) string {
	t.Helper()
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return string(b)
}

func TestNFPMShipsTheServiceFiles(t *testing.T) {
	y := read(t, "../.goreleaser.yaml")
	for _, want := range []string{
		"nfpms:",
		"formats: [deb, rpm]",
		"dst: /usr/lib/systemd/system/jumpgate.service",
		"dst: /usr/lib/sysusers.d/jumpgate.conf",
		"postinstall: packaging/linux/server/postinst.sh",
	} {
		if !strings.Contains(y, want) {
			t.Errorf(".goreleaser.yaml lacks %q", want)
		}
	}
	// The box's agent binary is never the package's to own or replace.
	if strings.Contains(y, "/usr/local/lib/jumpgate") {
		t.Error(".goreleaser.yaml ships into /usr/local/lib/jumpgate, the agent's path")
	}
}

func TestPostinstNeverEnablesOrRestartsAManualService(t *testing.T) {
	const path = "linux/server/postinst.sh"
	s := read(t, path)
	for i, line := range strings.Split(s, "\n") {
		code, _, _ := strings.Cut(line, "#")
		if strings.Contains(code, "enable") {
			t.Errorf("line %d enables something: %q", i+1, line)
		}
		if strings.Contains(code, "restart") {
			if strings.Contains(strings.ReplaceAll(code, "try-restart", ""), "restart") {
				t.Errorf("line %d restarts without try-restart: %q", i+1, line)
			}
			if !strings.Contains(code, "50-vault-seal.conf") {
				t.Errorf("line %d try-restarts a service that is not sealed: %q", i+1, line)
			}
		}
	}
	if _, err := exec.LookPath("sh"); err != nil {
		t.Skip("no sh to check the syntax with")
	}
	if out, err := exec.Command("sh", "-n", path).CombinedOutput(); err != nil {
		t.Errorf("sh -n %s: %v\n%s", path, err, out)
	}
}
