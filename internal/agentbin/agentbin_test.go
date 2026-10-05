package agentbin

import (
	"bytes"
	"compress/gzip"
	"crypto/sha256"
	"encoding/hex"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"testing/fstest"

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
// static Linux build, amd64.
func withSeams(t *testing.T) {
	t.Helper()
	oldFS, oldStatic, oldArch, oldExe := embeddedFS, selfIsStaticLinux, goarch, executable
	embeddedFS, selfIsStaticLinux, goarch = nil, func() bool { return false }, "amd64"
	executable = func() (string, error) { return "/self/jumpgate", nil }
	t.Cleanup(func() { embeddedFS, selfIsStaticLinux, goarch, executable = oldFS, oldStatic, oldArch, oldExe })
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
	if err := os.MkdirAll(dir, 0o700); err != nil {
		t.Fatal(err)
	}
	p := filepath.Join(dir, "jumpgate-linux-"+arch)
	if err := os.WriteFile(p, content, 0o755); err != nil {
		t.Fatal(err)
	}
	if sums != "" {
		if err := os.WriteFile(filepath.Join(dir, "SHA256SUMS"), []byte(sums), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	return p
}

func TestPathRejectsAnUnsupportedArch(t *testing.T) {
	withSeams(t)
	testutil.Home(t)
	if _, _, err := Path("riscv64"); err == nil || !strings.Contains(err.Error(), "linux/amd64 and linux/arm64") {
		t.Fatalf("Path(riscv64) = %v, want an error naming the supported arches", err)
	}
}

// Review Focus 4 / D14: the developer override wins, and says so.
func TestPathPrefersTheDevDirAndNamesIt(t *testing.T) {
	withSeams(t)
	home := testutil.Home(t)
	dev := []byte("dev agent")
	want := writeDevDir(t, home, "amd64", dev, sumLine("jumpgate-linux-amd64", dev))
	embeddedFS = embeddedWith(t, "amd64", []byte("embedded agent"), sumLine("jumpgate-linux-amd64", []byte("embedded agent")))
	got, src, err := Path("amd64")
	if err != nil || got != want || src != SourceDevDir {
		t.Fatalf("Path = %q, %q, %v; want %q, %q", got, src, err, want, SourceDevDir)
	}
}

// A dev binary whose sums are missing or wrong is an error, never a reason
// to fall through to another source silently.
func TestPathRefusesAnUnverifiableDevBinary(t *testing.T) {
	withSeams(t)
	home := testutil.Home(t)
	writeDevDir(t, home, "amd64", []byte("dev agent"), "")
	if _, _, err := Path("amd64"); err == nil || !strings.Contains(err.Error(), "SHA256SUMS") {
		t.Fatalf("no sums: Path = %v, want an error naming SHA256SUMS", err)
	}
	writeDevDir(t, home, "amd64", []byte("dev agent"), sumLine("jumpgate-linux-amd64", []byte("something else")))
	if _, _, err := Path("amd64"); err == nil || !strings.Contains(err.Error(), "does not match") {
		t.Fatalf("bad sums: Path = %v, want a mismatch error", err)
	}
}

func TestPathExtractsAndVerifiesTheEmbeddedAgent(t *testing.T) {
	withSeams(t)
	testutil.Home(t)
	content := []byte("embedded agent")
	embeddedFS = embeddedWith(t, "arm64", content, sumLine("jumpgate-linux-arm64", content))
	p, src, err := Path("arm64")
	if err != nil || src != SourceEmbedded {
		t.Fatalf("Path = %q, %q, %v; want the embedded source", p, src, err)
	}
	got, err := os.ReadFile(p)
	if err != nil || !bytes.Equal(got, content) {
		t.Fatalf("extracted %q, %v; want %q", got, err, content)
	}
	testutil.AssertPrivate(t, p)
	again, _, err := Path("arm64")
	if err != nil || again != p {
		t.Fatalf("second Path = %q, %v; want the cached %q", again, err, p)
	}
}

// The cache is verified on every use (D13): a cached file that was changed
// after extraction is rewritten from the embedded copy, never uploaded.
func TestPathRewritesADamagedCache(t *testing.T) {
	withSeams(t)
	testutil.Home(t)
	content := []byte("embedded agent")
	embeddedFS = embeddedWith(t, "amd64", content, sumLine("jumpgate-linux-amd64", content))
	p, _, err := Path("amd64")
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(p, []byte("tampered"), 0o600); err != nil {
		t.Fatal(err)
	}
	again, src, err := Path("amd64")
	if err != nil || again != p || src != SourceEmbedded {
		t.Fatalf("Path = %q, %q, %v; want the embedded agent at %q", again, src, err, p)
	}
	if got, _ := os.ReadFile(p); !bytes.Equal(got, content) {
		t.Fatalf("cache holds %q after Path, want %q", got, content)
	}
	testutil.AssertPrivate(t, p)
}

func TestPathRefusesADamagedEmbeddedAgent(t *testing.T) {
	withSeams(t)
	testutil.Home(t)
	embeddedFS = embeddedWith(t, "amd64", []byte("tampered"), sumLine("jumpgate-linux-amd64", []byte("original")))
	if _, _, err := Path("amd64"); err == nil || !strings.Contains(err.Error(), "damaged") {
		t.Fatalf("Path = %v, want a damaged-agent error", err)
	}
}

// B-2: the controller uploads itself only when it is a static Linux build of
// the very arch the box needs.
func TestPathUsesSelfOnlyForAStaticLinuxBuildOfTheSameArch(t *testing.T) {
	withSeams(t)
	testutil.Home(t)
	selfIsStaticLinux = func() bool { return true }
	if p, src, err := Path("amd64"); err != nil || src != SourceSelf || p != "/self/jumpgate" {
		t.Fatalf("same arch: Path = %q, %q, %v; want self", p, src, err)
	}
	if _, _, err := Path("arm64"); err == nil {
		t.Fatal("other arch: Path used self")
	}
	selfIsStaticLinux = func() bool { return false }
	if _, _, err := Path("amd64"); err == nil || !strings.Contains(err.Error(), "scripts/build-agents.sh") {
		t.Fatalf("cgo build: Path = %v, want the no-agent error naming scripts/build-agents.sh", err)
	}
}
