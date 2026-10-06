package agentbin

import (
	"bytes"
	"compress/gzip"
	"crypto/sha256"
	"debug/elf"
	"encoding/hex"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"testing/fstest"

	"github.com/valve-tech/jumpgate/internal/fsperm"
	"github.com/valve-tech/jumpgate/internal/testutil"
)

func gz(t *testing.T, b []byte) []byte {
	t.Helper()
	var buf bytes.Buffer
	w := gzip.NewWriter(&buf)
	if _, err := w.Write(b); err != nil {
		t.Fatal(err)
	}
	if err := w.Close(); err != nil {
		t.Fatal(err)
	}
	return buf.Bytes()
}

func sumLine(name string, b []byte) string {
	s := sha256.Sum256(b)
	return hex.EncodeToString(s[:]) + "  " + name + "\n"
}

// withSeams resets every seam for one test: no embedded agents, not a
// static Linux build, amd64, a static amd64 image as this binary, and a
// buffer for stderr. JUMPGATE_DEV_AGENTS is unset.
func withSeams(t *testing.T) *bytes.Buffer {
	t.Helper()
	t.Setenv(DevAgentsEnv, "")
	self := testutil.ELF(t, elf.ET_EXEC, elf.EM_X86_64, false)
	var errOut bytes.Buffer
	oldFS, oldStatic, oldArch, oldSelf, oldErr := embeddedFS, selfIsStaticLinux, goarch, readSelf, stderr
	embeddedFS, selfIsStaticLinux, goarch = nil, func() bool { return false }, "amd64"
	readSelf = func() ([]byte, error) { return self, nil }
	stderr = &errOut
	t.Cleanup(func() {
		embeddedFS, selfIsStaticLinux, goarch, readSelf, stderr = oldFS, oldStatic, oldArch, oldSelf, oldErr
	})
	return &errOut
}

func embeddedWith(t *testing.T, arch string, content []byte, sums string) fstest.MapFS {
	t.Helper()
	return fstest.MapFS{
		"embedded/jumpgate-linux-" + arch + ".gz": {Data: gz(t, content)},
		"embedded/SHA256SUMS":                     {Data: []byte(sums)},
	}
}

func writeDevDir(t *testing.T, home, arch string, content []byte, sums string) string {
	t.Helper()
	dir := filepath.Join(home, ".jumpgate", "agents")
	if err := fsperm.MkdirPrivate(dir); err != nil {
		t.Fatal(err)
	}
	p := filepath.Join(dir, "jumpgate-linux-"+arch)
	if err := fsperm.WriteFilePrivate(p, content); err != nil {
		t.Fatal(err)
	}
	if sums != "" {
		if err := fsperm.WriteFilePrivate(filepath.Join(dir, "SHA256SUMS"), []byte(sums)); err != nil {
			t.Fatal(err)
		}
	}
	return p
}

func TestLoadRejectsAnUnsupportedArch(t *testing.T) {
	withSeams(t)
	testutil.Home(t)
	if _, _, err := Load("riscv64"); err == nil || !strings.Contains(err.Error(), "linux/amd64 and linux/arm64") {
		t.Fatalf("Load(riscv64) = %v, want an error naming the supported arches", err)
	}
}

// In a build without embedded agents the developer directory is the normal
// source, and says so.
func TestLoadUsesTheDevDirInADevBuild(t *testing.T) {
	withSeams(t)
	home := testutil.Home(t)
	dev := []byte("dev agent")
	writeDevDir(t, home, "amd64", dev, sumLine("jumpgate-linux-amd64", dev))
	got, src, err := Load("amd64")
	if err != nil || !bytes.Equal(got, dev) || src != SourceDevDir {
		t.Fatalf("Load = %q, %q, %v; want %q from %q", got, src, err, dev, SourceDevDir)
	}
}

// P35: a release build ignores a forgotten ~/.jumpgate/agents unless
// JUMPGATE_DEV_AGENTS=1 asks for it.
func TestLoadIgnoresTheDevDirInAnEmbeddedBuildByDefault(t *testing.T) {
	withSeams(t)
	home := testutil.Home(t)
	dev, emb := []byte("dev agent"), []byte("embedded agent")
	writeDevDir(t, home, "amd64", dev, sumLine("jumpgate-linux-amd64", dev))
	embeddedFS = embeddedWith(t, "amd64", emb, sumLine("jumpgate-linux-amd64", emb))
	got, src, err := Load("amd64")
	if err != nil || !bytes.Equal(got, emb) || src != SourceEmbedded {
		t.Fatalf("Load = %q, %q, %v; want the embedded agent", got, src, err)
	}
}

func TestLoadHonoursTheDevDirInAnEmbeddedBuildOnRequest(t *testing.T) {
	withSeams(t)
	home := testutil.Home(t)
	t.Setenv(DevAgentsEnv, "1")
	dev, emb := []byte("dev agent"), []byte("embedded agent")
	writeDevDir(t, home, "amd64", dev, sumLine("jumpgate-linux-amd64", dev))
	embeddedFS = embeddedWith(t, "amd64", emb, sumLine("jumpgate-linux-amd64", emb))
	got, src, err := Load("amd64")
	if err != nil || !bytes.Equal(got, dev) || src != SourceDevDir {
		t.Fatalf("Load = %q, %q, %v; want the dev agent", got, src, err)
	}
}

// A dev binary whose sums are missing or wrong is an error, never a reason
// to fall through to another source silently.
func TestLoadRefusesAnUnverifiableDevBinary(t *testing.T) {
	withSeams(t)
	home := testutil.Home(t)
	writeDevDir(t, home, "amd64", []byte("dev agent"), "")
	if _, _, err := Load("amd64"); err == nil || !strings.Contains(err.Error(), "SHA256SUMS") {
		t.Fatalf("no sums: Load = %v, want an error naming SHA256SUMS", err)
	}
	writeDevDir(t, home, "amd64", []byte("dev agent"), sumLine("jumpgate-linux-amd64", []byte("something else")))
	if _, _, err := Load("amd64"); err == nil || !strings.Contains(err.Error(), "does not match") {
		t.Fatalf("bad sums: Load = %v, want a mismatch error", err)
	}
}

// P35: another user who can write the dev directory, or a file in it, could
// put their own agent (and its sum) there; it is refused, not uploaded.
func TestLoadRefusesADevDirOthersCanWrite(t *testing.T) {
	testutil.RequireUnix(t)
	for _, which := range []string{"dir", "binary", "SHA256SUMS"} {
		t.Run(which, func(t *testing.T) {
			withSeams(t)
			home := testutil.Home(t)
			dev := []byte("dev agent")
			bin := writeDevDir(t, home, "amd64", dev, sumLine("jumpgate-linux-amd64", dev))
			target := map[string]string{"dir": filepath.Dir(bin), "binary": bin, "SHA256SUMS": filepath.Join(filepath.Dir(bin), "SHA256SUMS")}[which]
			mode := os.FileMode(0o622)
			if which == "dir" {
				mode = 0o777
			}
			if err := os.Chmod(target, mode); err != nil {
				t.Fatal(err)
			}
			if _, _, err := Load("amd64"); !errors.Is(err, fsperm.ErrNotPrivate) {
				t.Fatalf("Load = %v, want a refusal wrapping fsperm.ErrNotPrivate", err)
			}
		})
	}
}

func TestLoadExtractsAndVerifiesTheEmbeddedAgent(t *testing.T) {
	withSeams(t)
	home := testutil.Home(t)
	content := []byte("embedded agent")
	embeddedFS = embeddedWith(t, "arm64", content, sumLine("jumpgate-linux-arm64", content))
	got, src, err := Load("arm64")
	if err != nil || src != SourceEmbedded || !bytes.Equal(got, content) {
		t.Fatalf("Load = %q, %q, %v; want %q embedded", got, src, err, content)
	}
	cached := filepath.Join(home, ".jumpgate", "agents-cache", "dev", "jumpgate-linux-arm64")
	if b, err := os.ReadFile(cached); err != nil || !bytes.Equal(b, content) {
		t.Fatalf("cache holds %q, %v; want %q", b, err, content)
	}
	testutil.AssertPrivate(t, cached)
	if again, _, err := Load("arm64"); err != nil || !bytes.Equal(again, content) {
		t.Fatalf("second Load = %q, %v", again, err)
	}
}

// The cache is verified on every use (D13): a cached file that was changed
// after extraction is rewritten from the embedded copy, never uploaded.
func TestLoadRewritesADamagedCache(t *testing.T) {
	withSeams(t)
	home := testutil.Home(t)
	content := []byte("embedded agent")
	embeddedFS = embeddedWith(t, "amd64", content, sumLine("jumpgate-linux-amd64", content))
	if _, _, err := Load("amd64"); err != nil {
		t.Fatal(err)
	}
	cached := filepath.Join(home, ".jumpgate", "agents-cache", "dev", "jumpgate-linux-amd64")
	if err := os.WriteFile(cached, []byte("tampered"), 0o600); err != nil {
		t.Fatal(err)
	}
	got, src, err := Load("amd64")
	if err != nil || src != SourceEmbedded || !bytes.Equal(got, content) {
		t.Fatalf("Load = %q, %q, %v; want the embedded agent", got, src, err)
	}
	if b, _ := os.ReadFile(cached); !bytes.Equal(b, content) {
		t.Fatalf("cache holds %q after Load, want %q", b, content)
	}
}

func TestLoadRefusesADamagedEmbeddedAgent(t *testing.T) {
	withSeams(t)
	testutil.Home(t)
	embeddedFS = embeddedWith(t, "amd64", []byte("tampered"), sumLine("jumpgate-linux-amd64", []byte("original")))
	if _, _, err := Load("amd64"); err == nil || !strings.Contains(err.Error(), "damaged") {
		t.Fatalf("Load = %v, want a damaged-agent error", err)
	}
}

// B-2: the controller uploads itself only when it is a static Linux build of
// the very arch the box needs, and the bytes it read are a static ELF.
func TestLoadUsesSelfOnlyForAStaticLinuxBuildOfTheSameArch(t *testing.T) {
	withSeams(t)
	testutil.Home(t)
	self, _ := readSelf()
	selfIsStaticLinux = func() bool { return true }
	if got, src, err := Load("amd64"); err != nil || src != SourceSelf || !bytes.Equal(got, self) {
		t.Fatalf("same arch: Load = %d bytes, %q, %v; want self", len(got), src, err)
	}
	if _, _, err := Load("arm64"); err == nil {
		t.Fatal("other arch: Load used self")
	}
	selfIsStaticLinux = func() bool { return false }
	if _, _, err := Load("amd64"); err == nil || !strings.Contains(err.Error(), "scripts/build-agents.sh") {
		t.Fatalf("cgo build: Load = %v, want the no-agent error naming scripts/build-agents.sh", err)
	}
}

// The bytes read from this binary are checked themselves: a dynamically
// linked image is never returned, whatever SelfIsStaticLinux said earlier.
func TestLoadRefusesADynamicSelfImage(t *testing.T) {
	withSeams(t)
	testutil.Home(t)
	selfIsStaticLinux = func() bool { return true }
	readSelf = func() ([]byte, error) { return testutil.ELF(t, elf.ET_DYN, elf.EM_X86_64, true), nil }
	if _, _, err := Load("amd64"); err == nil || !strings.Contains(err.Error(), "statically linked") {
		t.Fatalf("Load = %v, want a refusal of a dynamic self image", err)
	}
}
