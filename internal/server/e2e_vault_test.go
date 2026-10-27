//go:build e2e

package server

import (
	"bytes"
	"context"
	"encoding/hex"
	"encoding/json"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"testing/fstest"
	"time"

	"golang.org/x/crypto/ssh"
	"golang.org/x/crypto/ssh/knownhosts"

	"github.com/valve-tech/jumpgate/internal/api"
	"github.com/valve-tech/jumpgate/internal/bootstrap"
	"github.com/valve-tech/jumpgate/internal/catalog"
	"github.com/valve-tech/jumpgate/internal/config"
	"github.com/valve-tech/jumpgate/internal/executor"
	"github.com/valve-tech/jumpgate/internal/intent"
	"github.com/valve-tech/jumpgate/internal/secrets"
	"github.com/valve-tech/jumpgate/internal/signer"
	"github.com/valve-tech/jumpgate/internal/testutil"
	"github.com/valve-tech/jumpgate/internal/vault"
	"github.com/valve-tech/jumpgate/internal/vault/vaulttest"
)

// e2eVaultEnv is an environment variable scripts/e2e-agent.sh exports.
func e2eVaultEnv(t *testing.T, k string) string {
	t.Helper()
	v := os.Getenv(k)
	if v == "" {
		t.Skip(k + " not set; run scripts/e2e-agent.sh")
	}
	return v
}

// e2eVaultLegacyWire is the node a box set up before sub-project 2 runs
// (the recipe of bootstrap's e2e_jobs_test.go).
var e2eVaultLegacyWire = catalog.WireConfig{ChainID: 369, ExecID: "reth", BeaconID: "lighthouse-pulse",
	DataDir: catalog.LegacyNames.DataRoot + "/369", JWTPath: catalog.LegacyNames.DataRoot + "/369/jwt.hex"}

// e2eSlowStop is the legacy exec unit's drop-in directory: a drop-in there
// makes stopping it slow, so the test locks while stop-units runs.
var e2eSlowStop = catalog.UnitPath(catalog.LegacyNames.ExecUnit()) + ".d"

// resetToLegacyNode undoes what earlier e2e tests left (a migrated 369 node,
// the module test's 943 units under the current names) and seeds the legacy
// node bootstrap's migrate test starts from, running, with an exec unit that
// takes 15 s to stop. No process ignores SIGTERM, so stop-units asks nothing.
func resetToLegacyNode(ctx context.Context, t *testing.T, ex executor.Executor) {
	t.Helper()
	w := e2eVaultLegacyWire
	legacy := catalog.LegacyNames
	sh := func(cmd string) {
		t.Helper()
		if r, err := ex.Run(ctx, cmd, nil); err != nil || r.ExitCode != 0 {
			t.Fatalf("%s: %+v %v", cmd, r, err)
		}
	}
	cur := catalog.CurrentNames
	sh("systemctl disable --now " + strings.Join(cur.Units(), " ") + " 2>/dev/null; true")
	sh("rm -f " + catalog.UnitPath(cur.ExecUnit()) + " " + catalog.UnitPath(cur.BeaconUnit()) + " && systemctl daemon-reload")
	sh("pkill -KILL -u " + cur.User + " 2>/dev/null; sleep 1; userdel " + cur.User + " 2>/dev/null; groupdel " + cur.Group + " 2>/dev/null; true")
	sh("rm -rf " + cur.DataRoot + " /etc/jumpgate/names.json " + bootstrap.NodePath)

	execUnit, beaconUnit, err := catalog.RenderUnits(w)
	if err != nil {
		t.Fatal(err)
	}
	sh("useradd --system --user-group --home-dir " + legacy.DataRoot + " --no-create-home --shell /usr/sbin/nologin " + legacy.User)
	sh("mkdir -p " + w.DataDir + "/db && head -c 32 /dev/urandom | od -An -tx1 | tr -d ' \\n' > " + w.JWTPath)
	sh("chown -R " + legacy.User + ":" + legacy.Group + " " + legacy.DataRoot)
	for path, body := range map[string]string{
		catalog.UnitPath(legacy.ExecUnit()):   execUnit,
		catalog.UnitPath(legacy.BeaconUnit()): beaconUnit,
		e2eSlowStop + "/slow-stop.conf":       "[Service]\nExecStop=/bin/sleep 15\nTimeoutStopSec=60\n",
	} {
		sh("mkdir -p " + filepath.Dir(path))
		if err := ex.WriteFile(ctx, path, []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	b, err := json.Marshal(w)
	if err != nil {
		t.Fatal(err)
	}
	if err := ex.WriteFile(ctx, bootstrap.NodePath, b, 0o600); err != nil {
		t.Fatal(err)
	}
	sh(catalog.SnapshotMarkerCommand(w))
	sh(catalog.OwnerMarkerCommand(w))
	sh("systemctl daemon-reload && systemctl enable --now " + strings.Join(legacy.Units(), " "))
	t.Cleanup(func() {
		_, _ = ex.Run(context.Background(), "rm -rf "+e2eSlowStop+" && systemctl daemon-reload", nil)
	})
}

// e2eManifest reads job id's manifest on the box, as root, signing nothing.
func e2eManifest(ctx context.Context, t *testing.T, ex executor.Executor, id string) intent.JobRecord {
	t.Helper()
	b, err := ex.ReadFile(ctx, "/var/lib/jumpgate/jobs/"+id+".json")
	if err != nil {
		t.Fatal(err)
	}
	var rec intent.JobRecord
	if err := json.Unmarshal(b, &rec); err != nil {
		t.Fatal(err)
	}
	return rec
}

// TestE2EPlanWaitsAtTheStepBoundaryWhileLocked pairs the container box with
// a controller whose key and transport key live in a vault, starts
// node.migrate-names and locks while a step runs: the step finishes on the
// box, the job waits at the next boundary with nothing signed, and the
// unlock moves it on.
func TestE2EPlanWaitsAtTheStepBoundaryWhileLocked(t *testing.T) {
	root := e2eRoot(t)
	transportKey := e2eVaultEnv(t, "JUMPGATE_E2E_TRANSPORT_KEY")
	transportPub := e2eVaultEnv(t, "JUMPGATE_E2E_TRANSPORT_PUB")
	agentsDir := e2eVaultEnv(t, "JUMPGATE_E2E_AGENTS")
	home := testutil.Home(t)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
	defer cancel()

	// The box's host key, on record in this HOME's known_hosts, so the
	// server's strict dial trusts it.
	var hostKey ssh.PublicKey
	rootRec := root
	rootRec.HostKey = func(_ string, _ net.Addr, k ssh.PublicKey) error { hostKey = k; return nil }
	ex, err := executor.NewSSHContext(ctx, rootRec)
	if err != nil {
		t.Fatal(err)
	}
	defer ex.Close()
	hostport := net.JoinHostPort(root.Host, strconv.Itoa(root.Port))
	if err := os.MkdirAll(filepath.Join(home, ".ssh"), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(home, ".ssh", "known_hosts"), []byte(knownhosts.Line([]string{knownhosts.Normalize(hostport)}, hostKey)+"\n"), 0o600); err != nil {
		t.Fatal(err)
	}

	controller, err := signer.GenerateKey()
	if err != nil {
		t.Fatal(err)
	}
	agentAddr, err := bootstrap.Run(ctx, bootstrap.Options{Exec: ex, Controller: controller.Address(), ControllerLabel: "e2e-vault",
		TransportKey: transportPub, AgentBinary: func(arch string) ([]byte, error) {
			return os.ReadFile(filepath.Join(agentsDir, "jumpgate-linux-"+arch))
		}})
	if err != nil {
		t.Fatal(err)
	}
	resetToLegacyNode(ctx, t, ex)

	// The vault holds the controller key and the transport key.
	p, pass := vaulttest.Path(t)
	v, err := vault.Open(p)
	if err != nil {
		t.Fatal(err)
	}
	if err := v.Unlock(bytes.Clone(pass)); err != nil {
		t.Fatal(err)
	}
	tk, err := os.ReadFile(transportKey)
	if err != nil {
		t.Fatal(err)
	}
	if err := v.Put(secrets.NameTransportKey, tk, "test"); err != nil {
		t.Fatal(err)
	}
	if err := v.Put("controller-key", []byte(hex.EncodeToString(controller.Bytes())), "test"); err != nil {
		t.Fatal(err)
	}
	v.Lock()
	w := e2eVaultLegacyWire
	if _, err := config.Update(func(c *config.Config) error {
		c.SecretStore = config.SecretStoreVault
		c.Controller = &config.Controller{KeyStore: string(signer.StoreVault), KeyRef: "controller-key", Address: controller.Address().Hex()}
		c.Targets = []config.Target{{ID: "box", Mode: "ssh", Wire: &w,
			SSH:   &executor.SSHConfig{Host: root.Host, Port: root.Port, User: "root", KeyPath: root.KeyPath},
			Agent: &config.AgentPairing{Address: agentAddr.Hex(), Transport: "ssh", PairedAt: time.Now()}}}
		return nil
	}); err != nil {
		t.Fatal(err)
	}

	token := NewSessionToken()
	srv := New(Config{Token: token, UI: fstest.MapFS{}, Vault: v, SignerErr: signer.ErrVaultStore,
		Unlocked: func(v *vault.Vault) (signer.Signer, error) {
			k, err := signer.OpenVault(v, "controller-key")
			if err != nil {
				return nil, err
			}
			return k, nil
		}})
	ts := httptest.NewServer(srv.Handler())
	t.Cleanup(func() { ts.Close(); srv.fleet.stop(); srv.jobs.stopWithin(5 * time.Second) })
	if _, err := srv.Unlock(bytes.Clone(pass)); err != nil {
		t.Fatal(err)
	}

	raw, _ := json.Marshal(api.JobStart{Kind: intent.KindNodeMigrateNames, Inputs: json.RawMessage(`{}`)})
	req, _ := http.NewRequest("POST", ts.URL+"/api/jobs/box", bytes.NewReader(raw))
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	var jv api.JobView
	if err := json.NewDecoder(res.Body).Decode(&jv); err != nil || res.StatusCode/100 != 2 {
		t.Fatalf("POST /api/jobs/box = %d %v", res.StatusCode, err)
	}
	res.Body.Close()
	t.Logf("job %s started", jv.ID)

	poll := func(what string, d time.Duration, ok func(intent.JobRecord) bool) intent.JobRecord {
		t.Helper()
		deadline := time.Now().Add(d)
		for {
			rec := e2eManifest(ctx, t, ex, jv.ID)
			if ok(rec) {
				return rec
			}
			if intent.JobTerminal(rec.State) || time.Now().After(deadline) {
				t.Fatalf("waiting for %s: job is %s at step %d (%s)", what, rec.State, rec.Step, rec.Error)
			}
			time.Sleep(200 * time.Millisecond)
		}
	}
	stepID := func(rec intent.JobRecord) string {
		if rec.Step < len(rec.Steps) {
			return rec.Steps[rec.Step].ID
		}
		return ""
	}

	running := poll("stop-units to run", 2*time.Minute, func(r intent.JobRecord) bool {
		return r.State == intent.JobRunning && stepID(r) == "stop-units"
	})
	srv.Lock()
	t.Logf("locked while %s runs", running.Unit)

	// The step finishes on the box by itself; the job waits at the boundary.
	boundary := poll("the next boundary", time.Minute, func(r intent.JobRecord) bool { return r.State != intent.JobRunning })
	if boundary.State != intent.JobWaiting || boundary.Step != running.Step+1 {
		t.Fatalf("after the step the job is %s at step %d, want waiting at %d", boundary.State, boundary.Step, running.Step+1)
	}
	time.Sleep(3 * jobsDriveEvery)
	held := e2eManifest(ctx, t, ex, jv.ID)
	if held.State != intent.JobWaiting || held.Step != boundary.Step || held.LogLen != boundary.LogLen {
		t.Fatalf("while locked the job moved: %s step %d log %d, was waiting at %d log %d",
			held.State, held.Step, held.LogLen, boundary.Step, boundary.LogLen)
	}
	if st := srv.VaultState().State; st != "locked" {
		t.Fatalf("vault %s", st)
	}

	if _, err := srv.Unlock(bytes.Clone(pass)); err != nil {
		t.Fatal(err)
	}
	poll("the unlock to advance the job", 30*time.Second, func(r intent.JobRecord) bool {
		return r.Step > boundary.Step || r.State == intent.JobRunning
	})
	t.Log("the unlock advanced the job")
	deadline := time.Now().Add(3 * time.Minute)
	for rec := e2eManifest(ctx, t, ex, jv.ID); !intent.JobTerminal(rec.State); rec = e2eManifest(ctx, t, ex, jv.ID) {
		if time.Now().After(deadline) {
			t.Fatalf("job never finished: %s at %s", rec.State, stepID(rec))
		}
		time.Sleep(time.Second)
	}
	if rec := e2eManifest(ctx, t, ex, jv.ID); rec.State != intent.JobSucceeded {
		t.Fatalf("job ended %s: %s", rec.State, rec.Error)
	}
}
