package jobs

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/valve-tech/jumpgate/internal/eip712"
	"github.com/valve-tech/jumpgate/internal/intent"
	"github.com/valve-tech/jumpgate/internal/joblog"
)

// countRecoveries counts the entry signatures the store recovers from now on.
func countRecoveries(t *testing.T) *int {
	t.Helper()
	orig, n := verifyEntry, new(int)
	verifyEntry = func(agent eip712.Address, job string, e joblog.Entry) error {
		*n++
		return orig(agent, job, e)
	}
	t.Cleanup(func() { verifyEntry = orig })
	return n
}

// The agent's check of its own log is the digest chain against the
// manifest's count and head, with one signature recovery, the last entry's,
// per Load. An Append recovers none, so a job's life costs no O(n²)
// recoveries, and a List of the hundred jobs a box keeps (J6) recovers one
// signature per job, not one per entry of every log.
func TestTheStoreRecoversOneSignaturePerLoad(t *testing.T) {
	s, _, _ := testStore(t)
	ctx := context.Background()
	n := countRecoveries(t)
	const jobs, appends = 100, 6
	for i := 0; i < jobs; i++ {
		m, mbody := newManifest(t)
		mustOK(t, s.Create(ctx, &m, mbody))
		for j := 0; j < appends; j++ {
			mustOK(t, s.Append(ctx, &m, joblog.KindStepStarted, joblog.StepBody{Step: "one", Attempt: j}))
		}
	}
	if *n != 0 {
		t.Fatalf("creating and appending recovered %d signatures; want none", *n)
	}
	ms, err := s.List()
	if err != nil || len(ms) != jobs {
		t.Fatalf("List = %d jobs, %v", len(ms), err)
	}
	for _, m := range ms {
		if m.State == intent.JobDamaged {
			t.Fatalf("job %s listed damaged: %s", m.ID, m.Error)
		}
	}
	if *n != jobs {
		t.Fatalf("List of %d jobs of %d entries recovered %d signatures; want %d", jobs, appends+1, *n, jobs)
	}
}

// BenchmarkListOf100Jobs is List over the most finished jobs a box keeps.
func BenchmarkListOf100Jobs(b *testing.B) {
	s, _, _ := testStore(b)
	ctx := context.Background()
	for i := 0; i < 100; i++ {
		m, mbody := newManifest(b)
		mustOK(b, s.Create(ctx, &m, mbody))
		for j := 0; j < 10; j++ {
			mustOK(b, s.Append(ctx, &m, joblog.KindStepStarted, joblog.StepBody{Step: "one", Attempt: j}))
		}
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if _, err := s.List(); err != nil {
			b.Fatal(err)
		}
	}
}

// rewriteEntry replaces entry at of job id's log with edit applied to it.
func rewriteEntry(t *testing.T, s *Store, id string, at int, edit func(*intent.JobLogEntryJSON)) {
	t.Helper()
	logPath := filepath.Join(s.Dir(), id+".log")
	b, err := os.ReadFile(logPath)
	mustOK(t, err)
	lines := strings.Split(strings.TrimSpace(string(b)), "\n")
	var e intent.JobLogEntryJSON
	mustOK(t, json.Unmarshal([]byte(lines[at]), &e))
	edit(&e)
	l, err := json.Marshal(e)
	mustOK(t, err)
	lines[at] = string(l)
	mustOK(t, os.WriteFile(logPath, []byte(strings.Join(lines, "\n")+"\n"), 0o600))
}

// A bad signature on the last entry is the damage the digest chain cannot
// see (a signature is not part of the digest), so Load recovers it.
func TestLoadDetectsABadSignatureOnTheLastEntry(t *testing.T) {
	s, _, _ := testStore(t)
	m, mbody := newManifest(t)
	ctx := context.Background()
	mustOK(t, s.Create(ctx, &m, mbody))
	es, _, err := s.ReadLog(m.ID, 0, 1)
	mustOK(t, err)
	mustOK(t, s.Append(ctx, &m, joblog.KindStepStarted, joblog.StepBody{Step: "one"}))
	// A well-formed signature, over another entry.
	rewriteEntry(t, s, m.ID, 1, func(e *intent.JobLogEntryJSON) { e.Sig = es[0].Sig.Hex() })
	if _, err := s.Load(m.ID); !errors.Is(err, ErrDamaged) || !strings.Contains(err.Error(), "not signed") {
		t.Fatalf("Load with a bad last signature = %v, want ErrDamaged", err)
	}
	if ms, err := s.List(); err != nil || len(ms) != 1 || ms[0].State != intent.JobDamaged {
		t.Fatalf("List = %+v, %v", ms, err)
	}
}

// An edited entry breaks the digest chain at the next entry, or the
// manifest's head at the last, and is found without recovering a signature.
func TestTheChainCatchesAnEditedEntry(t *testing.T) {
	for _, at := range []int{0, 1, 2} {
		s, _, _ := testStore(t)
		m, mbody := newManifest(t)
		ctx := context.Background()
		mustOK(t, s.Create(ctx, &m, mbody))
		mustOK(t, s.Append(ctx, &m, joblog.KindStepStarted, joblog.StepBody{Step: "one"}))
		mustOK(t, s.Append(ctx, &m, joblog.KindStepFinished, joblog.StepBody{Step: "one", OK: true}))
		rewriteEntry(t, s, m.ID, at, func(e *intent.JobLogEntryJSON) { e.At++ })
		n := countRecoveries(t)
		if err := s.Append(ctx, &m, joblog.KindFinished, joblog.FinishedBody{State: intent.JobFailed}); err == nil {
			t.Fatalf("Append onto a log with entry %d edited", at)
		}
		if _, err := s.Load(m.ID); !errors.Is(err, ErrDamaged) {
			t.Fatalf("Load with entry %d edited = %v, want ErrDamaged", at, err)
		}
		if *n != 0 {
			t.Fatalf("entry %d's edit was found by %d recoveries, not by the chain", at, *n)
		}
	}
}
