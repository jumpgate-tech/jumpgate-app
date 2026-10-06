//go:build !unix && !windows

package signer

import (
	"fmt"
	"os"
)

func openKeyFile(path string) (*os.File, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("signer: %w", err)
	}
	return f, nil
}
