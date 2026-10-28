//go:build !linux

package main

import "errors"

// disableDumpable is Linux only, as is `serve --system`.
func disableDumpable() error {
	return errors.New("`serve --system` runs under systemd on Linux")
}
