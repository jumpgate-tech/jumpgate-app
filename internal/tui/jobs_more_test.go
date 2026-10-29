package tui

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/valve-tech/jumpgate/internal/api"
	"github.com/valve-tech/jumpgate/internal/apiclient"
	"github.com/valve-tech/jumpgate/internal/tui/tuitest"
)

// The open job follows its journal live; closing it stops the stream.
func TestJobDetailStreamsOutput(t *testing.T) {
	f := tuitest.NewFake()
	f.Output("box-a/0123456789abcdef") <- apiclient.Update[[]api.JobOutputLine]{Has: true, State: apiclient.Live,
		Value: []api.JobOutputLine{{Unit: "jg-job", Line: "stopping units\x1b[2J"}, {Unit: "jg-job", Line: "renamed account"}}}
	m := press(t, jobsApp(t, f, "unicode"), "enter")
	if !f.Called("WatchJobOutput box-a 0123456789abcdef") {
		t.Fatalf("calls %v", f.Calls)
	}
	fr := tuitest.Frame(m)
	for _, want := range []string{"output", "jg-job stopping units", "jg-job renamed account"} {
		if !strings.Contains(fr, want) {
			t.Errorf("detail lacks %q:\n%s", want, fr)
		}
	}
	if strings.Contains(fr, "\x1b[2J") {
		t.Fatal("output escapes reach the screen")
	}
	m = press(t, m, "enter")
	if fr := tuitest.Frame(m); strings.Contains(fr, "renamed account") {
		t.Fatalf("closed job still shows output:\n%s", fr)
	}
}

func TestStartJobNeedsTheBoxTyped(t *testing.T) {
	f := tuitest.NewFake()
	f.Routes["POST /api/jobs/box-a"] = api.JobView{Target: "box-a", ID: "abc", State: "running"}
	m := press(t, jobsApp(t, f, "unicode"), "n")
	m = typed(t, m, "rm-rf")
	m = press(t, m, "enter")
	if fr := tuitest.Frame(m); !strings.Contains(fr, "not a job kind") {
		t.Fatalf("unknown kind accepted:\n%s", fr)
	}
	m = press(t, m, "esc", "n")
	m = typed(t, m, "node.migrate-names")
	m = press(t, m, "enter", "enter")
	if f.Called("Do POST /api/jobs/box-a") {
		t.Fatal("started without the typed confirmation")
	}
	m = typed(t, m, "box-a")
	m = press(t, m, "enter")
	if b, _ := json.Marshal(f.Sent["POST /api/jobs/box-a"]); string(b) != `{"kind":"node.migrate-names"}` {
		t.Fatalf("sent %s; calls %v", b, f.Calls)
	}
	if fr := tuitest.Frame(m); !strings.Contains(fr, "job abc started on box-a") {
		t.Fatalf("frame:\n%s", fr)
	}
}

func TestGatewayModuleToggle(t *testing.T) {
	f := tuitest.NewFake()
	g := gateway("rpc-main", "box")
	f.GatewaysV = []api.GatewaySummary{g}
	f.CatalogV = []api.CatalogModule{{ID: "watch", Placement: "box"}, {ID: "private-send", Name: "Private send", Placement: "gateway"}}
	f.Routes["PUT /api/gateways/rpc-main/modules/private-send"] = map[string]any{}
	m := press(t, newTestApp(t, f, 80, 24, "unicode"), "4", "m")
	fr := tuitest.Frame(m)
	if !strings.Contains(fr, "1 [ ] Private send") || strings.Contains(fr, "watch") {
		t.Fatalf("modal:\n%s", fr)
	}
	m = press(t, m, "1")
	if b, _ := json.Marshal(f.Sent["PUT /api/gateways/rpc-main/modules/private-send"]); string(b) != `{"enabled":true}` {
		t.Fatalf("sent %s; calls %v", b, f.Calls)
	}
	if !strings.Contains(tuitest.Frame(m), "private-send on for rpc-main") {
		t.Fatalf("frame:\n%s", tuitest.Frame(m))
	}
}
