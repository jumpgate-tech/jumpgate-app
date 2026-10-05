// Package agentbin finds the Linux agent binary a pairing uploads.
//
// This is the lookup the server's pair handler used before the CLI needed it
// too (local pairing runs bootstrap in the foreground). Task 3 of the
// platform plan replaces this file: it adds the embedded agents and the D14
// order (developer override, then embedded, then a static Linux self).
// Reporting, in reporting.go, stays the one place both callers go through.
package agentbin

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/valve-tech/jumpgate/internal/config"
)

// Source says where an agent binary came from, for the pairing log.
type Source string

const (
	SourceDevDir Source = "dev override ~/.jumpgate/agents"
	SourceSelf   Source = "this binary"
)

// Path returns a local file holding the Linux agent for arch, and where it
// came from: this binary when it already is one, otherwise
// ~/.jumpgate/agents/jumpgate-linux-<arch>, checked against the SHA256SUMS
// written beside it by scripts/build-agents.sh.
func Path(arch string) (string, Source, error) {
	if runtime.GOOS == "linux" && runtime.GOARCH == arch {
		exe, err := os.Executable()
		return exe, SourceSelf, err
	}
	base, err := config.Dir()
	if err != nil {
		return "", "", err
	}
	dir := filepath.Join(base, "agents")
	name := "jumpgate-linux-" + arch
	path := filepath.Join(dir, name)
	content, err := os.ReadFile(path)
	if err != nil {
		return "", "", fmt.Errorf("no agent binary for linux/%s at %s; run scripts/build-agents.sh", arch, path)
	}
	sums, err := os.ReadFile(filepath.Join(dir, "SHA256SUMS"))
	if err != nil {
		return "", "", fmt.Errorf("no SHA256SUMS beside %s", path)
	}
	got := sha256.Sum256(content)
	for _, line := range strings.Split(string(sums), "\n") {
		f := strings.Fields(line)
		if len(f) == 2 && strings.TrimPrefix(f[1], "*") == name {
			if f[0] != hex.EncodeToString(got[:]) {
				return "", "", fmt.Errorf("%s does not match its SHA256SUMS entry", path)
			}
			return path, SourceDevDir, nil
		}
	}
	return "", "", fmt.Errorf("%s is not listed in SHA256SUMS", name)
}
