package main

import (
	"context"
	"errors"
	"io"
	"path/filepath"
	"strings"
	"testing"

	"github.com/valve-tech/jumpgate/internal/config"
	"github.com/valve-tech/jumpgate/internal/signer"
)

const otherAddress = "0x000000000000000000000000000000000000dEaD"

// fileController creates a file-store key in the test HOME and returns its
// ref and real address.
func fileController(t *testing.T) (ref, addr string) {
	t.Helper()
	ref = filepath.Join(t.TempDir(), "controller.key")
	k, err := signer.Create(context.Background(), signer.StoreFile, ref)
	if err != nil {
		t.Fatal(err)
	}
	return ref, k.Address().Hex()
}

// A1: the opened key must be the identity config.json recorded. A swapped
// keychain or 1Password item would otherwise sign as someone else, and every
// box would answer unauthorized_kind with the misleading "pair it" remedy.
func TestOpenControllerKeyRefusesAKeyWithAnotherAddress(t *testing.T) {
	shortHome(t)
	ref, _ := fileController(t)
	_, err := openControllerKey(config.Config{Controller: &config.Controller{KeyStore: "file", KeyRef: ref, Address: otherAddress}})
	if !errors.Is(err, signer.ErrAddressMismatch) {
		t.Fatalf("err = %v, want ErrAddressMismatch", err)
	}
	if !strings.Contains(err.Error(), otherAddress) {
		t.Fatalf("error %q does not name the recorded address", err)
	}
}

func TestOpenControllerKeyAcceptsTheRecordedAddressInAnyCase(t *testing.T) {
	shortHome(t)
	ref, addr := fileController(t)
	k, err := openControllerKey(config.Config{Controller: &config.Controller{KeyStore: "file", KeyRef: ref, Address: strings.ToLower(addr)}})
	if err != nil || k == nil {
		t.Fatalf("k = %v, err = %v", k, err)
	}
}

// Both entry points tolerate the mismatch the same way: the server comes up
// with no signer and the mismatch as the reason.
func TestBuildServerReportsAKeyMismatch(t *testing.T) {
	shortHome(t)
	ref, _ := fileController(t)
	writeControllerConfig(t, "file", ref, otherAddress)
	b, err := buildServer(serverOptions{Bind: freeAddr(t), ERPCURL: "http://127.0.0.1:4000"}, func() {}, io.Discard)
	if err != nil {
		t.Fatal(err)
	}
	if !errors.Is(b.signerErr, signer.ErrAddressMismatch) {
		t.Fatalf("signerErr = %v", b.signerErr)
	}
}

// `keys show` prints the key's real address, not just the recorded one.
func TestKeysShowPrintsTheRealAddress(t *testing.T) {
	shortHome(t)
	ref, addr := fileController(t)
	writeControllerConfig(t, "file", ref, strings.ToLower(addr))
	var out, errOut strings.Builder
	if code := keysShow(&out, &errOut); code != 0 {
		t.Fatalf("exit %d: %s", code, errOut.String())
	}
	if !strings.Contains(out.String(), addr) {
		t.Fatalf("stdout %q lacks the real address %s", out.String(), addr)
	}
}

// ...and flags a mismatch with the recorded one as a security failure.
func TestKeysShowFlagsAMismatch(t *testing.T) {
	shortHome(t)
	ref, addr := fileController(t)
	writeControllerConfig(t, "file", ref, otherAddress)
	var out, errOut strings.Builder
	code := keysShow(&out, &errOut)
	if code != exitCode("controller_key_mismatch") || code != 4 {
		t.Fatalf("exit %d, want 4", code)
	}
	if !strings.Contains(out.String(), addr) {
		t.Fatalf("stdout %q lacks the real address", out.String())
	}
	e := errOut.String()
	for _, want := range []string{"SECURITY", otherAddress, addr, "->"} {
		if !strings.Contains(e, want) {
			t.Errorf("stderr lacks %q:\n%s", want, e)
		}
	}
}

func TestServerKeyMismatchExitsAsSecurity(t *testing.T) {
	var w strings.Builder
	code := reportServerError(&w, "box", apiError{Status: 503, Error: "mismatch", Code: "controller_key_mismatch"})
	if code != 4 || !strings.Contains(w.String(), "SECURITY") || !strings.Contains(w.String(), "->") {
		t.Fatalf("exit %d, output %q", code, w.String())
	}
}
