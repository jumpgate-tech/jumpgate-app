package main

import (
	"net/http"
	"strings"
	"testing"

	"github.com/valve-tech/jumpgate/internal/buildinfo"
	"github.com/valve-tech/jumpgate/internal/daemon"
)

// A route this CLI needs that an older server lacks answers 404. From a server
// of another version that is the wire contract changing, and the operator is
// told to restart it rather than shown "404 Not Found".
func TestServerErrorFromAnotherVersionSaysRestart(t *testing.T) {
	var w strings.Builder
	old := daemon.Info{Version: "v0.0.1-old"}
	code := reportServerErrorFrom(&w, "box", old, apiError{Status: http.StatusNotFound, Error: "404 page not found"})
	if code != exitCode("failed") {
		t.Fatalf("exit %d", code)
	}
	if !strings.Contains(w.String(), "v0.0.1-old") || !strings.Contains(w.String(), "jumpgate stop") {
		t.Fatalf("output %q does not name the version and the way out", w.String())
	}

	// The same version answering 404 is an ordinary error.
	w.Reset()
	same := daemon.Info{Version: buildinfo.Version()}
	reportServerErrorFrom(&w, "box", same, apiError{Status: http.StatusNotFound, Error: "no such target"})
	if strings.Contains(w.String(), "jumpgate stop") {
		t.Fatalf("same-version 404 told to restart: %q", w.String())
	}
}
