package agentbin

import (
	"crypto/sha256"
	"encoding/hex"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/valve-tech/jumpgate/internal/testutil"
)

// otherArch is an agent arch this test binary is not, so Path never answers
// with "this binary" and the developer directory is what gets read.
func otherArch() string {
	if runtime.GOARCH == "arm64" {
		return "amd64"
	}
	return "arm64"
}

func TestReportingNamesTheSource(t *testing.T) {
	home := testutil.Home(t)
	arch := otherArch()
	dir := filepath.Join(home, ".jumpgate", "agents")
	if err := os.MkdirAll(dir, 0o700); err != nil {
		t.Fatal(err)
	}
	body := []byte("agent")
	sum := sha256.Sum256(body)
	name := "jumpgate-linux-" + arch
	if err := os.WriteFile(filepath.Join(dir, name), body, 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "SHA256SUMS"), []byte(hex.EncodeToString(sum[:])+"  "+name+"\n"), 0o600); err != nil {
		t.Fatal(err)
	}

	var lines []string
	p, err := Reporting(func(l string) { lines = append(lines, l) })(arch)
	if err != nil || p != filepath.Join(dir, name) {
		t.Fatalf("got %q, %v", p, err)
	}
	want := "agent binary for linux/" + arch + ": " + string(SourceDevDir)
	if len(lines) != 1 || lines[0] != want {
		t.Fatalf("reported %q, want [%q]", lines, want)
	}
}

func TestReportingReportsNothingWithoutAnAgent(t *testing.T) {
	testutil.Home(t)
	var lines []string
	_, err := Reporting(func(l string) { lines = append(lines, l) })(otherArch())
	if err == nil || !strings.Contains(err.Error(), "build-agents.sh") {
		t.Fatalf("err = %v, want one naming scripts/build-agents.sh", err)
	}
	if len(lines) != 0 {
		t.Fatalf("reported %q for a failed lookup", lines)
	}
}
