package vault

import (
	"bytes"
	"crypto/rand"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"slices"
	"sync"
	"time"

	"github.com/valve-tech/jumpgate/internal/filelock"
	"github.com/valve-tech/jumpgate/internal/fsatomic"
	"github.com/valve-tech/jumpgate/internal/fsperm"
)

// MaxSize bounds a vault file, on read and on write.
const MaxSize = 1 << 20

var (
	ErrNoVault    = errors.New("vault: no vault")
	ErrExists     = errors.New("vault: a vault already exists")
	ErrLocked     = errors.New("vault: locked")
	ErrUnsafeFile = errors.New("vault: the vault file is not safe to read")
	ErrNotFound   = errors.New("vault: no such entry")
	ErrTooLarge   = errors.New("vault: the vault would exceed its size limit")
)

// Params are argon2id parameters for a passphrase slot.
type Params struct {
	Memory  uint32 // KiB
	Time    uint32
	Threads uint8
}

// DefaultParams are also the floor: a vault made with them opens anywhere.
var DefaultParams = Params{Memory: 65536, Time: 3, Threads: 4}

// Item is one entry to write.
type Item struct {
	Name  string
	Value []byte
	From  string
}

// Entry is one stored secret and where it came from.
type Entry struct {
	Value   []byte
	Created time.Time
	From    string
}

// writeHook, when set by a test, changes the bytes about to be written.
var writeHook func([]byte) []byte

type stamp struct {
	fi os.FileInfo
}

func (s stamp) same(fi os.FileInfo) bool {
	return s.fi != nil && fi != nil && os.SameFile(s.fi, fi) &&
		s.fi.Size() == fi.Size() && s.fi.ModTime().Equal(fi.ModTime())
}

// Vault is one vault file. It is safe for concurrent use; every write takes
// an exclusive lock on <path>.lock and starts from the file on disk, so two
// writers never drop each other's entries.
type Vault struct {
	mu      sync.Mutex
	path    string
	hdr     header
	dek     []byte // nil when locked
	entries map[string]entryRec
	seen    stamp
}

func (p Params) kdf() (kdfParams, error) {
	k := kdfParams{Alg: kdfAlg, M: p.Memory, T: p.Time, P: p.Threads, Salt: make([]byte, saltLen)}
	if _, err := rand.Read(k.Salt); err != nil {
		return kdfParams{}, fmt.Errorf("vault: %w", err)
	}
	if err := checkKDF(k); err != nil {
		return kdfParams{}, fmt.Errorf("vault: argon2id parameters m=%d t=%d p=%d are outside the allowed range", p.Memory, p.Time, p.Threads)
	}
	return k, nil
}

// passSlot derives a KEK from pass and wraps dek into a new passphrase slot.
func passSlot(label string, pass, dek []byte, p Params) (slotRec, error) {
	k, err := p.kdf()
	if err != nil {
		return slotRec{}, err
	}
	kek := deriveKEK(pass, k)
	defer clear(kek)
	s := slotRec{Kind: string(SlotPassphrase), Label: label, KDF: &k}
	if err := wrapDEK(kek, dek, &s); err != nil {
		return slotRec{}, err
	}
	return s, nil
}

// Create makes a new vault at path with one "primary" passphrase slot and
// returns it unlocked. It clears passphrase.
func Create(path string, passphrase []byte, p Params) (*Vault, error) {
	defer clear(passphrase)
	if len(passphrase) == 0 {
		return nil, errors.New("vault: empty passphrase")
	}
	dek := make([]byte, keyLen)
	if _, err := rand.Read(dek); err != nil {
		return nil, fmt.Errorf("vault: %w", err)
	}
	s, err := passSlot("primary", passphrase, dek, p)
	if err != nil {
		clear(dek)
		return nil, err
	}
	b := body{Entries: map[string]entryRec{}}
	raw, err := seal(header{V: version, Slots: []slotRec{s}}, dek, b)
	if err != nil {
		clear(dek)
		return nil, err
	}
	lk, err := filelock.Lock(path+".lock", true)
	if err != nil {
		clear(dek)
		return nil, fmt.Errorf("vault: lock %s: %w", path, err)
	}
	defer lk.Unlock()
	if err := createFile(path, raw); err != nil {
		clear(dek)
		return nil, err
	}
	v := &Vault{path: path}
	if err := v.reload(dek); err != nil {
		clear(dek)
		return nil, err
	}
	return v, nil
}

func createFile(path string, raw []byte) error {
	f, err := fsperm.CreatePrivate(path)
	if errors.Is(err, fs.ErrExist) {
		return fmt.Errorf("%w at %s", ErrExists, path)
	}
	if err != nil {
		return fmt.Errorf("vault: %w", err)
	}
	_, werr := f.Write(raw)
	if werr == nil {
		werr = f.Sync()
	}
	if cerr := f.Close(); werr == nil {
		werr = cerr
	}
	if werr != nil {
		os.Remove(path)
		return fmt.Errorf("vault: write %s: %w", path, werr)
	}
	return fsatomic.SyncDir(filepath.Dir(path))
}

// Open checks the file and its header and returns the vault locked.
func Open(path string) (*Vault, error) {
	raw, fi, err := readFile(path)
	if err != nil {
		return nil, err
	}
	h, _, _, err := parse(raw)
	if err != nil {
		return nil, err
	}
	return &Vault{path: path, hdr: h, seen: stamp{fi}}, nil
}

// readFile reads path without following a link, after checking that the open
// handle is a private regular file of this user, no larger than MaxSize.
func readFile(path string) ([]byte, os.FileInfo, error) {
	f, err := openNoFollow(path)
	if errors.Is(err, fs.ErrNotExist) {
		return nil, nil, fmt.Errorf("%w at %s: %w", ErrNoVault, path, err)
	}
	if err != nil {
		return nil, nil, err
	}
	defer f.Close()
	fi, err := f.Stat()
	if err != nil {
		return nil, nil, fmt.Errorf("vault: %w", err)
	}
	if !fi.Mode().IsRegular() {
		return nil, nil, fmt.Errorf("%w: %s is not a regular file", ErrUnsafeFile, path)
	}
	if err := fsperm.CheckPrivateFile(f); err != nil {
		return nil, nil, fmt.Errorf("%w: %w", ErrUnsafeFile, err)
	}
	if err := checkOwner(path, fi); err != nil {
		return nil, nil, err
	}
	if fi.Size() > MaxSize {
		return nil, nil, corrupt("larger than 1 MiB")
	}
	raw, err := io.ReadAll(io.LimitReader(f, MaxSize+1))
	if err != nil {
		return nil, nil, fmt.Errorf("vault: read %s: %w", path, err)
	}
	if len(raw) > MaxSize {
		return nil, nil, corrupt("larger than 1 MiB")
	}
	return raw, fi, nil
}

// Path is the vault file's path.
func (v *Vault) Path() string { return v.path }

// Locked reports whether the vault holds no data key.
func (v *Vault) Locked() bool {
	v.mu.Lock()
	defer v.mu.Unlock()
	return v.dek == nil
}

// Unlock tries passphrase on every passphrase slot. On an unlocked vault it
// checks the passphrase and keeps the data key; a wrong one does not lock.
// It clears passphrase.
func (v *Vault) Unlock(passphrase []byte) error {
	defer clear(passphrase)
	return v.unlockWith(func(s slotRec) []byte {
		if SlotKind(s.Kind) != SlotPassphrase {
			return nil
		}
		return deriveKEK(passphrase, *s.KDF)
	}, ErrWrongPassphrase)
}

// unlockWith reads the file and tries kekFor on each slot (nil skips one).
// The vault's state changes only on success.
func (v *Vault) unlockWith(kekFor func(slotRec) []byte, refused error) error {
	v.mu.Lock()
	defer v.mu.Unlock()
	raw, fi, err := readFile(v.path)
	if err != nil {
		return err
	}
	h, line, ct, err := parse(raw)
	if err != nil {
		return err
	}
	for _, s := range h.Slots {
		kek := kekFor(s)
		if kek == nil {
			continue
		}
		dek, err := unwrapDEK(kek, s)
		clear(kek)
		if err != nil {
			continue
		}
		b, err := openBody(line, ct, dek, h.Nonce)
		if err != nil {
			clear(dek)
			return err
		}
		v.drop()
		v.hdr, v.dek, v.entries, v.seen = h, dek, b.Entries, stamp{fi}
		return nil
	}
	return refused
}

// reload re-reads the file and decrypts it with dek, which it takes over.
// Callers hold v.mu (or own v exclusively).
func (v *Vault) reload(dek []byte) error {
	raw, fi, err := readFile(v.path)
	if err != nil {
		return err
	}
	h, line, ct, err := parse(raw)
	if err != nil {
		return err
	}
	b, err := openBody(line, ct, dek, h.Nonce)
	if err != nil {
		return err
	}
	if v.dek != nil && &v.dek[0] != &dek[0] {
		clear(v.dek)
	}
	clearEntries(v.entries)
	v.hdr, v.dek, v.entries, v.seen = h, dek, b.Entries, stamp{fi}
	return nil
}

// fresh makes an unlocked vault current with the file. A file the held data
// key cannot decrypt locks the vault. Callers hold v.mu.
func (v *Vault) fresh() error {
	if v.dek == nil {
		return ErrLocked
	}
	fi, err := os.Lstat(v.path)
	if err == nil && v.seen.same(fi) {
		return nil
	}
	if err := v.reload(v.dek); err != nil {
		v.drop()
		return err
	}
	return nil
}

func clearEntries(m map[string]entryRec) {
	for _, e := range m {
		clear(e.Value)
	}
}

// drop clears the data key and every value. Callers hold v.mu.
func (v *Vault) drop() {
	clear(v.dek)
	v.dek = nil
	clearEntries(v.entries)
	v.entries = nil
}

// Lock clears the data key and every value held in memory.
func (v *Vault) Lock() {
	v.mu.Lock()
	defer v.mu.Unlock()
	v.drop()
}

// Get returns a copy of the named value.
func (v *Vault) Get(name string) ([]byte, error) {
	e, err := v.Entry(name)
	return e.Value, err
}

// Entry returns a copy of the named entry.
func (v *Vault) Entry(name string) (Entry, error) {
	v.mu.Lock()
	defer v.mu.Unlock()
	if err := v.fresh(); err != nil {
		return Entry{}, err
	}
	e, ok := v.entries[name]
	if !ok {
		return Entry{}, fmt.Errorf("%w: %s", ErrNotFound, name)
	}
	return Entry{Value: bytes.Clone(e.Value), Created: e.Created, From: e.From}, nil
}

// Names lists the entries, sorted.
func (v *Vault) Names() ([]string, error) {
	v.mu.Lock()
	defer v.mu.Unlock()
	if err := v.fresh(); err != nil {
		return nil, err
	}
	n := make([]string, 0, len(v.entries))
	for k := range v.entries {
		n = append(n, k)
	}
	slices.Sort(n)
	return n, nil
}

// Put stores one value; from records where it came from.
func (v *Vault) Put(name string, value []byte, from string) error {
	return v.PutAll([]Item{{Name: name, Value: value, From: from}})
}

// PutAll stores every item in one verified write.
func (v *Vault) PutAll(items []Item) error {
	for _, it := range items {
		if err := validName(it.Name); err != nil {
			return err
		}
	}
	now := time.Now().UTC()
	return v.update(func(_ *header, m map[string]entryRec) error {
		for _, it := range items {
			m[it.Name] = entryRec{Value: bytes.Clone(it.Value), Created: now, From: it.From}
		}
		return nil
	})
}

// Delete removes an entry; ErrNotFound when it is absent.
func (v *Vault) Delete(name string) error {
	return v.update(func(_ *header, m map[string]entryRec) error {
		e, ok := m[name]
		if !ok {
			return fmt.Errorf("%w: %s", ErrNotFound, name)
		}
		clear(e.Value)
		delete(m, name)
		return nil
	})
}

// update applies fn to a copy of the current header and entries, under the
// file lock and from the file on disk, then writes and verifies the result.
// On any error the vault in memory is as it was before. Callers must not
// hold v.mu.
func (v *Vault) update(fn func(h *header, m map[string]entryRec) error) error {
	v.mu.Lock()
	defer v.mu.Unlock()
	if v.dek == nil {
		return ErrLocked
	}
	lk, err := filelock.Lock(v.path+".lock", true)
	if err != nil {
		return fmt.Errorf("vault: lock %s: %w", v.path, err)
	}
	defer lk.Unlock()
	if err := v.reload(v.dek); err != nil {
		v.drop()
		return err
	}
	h := v.hdr
	h.Slots = slices.Clone(v.hdr.Slots)
	m := make(map[string]entryRec, len(v.entries))
	for k, e := range v.entries {
		m[k] = entryRec{Value: bytes.Clone(e.Value), Created: e.Created, From: e.From}
	}
	if err := fn(&h, m); err != nil {
		clearEntries(m)
		return err
	}
	b := body{Entries: m}
	raw, err := seal(h, v.dek, b)
	if err != nil {
		clearEntries(m)
		return err
	}
	if len(raw) > MaxSize {
		clearEntries(m)
		return ErrTooLarge
	}
	if writeHook != nil {
		raw = writeHook(raw)
	}
	if err := v.writeVerified(raw, b); err != nil {
		clearEntries(m)
		return err
	}
	clearEntries(m)
	if err := v.reload(v.dek); err != nil {
		v.drop()
		return err
	}
	return nil
}

// writeVerified writes raw to a private temp file, reads it back and checks
// that it decrypts to want under the held key, and only then renames it over
// the vault and syncs the directory.
func (v *Vault) writeVerified(raw []byte, want body) error {
	dir := filepath.Dir(v.path)
	tmp, err := fsperm.CreateTempPrivate(dir, "."+filepath.Base(v.path)+".tmp-*")
	if err != nil {
		return fmt.Errorf("vault: %w", err)
	}
	name := tmp.Name()
	ok := false
	defer func() {
		if !ok {
			tmp.Close()
			os.Remove(name)
		}
	}()
	if err := keepFileOwner(tmp, v.seen.fi); err != nil {
		return err
	}
	if _, err := tmp.Write(raw); err != nil {
		return fmt.Errorf("vault: write: %w", err)
	}
	if err := tmp.Sync(); err != nil {
		return fmt.Errorf("vault: write: %w", err)
	}
	if err := tmp.Close(); err != nil {
		return fmt.Errorf("vault: write: %w", err)
	}
	back, err := os.ReadFile(name)
	if err != nil {
		return fmt.Errorf("vault: read back: %w", err)
	}
	if err := v.verify(back, want); err != nil {
		return fmt.Errorf("vault: the written file does not read back: %w", err)
	}
	if err := fsperm.Rename(name, v.path); err != nil {
		return fmt.Errorf("vault: %w", err)
	}
	ok = true
	return fsatomic.SyncDir(dir)
}

func (v *Vault) verify(raw []byte, want body) error {
	h, line, ct, err := parse(raw)
	if err != nil {
		return err
	}
	got, err := openBody(line, ct, v.dek, h.Nonce)
	if err != nil {
		return err
	}
	defer clearEntries(got.Entries)
	a, err := json.Marshal(got)
	if err != nil {
		return err
	}
	defer clear(a)
	b, err := json.Marshal(want)
	if err != nil {
		return err
	}
	defer clear(b)
	if !bytes.Equal(a, b) {
		return corrupt("contents differ")
	}
	return nil
}
