package signer

import (
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"strings"
)

// ErrKeyFilePermissions refuses a key file other users can read: a signing
// key anyone on the machine can copy authorises nothing.
var ErrKeyFilePermissions = errors.New("signer: key file is readable by other users; chmod 600 it")

// maxKeyFileSize bounds what LoadKeyFile reads: a hex key is 65 bytes.
const maxKeyFileSize = 1024

// LoadKeyFile reads a hex key written by GenerateKeyFile. It opens the path
// once, refuses symlinks, and checks the open handle, so the file it checks is
// the file it reads.
func LoadKeyFile(path string) (*Key, error) {
	f, err := os.OpenFile(path, os.O_RDONLY|openNoFollow, 0)
	if err != nil {
		return nil, fmt.Errorf("signer: %w", err)
	}
	defer f.Close()
	fi, err := f.Stat()
	if err != nil {
		return nil, fmt.Errorf("signer: %w", err)
	}
	if !fi.Mode().IsRegular() {
		return nil, fmt.Errorf("signer: %s is not a regular file", path)
	}
	if runtime.GOOS != "windows" && fi.Mode().Perm()&0o077 != 0 {
		return nil, fmt.Errorf("%w (%s is %o)", ErrKeyFilePermissions, path, fi.Mode().Perm())
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

// GenerateKeyFile creates a new key at path, 0600, refusing to overwrite.
func GenerateKeyFile(path string) (*Key, error) {
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return nil, fmt.Errorf("signer: %w", err)
	}
	k, err := GenerateKey()
	if err != nil {
		return nil, err
	}
	f, err := os.OpenFile(path, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0o600)
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
