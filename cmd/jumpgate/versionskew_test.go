package main

import (
	"net/http"
	"strings"
	"testing"

	"github.com/valve-tech/jumpgate/internal/api"
	"github.com/valve-tech/jumpgate/internal/buildinfo"
	"github.com/valve-tech/jumpgate/internal/daemon"
)

// A route this CLI needs that an older server lacks answers 404. From a server
// of another version that is the wire contract changing, and the operator is
// told to restart it rather than shown "404 Not Found".
func TestServerErrorFromAnotherVersionSaysRestart(t *testing.T) {
	old := daemon.Info{Version: "v0.0.1-old"}
	// A bare 404 decodes to not_found since the error contract (Ruling T2b);
	// an older server's error JSON may carry no code at all.
	for _, code := range []api.Code{"", api.CodeNotFound, api.Decode(http.StatusNotFound, []byte("404 page not found\n")).Code} {
		var w strings.Builder
		got := reportServerErrorFrom(&w, "box", old, apiError{Status: http.StatusNotFound, Code: code, Message: "404 page not found"})
		if got != exitCode("failed") {
			t.Fatalf("code %q: exit %d", code, got)
		}
		if !strings.Contains(w.String(), "v0.0.1-old") || !strings.Contains(w.String(), "jumpgate stop") {
			t.Fatalf("code %q: output %q does not name the version and the way out", code, w.String())
		}
	}

	// The same version answering 404 is an ordinary error.
	var w strings.Builder
	same := daemon.Info{Version: buildinfo.Version()}
	reportServerErrorFrom(&w, "box", same, apiError{Status: http.StatusNotFound, Code: api.CodeNotFound, Message: "no such target"})
	if strings.Contains(w.String(), "jumpgate stop") {
		t.Fatalf("same-version 404 told to restart: %q", w.String())
	}

	// A specific code is an answer from a server that knew the route.
	w.Reset()
	reportServerErrorFrom(&w, "box", old, apiError{Status: http.StatusNotFound, Code: api.CodeTargetNotFound, Message: "target not found"})
	if strings.Contains(w.String(), "does not know this request") {
		t.Fatalf("target_not_found read as a skew: %q", w.String())
	}
}
