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
	_ = s.Create(ctx, &m, mbody)
	_ = s.Append(ctx, &m, joblog.KindStepStarted, joblog.StepBody{Step: "one"})
	_ = s.Save(&m)

	logPath := filepath.Join(s.Dir(), m.ID+".log")
	b, _ := os.ReadFile(logPath)
	lines := strings.SplitAfter(strings.TrimSpace(string(b)), "\n")
	// lines[0] already ends in "\n": the file keeps one whole, valid entry, so
	// the parse and the chain check pass and only the length check (one entry
	// where the manifest expects two) can refuse it.
	_ = os.WriteFile(logPath, []byte(lines[0]), 0o600)
	_, err := s.Load(m.ID)
	if !errors.Is(err, ErrDamaged) || !strings.Contains(err.Error(), "its manifest expects 2") {
		t.Fatalf("Load of a truncated log = %v, want ErrDamaged from the length check", err)
	}
}

func TestLoadDetectsALogCutMidEntry(t *testing.T) {
	s, _, _ := testStore(t)
	m, mbody := newManifest(t)
	ctx := context.Background()
	_ = s.Create(ctx, &m, mbody)
	_ = s.Append(ctx, &m, joblog.KindStepStarted, joblog.StepBody{Step: "one"})
	_ = s.Save(&m)

	logPath := filepath.Join(s.Dir(), m.ID+".log")
	b, _ := os.ReadFile(logPath)
	_ = os.WriteFile(logPath, b[:len(b)-10], 0o600) // a torn last line
	if _, err := s.Load(m.ID); !errors.Is(err, ErrDamaged) {
		t.Fatalf("Load of a truncated log = %v, want ErrDamaged", err)
	}
}

func TestLoadCompletesAnAppendTheManifestMissed(t *testing.T) {
	s, _, _ := testStore(t)
	m, mbody := newManifest(t)
	ctx := context.Background()
	_ = s.Create(ctx, &m, mbody)
	stale := m // the manifest as saved
	_ = s.Append(ctx, &m, joblog.KindStepStarted, joblog.StepBody{Step: "one"})
	_ = s.Save(&stale) // a crash between the append and the save
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
	_ = s.Create(ctx, &a, abody)
	b, bbody := newManifest(t)
	b.CreatedAt = now.Unix() + 10
	_ = s.Create(ctx, &b, bbody)
	_ = os.WriteFile(filepath.Join(s.Dir(), a.ID+".log"), nil, 0o600)
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
	_ = s.Create(ctx, &old, oldbody)
	running, runningbody := newManifest(t)
	running.State = intent.JobRunning
	_ = s.Create(ctx, &running, runningbody)
	*now = now.Add(31 * 24 * time.Hour)
	recent, recentbody := newManifest(t)
	recent.State = intent.JobSucceeded
	_ = s.Create(ctx, &recent, recentbody)
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
	_ = s.Create(ctx, &m, mbody)
	_ = s.Append(ctx, &m, joblog.KindStepStarted, joblog.StepBody{Step: "one"})
	m.LogHead = "0x" + strings.Repeat("11", 32) // same count, a different head
	_ = s.Save(&m)
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
