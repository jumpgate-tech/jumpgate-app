//go:build !notui

package main

import "github.com/charmbracelet/x/term"

// Controller builds read a passphrase from the terminal with echo off. The
// gate build (-tags notui) links no charmbracelet code.
var (
	termIsTerminal   = term.IsTerminal
	termReadPassword = term.ReadPassword
)
