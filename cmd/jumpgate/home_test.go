package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"sync/atomic"
	"testing"

	"github.com/valve-tech/jumpgate/internal/apiclient"
)

// triedToStartServer records a test that reached the real connect seam.
var triedToStartServer atomic.Bool

// TestMain points HOME at a throwaway directory for the whole package, so no
// test can read the developer's ~/.jumpgate or write into it (config lock,
// run directory, keys), even one that forgets its own t.Setenv("HOME", ...).
//
// It also replaces connect (Ruling T2e): the real one would exec this test
// binary as a detached `serve` under whatever HOME the test left. A test that
// needs a server points connect at an in-process one; any other test that
// reaches it fails the run.
func TestMain(m *testing.M) {
	home, err := os.MkdirTemp("", "jumpgate-test-home-")
	if err != nil {
		fmt.Fprintln(os.Stderr, "test home:", err)
		os.Exit(1)
	}
	os.Setenv("HOME", home)
	os.Setenv("USERPROFILE", home) // os.UserHomeDir on Windows
	connect = func(context.Context) (*apiclient.Client, error) {
		triedToStartServer.Store(true)
		return nil, errors.New("test tried to start a jumpgate server")
	}
	code := m.Run()
	if triedToStartServer.Load() {
		fmt.Fprintln(os.Stderr, "FAIL: a test tried to start a jumpgate server; point connect at an in-process server (see withServer)")
		code = 1
	}
	os.RemoveAll(home)
	os.Exit(code)
}
