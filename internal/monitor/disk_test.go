package monitor

import (
	"context"
	"encoding/json"
	"errors"
	"os/exec"
	"runtime"
	"testing"
	"time"

	"github.com/valve-tech/jumpgate/internal/catalog"
	"github.com/valve-tech/jumpgate/internal/executor"
)

// TestDiskCmd_ExitsNonZeroWhenDfFails runs the real probe command under sh
// with a df that fails. Piping df through `tail -1` made the pipeline's exit
// status tail's, so a failed df looked like a successful probe with no
// output, and the poll could not tell that apart from a reading.
func TestDiskCmd_ExitsNonZeroWhenDfFails(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("probe runs under a POSIX shell")
	}
	sh, err := exec.LookPath("sh")
	if err != nil {
		t.Skip("no sh on PATH")
	}

	// df is replaced by a shell function, which takes precedence over PATH,
	// rather than by a fake executable: macOS may hold a freshly written
	// executable for a security scan, which under a loaded `go test ./...`
	// stalled this test for minutes. The timeout bounds it regardless.
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	failingDf := `df() { echo "df: unrecognized option '--output=pcent'" >&2; return 1; }; `
	err = exec.CommandContext(ctx, sh, "-c", failingDf+diskCmd(t.TempDir())).Run()
	if ctx.Err() != nil {
		t.Fatalf("probe did not finish: %v", ctx.Err())
	}

	var exitErr *exec.ExitError
	if !errors.As(err, &exitErr) {
		t.Fatalf("probe exited %v, want a non-zero exit when df fails", err)
	}
}

// TestPoll_DiskKnownSeparatesAFailedProbeFromZeroPercent is the regression
// for a failed disk probe reading as 0% used: DiskUsedPct alone cannot say
// whether 0 is a measurement or a missing one, so DiskKnown carries that.
func TestPoll_DiskKnownSeparatesAFailedProbeFromZeroPercent(t *testing.T) {
	cases := []struct {
		name      string
		res       executor.Result
		wantKnown bool
		wantPct   float64
	}{
		{"a real reading", executor.Result{Stdout: "Use%\n 42%\n"}, true, 42},
		{"a real zero reading", executor.Result{Stdout: "Use%\n  0%\n"}, true, 0},
		{"df failed", executor.Result{ExitCode: 1, Stderr: "df: unrecognized option '--output=pcent'"}, false, 0},
		{"df succeeded but printed nothing parseable", executor.Result{Stdout: "Use%\n"}, false, 0},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			fe := newFakeExecutor().script("df --output=pcent", tc.res)
			m := New(Config{Exec: fe, Wire: catalog.WireConfig{DataDir: "/var/lib/valve-node-app/369"}})
			snap := m.poll(context.Background())
			if snap.DiskKnown != tc.wantKnown {
				t.Errorf("DiskKnown = %v, want %v", snap.DiskKnown, tc.wantKnown)
			}
			if snap.DiskUsedPct != tc.wantPct {
				t.Errorf("DiskUsedPct = %v, want %v", snap.DiskUsedPct, tc.wantPct)
			}
		})
	}

	t.Run("a transport error", func(t *testing.T) {
		fe := newFakeExecutor().errOn("df --output=pcent", errors.New("ssh: connection lost"))
		snap := New(Config{Exec: fe}).poll(context.Background())
		if snap.DiskKnown {
			t.Error("DiskKnown = true after the probe's transport failed")
		}
	})
}

// The web UI reads diskUsedPct as a number, so the existing field keeps its
// name and type and the new signal is an additional field.
func TestSnapshot_DiskFieldsSerializeBackwardCompatibly(t *testing.T) {
	b, err := json.Marshal(Snapshot{DiskUsedPct: 42, DiskKnown: true})
	if err != nil {
		t.Fatal(err)
	}
	var m map[string]any
	if err := json.Unmarshal(b, &m); err != nil {
		t.Fatal(err)
	}
	if pct, ok := m["diskUsedPct"].(float64); !ok || pct != 42 {
		t.Errorf("diskUsedPct = %#v, want the number 42", m["diskUsedPct"])
	}
	if known, ok := m["diskKnown"].(bool); !ok || !known {
		t.Errorf("diskKnown = %#v, want true", m["diskKnown"])
	}
}
