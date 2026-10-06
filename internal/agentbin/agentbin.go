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

// DevAgentsEnv, set to 1, lets a build with embedded agents use the
// developer agents in ~/.jumpgate/agents instead (ruling P35). Without it a
// release ignores that directory, so a forgotten one never replaces the
// agents it was built with.
const DevAgentsEnv = "JUMPGATE_DEV_AGENTS"

// maxAgentSize bounds decompression of an embedded agent.
const maxAgentSize = 256 << 20

// Seams for tests; production never reassigns them. embeddedFS is set by
// embed_on.go in builds made with -tags embedagents and is nil otherwise.
var (
	embeddedFS        fs.FS
	selfIsStaticLinux = buildinfo.SelfIsStaticLinux
	goarch            = runtime.GOARCH
	readSelf          = func() ([]byte, error) {
		exe, err := os.Executable()
		if err != nil {
			return nil, err
		}
		return os.ReadFile(exe)
	}
	stderr io.Writer = os.Stderr
)

// Load returns the verified Linux agent for arch ("amd64" or "arm64") and
// where it came from. It returns the bytes themselves, not a path, so the
// bytes checked are the bytes uploaded: nothing reads the file again between
// the check and the upload.
//
// The order (spec D14 as amended by ruling P35): the developer agents in
// ~/.jumpgate/agents, in a build without embedded agents, or in one with them
// only when JUMPGATE_DEV_AGENTS=1; then the agents embedded in a release
// build, also cached under ~/.jumpgate/agents-cache/<version>/ (D13); then
// this binary, only when it is a static Linux build of that arch (B-2);
// otherwise an error saying how to get one.
func Load(arch string) ([]byte, Source, error) {
	if arch != "amd64" && arch != "arm64" {
		return nil, "", fmt.Errorf("agentbin: no agent for linux/%s; the jumpgate agent runs on linux/amd64 and linux/arm64", arch)
	}
	dev, err := devDir()
	if err != nil {
		return nil, "", err
	}
	if embeddedFS == nil || os.Getenv(DevAgentsEnv) == "1" {
		if b, err := fromDir(dev, arch); err == nil {
			return b, SourceDevDir, nil
		} else if !errors.Is(err, fs.ErrNotExist) {
			return nil, "", err
		}
	}
	if embeddedFS != nil {
		base, err := config.Dir()
		if err != nil {
			return nil, "", err
		}
		b, err := extractFrom(embeddedFS, arch, filepath.Join(base, "agents-cache", buildinfo.Version()))
		if err != nil {
			return nil, "", err
		}
		return b, SourceEmbedded, nil
	}
	if selfIsStaticLinux() && goarch == arch {
		b, err := readSelf()
		if err != nil {
			return nil, "", fmt.Errorf("agentbin: read this binary: %w", err)
		}
		// The bytes read are checked, not just the build: they are what goes
		// to the box.
		if !buildinfo.IsStaticELF(bytes.NewReader(b)) {
			return nil, "", fmt.Errorf("agentbin: this binary is not a statically linked executable, so it cannot be the agent for linux/%s; run scripts/build-agents.sh", arch)
		}
		return b, SourceSelf, nil
	}
	return nil, "", fmt.Errorf("agentbin: this build of jumpgate carries no agent for linux/%s. Use a release build (they embed both agents), or for development run scripts/build-agents.sh, which writes them to %s", arch, dev)
}

// devDir is ~/.jumpgate/agents, where scripts/build-agents.sh writes by
// default.
func devDir() (string, error) {
	base, err := config.Dir()
	if err != nil {
		return "", err
	}
	return filepath.Join(base, "agents"), nil
}

// fromDir returns dir's agent for arch after checking it against the
// SHA256SUMS beside it. Only a missing binary reports fs.ErrNotExist, so the
// caller falls through; anything else about a present binary is an error.
// The directory, the binary and the sums must be the user's own and private
// (P35): anyone else who could write them could choose what gets uploaded,
// sums and all.
func fromDir(dir, arch string) ([]byte, error) {
	name := "jumpgate-linux-" + arch
	path := filepath.Join(dir, name)
	f, err := os.Open(path)
	if errors.Is(err, fs.ErrNotExist) {
		return nil, err
	}
	if err != nil {
		return nil, fmt.Errorf("agentbin: %v", err)
	}
	defer f.Close()
	if err := fsperm.CheckPrivate(dir); err != nil {
		return nil, fmt.Errorf("agentbin: refusing the development agents: %w", err)
	}
	content, err := readPrivate(f)
	if err != nil {
		return nil, err
	}
	sf, err := os.Open(filepath.Join(dir, "SHA256SUMS"))
	if err != nil {
		return nil, fmt.Errorf("agentbin: %s has no readable SHA256SUMS beside it (%v); rerun scripts/build-agents.sh", path, err)
	}
	defer sf.Close()
	sums, err := readPrivate(sf)
	if err != nil {
		return nil, err
	}
	if err := checkSum(sums, name, content); err != nil {
		return nil, fmt.Errorf("agentbin: %s: %v", path, err)
	}
	return content, nil
}

// readPrivate reads f after checking, on the open file, that only its owner
// can change it.
func readPrivate(f *os.File) ([]byte, error) {
	if err := fsperm.CheckPrivateFile(f); err != nil {
		return nil, fmt.Errorf("agentbin: refusing the development agents: %w", err)
	}
	b, err := io.ReadAll(f)
	if err != nil {
		return nil, fmt.Errorf("agentbin: %v", err)
	}
	return b, nil
}

// extractFrom decompresses the embedded agent for arch, checks it against
// the embedded sums, caches it owner-only under cacheDir and returns the
// verified bytes. The cache is rewritten if its content differs, so a damaged
// cache heals itself; it is never what gets uploaded.
func extractFrom(fsys fs.FS, arch, cacheDir string) ([]byte, error) {
	name := "jumpgate-linux-" + arch
	damaged := func(err error) error {
		return fmt.Errorf("agentbin: the embedded agent for linux/%s is damaged: %v", arch, err)
	}
	sums, err := fs.ReadFile(fsys, "embedded/SHA256SUMS")
	if err != nil {
		return nil, fmt.Errorf("agentbin: embedded SHA256SUMS: %w", err)
	}
	f, err := fsys.Open("embedded/" + name + ".gz")
	if err != nil {
		return nil, fmt.Errorf("agentbin: embedded agent for linux/%s: %w", arch, err)
	}
	defer f.Close()
	zr, err := gzip.NewReader(f)
	if err != nil {
		return nil, damaged(err)
	}
	content, err := io.ReadAll(io.LimitReader(zr, maxAgentSize+1))
	if err != nil {
		return nil, damaged(err)
	}
	if len(content) > maxAgentSize {
		return nil, damaged(fmt.Errorf("it decompresses to more than %d MiB", maxAgentSize>>20))
	}
	if err := checkSum(sums, name, content); err != nil {
		return nil, damaged(err)
	}
	path := filepath.Join(cacheDir, name)
	if cached, err := os.ReadFile(path); err == nil && bytes.Equal(cached, content) {
		return content, nil
	}
	if err := fsperm.MkdirPrivate(cacheDir); err != nil {
		return nil, err
	}
	if err := fsperm.WriteFilePrivate(path, content); err != nil {
		return nil, err
	}
	return content, nil
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
