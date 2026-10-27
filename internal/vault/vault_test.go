package vault

import (
	"bytes"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sync"
	"testing"
)

func pass() []byte { return bytes.Clone(testPass) }

// newVault creates an unlocked vault in a fresh directory.
func newVault(t *testing.T) (*Vault, string) {
	t.Helper()
	p := filepath.Join(t.TempDir(), "vault")
	v, err := Create(p, pass(), DefaultParams)
	if err != nil {
		t.Fatal(err)
	}
	return v, p
}

func openUnlocked(t *testing.T, p string) *Vault {
	t.Helper()
	v, err := Open(p)
	if err != nil {
		t.Fatal(err)
	}
	if err := v.Unlock(pass()); err != nil {
		t.Fatal(err)
	}
	return v
}

func mustGet(t *testing.T, v *Vault, name string) []byte {
	t.Helper()
	b, err := v.Get(name)
	if err != nil {
		t.Fatalf("Get(%q): %v", name, err)
	}
	return b
}

func TestCreateThenOpenUnlockGet(t *testing.T) {
	v, p := newVault(t)
	if v.Locked() || v.Path() != p {
		t.Fatalf("Create: locked=%v path=%q", v.Locked(), v.Path())
	}
	if err := v.Put("ai-key", []byte("k"), "config"); err != nil {
		t.Fatal(err)
	}
	w, err := Open(p)
	if err != nil {
		t.Fatal(err)
	}
	if !w.Locked() {
		t.Fatal("Open returned an unlocked vault")
	}
	if err := w.Unlock(pass()); err != nil {
		t.Fatal(err)
	}
	if got := mustGet(t, w, "ai-key"); string(got) != "k" {
		t.Fatalf("Get = %q", got)
	}
	e, err := w.Entry("ai-key")
	if err != nil || e.From != "config" || e.Created.IsZero() {
		t.Fatalf("Entry = %+v, %v", e, err)
	}
	if _, err := w.Get("nope"); !errors.Is(err, ErrNotFound) {
		t.Fatalf("Get(missing) = %v", err)
	}
	if err := w.Delete("ai-key"); err != nil {
		t.Fatal(err)
	}
	if err := w.Delete("ai-key"); !errors.Is(err, ErrNotFound) {
		t.Fatalf("Delete twice = %v", err)
	}
}

func TestCreateRefusesAnExistingFile(t *testing.T) {
	_, p := newVault(t)
	before, _ := os.ReadFile(p)
	if _, err := Create(p, pass(), DefaultParams); !errors.Is(err, ErrExists) {
		t.Fatalf("err = %v, want ErrExists", err)
	}
	after, _ := os.ReadFile(p)
	if !bytes.Equal(before, after) {
		t.Fatal("the existing file changed")
	}
}

func TestCreateRefusesWeakParams(t *testing.T) {
	p := filepath.Join(t.TempDir(), "vault")
	if _, err := Create(p, pass(), Params{Memory: 1024, Time: 1, Threads: 1}); err == nil {
		t.Fatal("weak parameters accepted")
	}
	if _, err := os.Stat(p); !errors.Is(err, fs.ErrNotExist) {
		t.Fatal("a file was left behind")
	}
}

func TestPassphraseSlicesAreCleared(t *testing.T) {
	p := filepath.Join(t.TempDir(), "vault")
	a := pass()
	v, err := Create(p, a, DefaultParams)
	if err != nil {
		t.Fatal(err)
	}
	b, c := pass(), []byte("wrong")
	if err := v.Unlock(b); err != nil {
		t.Fatal(err)
	}
	if err := v.Unlock(c); !errors.Is(err, ErrWrongPassphrase) {
		t.Fatalf("wrong passphrase: %v", err)
	}
	for _, s := range [][]byte{a, b, c} {
		if !bytes.Equal(s, make([]byte, len(s))) {
			t.Fatalf("passphrase not cleared: %q", s)
		}
	}
	if v.Locked() {
		t.Fatal("a wrong passphrase locked an unlocked vault")
	}
}

func TestWrongPassphraseKeepsItLocked(t *testing.T) {
	_, p := newVault(t)
	v, err := Open(p)
	if err != nil {
		t.Fatal(err)
	}
	if err := v.Unlock([]byte("nope")); !errors.Is(err, ErrWrongPassphrase) || errors.Is(err, ErrCorrupt) {
		t.Fatalf("err = %v", err)
	}
	if !v.Locked() {
		t.Fatal("unlocked by a wrong passphrase")
	}
}

func TestLockedVaultRefusesReadsAndWrites(t *testing.T) {
	_, p := newVault(t)
	v, err := Open(p)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := v.Get("x"); !errors.Is(err, ErrLocked) {
		t.Fatalf("Get: %v", err)
	}
	if _, err := v.Entry("x"); !errors.Is(err, ErrLocked) {
		t.Fatalf("Entry: %v", err)
	}
	if err := v.Put("x", []byte("y"), ""); !errors.Is(err, ErrLocked) {
		t.Fatalf("Put: %v", err)
	}
	if err := v.PutAll([]Item{{Name: "x", Value: []byte("y")}}); !errors.Is(err, ErrLocked) {
		t.Fatalf("PutAll: %v", err)
	}
	if err := v.Delete("x"); !errors.Is(err, ErrLocked) {
		t.Fatalf("Delete: %v", err)
	}
	if _, err := v.Names(); !errors.Is(err, ErrLocked) {
		t.Fatalf("Names: %v", err)
	}
}

func TestLockDropsValues(t *testing.T) {
	v, _ := newVault(t)
	if err := v.Put("ai-key", []byte("secret"), ""); err != nil {
		t.Fatal(err)
	}
	got := mustGet(t, v, "ai-key")
	v.mu.Lock()
	held := v.entries["ai-key"].Value
	dek := v.dek
	v.mu.Unlock()
	v.Lock()
	if string(got) != "secret" {
		t.Fatalf("the returned copy changed: %q", got)
	}
	if !bytes.Equal(held, make([]byte, len(held))) || !bytes.Equal(dek, make([]byte, len(dek))) {
		t.Fatal("Lock left key material in memory")
	}
	if _, err := v.Get("ai-key"); !errors.Is(err, ErrLocked) {
		t.Fatalf("Get after Lock: %v", err)
	}
	if !v.Locked() {
		t.Fatal("not locked")
	}
}

func TestNamesAreSorted(t *testing.T) {
	v, _ := newVault(t)
	if err := v.PutAll([]Item{{Name: "vpn/b", Value: []byte("1")}, {Name: "ai-key", Value: []byte("2")}, {Name: "controller-key", Value: []byte("3")}}); err != nil {
		t.Fatal(err)
	}
	n, err := v.Names()
	if err != nil {
		t.Fatal(err)
	}
	if fmt.Sprint(n) != "[ai-key controller-key vpn/b]" {
		t.Fatalf("Names = %v", n)
	}
	if err := v.Put("bad name", []byte("x"), ""); !errors.Is(err, ErrBadName) {
		t.Fatalf("bad name: %v", err)
	}
	if err := v.PutAll([]Item{{Name: "ok", Value: []byte("1")}, {Name: "", Value: nil}}); !errors.Is(err, ErrBadName) {
		t.Fatalf("PutAll bad name: %v", err)
	}
	if _, err := v.Get("ok"); !errors.Is(err, ErrNotFound) {
		t.Fatal("a refused PutAll wrote part of its items")
	}
}

func TestOpenMissingIsErrNoVault(t *testing.T) {
	_, err := Open(filepath.Join(t.TempDir(), "vault"))
	if !errors.Is(err, ErrNoVault) || !errors.Is(err, fs.ErrNotExist) {
		t.Fatalf("err = %v", err)
	}
}

func TestOpenRefusesOversizeFile(t *testing.T) {
	p := filepath.Join(t.TempDir(), "vault")
	big := append([]byte(magic), bytes.Repeat([]byte{'x'}, MaxSize+1-len(magic))...)
	if err := os.WriteFile(p, big, 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := Open(p); !errors.Is(err, ErrCorrupt) {
		t.Fatalf("Open = %v, want ErrCorrupt", err)
	}
	v, _ := newVault(t)
	if err := v.PutAll([]Item{{Name: "big", Value: make([]byte, MaxSize)}}); !errors.Is(err, ErrTooLarge) {
		t.Fatalf("PutAll = %v, want ErrTooLarge", err)
	}
	if _, err := v.Get("big"); !errors.Is(err, ErrNotFound) {
		t.Fatal("a refused write changed memory")
	}
}

func TestOpenRejectsCorruptFiles(t *testing.T) {
	v, p := newVault(t)
	if err := v.Put("ai-key", []byte("k"), ""); err != nil {
		t.Fatal(err)
	}
	good, _ := os.ReadFile(p)
	for name, raw := range map[string][]byte{
		"empty":     {},
		"magic":     []byte("JGVAULT2\n{}\n"),
		"truncated": good[:len(good)-5],
		"bit":       append(bytes.Clone(good[:len(good)-1]), good[len(good)-1]^1),
	} {
		t.Run(name, func(t *testing.T) {
			if err := os.WriteFile(p, raw, 0o600); err != nil {
				t.Fatal(err)
			}
			w, err := Open(p)
			if err == nil {
				err = w.Unlock(pass())
				if !w.Locked() {
					t.Fatal("unlocked a corrupt vault")
				}
			}
			if !errors.Is(err, ErrCorrupt) {
				t.Fatalf("err = %v, want ErrCorrupt", err)
			}
		})
	}
}

func TestPutKeepsAnotherWritersEntry(t *testing.T) {
	a, p := newVault(t)
	b := openUnlocked(t, p)
	if err := b.Put("controller-key", []byte("x"), ""); err != nil {
		t.Fatal(err)
	}
	if err := a.Put("ai-key", []byte("y"), ""); err != nil {
		t.Fatal(err)
	}
	c := openUnlocked(t, p)
	if string(mustGet(t, c, "controller-key")) != "x" || string(mustGet(t, c, "ai-key")) != "y" {
		t.Fatal("a writer lost the other's entry")
	}
	// a's reads see b's write too.
	if string(mustGet(t, a, "controller-key")) != "x" {
		t.Fatal("a did not reload")
	}
}

func TestReloadWithAnotherDEKLocks(t *testing.T) {
	a, p := newVault(t)
	if err := a.Put("ai-key", []byte("y"), ""); err != nil {
		t.Fatal(err)
	}
	_, q := newVault(t)
	raw, err := os.ReadFile(q)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(p+".new", raw, 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.Rename(p+".new", p); err != nil {
		t.Fatal(err)
	}
	if _, err := a.Get("ai-key"); !errors.Is(err, ErrCorrupt) {
		t.Fatalf("Get = %v, want ErrCorrupt", err)
	}
	if !a.Locked() {
		t.Fatal("still unlocked")
	}
}

func TestWriteVerifiesWhatItWrote(t *testing.T) {
	v, p := newVault(t)
	if err := v.Put("ai-key", []byte("old"), ""); err != nil {
		t.Fatal(err)
	}
	before, _ := os.ReadFile(p)
	writeHook = func(b []byte) []byte { c := bytes.Clone(b); c[len(c)-1] ^= 1; return c }
	t.Cleanup(func() { writeHook = nil })
	if err := v.Put("ai-key", []byte("new"), ""); err == nil {
		t.Fatal("a corrupted write succeeded")
	}
	writeHook = nil
	if got := mustGet(t, v, "ai-key"); string(got) != "old" {
		t.Fatalf("Get = %q, want old", got)
	}
	after, _ := os.ReadFile(p)
	if !bytes.Equal(before, after) {
		t.Fatal("the file changed")
	}
	left, _ := filepath.Glob(filepath.Join(filepath.Dir(p), ".vault.tmp-*"))
	if len(left) != 0 {
		t.Fatalf("temp files left: %v", left)
	}
}

// TestConcurrentUseOfOneVault runs readers, writers and lock/unlock cycles on
// one Vault at once; -race checks the locking.
func TestConcurrentUseOfOneVault(t *testing.T) {
	v, _ := newVault(t)
	var wg sync.WaitGroup
	for i := range 4 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := range 5 {
				name := fmt.Sprintf("w%d/%d", i, j)
				if err := v.Put(name, []byte(name), ""); err != nil && !errors.Is(err, ErrLocked) {
					t.Errorf("Put: %v", err)
				}
				if _, err := v.Get(name); err != nil && !errors.Is(err, ErrLocked) && !errors.Is(err, ErrNotFound) {
					t.Errorf("Get: %v", err)
				}
				v.Names()
				v.Locked()
			}
		}()
	}
	wg.Add(1)
	go func() {
		defer wg.Done()
		for range 2 {
			v.Lock()
			if err := v.Unlock(pass()); err != nil {
				t.Errorf("Unlock: %v", err)
			}
		}
	}()
	wg.Wait()
}

// TestConcurrentWritersOnOnePath has two Vault instances write disjoint
// entries at once; every entry must survive.
func TestConcurrentWritersOnOnePath(t *testing.T) {
	a, p := newVault(t)
	b := openUnlocked(t, p)
	var wg sync.WaitGroup
	for k, v := range map[string]*Vault{"a": a, "b": b} {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := range 10 {
				if err := v.Put(fmt.Sprintf("%s-%d", k, j), []byte(k), ""); err != nil {
					t.Errorf("Put: %v", err)
				}
			}
		}()
	}
	wg.Wait()
	c := openUnlocked(t, p)
	n, err := c.Names()
	if err != nil {
		t.Fatal(err)
	}
	if len(n) != 20 {
		t.Fatalf("Names = %v, want 20 entries", n)
	}
}
