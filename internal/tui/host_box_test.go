package tui

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/valve-tech/jumpgate/internal/api"
	"github.com/valve-tech/jumpgate/internal/apiclient"
	"github.com/valve-tech/jumpgate/internal/tui/tuitest"
)

const ctrPath = "/api/targets/box/containers"

func sampleContainers() map[string]any {
	return map[string]any{
		"docker": map[string]any{"present": true, "reachable": true, "flavor": "docker", "serverVersion": "27.1"},
		"services": []any{map[string]any{
			"id": "devnet", "label": "Devnet", "containerName": "jumpgate-devnet", "configured": true,
			"status":       map[string]any{"State": "running", "Image": "ghcr.io/paradigmxyz/reth:v1"},
			"endpoints":    []any{map[string]any{"label": "JSON-RPC", "url": "http://user:pw@127.0.0.1:8545/"}},
			"actions":      []any{"stop", "restart", "recreate", "wipe"},
			"wipeDiscards": "the chain's data volume", "restartsOnWipe": []any{"gw1"},
		}},
	}
}

func boxFake() *tuitest.Fake {
	f := tuitest.NewFake()
	f.Routes["GET "+ctrPath] = sampleContainers()
	f.Routes["GET /api/targets/box/diagnostics/latest"] = map[string]any{"at": "2026-10-29T10:00:00Z", "checks": []any{map[string]any{"name": "rpc", "ok": true}}}
	return f
}

func containersHost(t *testing.T, f *tuitest.Fake) *App {
	t.Helper()
	return tab(t, openTestHost(t, f, "unicode"), 6)
}

func toolsHost(t *testing.T, f *tuitest.Fake) *App {
	t.Helper()
	return tab(t, openTestHost(t, f, "unicode"), 7)
}

func TestJSONLinesFlattenSanitizeMask(t *testing.T) {
	got := strings.Join(jsonLines(json.RawMessage(`{"b":{"url":"https://u:secret@h.example/x","n":3},"a":["x\u001b[31m",true],"z":null}`)), "\n")
	for _, want := range []string{"a[0]: x", "a[1]: true", "b.n: 3", "z: n/a"} {
		if !strings.Contains(got, want) {
			t.Errorf("lacks %q:\n%s", want, got)
		}
	}
	if strings.Contains(got, "secret") || strings.Contains(got, "\x1b") {
		t.Fatalf("leaked:\n%q", got)
	}
	if strings.Index(got, "a[0]") > strings.Index(got, "b.n") {
		t.Fatalf("keys not sorted:\n%s", got)
	}
	if l := jsonLines(json.RawMessage(`not json`)); len(l) != 1 || !strings.Contains(l[0], "FAIL") {
		t.Fatalf("bad JSON %q", l)
	}
}

func TestContainersTab(t *testing.T) {
	f := boxFake()
	a := containersHost(t, f)
	fr := tuitest.Frame(a)
	for _, want := range []string{"[Docker]", "docker 27.1 reachable", "devnet", "running", "stop restart recreate wipe", "JSON-RPC"} {
		if !strings.Contains(fr, want) {
			t.Errorf("containers lack %q:\n%s", want, fr)
		}
	}
	if strings.Contains(fr, "pw@") {
		t.Fatalf("an endpoint's credentials are shown:\n%s", fr)
	}
	tuitest.Golden(t, "host_containers", fr)
}

func TestContainerStartAtOnceStopTyped(t *testing.T) {
	f := boxFake()
	f.Routes["POST "+ctrPath+"/devnet/start"] = map[string]any{"status": map[string]any{"State": "running"}}
	f.Routes["POST "+ctrPath+"/devnet/stop"] = map[string]any{"status": map[string]any{"State": "stopped"}}
	m := press(t, containersHost(t, f), "s")
	if !f.Called("Do POST " + ctrPath + "/devnet/start") {
		t.Fatalf("calls %v", f.Calls)
	}
	m = press(t, m, "t")
	m = typed(t, m, "devne")
	m = press(t, m, "enter")
	if f.Called("Do POST " + ctrPath + "/devnet/stop") {
		t.Fatal("stop ran on a near miss")
	}
	m = typed(t, m, "devnet")
	m = press(t, m, "enter")
	if !f.Called("Do POST " + ctrPath + "/devnet/stop") {
		t.Fatalf("calls %v", f.Calls)
	}
	if fr := tuitest.Frame(m); !strings.Contains(fr, "status.State: stopped") {
		t.Fatalf("the answer is not shown:\n%s", fr)
	}
}

func TestContainerWipeAndResetNeedTheName(t *testing.T) {
	for _, c := range []struct{ key, route string }{{"W", "wipe"}, {"X", "reset"}} {
		f := boxFake()
		f.Routes["POST "+ctrPath+"/devnet/"+c.route] = map[string]any{"report": map[string]any{"ContainerRemoved": true}}
		m := press(t, containersHost(t, f), c.key)
		if fr := tuitest.Frame(m); !strings.Contains(fr, "Type devnet") {
			t.Fatalf("%s: no typed confirmation:\n%s", c.route, fr)
		}
		m = press(t, m, "enter")
		if f.Called("Do POST " + ctrPath + "/devnet/" + c.route) {
			t.Fatalf("%s ran on a bare enter", c.route)
		}
		m = typed(t, m, "devnet")
		m = press(t, m, "enter")
		if !f.Called("Do POST " + ctrPath + "/devnet/" + c.route) {
			t.Fatalf("%s: calls %v", c.route, f.Calls)
		}
		if c.route == "wipe" {
			if b, _ := json.Marshal(f.Sent["POST "+ctrPath+"/devnet/wipe"]); string(b) != `{"Confirm":"devnet"}` {
				t.Fatalf("wipe body %s", b)
			}
		}
		if fr := tuitest.Frame(m); !strings.Contains(fr, "report.ContainerRemoved: true") {
			t.Fatalf("%s: the report is not shown:\n%s", c.route, fr)
		}
	}
}

func TestContainerProvisionFollowsTheRun(t *testing.T) {
	f := boxFake()
	f.Routes["POST "+ctrPath+"/devnet/provision"] = map[string]string{"status": "started"}
	f.SetupV = []api.SetupEvent{{StepID: "pull", Line: "pulling image"}, {StepID: "pull", Done: true}, {Done: true}}
	m := press(t, containersHost(t, f), "p")
	if !f.Called("WatchSetup box") {
		t.Fatalf("calls %v", f.Calls)
	}
	fr := tuitest.Frame(m)
	for _, want := range []string{"provision devnet", "pull pulling image", "done"} {
		if !strings.Contains(fr, want) {
			t.Errorf("run lacks %q:\n%s", want, fr)
		}
	}
}

func TestContainerConfigShowAndSet(t *testing.T) {
	f := boxFake()
	f.Routes["GET "+ctrPath+"/devnet/config"] = map[string]any{"id": "devnet", "configured": true, "devnet": map[string]any{"ChainID": 1337, "BlockTime": "2s"}}
	f.Routes["PUT "+ctrPath+"/devnet/config"] = map[string]any{"id": "devnet", "configured": true, "devnet": map[string]any{"ChainID": 1337, "BlockTime": "5s"}}
	m := press(t, containersHost(t, f), "v")
	if fr := tuitest.Frame(m); !strings.Contains(fr, "devnet.BlockTime: 2s") {
		t.Fatalf("config not shown:\n%s", fr)
	}
	m = press(t, m, "e")
	m = typed(t, m, "{nope")
	m = press(t, m, "enter")
	if f.Called("Do PUT") {
		t.Fatal("bad JSON was sent")
	}
	m = press(t, m, "esc", "e")
	m = typed(t, m, `{"BlockTime":"5s"}`)
	m = press(t, m, "enter")
	if b, _ := json.Marshal(f.Sent["PUT "+ctrPath+"/devnet/config"]); string(b) != `{"BlockTime":"5s"}` {
		t.Fatalf("sent %s; calls %v", b, f.Calls)
	}
	if fr := tuitest.Frame(m); !strings.Contains(fr, "devnet.BlockTime: 5s") {
		t.Fatalf("saved config not shown:\n%s", fr)
	}
}

func TestClearServiceStateNeedsTheName(t *testing.T) {
	f := tuitest.NewFake()
	f.Routes["POST /api/targets/box/services/exec/clear"] = map[string]any{"cleared": true}
	m := press(t, servicesHost(t, f), "C")
	m = typed(t, m, "beacon")
	m = press(t, m, "enter")
	if f.Called("Do POST") {
		t.Fatal("the wrong name cleared a service")
	}
	m = typed(t, m, "exec")
	m = press(t, m, "enter")
	if b, _ := json.Marshal(f.Sent["POST /api/targets/box/services/exec/clear"]); string(b) != `{"Confirm":"exec"}` {
		t.Fatalf("sent %s; calls %v", b, f.Calls)
	}
	if fr := tuitest.Frame(m); !strings.Contains(fr, "cleared: true") {
		t.Fatalf("frame:\n%s", fr)
	}
}

func TestStorageDiskUsage(t *testing.T) {
	f := tuitest.NewFake()
	f.DiskV["box"] = sampleDisk()
	f.Routes["GET /api/targets/box/du"] = map[string]any{"execBytes": 123, "beaconBytes": 45}
	m := press(t, tab(t, openTestHost(t, f, "unicode"), 1), "u")
	if fr := tuitest.Frame(m); !strings.Contains(fr, "execBytes: 123") {
		t.Fatalf("frame:\n%s", fr)
	}
}

func TestToolsTab(t *testing.T) {
	f := boxFake()
	a := toolsHost(t, f)
	fr := tuitest.Frame(a)
	for _, want := range []string{"[Tools]", "D run diagnostics", "checks[0].name: rpc"} {
		if !strings.Contains(fr, want) {
			t.Errorf("tools lack %q:\n%s", want, fr)
		}
	}
	tuitest.Golden(t, "host_tools", fr)
	f.Routes["GET /api/targets/box/diagnostics"] = map[string]any{"checks": []any{map[string]any{"name": "disk", "ok": false}}}
	if fr := tuitest.Frame(press(t, a, "D")); !strings.Contains(fr, "checks[0].name: disk") {
		t.Fatalf("run not shown:\n%s", fr)
	}
}

// The monitor is numbers, latest value only; M again stops it.
func TestMonitorShowsNumbers(t *testing.T) {
	f := boxFake()
	const p = "/api/targets/box/monitor/stream"
	f.Stream(p) <- apiclient.Update[json.RawMessage]{Value: json.RawMessage(`{"exec":{"head":100,"peers":9}}`), Has: true, State: apiclient.Live}
	m := press(t, toolsHost(t, f), "M")
	if !f.Called("Watch " + p) {
		t.Fatalf("calls %v", f.Calls)
	}
	fr := tuitest.Frame(m)
	for _, want := range []string{"monitor live", "exec.head: 100", "exec.peers: 9"} {
		if !strings.Contains(fr, want) {
			t.Errorf("monitor lacks %q:\n%s", want, fr)
		}
	}
	m = press(t, m, "M")
	if fr := tuitest.Frame(m); strings.Contains(fr, "exec.peers") {
		t.Fatalf("monitor still shown:\n%s", fr)
	}
}

func TestIntentRunsAKnownKind(t *testing.T) {
	f := boxFake()
	f.Routes["POST /api/targets/box/intent/disk.read"] = map[string]any{"status": "ok", "result": map[string]any{"free": 9}}
	m := press(t, toolsHost(t, f), "I")
	m = typed(t, m, "job.decide")
	m = press(t, m, "enter")
	if f.Called("Do POST /api/targets/box/intent") {
		t.Fatal("a kind outside the list was sent")
	}
	m = press(t, m, "esc", "I")
	m = typed(t, m, "disk.read")
	m = press(t, m, "enter")
	if fr := tuitest.Frame(m); !strings.Contains(fr, "result.free: 9") {
		t.Fatalf("frame:\n%s\ncalls %v", fr, f.Calls)
	}
}

func TestSetupWizardStreams(t *testing.T) {
	f := boxFake()
	f.Routes["POST /api/targets/box/setup"] = map[string]string{"status": "started"}
	f.SetupV = []api.SetupEvent{{StepID: "install", Line: "installing reth"}, {StepID: "install", Err: "no space left"}}
	m := press(t, toolsHost(t, f), "w")
	m = typed(t, m, `{"ChainID":369}`)
	m = press(t, m, "enter")
	if b, _ := json.Marshal(f.Sent["POST /api/targets/box/setup"]); string(b) != `{"ChainID":369}` {
		t.Fatalf("sent %s; calls %v", b, f.Calls)
	}
	fr := tuitest.Frame(m)
	for _, want := range []string{"install installing reth", "FAIL", "no space left"} {
		if !strings.Contains(fr, want) {
			t.Errorf("run lacks %q:\n%s", want, fr)
		}
	}
}
