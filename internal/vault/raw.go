package vault

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"

	"github.com/valve-tech/jumpgate/internal/fsatomic"
	"github.com/valve-tech/jumpgate/internal/fsperm"
)

// LockPath takes the exclusive file lock (<path>.lock) that every vault
// write holds, for a caller that copies the file whole.
func LockPath(path string) (unlock func(), err error) { return lockFile(path + ".lock") }

// ReadFile returns the bytes of the vault file at path after the checks Open
// makes: no link, a private regular file of this user, at most MaxSize, a
// header this build reads. It decrypts nothing.
func ReadFile(path string) ([]byte, error) {
	raw, _, err := readFile(path)
	if err != nil {
		return nil, err
	}
	if _, _, _, err := parse(raw); err != nil {
		return nil, err
	}
	return raw, nil
}

// Install puts raw, a whole vault file, at path under the file lock. raw must
// parse as a vault. An existing vault is replaced only with replace; root
// never replaces a vault another user owns.
func Install(path string, raw []byte, replace bool) error {
	if _, _, _, err := parse(raw); err != nil {
		return err
	}
	unlock, err := lockFile(path + ".lock")
	if err != nil {
		return err
	}
	defer unlock()
	fi, err := os.Lstat(path)
	switch {
	case err == nil && !replace:
		return fmt.Errorf("%w at %s", ErrExists, path)
	case err == nil:
		if err := checkWriteOwner(path, fi); err != nil {
			return err
		}
	case !errors.Is(err, fs.ErrNotExist):
		return fmt.Errorf("vault: %w", err)
	}
	if err := fsperm.WriteFilePrivate(path, raw); err != nil {
		return fmt.Errorf("vault: %w", err)
	}
	return fsatomic.SyncDir(filepath.Dir(path))
}

// Check reports parameters outside the floor and ceiling, before any
// passphrase is asked for.
func (p Params) Check() error {
	_, err := p.kdf()
	return err
}

// CheckWrite reports, before any work, whether this process may write the
// vault: root never writes a vault another user owns.
func (v *Vault) CheckWrite() error {
	v.mu.Lock()
	defer v.mu.Unlock()
	if v.seen.fi == nil {
		return nil
	}
	return checkWriteOwner(v.path, v.seen.fi)
}
