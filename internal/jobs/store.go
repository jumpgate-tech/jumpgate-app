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

// ResultPath is where step step's attempt writes its result.
func ResultPath(dir, id, step string, attempt int) string {
	return filepath.Join(resultDir(dir, id), step+"-"+strconv.Itoa(attempt)+".json")
}

// Create writes entry 0 (job.created) and then the manifest. The order
// matters: a crash in between leaves a log Load completes, never a manifest
// whose history is missing.
func (s *Store) Create(ctx context.Context, m *Manifest, created joblog.IntentBody) error {
	if !ValidID(m.ID) {
		return refuse(intent.ReasonValidation, fmt.Sprintf("%q is not a job id", m.ID))
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
	if err := os.MkdirAll(resultDir(s.dir, m.ID), 0o700); err != nil {
		return err
	}
	m.LogLen, m.LogHead = 0, ""
	if err := s.Append(ctx, m, joblog.KindCreated, created); err != nil {
		return err
	}
	return s.Save(m)
}

// Append signs body as the next entry of m's log, writes it and fsyncs, and
// moves m's LogLen and LogHead on. The caller saves m.
func (s *Store) Append(ctx context.Context, m *Manifest, kind string, body any) error {
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
	line, _ := json.Marshal(joblog.ToJSON(e))
	s.mu.Lock()
	defer s.mu.Unlock()
	f, err := os.OpenFile(logPath(s.dir, m.ID), os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o600)
	if err != nil {
		return err
	}
	if _, err := f.Write(append(line, '\n')); err != nil {
		f.Close()
		return err
	}
	if err := f.Sync(); err != nil {
		f.Close()
		return err
	}
	if err := f.Close(); err != nil {
		return err
	}
	m.LogLen++
	m.LogHead = "0x" + hex.EncodeToString(d[:])
	m.UpdatedAt = s.now().Unix()
	return nil
}

// Save replaces m's manifest atomically, 0600.
func (s *Store) Save(m *Manifest) error {
	b, err := json.MarshalIndent(m, "", "  ")
	if err != nil {
		return err
	}
	return fsatomic.WriteFile(manifestPath(s.dir, m.ID), b)
}

// LoadManifest reads a manifest without checking its log: for step processes,
// which hold no key and only read.
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
	return m, nil
}

// Load reads a manifest and checks its log against it. The log may be one
// entry longer (an append whose manifest save a crash cut off), which Load
// completes; any other mismatch is ErrDamaged.
func (s *Store) Load(id string) (Manifest, error) {
	m, err := LoadManifest(s.dir, id)
	if err != nil {
		return m, err
	}
	entries, err := s.readAll(id)
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
	case n == m.LogLen && headHex == m.LogHead:
		return m, nil
	case n == m.LogLen+1 && n > 0 && "0x"+hex.EncodeToString(entries[n-1].Prev[:]) == manifestPrev(m):
		// One entry past the manifest, chained onto the head it recorded: an
		// append whose manifest save a crash cut off.
		m.LogLen, m.LogHead = n, headHex
		return m, s.Save(&m)
	}
	return m, fmt.Errorf("%w: job %s's log has %d entries (head %s), its manifest expects %d (head %s)", ErrDamaged, id, n, headHex, m.LogLen, m.LogHead)
}

// manifestPrev is the head the manifest recorded, with the empty head of a
// log that has no entries spelled as the zero digest.
func manifestPrev(m Manifest) string {
	if m.LogHead == "" {
		return "0x" + strings.Repeat("00", 32)
	}
	return m.LogHead
}

func (s *Store) readAll(id string) ([]joblog.Entry, error) {
	es, _, err := s.ReadLog(id, 0, 0)
	return es, err
}

// ReadLog returns up to max entries from index from (max 0: all), and
// whether more follow.
func (s *Store) ReadLog(id string, from uint64, max int) ([]joblog.Entry, bool, error) {
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
	return out, false, sc.Err()
}

// List is every job, newest first. A job whose log does not check out is
// listed as damaged with the reason, never hidden.
func (s *Store) List() ([]Manifest, error) {
	ents, err := os.ReadDir(s.dir)
	if err != nil {
		return nil, err
	}
	var out []Manifest
	for _, e := range ents {
		id, ok := strings.CutSuffix(e.Name(), ".json")
		if !ok || !ValidID(id) {
			continue
		}
		m, err := s.Load(id)
		if errors.Is(err, ErrDamaged) {
			if m.ID == "" {
				m.ID = id // the manifest itself did not parse
			}
			m.State, m.Error = intent.JobDamaged, err.Error()
		} else if err != nil {
			continue
		}
		out = append(out, m)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].CreatedAt > out[j].CreatedAt })
	return out, nil
}

// Prune removes finished jobs last changed more than maxAge ago, and all but
// the newest keep finished jobs (D-J6). Unfinished jobs are never pruned.
func (s *Store) Prune(maxAge time.Duration, keep int) error {
	ms, err := s.List()
	if err != nil {
		return err
	}
	cutoff, kept := s.now().Add(-maxAge).Unix(), 0
	for _, m := range ms {
		if !intent.JobTerminal(m.State) {
			continue
		}
		kept++
		if m.UpdatedAt >= cutoff && kept <= keep {
			continue
		}
		_ = os.RemoveAll(resultDir(s.dir, m.ID))
		_ = os.Remove(logPath(s.dir, m.ID))
		if err := os.Remove(manifestPath(s.dir, m.ID)); err != nil && !errors.Is(err, fs.ErrNotExist) {
			return err
		}
	}
	return fsatomic.SyncDir(s.dir)
}
