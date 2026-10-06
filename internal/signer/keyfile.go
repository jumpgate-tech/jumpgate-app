package signer

import (
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"github.com/valve-tech/jumpgate/internal/fsperm"
)

// ErrKeyFilePermissions refuses a key file other users can read: a signing
// key anyone on the machine can copy authorises nothing.
var ErrKeyFilePermissions = errors.New("signer: key file is readable by other users; restrict it to your user (chmod 600 on macOS and Linux)")

// maxKeyFileSize bounds what LoadKeyFile reads: a hex key is 65 bytes.
const maxKeyFileSize = 1024

// LoadKeyFile reads a hex key written by GenerateKeyFile. It opens the path
// once, refuses symlinks and Windows reparse points, and checks the open
// handle, so the file it checks is the file it reads.
func LoadKeyFile(path string) (*Key, error) {
	// openKeyFile refuses a link or reparse point as it opens (M-4), so
	// nothing can be swapped in between a check and the open.
	f, err := openKeyFile(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	fi, err := f.Stat()
	if err != nil {
		return nil, fmt.Errorf("signer: %w", err)
	}
	if !fi.Mode().IsRegular() {
		return nil, fmt.Errorf("signer: %s is not a regular file", path)
	}
	if err := fsperm.CheckPrivateFile(f); err != nil {
		return nil, fmt.Errorf("%w (%v)", ErrKeyFilePermissions, err)
	}
	raw, err := io.ReadAll(io.LimitReader(f, maxKeyFileSize))
	if err != nil {
		return nil, fmt.Errorf("signer: %w", err)
	}
	b, err := hex.DecodeString(strings.TrimSpace(strings.TrimPrefix(strings.TrimSpace(string(raw)), "0x")))
	if err != nil {
		return nil, fmt.Errorf("signer: %s is not a hex key", path)
	}
	return KeyFromBytes(b)
}

// GenerateKeyFile creates a new owner-only key at path, refusing to
// overwrite. A missing parent directory is created owner-only; an existing
// one the operator chose (their home, say) is left as it is.
func GenerateKeyFile(path string) (*Key, error) {
	dir := filepath.Dir(path)
	if _, err := os.Stat(dir); errors.Is(err, fs.ErrNotExist) {
		if err := fsperm.MkdirPrivate(dir); err != nil {
			return nil, fmt.Errorf("signer: %w", err)
		}
	}
	k, err := GenerateKey()
	if err != nil {
		return nil, err
	}
	// Private from creation, never replacing anything already at path: the
	// directory may be one the operator chose and other users can read.
	f, err := fsperm.CreatePrivate(path)
	if err != nil {
		return nil, fmt.Errorf("signer: %w", err)
	}
	if _, err := f.WriteString(hex.EncodeToString(k.Bytes()) + "\n"); err != nil {
		f.Close()
		os.Remove(path)
		return nil, fmt.Errorf("signer: %w", err)
	}
	return k, f.Close()
}
