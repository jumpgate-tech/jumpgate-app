package agentbin

import (
	"path/filepath"
	"strings"
	"testing"

	"github.com/valve-tech/jumpgate/internal/testutil"
)

func TestReportingNamesTheSource(t *testing.T) {
	withSeams(t)
	home := testutil.Home(t)
	body := []byte("agent")
	want := writeDevDir(t, home, "arm64", body, sumLine("jumpgate-linux-arm64", body))

	var lines []string
	p, err := Reporting(func(l string) { lines = append(lines, l) })("arm64")
	if err != nil || p != want {
		t.Fatalf("got %q, %v; want %q", p, err, want)
	}
	if line := "agent binary for linux/arm64: " + string(SourceDevDir); len(lines) != 1 || lines[0] != line {
		t.Fatalf("reported %q, want [%q]", lines, line)
	}
}

// A release build says its agent came from the build itself, so the person
// pairing can tell it from a developer override.
func TestReportingNamesTheEmbeddedSource(t *testing.T) {
	withSeams(t)
	home := testutil.Home(t)
	body := []byte("embedded agent")
	embeddedFS = embeddedWith(t, "amd64", body, sumLine("jumpgate-linux-amd64", body))

	var lines []string
	p, err := Reporting(func(l string) { lines = append(lines, l) })("amd64")
	if err != nil || !strings.HasPrefix(p, filepath.Join(home, ".jumpgate", "agents-cache")) {
		t.Fatalf("got %q, %v; want a file in the agents cache", p, err)
	}
	if line := "agent binary for linux/amd64: " + string(SourceEmbedded); len(lines) != 1 || lines[0] != line {
		t.Fatalf("reported %q, want [%q]", lines, line)
	}
}

func TestReportingReportsNothingWithoutAnAgent(t *testing.T) {
	withSeams(t)
	testutil.Home(t)
	var lines []string
	_, err := Reporting(func(l string) { lines = append(lines, l) })("arm64")
	if err == nil || !strings.Contains(err.Error(), "build-agents.sh") {
		t.Fatalf("err = %v, want one naming scripts/build-agents.sh", err)
	}
	if len(lines) != 0 {
		t.Fatalf("reported %q for a failed lookup", lines)
	}
}
