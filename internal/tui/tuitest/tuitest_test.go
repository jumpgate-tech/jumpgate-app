package tuitest

import (
	"context"
	"testing"
	"time"

	tea "charm.land/bubbletea/v2"

	"github.com/valve-tech/jumpgate/internal/api"
	"github.com/valve-tech/jumpgate/internal/apiclient"
)

func TestKeyNames(t *testing.T) {
	for _, n := range []string{"enter", "esc", "tab", "shift+tab", "up", "down", "backspace", "space", "ctrl+c", "q", "/", "?", "1"} {
		if got := Key(n).String(); got != n {
			t.Errorf("Key(%q).String() = %q", n, got)
		}
	}
}

func TestRunExpandsBatchesAndDropsBlockedCommands(t *testing.T) {
	block := make(chan struct{})
	defer close(block)
	msgs := Run(tea.Batch(
		func() tea.Msg { return "a" },
		func() tea.Msg { <-block; return "never" },
		func() tea.Msg { return "b" },
	))
	if len(msgs) != 2 {
		t.Fatalf("msgs %v", msgs)
	}
}

func TestNormalizeTrimsTrailingSpacesAndCRLF(t *testing.T) {
	if got := normalize("a  \r\nb\t \n  c"); got != "a\nb\t\n  c" {
		t.Errorf("normalize = %q", got)
	}
}

// The fake's streams behave like the client's: they deliver what the test
// sends and close when the context ends, so no goroutine outlives a test.
func TestFakeStreamsCloseWithTheContext(t *testing.T) {
	f := NewFake()
	ctx, cancel := context.WithCancel(context.Background())
	ch := f.WatchFleet(ctx)
	f.FleetCh <- apiclient.Update[api.Fleet]{State: apiclient.Live}
	if u := <-ch; u.State != apiclient.Live {
		t.Fatalf("update %+v", u)
	}
	cancel()
	select {
	case _, ok := <-ch:
		if ok {
			t.Fatal("an update after cancel")
		}
	case <-time.After(2 * time.Second):
		t.Fatal("the stream did not close when the context ended")
	}
	if !f.Called("WatchFleet") {
		t.Fatal("WatchFleet was not recorded")
	}

	logs := f.WatchLogs(context.Background(), "box", 10)
	close(f.Logs("box"))
	select {
	case _, ok := <-logs:
		if ok {
			t.Fatal("an update from a closed source")
		}
	case <-time.After(2 * time.Second):
		t.Fatal("the stream did not close with its source")
	}
}
