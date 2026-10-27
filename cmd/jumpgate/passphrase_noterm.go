//go:build notui

package main

import "errors"

// The gate build (-tags notui) links no terminal library, so it cannot turn
// echo off; a passphrase comes only from --stdin.
var errNoTermPrompt = errors.New("this build cannot prompt for a passphrase")

func termIsTerminal(uintptr) bool { return false }

func termReadPassword(uintptr) ([]byte, error) { return nil, errNoTermPrompt }
