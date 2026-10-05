package agentbin

import (
	"bytes"
	"path/filepath"
	"strings"
	"testing"

	"github.com/valve-tech/jumpgate/internal/testutil"
)

func TestReportingNamesTheSource(t *testing.T) {
	errOut := withSeams(t)
	home := testutil.Home(t)
	body := []byte("agent")
	writeDevDir(t, home, "arm64", body, sumLine("jumpgate-linux-arm64", body))

	var lines []string
	got, err := Reporting(func(l string) { lines = append(lines, l) })("arm64")
	if err != nil || !bytes.Equal(got, body) {
		t.Fatalf("got %q, %v; want %q", got, err, body)
	}
	if line := "agent binary for linux/arm64: " + string(SourceDevDir); len(lines) != 1 || lines[0] != line {
		t.Fatalf("reported %q, want [%q]", lines, line)
	}
	if errOut.Len() != 0 {
		t.Fatalf("a dev build's own dev dir printed %q on stderr", errOut)
	}
}

// A release build says its agent came from the build itself, so the person
// pairing can tell it from a developer override.
func TestReportingNamesTheEmbeddedSource(t *testing.T) {
	withSeams(t)
	testutil.Home(t)
	body := []byte("embedded agent")
	embeddedFS = embeddedWith(t, "amd64", body, sumLine("jumpgate-linux-amd64", body))

	var lines []string
	if _, err := Reporting(func(l string) { lines = append(lines, l) })("amd64"); err != nil {
		t.Fatal(err)
	}
	if line := "agent binary for linux/amd64: " + string(SourceEmbedded); len(lines) != 1 || lines[0] != line {
		t.Fatalf("reported %q, want [%q]", lines, line)
	}
}

// P35: a release build using development agents says so loudly, on the
// pairing stream and on stderr.
func TestReportingWarnsWhenDevAgentsReplaceTheEmbeddedOnes(t *testing.T) {
	errOut := withSeams(t)
	home := testutil.Home(t)
	t.Setenv(DevAgentsEnv, "1")
	dev, emb := []byte("dev agent"), []byte("embedded agent")
	writeDevDir(t, home, "amd64", dev, sumLine("jumpgate-linux-amd64", dev))
	embeddedFS = embeddedWith(t, "amd64", emb, sumLine("jumpgate-linux-amd64", emb))

	var lines []string
	if _, err := Reporting(func(l string) { lines = append(lines, l) })("amd64"); err != nil {
		t.Fatal(err)
	}
	warning := "WARNING: using development agents from " + filepath.Join(home, ".jumpgate", "agents") + ", not the agents built into this jumpgate"
	if len(lines) != 2 || lines[0] != warning || lines[1] != "agent binary for linux/amd64: "+string(SourceDevDir) {
		t.Fatalf("reported %q, want the warning then the source", lines)
	}
	if strings.TrimSpace(errOut.String()) != warning {
		t.Fatalf("stderr %q, want %q", errOut, warning)
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
