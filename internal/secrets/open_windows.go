//go:build windows

package secrets

import (
	"fmt"
	"os"
)

// openNoFollow opens path with flag, refusing a link or reparse point at the
// final path element.
func openNoFollow(path string, flag int) (*os.File, error) {
	fi, err := os.Lstat(path)
	if err != nil {
		return nil, err
	}
	if fi.Mode()&(os.ModeSymlink|os.ModeIrregular) != 0 {
		return nil, fmt.Errorf("%s is a link, not a secret file", path)
	}
	return os.OpenFile(path, flag, 0)
}
