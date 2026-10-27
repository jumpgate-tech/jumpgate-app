package secrets

import (
	"bytes"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"github.com/valve-tech/jumpgate/internal/config"
	"github.com/valve-tech/jumpgate/internal/fsatomic"
	"github.com/valve-tech/jumpgate/internal/signer"
	"github.com/valve-tech/jumpgate/internal/vault"
)

// MigrateNotice is the notice Migrate records whenever it moves or removes a
// plain secret.
const MigrateNotice = "secrets moved into the vault and their plain copies were overwritten and removed; overwriting is best effort (journaling, copy-on-write filesystems and SSDs may keep old blocks), so treat a key that was ever stored plain on a disk that may have been imaged as exposed and rotate it"

// conflictNotice names the fields (never the values) kept plain because the
// vault already holds a different value under their entry.
func conflictNotice(fields []string) string {
	return "these secrets differ from the copy the vault already holds and were left plain, so secretStore stays as it was: " +
		strings.Join(fields, ", ") + "; resolve each by hand, then run `jumpgate vault migrate` again"
}

// maxSecretFile bounds a plain key file Migrate reads.
const maxSecretFile = 64 << 10

// MigrateReport says what Migrate did. Moved and Conflicts name a config
// field path ("aiKey", "providerKeys.NAME", "vpns.ID.config") or a file
// path; Removed names the plain files overwritten and removed.
type MigrateReport struct{ Moved, Removed, Conflicts []string }

// Test seams: migrateCrash fails the migration after a step (1 after the
// vault write, 2 after the config write, 3 after a file is overwritten, 4
// after it is removed); wipeHook sees a file once it is overwritten, before
// it is removed.
var (
	migrateCrash func(step int) error
	wipeHook     func(path string)
)

func crash(step int) error {
	if migrateCrash != nil {
		return migrateCrash(step)
	}
	return nil
}

// plainItem is one plain secret found outside the vault.
type plainItem struct {
	name  string
	value []byte
	from  string // a config field path or a file path
	file  bool
}

// Migrate moves every plain secret into the unlocked vault v, in the order
// that a crash at any point can be resumed by running it again:
//
//  1. collect the plain values (one equal to the vault's copy needs no
//     write; one that differs is a conflict and stays plain);
//  2. write them all in one verified vault write;
//  3. swap config.json's fields for references in one config write, setting
//     secretStore "vault" only when nothing conflicts;
//  4. overwrite each plain key file the vault now holds with zeros, then
//     remove it.
//
// Values are never logged: notices and the report name fields and paths.
func Migrate(v *vault.Vault) (MigrateReport, error) {
	var rep MigrateReport
	if v == nil || v.Locked() {
		return rep, vault.ErrLocked
	}
	cfg, err := config.Load()
	if err != nil {
		return rep, err
	}
	dir, err := config.Dir()
	if err != nil {
		return rep, err
	}
	found, err := collect(cfg, dir)
	defer func() {
		for _, it := range found {
			clear(it.value)
		}
	}()
	if err != nil {
		return rep, err
	}

	// Step 1: sort into new, already held, and conflicting.
	var put []vault.Item
	held := map[string]bool{} // from → the vault holds this value
	for _, it := range found {
		have, err := v.Get(it.name)
		switch {
		case errors.Is(err, vault.ErrNotFound):
			put = append(put, vault.Item{Name: it.name, Value: it.value, From: it.from})
			held[it.from] = true
		case err != nil:
			return rep, err
		case bytes.Equal(have, it.value):
			held[it.from] = true
		default:
			rep.Conflicts = append(rep.Conflicts, it.from)
		}
		clear(have)
	}

	// Step 2: one verified write.
	if len(put) > 0 {
		if err := v.PutAll(put); err != nil {
			return rep, err
		}
	}
	if err := crash(1); err != nil {
		return rep, err
	}

	// Files to finish: the plain copies the vault now holds, and any left
	// by an earlier run that stopped after its config write.
	wipe, err := leftoverFiles(v, cfg, dir)
	if err != nil {
		return rep, err
	}
	for _, it := range found {
		if it.file && held[it.from] && !slices.Contains(wipe, it.from) {
			wipe = append(wipe, it.from)
		}
	}
	for _, it := range found {
		if held[it.from] {
			rep.Moved = append(rep.Moved, it.from)
		}
	}

	// Step 3: one config write, only when something changes.
	if len(rep.Moved) > 0 || len(wipe) > 0 || len(rep.Conflicts) > 0 {
		if _, err := config.Update(func(c *config.Config) error {
			swapRefs(c, found, held)
			if len(rep.Moved) > 0 || len(wipe) > 0 {
				c.AddNotice(MigrateNotice)
			}
			if len(rep.Conflicts) > 0 {
				c.AddNotice(conflictNotice(rep.Conflicts))
			} else if len(c.PlainSecrets()) == 0 && controllerNotPlain(c) {
				c.SecretStore = config.SecretStoreVault
			}
			return nil
		}); err != nil {
			return rep, err
		}
	}
	if err := crash(2); err != nil {
		return rep, err
	}

	// Step 4: overwrite, remove, sync the directory.
	for _, p := range wipe {
		removed, err := wipeFile(p)
		if err != nil {
			return rep, err
		}
		if removed {
			rep.Removed = append(rep.Removed, p)
		}
	}
	return rep, nil
}

// controllerNotPlain reports whether the controller key is not a plain file.
func controllerNotPlain(c *config.Config) bool {
	return c.Controller == nil || c.Controller.KeyStore != string(signer.StoreFile)
}

// transportFile is the plain transport key's path.
func transportFile(dir string) string { return filepath.Join(dir, "ssh", "jumpgate_ed25519") }

// collect finds every plain secret: the controller key file (the "file"
// store only; an OS store is left alone), the transport key file, and the
// config's plain secret fields.
func collect(cfg config.Config, dir string) ([]plainItem, error) {
	var out []plainItem
	if c := cfg.Controller; c != nil && c.KeyStore == string(signer.StoreFile) {
		k, err := signer.LoadKeyFile(c.KeyRef)
		if err != nil {
			return out, fmt.Errorf("controller key %s: %w", c.KeyRef, err)
		}
		kb := k.Bytes()
		out = append(out, plainItem{name: NameControllerKey, value: []byte(hex.EncodeToString(kb)), from: c.KeyRef, file: true})
		clear(kb)
	}
	tp := transportFile(dir)
	raw, err := readSecretFile(tp)
	switch {
	case err == nil && allZero(raw):
		// Overwritten by an interrupted run; leftoverFiles removes it.
	case err == nil:
		out = append(out, plainItem{name: NameTransportKey, value: raw, from: tp, file: true})
	case !errors.Is(err, os.ErrNotExist):
		return out, fmt.Errorf("transport key %s: %w", tp, err)
	}
	field := func(name, from, v string) {
		if v != "" && !config.IsSecretRef(v) {
			out = append(out, plainItem{name: name, value: []byte(v), from: from})
		}
	}
	field(NameAIKey, "aiKey", cfg.AIKey)
	for _, ph := range sortedKeys(cfg.ProviderKeys) {
		field(ProviderName(ph), "providerKeys."+ph, cfg.ProviderKeys[ph])
	}
	for _, vp := range cfg.VPNs {
		field(VPNName(vp.ID), "vpns."+vp.ID+".config", vp.Config)
	}
	return out, nil
}

func sortedKeys(m map[string]string) []string {
	k := make([]string, 0, len(m))
	for n := range m {
		k = append(k, n)
	}
	slices.Sort(k)
	return k
}

// swapRefs replaces each field the vault now holds with its reference. A
// field changed since it was collected is left for the next run.
func swapRefs(c *config.Config, found []plainItem, held map[string]bool) {
	for _, it := range found {
		if !held[it.from] {
			continue
		}
		ref := config.SecretRef(it.name)
		switch {
		case it.name == NameControllerKey:
			if c.Controller != nil && c.Controller.KeyStore == string(signer.StoreFile) && c.Controller.KeyRef == it.from {
				c.Controller.KeyStore, c.Controller.KeyRef = string(signer.StoreVault), NameControllerKey
			}
		case it.file:
		case it.from == "aiKey":
			if c.AIKey == string(it.value) {
				c.AIKey = ref
			}
		case strings.HasPrefix(it.from, "providerKeys."):
			ph := strings.TrimPrefix(it.from, "providerKeys.")
			if c.ProviderKeys[ph] == string(it.value) {
				c.ProviderKeys[ph] = ref
			}
		default:
			for i := range c.VPNs {
				if "vpns."+c.VPNs[i].ID+".config" == it.from && c.VPNs[i].Config == string(it.value) {
					c.VPNs[i].Config = ref
				}
			}
		}
	}
}

// leftoverFiles are plain key files an earlier run moved into the vault and
// did not finish removing: the vault entry's origin path, once config.json
// no longer points at it.
func leftoverFiles(v *vault.Vault, cfg config.Config, dir string) ([]string, error) {
	var out []string
	for _, name := range []string{NameControllerKey, NameTransportKey} {
		e, err := v.Entry(name)
		if errors.Is(err, vault.ErrNotFound) {
			continue
		}
		if err != nil {
			return nil, err
		}
		clear(e.Value)
		p := e.From
		if !filepath.IsAbs(p) {
			continue
		}
		if name == NameTransportKey && p != transportFile(dir) {
			continue
		}
		if name == NameControllerKey && cfg.Controller != nil && cfg.Controller.KeyStore == string(signer.StoreFile) {
			continue // still in use; collect decides
		}
		ok, err := leftover(name, p, v)
		if err != nil {
			return nil, err
		}
		if ok {
			out = append(out, p)
		}
	}
	return out, nil
}

// leftover reports whether the file at p is the vault's copy of entry name,
// or a copy an earlier run already overwritten with zeros. Any other content
// is never wiped.
func leftover(name, p string, v *vault.Vault) (bool, error) {
	raw, err := readSecretFile(p)
	if errors.Is(err, os.ErrNotExist) {
		return false, nil
	}
	if err != nil {
		return false, fmt.Errorf("%s: %w", p, err)
	}
	defer clear(raw)
	if allZero(raw) {
		return true, nil
	}
	want, err := v.Get(name)
	if err != nil {
		return false, err
	}
	defer clear(want)
	if name == NameTransportKey {
		return bytes.Equal(raw, want), nil
	}
	k, err := signer.LoadKeyFile(p)
	if err != nil {
		return false, nil
	}
	kb := k.Bytes()
	defer clear(kb)
	got := []byte(hex.EncodeToString(kb))
	defer clear(got)
	return bytes.Equal(got, want), nil
}

// allZero is a file an interrupted run overwrote but did not remove.
func allZero(b []byte) bool {
	if len(b) == 0 {
		return false
	}
	for _, c := range b {
		if c != 0 {
			return false
		}
	}
	return true
}

// readSecretFile reads a plain key file without following a link.
func readSecretFile(path string) ([]byte, error) {
	f, err := openNoFollow(path, os.O_RDONLY)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	fi, err := f.Stat()
	if err != nil {
		return nil, err
	}
	if !fi.Mode().IsRegular() {
		return nil, fmt.Errorf("%s is not a regular file", path)
	}
	raw, err := io.ReadAll(io.LimitReader(f, maxSecretFile+1))
	if err != nil {
		clear(raw)
		return nil, err
	}
	if len(raw) > maxSecretFile {
		clear(raw)
		return nil, fmt.Errorf("%s is too large to be a key file", path)
	}
	return raw, nil
}

// wipeFile overwrites path with zeros over its whole length, syncs it,
// removes it and syncs the directory. It opens the file O_WRONLY without
// following a link, so a link swapped in is refused, never its target
// overwritten. A path already gone is not an error (removed is false).
func wipeFile(path string) (removed bool, err error) {
	f, err := openNoFollow(path, os.O_WRONLY)
	if errors.Is(err, os.ErrNotExist) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	fi, err := f.Stat()
	if err == nil && !fi.Mode().IsRegular() {
		err = fmt.Errorf("%s is not a regular file", path)
	}
	if err == nil {
		_, err = f.WriteAt(make([]byte, fi.Size()), 0)
	}
	if err == nil {
		err = f.Sync()
	}
	if cerr := f.Close(); err == nil {
		err = cerr
	}
	if err != nil {
		return false, fmt.Errorf("overwrite %s: %w", path, err)
	}
	if wipeHook != nil {
		wipeHook(path)
	}
	if err := crash(3); err != nil {
		return false, err
	}
	if err := os.Remove(path); err != nil {
		return false, err
	}
	if err := crash(4); err != nil {
		return true, err
	}
	return true, fsatomic.SyncDir(filepath.Dir(path))
}
