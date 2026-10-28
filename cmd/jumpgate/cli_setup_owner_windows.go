package main

import (
	"errors"
	"os"
)

// lstatOwner is unsupported on Windows; install-service is Linux only.
func lstatOwner(string) (int, os.FileMode, error) {
	return 0, 0, errors.New("file ownership is not available on windows")
}
