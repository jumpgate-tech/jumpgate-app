package jobs

import (
	"bufio"
	"context"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/valve-tech/jumpgate/internal/fsatomic"
	"github.com/valve-tech/jumpgate/internal/intent"
	"github.com/valve-tech/jumpgate/internal/joblog"
	"github.com/valve-tech/jumpgate/internal/signer"
)

// Store is the jobs directory. Every write is the agent's: Save replaces a
// manifest atomically, Append adds one signed line to a log and fsyncs it.
type Store struct {
	dir string
	key signer.Signer
	now func() time.Time
	mu  sync.Mutex
}

// OpenStore creates dir (0700) if absent.
func OpenStore(dir string, key signer.Signer, now func() time.Time) (*Store, error) {
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return nil, err
	}
	if err := os.Chmod(dir, 0o700); err != nil {
		return nil, err
	}
	if now == nil {
		now = time.Now
	}
	return &Store{dir: dir, key: key, now: now}, nil
}

func (s *Store) Dir() string { return s.dir }

func manifestPath(dir, id string) string { return filepath.Join(dir, id+".json") }
func logPath(dir, id string) string      { return filepath.Join(dir, id+".log") }
func resultDir(dir, id string) string    { return filepath.Join(dir, id+".d") }

// ResultPath is where step step's attempt writes its result. A bad id, step
// or attempt is refused rather than built into a path.
func ResultPath(dir, id, step string, attempt int) (string, error) {
	if !ValidID(id) {
		return "", refuse(intent.ReasonValidation, fmt.Sprintf("%q is not a job id", id))
	}
	if !ValidStepID(step) {
		return "", refuse(intent.ReasonValidation, fmt.Sprintf("%q is not a step id", step))
	}
	if attempt < 0 {
		return "", refuse(intent.ReasonValidation, fmt.Sprintf("attempt %d is negative", attempt))
	}
	return filepath.Join(resultDir(dir, id), step+"-"+strconv.Itoa(attempt)+".json"), nil
}

func badID(id string) error {
	return refuse(intent.ReasonValidation, fmt.Sprintf("%q is not a job id", id))
}

// syncFile fsyncs a log file. It is a variable so a test can make the fsync
// of an append fail.
var syncFile = func(f *os.File) error { return f.Sync() }

// Locking: every method that touches a job's files holds s.mu, including
// Load's read-and-repair, so a repair can never overwrite a manifest a writer
// saved a moment later, and two appenders can never sign the same index.

// Create starts a job. It saves the manifest with an empty log first, then
// appends entry 0 (job.created), then saves again. A crash between the steps
// leaves a manifest Load accepts (an empty log it expects) or a log one entry
// ahead of its manifest, which Load completes; never a manifest that claims
// history the log lacks. Creating a job that already exists is refused.
func (s *Store) Create(ctx context.Context, m *Manifest, created joblog.IntentBody) error {
	if !ValidID(m.ID) {
		return badID(m.ID)
	}
	// The id must be the one the creating intent derives, so the first entry
	// of the log proves which signed request started the job.
	in, err := created.Envelope.Intent.Parse()
	if err != nil {
		return refuse(intent.ReasonValidation, "the creating intent does not parse: "+err.Error())
	}
	if want, err := joblog.JobIDFor(in); err != nil || want != m.ID {
		return refuse(intent.ReasonValidation, fmt.Sprintf("job id %s is not the id its creating intent derives", m.ID))
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, p := range []string{manifestPath(s.dir, m.ID), logPath(s.dir, m.ID)} {
		if _, err := os.Lstat(p); err == nil {
			return refuse(intent.ReasonValidation, fmt.Sprintf("job %s already exists", m.ID))
		} else if !errors.Is(err, fs.ErrNotExist) {
			return err
		}
	}
	if err := os.MkdirAll(resultDir(s.dir, m.ID), 0o700); err != nil {
		return err
	}
	m.LogLen, m.LogHead = 0, ""
	if err := s.saveLocked(m); err != nil {
		return err
	}
	return s.appendLocked(ctx, m, joblog.KindCreated, created)
}

// Append signs body as the next entry of m's log, writes and fsyncs it, moves
// m's LogLen and LogHead on and saves the manifest, as one operation under the
// store lock. If the write or fsync fails the log is cut back to its old
// length, so the next Append signs the same index onto a log that still
// verifies. If only the manifest save fails the entry stays in the log, one
// ahead of the manifest, which Load completes.
func (s *Store) Append(ctx context.Context, m *Manifest, kind string, body any) error {
	if !ValidID(m.ID) {
		return badID(m.ID)
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.appendLocked(ctx, m, kind, body)
}

func (s *Store) appendLocked(ctx context.Context, m *Manifest, kind string, body any) error {
	b, err := json.Marshal(body)
	if err != nil {
		return err
	}
	var prev [32]byte
	if m.LogHead != "" {
		p, err := hex.DecodeString(strings.TrimPrefix(m.LogHead, "0x"))
		if err != nil || len(p) != 32 {
			return fmt.Errorf("%w: job %s has a bad log head", ErrDamaged, m.ID)
		}
		copy(prev[:], p)
	}
	e, d, err := joblog.Sign(ctx, s.key, m.ID, m.LogLen, prev, kind, b, s.now())
	if err != nil {
		return err
	}
	line, err := json.Marshal(joblog.ToJSON(e))
	if err != nil {
		return err
	}
	if err := s.writeLine(logPath(s.dir, m.ID), append(line, '\n')); err != nil {
		return err
	}
	m.LogLen++
	m.LogHead = "0x" + hex.EncodeToString(d[:])
	m.UpdatedAt = s.now().Unix()
	return s.saveLocked(m)
}

// writeLine appends line to the log at path and fsyncs. On any failure it
// truncates the file back to its size before the write, so a half-written or
// unsynced line never stays on disk ahead of the manifest.
func (s *Store) writeLine(path string, line []byte) error {
	f, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o600)
	if err != nil {
		return err
	}
	fi, err := f.Stat()
	if err != nil {
		f.Close()
		return err
	}
	size := fi.Size()
	werr := func() error {
		if _, err := f.Write(line); err != nil {
			return err
		}
		return syncFile(f)
	}()
	if werr != nil {
		if terr := f.Truncate(size); terr != nil {
			werr = fmt.Errorf("%w (and cutting the log back failed: %v)", werr, terr)
		} else if serr := f.Sync(); serr != nil {
			werr = fmt.Errorf("%w (and syncing the cut failed: %v)", werr, serr)
		}
		f.Close()
		return werr
	}
	return f.Close()
}

// Save replaces m's manifest atomically, 0600.
func (s *Store) Save(m *Manifest) error {
	if !ValidID(m.ID) {
		return badID(m.ID)
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.saveLocked(m)
}

func (s *Store) saveLocked(m *Manifest) error {
	b, err := json.MarshalIndent(m, "", "  ")
	if err != nil {
		return err
	}
	return fsatomic.WriteFile(manifestPath(s.dir, m.ID), b)
}

// LoadManifest reads a manifest without checking its log: for step processes,
// which hold no key and only read. A manifest that names another job than its
// filename is damaged.
func LoadManifest(dir, id string) (Manifest, error) {
	var m Manifest
	if !ValidID(id) {
		return m, ErrNotFound
	}
	b, err := os.ReadFile(manifestPath(dir, id))
	if errors.Is(err, fs.ErrNotExist) {
		return m, ErrNotFound
	}
	if err != nil {
		return m, err
	}
	if err := json.Unmarshal(b, &m); err != nil {
		return m, fmt.Errorf("%w: manifest %s: %v", ErrDamaged, id, err)
	}
	if m.ID != id {
		return m, fmt.Errorf("%w: manifest %s names job %q", ErrDamaged, id, m.ID)
	}
	return m, nil
}

// Load reads a manifest and checks its log against it, under the store lock.
// The log may be one entry longer (an append whose manifest save a crash cut
// off), which Load completes and saves; any other mismatch is ErrDamaged.
func (s *Store) Load(id string) (Manifest, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.loadLocked(id)
}

func (s *Store) loadLocked(id string) (Manifest, error) {
	m, err := LoadManifest(s.dir, id)
	if err != nil {
		return m, err
	}
	entries, err := s.readLog(id, 0, 0)
	if err != nil {
		return m, err
	}
	head, err := joblog.Verify(s.key.Address(), id, entries, 0, [32]byte{})
	if err != nil {
		return m, fmt.Errorf("%w: %v", ErrDamaged, err)
	}
	// Verify proves only a valid prefix, so the manifest's own record of the
	// log is what catches a dropped tail: both the count and the head digest
	// must agree.
	n, headHex := uint64(len(entries)), "0x"+hex.EncodeToString(head[:])
	switch {
	case n == m.LogLen && headHex == manifestPrev(m):
		return m, nil
	case n == m.LogLen+1 && n > 0 && "0x"+hex.EncodeToString(entries[n-1].Prev[:]) == manifestPrev(m):
		// One entry past the manifest, chained onto the head it recorded: an
		// append whose manifest save a crash cut off.
		m.LogLen, m.LogHead = n, headHex
		return m, s.saveLocked(&m)
	}
	return m, fmt.Errorf("%w: job %s's log has %d entries (head %s), its manifest expects %d (head %s)", ErrDamaged, id, n, headHex, m.LogLen, manifestPrev(m))
}

// manifestPrev is the head the manifest recorded, with the empty head of a
// log that has no entries spelled as the zero digest.
func manifestPrev(m Manifest) string {
	if m.LogHead == "" {
		return "0x" + strings.Repeat("00", 32)
	}
	return m.LogHead
}

// ReadLog returns up to max entries from index from (max 0: all), and
// whether more follow.
func (s *Store) ReadLog(id string, from uint64, max int) ([]joblog.Entry, bool, error) {
	if !ValidID(id) {
		return nil, false, badID(id)
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.readLogMore(id, from, max)
}

func (s *Store) readLog(id string, from uint64, max int) ([]joblog.Entry, error) {
	es, _, err := s.readLogMore(id, from, max)
	return es, err
}

func (s *Store) readLogMore(id string, from uint64, max int) ([]joblog.Entry, bool, error) {
	f, err := os.Open(logPath(s.dir, id))
	if errors.Is(err, fs.ErrNotExist) {
		return nil, false, nil
	}
	if err != nil {
		return nil, false, err
	}
	defer f.Close()
	sc := bufio.NewScanner(f)
	sc.Buffer(make([]byte, 64<<10), 4<<20)
	var out []joblog.Entry
	var i uint64
	for sc.Scan() {
		if i < from {
			i++
			continue
		}
		if max > 0 && len(out) == max {
			return out, true, nil
		}
		var j intent.JobLogEntryJSON
		if err := json.Unmarshal(sc.Bytes(), &j); err != nil {
			return out, false, fmt.Errorf("%w: job %s entry %d: %v", ErrDamaged, id, i, err)
		}
		e, err := joblog.FromJSON(j)
		if err != nil {
			return out, false, fmt.Errorf("%w: %v", ErrDamaged, err)
		}
		out = append(out, e)
		i++
	}
	if err := sc.Err(); err != nil {
		return out, false, fmt.Errorf("%w: job %s's log cannot be read past entry %d: %v", ErrDamaged, id, i, err)
	}
	return out, false, nil
}

// List is every job, newest first. A job whose log or manifest does not check
// out is listed as damaged with the reason, never hidden.
func (s *Store) List() ([]Manifest, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	js, err := s.listLocked()
	out := make([]Manifest, len(js))
	for i, j := range js {
		out[i] = j.m
	}
	return out, err
}

// listed is a job with whether it checked out. Prune must tell the two apart:
// a damaged job is evidence, not clutter.
type listed struct {
	m       Manifest
	damaged bool
}

func (s *Store) listLocked() ([]listed, error) {
	ents, err := os.ReadDir(s.dir)
	if err != nil {
		return nil, err
	}
	var out []listed
	for _, e := range ents {
		id, ok := strings.CutSuffix(e.Name(), ".json")
		if !ok || !ValidID(id) {
			continue
		}
		m, err := s.loadLocked(id)
		if errors.Is(err, ErrNotFound) {
			continue
		}
		damaged := err != nil
		if damaged {
			m.ID = id // the manifest itself may not have parsed
			m.State, m.Error = intent.JobDamaged, err.Error()
		}
		out = append(out, listed{m, damaged})
	}
	sort.Slice(out, func(i, j int) bool { return out[i].m.CreatedAt > out[j].m.CreatedAt })
	return out, nil
}

// Prune removes finished jobs last changed more than maxAge ago, and all but
// the newest keep finished jobs (D-J6). Unfinished jobs are never pruned, and
// neither is a job that failed its check: its log is evidence. A job's files
// go before its manifest, so a failed removal leaves a job that is still
// listed and can be retried.
func (s *Store) Prune(maxAge time.Duration, keep int) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	js, err := s.listLocked()
	if err != nil {
		return err
	}
	cutoff, kept := s.now().Add(-maxAge).Unix(), 0
	for _, j := range js {
		m := j.m
		if j.damaged || !intent.JobTerminal(m.State) {
			continue
		}
		kept++
		if m.UpdatedAt >= cutoff && kept <= keep {
			continue
		}
		if err := os.RemoveAll(resultDir(s.dir, m.ID)); err != nil {
			return err
		}
		if err := os.Remove(logPath(s.dir, m.ID)); err != nil && !errors.Is(err, fs.ErrNotExist) {
			return err
		}
		if err := os.Remove(manifestPath(s.dir, m.ID)); err != nil && !errors.Is(err, fs.ErrNotExist) {
			return err
		}
	}
	return fsatomic.SyncDir(s.dir)
}
