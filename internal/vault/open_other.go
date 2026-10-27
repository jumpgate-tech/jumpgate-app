//go:build !unix && !windows

package vault

import (
	"fmt"
	"os"
)

func openNoFollow(path string) (*os.File, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("vault: %w", err)
	}
	return f, nil
}

func checkOwner(string, os.FileInfo) error { return nil }
