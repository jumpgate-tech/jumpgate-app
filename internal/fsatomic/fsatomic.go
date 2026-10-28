// Package fsatomic writes a file so that a crash leaves either the old
// contents or the new, never a torn mix, and the new name is durable once
// WriteFile returns. The agent's policy and replay record and the jobs store
// all persist through it.
package fsatomic

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
)

// WriteFile writes b to path (0600) through a fsynced temp file and a rename,
// then fsyncs the directory so the rename itself survives a power cut.
func WriteFile(path string, b []byte) error {
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return err
	}
	f, err := os.CreateTemp(dir, filepath.Base(path)+".tmp-*")
	if err != nil {
		return err
	}
	tmp := f.Name()
	defer os.Remove(tmp)
	if err := f.Chmod(0o600); err != nil {
		f.Close()
		return err
	}
	if _, err := f.Write(b); err != nil {
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
	if err := os.Rename(tmp, path); err != nil {
		return err
	}
	return SyncDir(dir)
}

// SyncDir fsyncs a directory. It is a variable so a test can make the
// directory fsync fail. On Windows the directory is opened but not flushed:
// a read-only directory handle cannot be flushed (Access is denied), and
// NTFS journals the rename.
var SyncDir = func(dir string) error {
	d, err := os.Open(dir)
	if err != nil {
		return fmt.Errorf("fsatomic: open %s to sync it: %w", dir, err)
	}
	if runtime.GOOS == "windows" {
		return d.Close()
	}
	if err := d.Sync(); err != nil {
		d.Close()
		return fmt.Errorf("fsatomic: sync directory %s: %w", dir, err)
	}
	if err := d.Close(); err != nil {
		return fmt.Errorf("fsatomic: close directory %s: %w", dir, err)
	}
	return nil
}
