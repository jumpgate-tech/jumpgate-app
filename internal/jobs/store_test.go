package jobs

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/valve-tech/jumpgate/internal/intent"
	"github.com/valve-tech/jumpgate/internal/joblog"
	"github.com/valve-tech/jumpgate/internal/signer"
	"github.com/valve-tech/jumpgate/internal/testutil"
)

func testStore(t *testing.T) (*Store, *signer.Key, *time.Time) {
	t.Helper()
	testutil.RequireUnix(t) // the store fsyncs directories, as the agent's replay record does
	k, _ := signer.GenerateKey()
	now := time.Unix(1_800_000_000, 0)
	s, err := OpenStore(filepath.Join(t.TempDir(), "jobs"), k, func() time.Time { return now })
	if err != nil {
		t.Fatal(err)
	}
	return s, k, &now
}

// newManifest builds a job the way the agent does: a signed creating intent
// whose digest names the job (ruling R-J5). The intent's random nonce keeps
// every fixture's id distinct.
func mustOK(t *testing.T, err error) {
	t.Helper()
	if err != nil {
		t.Fatal(err)
	}
}

func newManifest(t *testing.T) (Manifest, joblog.IntentBody) {
	t.Helper()
	ctl, _ := signer.GenerateKey()
	agent, _ := signer.GenerateKey()
	nonce, err := intent.NewNonce()
	if err != nil {
		t.Fatal(err)
	}
	in := intent.Intent{Agent: agent.Address(), Controller: ctl.Address(), Nonce: nonce, IssuedAt: 1_800_000_000, Expiry: 1_800_000_600, Kind: "node.migrate-names"}
	id, err := joblog.JobIDFor(in)
	if err != nil || !ValidID(id) {
		t.Fatalf("JobIDFor = %q, %v", id, err)
	}
	m := Manifest{JobRecord: intent.JobRecord{Format: FormatVersion, ID: id, Kind: "test.steps", PlanVersion: "test.steps/1",
		Driver: DriverController, State: intent.JobWaiting}, Inputs: json.RawMessage(`{}`)}
	return m, joblog.IntentBody{Envelope: intent.Envelope{Intent: in.JSON()}, Outcome: "created"}
}

func TestCreateWritesTheManifestAndASignedFirstEntry(t *testing.T) {
	s, k, _ := testStore(t)
	m, mbody := newManifest(t)
	if err := s.Create(context.Background(), &m, mbody); err != nil {
		t.Fatal(err)
	}
	fi, err := os.Stat(filepath.Join(s.Dir(), m.ID+".json"))
	if err != nil || fi.Mode().Perm() != 0o600 {
		t.Fatalf("manifest: %v, mode %v", err, fi.Mode())
	}
	got, err := s.Load(m.ID)
	if err != nil || got.LogLen != 1 {
		t.Fatalf("Load = %+v, %v", got.JobRecord, err)
	}
	es, more, err := s.ReadLog(m.ID, 0, 10)
	if err != nil || more || len(es) != 1 || es[0].Kind != joblog.KindCreated {
		t.Fatalf("ReadLog = %v, %v, %v", es, more, err)
	}
	if _, err := joblog.Verify(k.Address(), m.ID, es, 0, [32]byte{}); err != nil {
		t.Fatal(err)
	}
}

// Review Focus 5: a log shorter than the manifest says is damage, never a
// fresh start; one entry longer is an append the crash cut off before the
// manifest save, which Load completes.
func TestLoadDetectsATruncatedLog(t *testing.T) {
	s, _, _ := testStore(t)
	m, mbody := newManifest(t)
	ctx := context.Background()
	mustOK(t, s.Create(ctx, &m, mbody))
	mustOK(t, s.Append(ctx, &m, joblog.KindStepStarted, joblog.StepBody{Step: "one"}))
	mustOK(t, s.Save(&m))

	logPath := filepath.Join(s.Dir(), m.ID+".log")
	b, _ := os.ReadFile(logPath)
	lines := strings.SplitAfter(strings.TrimSpace(string(b)), "\n")
	// lines[0] already ends in "\n": the file keeps one whole, valid entry, so
	// the parse and the chain check pass and only the length check (one entry
	// where the manifest expects two) can refuse it.
	mustOK(t, os.WriteFile(logPath, []byte(lines[0]), 0o600))
	_, err := s.Load(m.ID)
	if !errors.Is(err, ErrDamaged) || !strings.Contains(err.Error(), "its manifest expects 2") {
		t.Fatalf("Load of a truncated log = %v, want ErrDamaged from the length check", err)
	}
}

func TestLoadDetectsALogCutMidEntry(t *testing.T) {
	s, _, _ := testStore(t)
	m, mbody := newManifest(t)
	ctx := context.Background()
	mustOK(t, s.Create(ctx, &m, mbody))
	mustOK(t, s.Append(ctx, &m, joblog.KindStepStarted, joblog.StepBody{Step: "one"}))
	mustOK(t, s.Save(&m))

	logPath := filepath.Join(s.Dir(), m.ID+".log")
	b, _ := os.ReadFile(logPath)
	mustOK(t, os.WriteFile(logPath, b[:len(b)-10], 0o600)) // a torn last line
	if _, err := s.Load(m.ID); !errors.Is(err, ErrDamaged) {
		t.Fatalf("Load of a truncated log = %v, want ErrDamaged", err)
	}
}

func TestLoadCompletesAnAppendTheManifestMissed(t *testing.T) {
	s, _, _ := testStore(t)
	m, mbody := newManifest(t)
	ctx := context.Background()
	mustOK(t, s.Create(ctx, &m, mbody))
	stale := m // the manifest as saved
	mustOK(t, s.Append(ctx, &m, joblog.KindStepStarted, joblog.StepBody{Step: "one"}))
	mustOK(t, s.Save(&stale)) // a crash between the append and the save
	got, err := s.Load(m.ID)
	if err != nil || got.LogLen != 2 || got.LogHead != m.LogHead {
		t.Fatalf("Load = len %d head %s, %v; want len 2 head %s", got.LogLen, got.LogHead, err, m.LogHead)
	}
}

func TestListIsNewestFirstAndShowsDamagedJobs(t *testing.T) {
	s, _, now := testStore(t)
	ctx := context.Background()
	a, abody := newManifest(t)
	a.CreatedAt = now.Unix()
	mustOK(t, s.Create(ctx, &a, abody))
	b, bbody := newManifest(t)
	b.CreatedAt = now.Unix() + 10
	mustOK(t, s.Create(ctx, &b, bbody))
	mustOK(t, os.WriteFile(filepath.Join(s.Dir(), a.ID+".log"), nil, 0o600))
	ms, err := s.List()
	if err != nil || len(ms) != 2 || ms[0].ID != b.ID || ms[1].State != intent.JobDamaged {
		t.Fatalf("List = %+v, %v", ms, err)
	}
}

func TestPruneKeepsRecentAndUnfinishedJobs(t *testing.T) {
	s, _, now := testStore(t)
	ctx := context.Background()
	// Age is judged from a job's last change (D-J6), and every append stamps
	// UpdatedAt with the store's clock. So the test writes the old jobs first
	// and then moves the store's clock on 31 days, instead of presetting
	// UpdatedAt (Create would overwrite it).
	old, oldbody := newManifest(t)
	old.State = intent.JobSucceeded
	mustOK(t, s.Create(ctx, &old, oldbody))
	running, runningbody := newManifest(t)
	running.State = intent.JobRunning
	mustOK(t, s.Create(ctx, &running, runningbody))
	*now = now.Add(31 * 24 * time.Hour)
	recent, recentbody := newManifest(t)
	recent.State = intent.JobSucceeded
	mustOK(t, s.Create(ctx, &recent, recentbody))
	if err := s.Prune(30*24*time.Hour, 100); err != nil {
		t.Fatal(err)
	}
	if _, err := s.Load(old.ID); !errors.Is(err, ErrNotFound) {
		t.Errorf("an old finished job survived: %v", err)
	}
	if _, err := s.Load(running.ID); err != nil {
		t.Errorf("an unfinished job was pruned: %v", err)
	}
	if _, err := s.Load(recent.ID); err != nil {
		t.Errorf("a recent finished job was pruned: %v", err)
	}
}

func TestCreateRefusesAnIdItsIntentDoesNotDerive(t *testing.T) {
	s, _, _ := testStore(t)
	m, body := newManifest(t)
	m.ID = "0123456789abcdef" // well formed, but not this intent's id
	var r *Refusal
	if err := s.Create(context.Background(), &m, body); !errors.As(err, &r) || r.Code != intent.ReasonValidation {
		t.Fatalf("Create with a foreign id = %v, want a validation Refusal", err)
	}
	if _, err := s.Load(m.ID); !errors.Is(err, ErrNotFound) {
		t.Fatalf("a refused job left a manifest behind: %v", err)
	}
	bad := body
	bad.Envelope.Intent.Agent = "nonsense"
	good, _ := newManifest(t)
	if err := s.Create(context.Background(), &good, bad); !errors.As(err, &r) {
		t.Fatalf("Create with an unparsable intent = %v, want a Refusal", err)
	}
}

func TestLoadDetectsAHeadThatDisagreesWithTheManifest(t *testing.T) {
	s, _, _ := testStore(t)
	m, mbody := newManifest(t)
	ctx := context.Background()
	mustOK(t, s.Create(ctx, &m, mbody))
	mustOK(t, s.Append(ctx, &m, joblog.KindStepStarted, joblog.StepBody{Step: "one"}))
	m.LogHead = "0x" + strings.Repeat("11", 32) // same count, a different head
	mustOK(t, s.Save(&m))
	if _, err := s.Load(m.ID); !errors.Is(err, ErrDamaged) {
		t.Fatalf("Load with a wrong head = %v, want ErrDamaged", err)
	}
}

func TestJobIDForIdsAreValidIDs(t *testing.T) {
	for i := 0; i < 20; i++ {
		m, _ := newManifest(t)
		if !ValidID(m.ID) {
			t.Fatalf("ValidID(%q) = false for a JobIDFor id", m.ID)
		}
	}
}

func TestValidIDRefusesAnythingButSixteenHex(t *testing.T) {
	for _, id := range []string{"", "0123456789ABCDEF", "0123456789abcde", "../../etc/passwd", "0123456789abcdef0"} {
		if ValidID(id) {
			t.Errorf("ValidID(%q) = true", id)
		}
	}
}

// Review I1: a failed append must not leave a line on disk that the manifest
// does not know, or the next append would sign the same index twice.
func TestFailedAppendLeavesTheLogVerifiable(t *testing.T) {
	s, k, _ := testStore(t)
	m, mbody := newManifest(t)
	ctx := context.Background()
	mustOK(t, s.Create(ctx, &m, mbody))
	orig := syncFile
	defer func() { syncFile = orig }()
	syncFile = func(f *os.File) error {
		_, _ = f.Write([]byte("torn")) // a short, unsynced tail as well
		return errors.New("injected fsync failure")
	}
	before := m
	if err := s.Append(ctx, &m, joblog.KindStepStarted, joblog.StepBody{Step: "one"}); err == nil {
		t.Fatal("Append succeeded although the fsync failed")
	}
	if m.LogLen != before.LogLen || m.LogHead != before.LogHead {
		t.Fatalf("a failed Append moved the manifest: %d %s", m.LogLen, m.LogHead)
	}
	syncFile = orig
	mustOK(t, s.Append(ctx, &m, joblog.KindStepStarted, joblog.StepBody{Step: "one"}))
	es, _, err := s.ReadLog(m.ID, 0, 0)
	mustOK(t, err)
	if len(es) != 2 {
		t.Fatalf("log has %d entries, want 2", len(es))
	}
	if _, err := joblog.Verify(k.Address(), m.ID, es, 0, [32]byte{}); err != nil {
		t.Fatal(err)
	}
	if _, err := s.Load(m.ID); err != nil {
		t.Fatal(err)
	}
}

func TestCreateRefusesAJobThatExists(t *testing.T) {
	s, _, _ := testStore(t)
	m, mbody := newManifest(t)
	ctx := context.Background()
	mustOK(t, s.Create(ctx, &m, mbody))
	again := m
	var r *Refusal
	if err := s.Create(ctx, &again, mbody); !errors.As(err, &r) {
		t.Fatalf("re-creating a job = %v, want a Refusal", err)
	}
	if got, err := s.Load(m.ID); err != nil || got.LogLen != 1 {
		t.Fatalf("after a refused re-create: %+v, %v", got.JobRecord, err)
	}
}

// Review I2: Create saves the manifest before it appends, so a crash between
// the two leaves a job that loads cleanly.
func TestLoadAcceptsACrashBetweenCreatesSaveAndAppend(t *testing.T) {
	s, _, _ := testStore(t)
	m, _ := newManifest(t)
	mustOK(t, s.Save(&m)) // LogLen 0, no log file
	got, err := s.Load(m.ID)
	if err != nil || got.LogLen != 0 {
		t.Fatalf("Load = %+v, %v", got.JobRecord, err)
	}
}

// Review I3: Load's repair, List and a writer on one job must stay consistent.
func TestConcurrentListAndAppendStayConsistent(t *testing.T) {
	s, _, _ := testStore(t)
	m, mbody := newManifest(t)
	ctx := context.Background()
	mustOK(t, s.Create(ctx, &m, mbody))
	done := make(chan struct{})
	errc := make(chan error, 1)
	go func() {
		defer close(done)
		for i := 0; i < 50; i++ {
			if err := s.Append(ctx, &m, joblog.KindStepStarted, joblog.StepBody{Step: "one", Attempt: i}); err != nil {
				errc <- err
				return
			}
		}
	}()
	for running := true; running; {
		select {
		case <-done:
			running = false
		default:
		}
		ms, err := s.List()
		if err != nil {
			t.Fatal(err)
		}
		for _, x := range ms {
			if x.State == intent.JobDamaged {
				t.Fatalf("a concurrent writer made the job look damaged: %s", x.Error)
			}
		}
	}
	select {
	case err := <-errc:
		t.Fatal(err)
	default:
	}
	got, err := s.Load(m.ID)
	if err != nil || got.LogLen != 51 {
		t.Fatalf("Load = %d, %v", got.LogLen, err)
	}
}

// Review I4: a damaged job is evidence and outlives the retention window.
func TestPruneNeverRemovesADamagedJob(t *testing.T) {
	s, _, now := testStore(t)
	ctx := context.Background()
	m, mbody := newManifest(t)
	m.State = intent.JobSucceeded
	mustOK(t, s.Create(ctx, &m, mbody))
	mustOK(t, os.WriteFile(filepath.Join(s.Dir(), m.ID+".log"), nil, 0o600))
	torn, tbody := newManifest(t)
	mustOK(t, s.Create(ctx, &torn, tbody))
	mustOK(t, os.WriteFile(filepath.Join(s.Dir(), torn.ID+".json"), []byte("{not json"), 0o600))
	*now = now.Add(60 * 24 * time.Hour)
	mustOK(t, s.Prune(30*24*time.Hour, 100))
	for _, id := range []string{m.ID, torn.ID} {
		if _, err := os.Stat(filepath.Join(s.Dir(), id+".json")); err != nil {
			t.Errorf("damaged job %s was pruned: %v", id, err)
		}
	}
	ms, err := s.List()
	if err != nil || len(ms) != 2 {
		t.Fatalf("List = %d jobs, %v", len(ms), err)
	}
	for _, x := range ms {
		if x.State != intent.JobDamaged || !ValidID(x.ID) {
			t.Errorf("listed %+v", x.JobRecord)
		}
	}
}

// Review I5: a log too long to scan is damage, not a skipped job.
func TestListShowsAJobWhoseLogCannotBeScanned(t *testing.T) {
	s, _, _ := testStore(t)
	m, mbody := newManifest(t)
	mustOK(t, s.Create(context.Background(), &m, mbody))
	mustOK(t, os.WriteFile(filepath.Join(s.Dir(), m.ID+".log"), []byte(strings.Repeat("x", 5<<20)), 0o600))
	ms, err := s.List()
	if err != nil || len(ms) != 1 || ms[0].State != intent.JobDamaged {
		t.Fatalf("List = %+v, %v", ms, err)
	}
	if _, err := s.Load(m.ID); !errors.Is(err, ErrDamaged) {
		t.Fatalf("Load = %v, want ErrDamaged", err)
	}
}

// Review I6: ids and steps reach paths only after validation.
func TestBadIDsAndStepsAreRefusedBeforeTheyReachAPath(t *testing.T) {
	s, _, _ := testStore(t)
	ctx := context.Background()
	bad := Manifest{JobRecord: intent.JobRecord{ID: "../x"}}
	var r *Refusal
	if err := s.Save(&bad); !errors.As(err, &r) {
		t.Errorf("Save = %v", err)
	}
	if err := s.Append(ctx, &bad, joblog.KindStepStarted, joblog.StepBody{}); !errors.As(err, &r) {
		t.Errorf("Append = %v", err)
	}
	if _, _, err := s.ReadLog("../x", 0, 0); !errors.As(err, &r) {
		t.Errorf("ReadLog = %v", err)
	}
	if err := s.Create(ctx, &bad, joblog.IntentBody{}); !errors.As(err, &r) {
		t.Errorf("Create = %v", err)
	}
	for _, c := range []struct {
		id, step string
		attempt  int
	}{{"../x", "one", 0}, {"0123456789abcdef", "../x", 0}, {"0123456789abcdef", "", 0}, {"0123456789abcdef", "one", -1}} {
		if p, err := ResultPath(s.Dir(), c.id, c.step, c.attempt); err == nil {
			t.Errorf("ResultPath(%q, %q, %d) = %q, want a refusal", c.id, c.step, c.attempt, p)
		}
	}
	p, err := ResultPath(s.Dir(), "0123456789abcdef", "one", 2)
	if err != nil || p != filepath.Join(s.Dir(), "0123456789abcdef.d", "one-2.json") {
		t.Errorf("ResultPath = %q, %v", p, err)
	}
}

// Review I7: if a job's files cannot be removed its manifest stays, so the
// job is still listed and the prune can be retried.
func TestPruneKeepsTheManifestWhenTheLogCannotBeRemoved(t *testing.T) {
	s, _, now := testStore(t)
	m, mbody := newManifest(t)
	m.State = intent.JobSucceeded
	mustOK(t, s.Create(context.Background(), &m, mbody))
	*now = now.Add(60 * 24 * time.Hour)
	// A read-only directory makes every removal fail.
	mustOK(t, os.Chmod(s.Dir(), 0o500))
	defer os.Chmod(s.Dir(), 0o700)
	if os.Geteuid() == 0 {
		t.Skip("root ignores directory permissions")
	}
	if err := s.Prune(time.Hour, 100); err == nil {
		t.Fatal("Prune succeeded although nothing could be removed")
	}
	if _, err := os.Stat(filepath.Join(s.Dir(), m.ID+".json")); err != nil {
		t.Fatalf("the manifest was removed: %v", err)
	}
}

func TestLoadRefusesAManifestForAnotherJob(t *testing.T) {
	s, _, _ := testStore(t)
	m, mbody := newManifest(t)
	mustOK(t, s.Create(context.Background(), &m, mbody))
	other, _ := newManifest(t)
	b, err := os.ReadFile(filepath.Join(s.Dir(), m.ID+".json"))
	mustOK(t, err)
	mustOK(t, os.WriteFile(filepath.Join(s.Dir(), other.ID+".json"), b, 0o600))
	if _, err := s.Load(other.ID); !errors.Is(err, ErrDamaged) {
		t.Fatalf("Load = %v, want ErrDamaged", err)
	}
}
