//go:build windows

package signer

import (
	"fmt"
	"os"

	"golang.org/x/sys/windows"
)

// openKeyFile opens path read-only without following a reparse point
// (FILE_FLAG_OPEN_REPARSE_POINT), then refuses the handle if it is one: a
// symlink or junction is a link, not a key file. Windows has no O_NOFOLLOW,
// and checking with Lstat first would leave a window to swap one in.
func openKeyFile(path string) (*os.File, error) {
	p, err := windows.UTF16PtrFromString(path)
	if err != nil {
		return nil, fmt.Errorf("signer: %w", err)
	}
	h, err := windows.CreateFile(p, windows.GENERIC_READ, windows.FILE_SHARE_READ, nil,
		windows.OPEN_EXISTING, windows.FILE_FLAG_OPEN_REPARSE_POINT, 0)
	if err != nil {
		return nil, fmt.Errorf("signer: open %s: %w", path, err)
	}
	var info windows.ByHandleFileInformation
	if err := windows.GetFileInformationByHandle(h, &info); err != nil {
		windows.CloseHandle(h)
		return nil, fmt.Errorf("signer: %s: %w", path, err)
	}
	if info.FileAttributes&windows.FILE_ATTRIBUTE_REPARSE_POINT != 0 {
		windows.CloseHandle(h)
		return nil, fmt.Errorf("signer: %s is a link, not a key file", path)
	}
	return os.NewFile(uintptr(h), path), nil
}
