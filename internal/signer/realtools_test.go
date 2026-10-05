package signer

import (
	"errors"
	"testing"
)

// Fixtures captured from the real tools on macOS (op 2.34.0, security).
func TestRealToolNotFoundOutputIsRecognised(t *testing.T) {
	sec := &cmdError{Name: "security", ExitCode: 44, Stderr: "security: SecKeychainSearchCopyNext: The specified item could not be found in the keychain.\n", Err: errors.New("exit status 44")}
	if !keychainNotFound("security", sec) {
		t.Fatal("real `security` not-found was not recognised")
	}
	secI := &cmdError{Name: "security", ExitCode: 44, Stderr: "security: SecKeychainSearchCopyNext: The specified item could not be found in the keychain.\nfind-generic-password: returned -25300\n", Err: errors.New("exit status 44")}
	if !keychainNotFound("security", secI) {
		t.Fatal("real `security -i` not-found was not recognised")
	}
	const opMsg = `[ERROR] 2026/10/05 11:36:52 "jumpgate-verify-1" isn't an item in the "Private" vault. Specify the item with its UUID, name, or domain.`
	if !opItemAbsent(&cmdError{Name: "op", ExitCode: 1, Stderr: opMsg, Err: errors.New("exit status 1")}) {
		t.Fatal("real `op item get` not-found was not recognised")
	}
}

// Fixtures captured from the real secret-tool 0.20.5 (Debian 12, gnome-keyring).
func TestRealSecretToolOutputsAreClassified(t *testing.T) {
	missing := &cmdError{Name: "secret-tool", ExitCode: 1, Err: errors.New("exit status 1")}
	if !keychainNotFound("secret-tool", missing) {
		t.Fatal("real `secret-tool lookup` of a missing item was not recognised")
	}
	headless := &cmdError{Name: "secret-tool", ExitCode: 1, Stderr: "secret-tool: Cannot autolaunch D-Bus without X11 $DISPLAY\n", Err: errors.New("exit status 1")}
	if keychainNotFound("secret-tool", headless) {
		t.Fatal("a missing D-Bus session was mistaken for an absent item")
	}
	locked := &cmdError{Name: "secret-tool", ExitCode: 1, Stderr: "secret-tool: Cannot create an item in a locked collection\n", Err: errors.New("exit status 1")}
	if keychainNotFound("secret-tool", locked) {
		t.Fatal("a locked-collection store failure was mistaken for an absent item")
	}
}
