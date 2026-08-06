package vpn

import (
	"context"
	"io/fs"
	"strings"
	"sync"
	"testing"

	"github.com/valve-tech/valve-node-app/internal/executor"
)

// fakeExecutor mirrors internal/ops and internal/setup test doubles: scripted
// results keyed by command substring (longest match wins), every call and
// written file recorded for assertions.
type fakeExecutor struct {
	mu      sync.Mutex
	scripts map[string]executor.Result
	calls   []string
	files   map[string]struct {
		content []byte
		mode    fs.FileMode
	}
}

func newFake() *fakeExecutor {
	return &fakeExecutor{
		scripts: map[string]executor.Result{},
		files: map[string]struct {
			content []byte
			mode    fs.FileMode
		}{},
	}
}

func (f *fakeExecutor) script(substr string, res executor.Result) *fakeExecutor {
	f.scripts[substr] = res
	return f
}

func (f *fakeExecutor) Run(ctx context.Context, cmd string, opts *executor.RunOpts) (executor.Result, error) {
	f.mu.Lock()
	f.calls = append(f.calls, cmd)
	f.mu.Unlock()
	best := ""
	for k := range f.scripts {
		if strings.Contains(cmd, k) && len(k) > len(best) {
			best = k
		}
	}
	if best != "" {
		return f.scripts[best], nil
	}
	return executor.Result{ExitCode: 0}, nil
}

func (f *fakeExecutor) WriteFile(ctx context.Context, path string, content []byte, mode fs.FileMode) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.files[path] = struct {
		content []byte
		mode    fs.FileMode
	}{content, mode}
	return nil
}

func (f *fakeExecutor) ReadFile(ctx context.Context, path string) ([]byte, error) {
	return f.files[path].content, nil
}
func (f *fakeExecutor) Close() error { return nil }

func (f *fakeExecutor) called(substr string) bool {
	f.mu.Lock()
	defer f.mu.Unlock()
	for _, c := range f.calls {
		if strings.Contains(c, substr) {
			return true
		}
	}
	return false
}

func mustProvider(t *testing.T) StaticProvider {
	t.Helper()
	p, err := NewStaticProvider("bring-your-own", protonConf)
	if err != nil {
		t.Fatalf("NewStaticProvider: %v", err)
	}
	return p
}

// wg dump: interface line + one peer that has handshaked (non-zero timestamp).
const wgDumpUp = "PRIVKEY\tPUBKEY\t51820\toff\n" +
	"peerpub\t(none)\t203.0.113.7:51820\t0.0.0.0/0\t1785000000\t128\t256\t25\n"

func TestWgQuick_Up_WritesConfigAndVerifies(t *testing.T) {
	f := newFake().
		script("wg-quick up", executor.Result{ExitCode: 0}).
		script("wg show", executor.Result{ExitCode: 0, Stdout: wgDumpUp})
	w := WgQuick{Exec: f, Iface: "jumpgate0", Provider: mustProvider(t)}

	st, err := w.Up(context.Background())
	if err != nil {
		t.Fatalf("Up: %v", err)
	}
	if !st.Up || st.Peers != 1 {
		t.Errorf("state = %+v, want Up with 1 peer", st)
	}
	if !st.Handshaked() {
		t.Errorf("expected a handshake to be reported")
	}
	if st.Provider != "bring-your-own" {
		t.Errorf("provider = %q", st.Provider)
	}
	// config written to the right path, locked down 0600
	wf, ok := f.files["/etc/wireguard/jumpgate0.conf"]
	if !ok {
		t.Fatalf("config was not written")
	}
	if wf.mode != 0o600 {
		t.Errorf("config mode = %o, want 600", wf.mode)
	}
	if !strings.Contains(string(wf.content), "[Interface]") {
		t.Errorf("written config missing [Interface]:\n%s", wf.content)
	}
	if !f.called("wg-quick up 'jumpgate0'") {
		t.Errorf("wg-quick up was not invoked with the quoted iface; calls=%v", f.calls)
	}
}

// The load-bearing test: wg-quick exits 0 but the interface never actually came
// up (wg show fails). Up must FAIL, not report success — this app's worst bugs
// report success over a broken state.
func TestWgQuick_Up_FailsWhenInterfaceAbsentDespiteExit0(t *testing.T) {
	f := newFake().
		script("wg-quick up", executor.Result{ExitCode: 0}).
		script("wg show", executor.Result{ExitCode: 1, Stderr: "Unable to access interface: No such device"})
	w := WgQuick{Exec: f, Iface: "jumpgate0", Provider: mustProvider(t)}

	if _, err := w.Up(context.Background()); err == nil {
		t.Fatalf("Up reported success even though the interface is absent")
	}
}

func TestWgQuick_Up_PropagatesWgQuickFailure(t *testing.T) {
	f := newFake().script("wg-quick up", executor.Result{ExitCode: 1, Stderr: "RTNETLINK answers: Operation not permitted"})
	w := WgQuick{Exec: f, Iface: "jumpgate0", Provider: mustProvider(t)}
	if _, err := w.Up(context.Background()); err == nil {
		t.Errorf("expected Up to fail when wg-quick exits non-zero")
	}
}

func TestWgQuick_Status_DownWhenInterfaceAbsent(t *testing.T) {
	f := newFake().script("wg show", executor.Result{ExitCode: 1})
	w := WgQuick{Exec: f, Iface: "jumpgate0", Provider: mustProvider(t)}
	st, err := w.Status(context.Background())
	if err != nil {
		t.Fatalf("Status returned error for an absent interface: %v", err)
	}
	if st.Up {
		t.Errorf("expected Up=false for an absent interface")
	}
}

func TestWgQuick_Down_IgnoresNotAnInterface(t *testing.T) {
	f := newFake().script("wg-quick down", executor.Result{ExitCode: 1, Stderr: "`jumpgate0' is not a WireGuard interface"})
	w := WgQuick{Exec: f, Iface: "jumpgate0", Provider: mustProvider(t)}
	if err := w.Down(context.Background()); err != nil {
		t.Errorf("Down should treat 'not a WireGuard interface' as already-down, got %v", err)
	}
}

func TestParseWgDump_MultiPeerHandshake(t *testing.T) {
	dump := "PRIVKEY\tPUBKEY\t51820\toff\n" +
		"peerA\t(none)\tx:1\t10.0.0.0/24\t0\t0\t0\t0\n" +
		"peerB\t(none)\ty:2\t10.0.1.0/24\t1785000123\t1\t2\t25\n"
	st := parseWgDump(dump)
	if !st.Up || st.Peers != 2 {
		t.Fatalf("state = %+v, want Up with 2 peers", st)
	}
	if st.LastHandshake.Unix() != 1785000123 {
		t.Errorf("LastHandshake = %d, want the newer peer's ts", st.LastHandshake.Unix())
	}
}
