//go:build !unix

package signer

import "os/exec"

// isolateTool is a no-op where there are no process groups to kill; the
// context still kills the tool itself.
func isolateTool(*exec.Cmd, string) {}
