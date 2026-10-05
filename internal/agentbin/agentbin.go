// Package agentbin finds the Linux agent binary that pairing uploads to a
// box. A controller on any OS can pair any Linux box, so it needs the agent
// for the box's arch, built statically and from the same commit (spec D1).
// Reporting, in reporting.go, is the one place the server's pair handler and
// the CLI's foreground local pairing go through.
package agentbin

import (
	"bytes"
	"compress/gzip"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/valve-tech/jumpgate/internal/buildinfo"
	"github.com/valve-tech/jumpgate/internal/config"
	"github.com/valve-tech/jumpgate/internal/fsperm"
)

// Source says where an agent binary came from, for the pairing log.
type Source string

const (
	SourceDevDir   Source = "dev override ~/.jumpgate/agents"
	SourceEmbedded Source = "embedded in this build"
	SourceSelf     Source = "this binary"
)

// maxAgentSize bounds decompression of an embedded agent.
const maxAgentSize = 256 << 20

// Seams for tests; production never reassigns them. embeddedFS is set by
// embed_on.go in builds made with -tags embedagents and is nil otherwise.
var (
	embeddedFS        fs.FS
	selfIsStaticLinux = buildinfo.SelfIsStaticLinux
	goarch            = runtime.GOARCH
	executable        = os.Executable
)

// Path returns a local file holding the verified Linux agent for arch
// ("amd64" or "arm64"), and where it came from. The order is: the developer
// override in ~/.jumpgate/agents (spec D14); the agent embedded in a release
// build, extracted to ~/.jumpgate/agents-cache/<version>/ (D13); this binary,
// only when it is a static Linux build of that arch (B-2); otherwise an
// error saying how to get one.
func Path(arch string) (string, Source, error) {
	if arch != "amd64" && arch != "arm64" {
		return "", "", fmt.Errorf("agentbin: no agent for linux/%s; the jumpgate agent runs on linux/amd64 and linux/arm64", arch)
	}
	base, err := config.Dir()
	if err != nil {
		return "", "", err
	}
	dev := filepath.Join(base, "agents")
	if p, err := fromDir(dev, arch); err == nil {
		return p, SourceDevDir, nil
	} else if !errors.Is(err, fs.ErrNotExist) {
		return "", "", err
	}
	if embeddedFS != nil {
		p, err := extractFrom(embeddedFS, arch, filepath.Join(base, "agents-cache", buildinfo.Version()))
		if err != nil {
			return "", "", err
		}
		return p, SourceEmbedded, nil
	}
	if selfIsStaticLinux() && goarch == arch {
		exe, err := executable()
		if err != nil {
			return "", "", err
		}
		return exe, SourceSelf, nil
	}
	return "", "", fmt.Errorf("agentbin: this build of jumpgate carries no agent for linux/%s. Use a release build (they embed both agents), or for development run scripts/build-agents.sh, which writes them to %s", arch, dev)
}

// fromDir returns dir's agent for arch after checking it against the
// SHA256SUMS beside it. Only a missing binary reports fs.ErrNotExist, so the
// caller falls through; anything else about a present binary is an error.
func fromDir(dir, arch string) (string, error) {
	name := "jumpgate-linux-" + arch
	path := filepath.Join(dir, name)
	content, err := os.ReadFile(path)
	if errors.Is(err, fs.ErrNotExist) {
		return "", err
	}
	if err != nil {
		return "", fmt.Errorf("agentbin: %v", err)
	}
	sums, err := os.ReadFile(filepath.Join(dir, "SHA256SUMS"))
	if err != nil {
		return "", fmt.Errorf("agentbin: %s has no readable SHA256SUMS beside it (%v); rerun scripts/build-agents.sh", path, err)
	}
	if err := checkSum(sums, name, content); err != nil {
		return "", fmt.Errorf("agentbin: %s: %v", path, err)
	}
	return path, nil
}

// extractFrom decompresses the embedded agent for arch, checks it against
// the embedded sums, and caches it owner-only under cacheDir. The cache is
// rewritten if its content differs, so a damaged cache heals itself.
func extractFrom(fsys fs.FS, arch, cacheDir string) (string, error) {
	name := "jumpgate-linux-" + arch
	damaged := func(err error) error {
		return fmt.Errorf("agentbin: the embedded agent for linux/%s is damaged: %v", arch, err)
	}
	sums, err := fs.ReadFile(fsys, "embedded/SHA256SUMS")
	if err != nil {
		return "", fmt.Errorf("agentbin: embedded SHA256SUMS: %w", err)
	}
	f, err := fsys.Open("embedded/" + name + ".gz")
	if err != nil {
		return "", fmt.Errorf("agentbin: embedded agent for linux/%s: %w", arch, err)
	}
	defer f.Close()
	zr, err := gzip.NewReader(f)
	if err != nil {
		return "", damaged(err)
	}
	content, err := io.ReadAll(io.LimitReader(zr, maxAgentSize+1))
	if err != nil {
		return "", damaged(err)
	}
	if len(content) > maxAgentSize {
		return "", damaged(fmt.Errorf("it decompresses to more than %d MiB", maxAgentSize>>20))
	}
	if err := checkSum(sums, name, content); err != nil {
		return "", damaged(err)
	}
	path := filepath.Join(cacheDir, name)
	if cached, err := os.ReadFile(path); err == nil && bytes.Equal(cached, content) {
		return path, nil
	}
	if err := fsperm.MkdirPrivate(cacheDir); err != nil {
		return "", err
	}
	if err := fsperm.WriteFilePrivate(path, content); err != nil {
		return "", err
	}
	return path, nil
}

// checkSum finds name in a sha256sum-format list and compares content to it.
func checkSum(sums []byte, name string, content []byte) error {
	got := sha256.Sum256(content)
	for _, line := range strings.Split(string(sums), "\n") {
		f := strings.Fields(line)
		if len(f) == 2 && strings.TrimPrefix(f[1], "*") == name {
			if !strings.EqualFold(f[0], hex.EncodeToString(got[:])) {
				return fmt.Errorf("%s does not match its SHA256SUMS entry", name)
			}
			return nil
		}
	}
	return fmt.Errorf("%s is not listed in SHA256SUMS", name)
}
