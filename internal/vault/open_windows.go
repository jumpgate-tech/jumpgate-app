//go:build windows

package vault

import (
	"errors"
	"fmt"
	"os"

	"golang.org/x/sys/windows"
)

// openNoFollow opens path read-only without following a reparse point, then
// refuses the handle if it is one: a symlink or junction is not a vault file.
func openNoFollow(path string) (*os.File, error) {
	p, err := windows.UTF16PtrFromString(path)
	if err != nil {
		return nil, fmt.Errorf("vault: %w", err)
	}
	h, err := windows.CreateFile(p, windows.GENERIC_READ, windows.FILE_SHARE_READ, nil,
		windows.OPEN_EXISTING, windows.FILE_FLAG_OPEN_REPARSE_POINT, 0)
	if errors.Is(err, windows.ERROR_FILE_NOT_FOUND) || errors.Is(err, windows.ERROR_PATH_NOT_FOUND) {
		return nil, fmt.Errorf("vault: open %s: %w", path, os.ErrNotExist)
	}
	if err != nil {
		return nil, fmt.Errorf("vault: open %s: %w", path, err)
	}
	var info windows.ByHandleFileInformation
	if err := windows.GetFileInformationByHandle(h, &info); err != nil {
		windows.CloseHandle(h)
		return nil, fmt.Errorf("vault: %s: %w", path, err)
	}
	if info.FileAttributes&windows.FILE_ATTRIBUTE_REPARSE_POINT != 0 {
		windows.CloseHandle(h)
		return nil, fmt.Errorf("%w: %s is a link, not a vault file", ErrUnsafeFile, path)
	}
	return os.NewFile(uintptr(h), path), nil
}

// checkOwner is a no-op: fsperm.CheckPrivateFile already checks the owner
// and the DACL of the handle.
func checkOwner(string, os.FileInfo) error { return nil }
