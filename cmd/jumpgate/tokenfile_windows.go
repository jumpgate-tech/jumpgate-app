//go:build windows

package main

import (
	"fmt"
	"io"
	"os"

	"golang.org/x/sys/windows"

	"github.com/valve-tech/jumpgate/internal/fsperm"
)

// readPrivateFile reads a credential file only if it is private to this user.
// It opens path without following a reparse point
// (FILE_FLAG_OPEN_REPARSE_POINT) and refuses the handle if it is one, so a
// symlink or junction is refused rather than followed. It then checks the
// OPENED handle: it must be a regular file whose owner and DACL
// fsperm.CheckPrivateFile accepts (no other user can read, change or delete
// it), so the file checked is the file read.
func readPrivateFile(path string) ([]byte, error) {
	fix := fmt.Sprintf("make it a regular file only you can read (`icacls %s /inheritance:r /grant:r %%USERNAME%%:F`)", path)
	p, err := windows.UTF16PtrFromString(path)
	if err != nil {
		return nil, &os.PathError{Op: "open", Path: path, Err: err}
	}
	h, err := windows.CreateFile(p, windows.GENERIC_READ|windows.READ_CONTROL, windows.FILE_SHARE_READ, nil,
		windows.OPEN_EXISTING, windows.FILE_FLAG_OPEN_REPARSE_POINT|windows.FILE_FLAG_BACKUP_SEMANTICS, 0)
	if err != nil {
		return nil, &os.PathError{Op: "open", Path: path, Err: err}
	}
	var info windows.ByHandleFileInformation
	if err := windows.GetFileInformationByHandle(h, &info); err != nil {
		windows.CloseHandle(h)
		return nil, &os.PathError{Op: "stat", Path: path, Err: err}
	}
	if info.FileAttributes&windows.FILE_ATTRIBUTE_REPARSE_POINT != 0 {
		windows.CloseHandle(h)
		return nil, fmt.Errorf("%s is a link; a token file must be the file itself: %s", path, fix)
	}
	f := os.NewFile(uintptr(h), path)
	defer f.Close()

	fi, err := f.Stat()
	if err != nil {
		return nil, err
	}
	if !fi.Mode().IsRegular() {
		return nil, fmt.Errorf("%s is not a regular file (%s): %s", path, fi.Mode().Type(), fix)
	}
	if err := fsperm.CheckPrivateFile(f); err != nil {
		return nil, fmt.Errorf("%s is not private to you (%v): %s", path, err, fix)
	}
	// The token is one short line; bound the read anyway.
	return io.ReadAll(io.LimitReader(f, 64<<10))
}
