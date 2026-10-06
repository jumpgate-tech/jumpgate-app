package setup

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"testing"

	"github.com/valve-tech/jumpgate/internal/catalog"
	"github.com/valve-tech/jumpgate/internal/executor"
	"github.com/valve-tech/jumpgate/internal/executor/argvfake"
)

// rpcServer answers eth_chainId with chain and eth_blockNumber with 5.
func rpcServer(t *testing.T, chain string) *httptest.Server {
	t.Helper()
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		b, _ := io.ReadAll(r.Body)
		var req struct{ Method string }
		_ = json.Unmarshal(b, &req)
		result := chain
		if req.Method == "eth_blockNumber" {
			result = "0x5"
		}
		_, _ = w.Write([]byte(`{"jsonrpc":"2.0","id":1,"result":"` + result + `"}`))
	}))
	t.Cleanup(ts.Close)
	return ts
}

// windowsDesktop is a Windows controller whose Docker Desktop runs Linux
// containers, and whose container (by name) is already running, so the
// port checks defer to it the way they do for our own live container.
func windowsDesktop() *argvfake.Fake {
	return argvfake.New().
		Script("docker --version", executor.Result{Stdout: "Docker version 27.4.0, build bde2b89\n"}).
		Script("docker info --format", executor.Result{Stdout: "27.4.0|linux|x86_64|docker-desktop|Docker Desktop\n"}).
		Script("docker image inspect", executor.Result{Stdout: "sha256:abc\n"}).
		Script("docker inspect -f {{.State.Running}}", executor.Result{Stdout: "true\n"})
}

// The whole devnet plan runs on a machine with no shell at all.
func TestDevnetPlanNeedsNoShell(t *testing.T) {
	f := windowsDesktop()
	d := testDevnet()
	f.Route("127.0.0.1:"+itoa(d.HTTP()), strings.TrimPrefix(rpcServer(t, "0x539").URL, "http://"))
	steps, err := PlanDevnet(d)
	if err != nil {
		t.Fatal(err)
	}
	if err := RunAll(context.Background(), f, steps, &State{}); err != nil {
		t.Fatalf("devnet on a shell-less controller: %v", err)
	}
	if sh := f.ShellCalls(); len(sh) != 0 {
		t.Fatalf("the devnet plan used a shell: %q", sh)
	}
}

// The whole docker-backend gateway plan, too. The config lands under the
// Windows home, and docker is given that path in a --mount.
func TestGatewayPlanNeedsNoShell(t *testing.T) {
	f := windowsDesktop()
	g := testGateway()
	f.Route("127.0.0.1:4100", strings.TrimPrefix(rpcServer(t, "0x171").URL, "http://"))
	steps := mustPlanGateway(t, g, BackendDocker)
	if err := RunAll(context.Background(), f, steps, &State{}); err != nil {
		t.Fatalf("gateway on a shell-less controller: %v", err)
	}
	if sh := f.ShellCalls(); len(sh) != 0 {
		t.Fatalf("the gateway plan used a shell: %q", sh)
	}
	cfg := filepath.Join(f.Home, ".valve-node-app", "erpc.yaml")
	if _, ok := f.Files[cfg]; !ok {
		t.Fatalf("erpc.yaml not written at %s; files: %v", cfg, keys(f.Files))
	}
	var run []string
	for _, a := range f.Argvs() {
		if len(a) > 1 && a[0] == "docker" && a[1] == "run" {
			run = a
		}
	}
	if !strings.Contains(strings.Join(run, " "), "--mount type=bind,source="+cfg+",target=/erpc.yaml,readonly") {
		t.Fatalf("docker run %q does not bind %s", run, cfg)
	}
}

// The message the gap analysis found unreachable now reaches a Windows user.
func TestPreflight_WindowsContainerModeReachesAShellLessController(t *testing.T) {
	f := argvfake.New().
		Script("docker --version", executor.Result{Stdout: "Docker version 27.4.0\n"}).
		Script("docker info --format", executor.Result{Stdout: "27.4.0|windows|x86_64|WIN-RUNNER|Microsoft Windows Server 2022 Datacenter\n"})
	for name, step := range map[string]Step{
		"gateway": stepByID(t, mustPlanGateway(t, testGateway(), BackendDocker), "preflight"),
		"devnet":  stepByID(t, mustPlanDevnet(t, testDevnet()), "preflight"),
	} {
		err := step.Verify(context.Background(), f, &State{})
		// "Linux containers" is in both hints; which one is ops' concern
		// (TestWindowsContainersHintNamesTheRightFix), and it depends on
		// whether Docker Desktop is installed on the machine running this.
		if err == nil || !strings.Contains(err.Error(), "Linux containers") {
			t.Errorf("%s: %v, want the Windows-container hint", name, err)
		}
	}
	if sh := f.ShellCalls(); len(sh) != 0 {
		t.Fatalf("preflight used a shell: %q", sh)
	}
}

func TestGatewayPreflight_SystemdOnAShellLessControllerNamesTheDockerBackend(t *testing.T) {
	step := stepByID(t, mustPlanGateway(t, testGateway(), BackendSystemd), "preflight")
	err := step.Verify(context.Background(), argvfake.New(), &State{})
	if err == nil || !strings.Contains(err.Error(), `"docker" backend`) {
		t.Fatalf("got %v, want a refusal naming the docker backend", err)
	}
}

// D34: the TLS front would mount a certificate file at its own host path
// inside a Linux container, and a Windows path cannot be one.
func TestGatewayPreflight_RefusesCertFilesOnWindows(t *testing.T) {
	g := testGateway()
	g.TLS = &catalog.GatewayTLS{Enabled: true, Hostname: "rpc.lan", CertSource: catalog.CertFiles, CertFile: `C:\certs\rpc.pem`, KeyFile: `C:\certs\rpc.key`}
	step := stepByID(t, mustPlanGateway(t, g, BackendDocker), "preflight")
	err := step.Verify(context.Background(), windowsDesktop(), &State{})
	if err == nil || !strings.Contains(err.Error(), "certificate files") {
		t.Fatalf("got %v, want the cert-files refusal", err)
	}
}

func itoa(n int) string { return strconv.Itoa(n) }

// keys is m's keys, sorted, for a readable failure message.
func keys(m map[string][]byte) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}

// The buildx preflight (gap W1) reaches a shell-less controller: an absent
// image and a CLI without buildx name the fix, through argv.
func TestGatewayPreflight_BuildxCheckRunsAsArgv(t *testing.T) {
	f := windowsDesktop().
		Script("docker image inspect", executor.Result{ExitCode: 1, Stderr: "Error: No such image\n"}).
		Script("docker buildx version", executor.Result{ExitCode: 1, Stderr: "docker: 'buildx' is not a docker command.\n"})
	step := stepByID(t, mustPlanGateway(t, testGateway(), BackendDocker), "preflight")
	err := step.Verify(context.Background(), f, &State{})
	if err == nil || !strings.Contains(err.Error(), "buildx") {
		t.Fatalf("got %v, want the buildx refusal", err)
	}
	if sh := f.ShellCalls(); len(sh) != 0 {
		t.Fatalf("preflight used a shell: %q", sh)
	}
}

// The crash-loop diagnosis (gap W3) replaces a refused in-process probe the
// way it replaces curl's exit 7, and its logs are still redacted.
func TestGatewayCheck_CrashLoopDiagnosisOverArgv(t *testing.T) {
	f := windowsDesktop().
		Script("docker inspect -f {{.State.Status}}", executor.Result{Stdout: "restarting|5\n"}).
		Script("docker logs", executor.Result{Stdout: "read /erpc.yaml: is a directory\nupstream https://user:hunter2@rpc.example\n"})
	p := &gatewayPlan{id: testGatewayID, gw: testGateway(), backend: BackendDocker}
	err := p.gatewayCheck(context.Background(), f)
	if err == nil {
		t.Fatal("want an error: nothing answers the probe")
	}
	for _, want := range []string{"restarting", "(5 restarts)", "read /erpc.yaml: is a directory"} {
		if !strings.Contains(err.Error(), want) {
			t.Errorf("want %q in %v", want, err)
		}
	}
	if strings.Contains(err.Error(), "hunter2") {
		t.Errorf("logs not redacted: %v", err)
	}
	if sh := f.ShellCalls(); len(sh) != 0 {
		t.Fatalf("the check used a shell: %q", sh)
	}
}
