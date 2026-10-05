//go:build !windows

package signer

import "errors"

var errWinCredUnsupported = errors.New("signer: Windows Credential Manager exists only on Windows; use --store keychain or --store file")

type noCreds struct{}

func (noCreds) read(string) ([]byte, error) { return nil, errWinCredUnsupported }
func (noCreds) write(string, []byte) error  { return errWinCredUnsupported }
func (noCreds) del(string) error            { return errWinCredUnsupported }

func platformCreds() credStore { return noCreds{} }
