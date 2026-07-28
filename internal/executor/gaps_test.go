package executor

// The local executor's file writes, and the line streamer's final unterminated
// line — the one a command that does not end in '\n' produces, which is most
// of them when the process is killed partway.

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

// ---------------------------------------------------------------------
// local WriteFile / ReadFile
// ---------------------------------------------------------------------

// WriteFile creates the parent directory, because every caller writes into a
// path that may not exist yet (a data dir, a unit directory, a config sibling).
func TestLocalWriteFile_CreatesTheParentDirectory(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "a", "b", "c", "erpc.yaml")
	e := NewLocal()

	if err := e.WriteFile(context.Background(), path, []byte("networks: []\n"), 0o644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	got, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("the file was not written: %v", err)
	}
	if string(got) != "networks: []\n" {
		t.Errorf("content = %q, want what was written", got)
	}
}

// os.WriteFile only applies mode on CREATE: an existing file keeps its old
// permissions, and umask can narrow them even on create. The chmod is
// unconditional so the mode asked for is the mode on disk either way.
func TestLocalWriteFile_ModeIsAuthoritativeOnRewrite(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("unix file modes")
	}
	dir := t.TempDir()
	path := filepath.Join(dir, "jwt.hex")
	e := NewLocal()

	// First write wide open, then rewrite it as a secret.
	if err := e.WriteFile(context.Background(), path, []byte("first"), 0o644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}
	if err := e.WriteFile(context.Background(), path, []byte("second"), 0o600); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	fi, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	if perm := fi.Mode().Perm(); perm != 0o600 {
		t.Errorf("mode = %04o, want 0600 — a rewritten JWT kept its old permissions", perm)
	}

	got, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != "second" {
		t.Errorf("content = %q, want the rewrite", got)
	}
}

func TestLocalWriteFile_ReportsAPathItCannotWrite(t *testing.T) {
	dir := t.TempDir()
	// A regular file where a parent directory would have to be.
	blocker := filepath.Join(dir, "blocker")
	if err := os.WriteFile(blocker, []byte("in the way"), 0o600); err != nil {
		t.Fatal(err)
	}

	err := NewLocal().WriteFile(context.Background(), filepath.Join(blocker, "child", "f"), []byte("x"), 0o644)
	if err == nil {
		t.Fatal("writing under a regular file reported success")
	}
}

func TestLocalReadFile_RoundTripsAndReportsAMissingFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "erpc.yaml")
	e := NewLocal()

	want := []byte("networks:\n  - chainId: 369\n")
	if err := e.WriteFile(context.Background(), path, want, 0o644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}
	got, err := e.ReadFile(context.Background(), path)
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}
	if string(got) != string(want) {
		t.Errorf("got %q, want %q", got, want)
	}

	if _, err := e.ReadFile(context.Background(), filepath.Join(dir, "not-there")); err == nil {
		t.Error("reading a missing file reported success")
	}
}

// ---------------------------------------------------------------------
// writeFileCmd / readFileCmd
// ---------------------------------------------------------------------

// The remote path is always POSIX, even when the control plane is Windows —
// filepath.Dir there would emit `\var\lib\…` into the mkdir -p and break
// every unit, config and JWT write against every Linux target.
func TestWriteFileCmd_UsesPOSIXPathsAndQuotesEverything(t *testing.T) {
	cmd := writeFileCmd("/var/lib/valve-node-app/erpc.yaml", []byte("hello"), 0o600)

	if strings.Contains(cmd, `\`) {
		t.Errorf("a host separator leaked into the remote command: %s", cmd)
	}
	if !strings.Contains(cmd, "mkdir -p '/var/lib/valve-node-app'") {
		t.Errorf("the parent directory is wrong: %s", cmd)
	}
	if !strings.Contains(cmd, "chmod 600") {
		t.Errorf("the mode is not applied: %s", cmd)
	}
	// The content travels base64-encoded, so a payload with quotes, newlines
	// or metacharacters cannot break out of the command.
	if strings.Contains(cmd, "hello") {
		t.Errorf("the payload was inlined rather than encoded: %s", cmd)
	}
}

func TestWriteFileCmd_AHostilePathAndPayloadStayQuoted(t *testing.T) {
	cmd := writeFileCmd("/tmp/it's there/f", []byte("$(whoami)\n'; rm -rf /"), 0o644)

	// Single quotes in the path are escaped rather than terminating the
	// quoted string.
	if strings.Contains(cmd, "rm -rf /") {
		t.Errorf("the payload reached the command line verbatim: %s", cmd)
	}
	if !strings.Contains(cmd, "base64 -d") {
		t.Errorf("the payload is not decoded remotely: %s", cmd)
	}
}

func TestReadFileCmd_QuotesThePath(t *testing.T) {
	cmd := readFileCmd("/tmp/it's there/f")
	if !strings.Contains(cmd, "base64") {
		t.Errorf("the read is not encoded: %s", cmd)
	}
	if strings.Contains(cmd, "; ") {
		t.Errorf("the path escaped its quoting: %s", cmd)
	}
}

// ---------------------------------------------------------------------
// the line streamer
// ---------------------------------------------------------------------

func TestLineStreamer_FlushDeliversAnUnterminatedFinalLine(t *testing.T) {
	var got []string
	w := &lineStreamer{buf: &bytes.Buffer{}, fn: func(s string) { got = append(got, s) }}

	if _, err := w.Write([]byte("first\nsecond\nno newline here")); err != nil {
		t.Fatalf("Write: %v", err)
	}
	// Before the flush the last line is still buffered — it might yet get
	// its newline from the next write.
	if len(got) != 2 {
		t.Fatalf("got %v, want only the two terminated lines", got)
	}

	w.Flush()
	if len(got) != 3 || got[2] != "no newline here" {
		t.Fatalf("got %v, want the final unterminated line delivered", got)
	}

	// Flushing twice must not repeat it.
	w.Flush()
	if len(got) != 3 {
		t.Errorf("a second flush re-delivered the line: %v", got)
	}
}

func TestLineStreamer_FlushOnAnEmptyBufferDeliversNothing(t *testing.T) {
	var got []string
	w := &lineStreamer{buf: &bytes.Buffer{}, fn: func(s string) { got = append(got, s) }}

	w.Flush()
	if len(got) != 0 {
		t.Fatalf("got %v, want nothing from an empty buffer", got)
	}

	// A stream that ended exactly on a newline has nothing left either — a
	// spurious "" here would render as a blank line in the setup log.
	if _, err := w.Write([]byte("done\n")); err != nil {
		t.Fatal(err)
	}
	w.Flush()
	if len(got) != 1 {
		t.Errorf("got %v, want just the one line", got)
	}
}

// A nil callback is the "nobody is listening" case, and must not panic.
func TestLineStreamer_ANilCallbackIsSafe(t *testing.T) {
	w := &lineStreamer{buf: &bytes.Buffer{}}
	if _, err := w.Write([]byte("a line\nand a partial")); err != nil {
		t.Fatalf("Write: %v", err)
	}
	w.Flush()
}

// Carriage returns are trimmed, so a target emitting CRLF does not render
// every line with a stray \r in the UI.
func TestLineStreamer_TrimsCarriageReturns(t *testing.T) {
	var got []string
	w := &lineStreamer{buf: &bytes.Buffer{}, fn: func(s string) { got = append(got, s) }}

	if _, err := w.Write([]byte("windows line\r\nunix line\n")); err != nil {
		t.Fatal(err)
	}
	if len(got) != 2 || got[0] != "windows line" || got[1] != "unix line" {
		t.Errorf("got %#v, want the CR trimmed", got)
	}
}
