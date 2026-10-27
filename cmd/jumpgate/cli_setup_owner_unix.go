//go:build !windows

package main

import (
	"errors"
	"os"
	"syscall"
)

// lstatOwner is p's owner uid and mode, without following a symlink.
func lstatOwner(p string) (int, os.FileMode, error) {
	fi, err := os.Lstat(p)
	if err != nil {
		return 0, 0, err
	}
	st, ok := fi.Sys().(*syscall.Stat_t)
	if !ok {
		return 0, 0, errors.New("no owner information")
	}
	return int(st.Uid), fi.Mode(), nil
}
