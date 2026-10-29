package tui

import (
	"encoding/base64"
	"encoding/json"
	"os/exec"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"

	"github.com/valve-tech/jumpgate/internal/api"
	"github.com/valve-tech/jumpgate/internal/apiclient"
	"github.com/valve-tech/jumpgate/internal/tui/tuitest"
)

func vaultApp(t *testing.T, f *tuitest.Fake, goos string) *App {
	t.Helper()
	if f.VaultV.State == "" {
		f.VaultV = api.VaultView{State: "locked", Slots: []string{"passphrase", "keystore"}, Plain: []string{"relay-token"}}
	}
	f.SettingsV.VaultIdleLockMinutes = 30
	a := dashApp(t, f, 120, 40, "ascii", "auto")
	a.o.GOOS = goos
	return press(t, a, "6").(*App)
}

// recordSelf makes Command record each argv and run true.
func recordSelf(a *App) *[][]string {
	var ran [][]string
	a.o.Self = "/usr/local/bin/jumpgate"
	a.o.Command = func(name string, args ...string) *exec.Cmd {
		ran = append(ran, append([]string{name}, args...))
		return exec.Command("true")
	}
	return &ran
}

// selfArgs is the jumpgate argv a recorded run passed through pauseScript.
func selfArgs(t *testing.T, r []string) string {
	t.Helper()
	if len(r) < 4 || r[0] != "/bin/sh" || r[1] != "-c" || r[2] != pauseScript || r[3] != "/usr/local/bin/jumpgate" {
		t.Fatalf("not a paused self run: %q", r)
	}
	return strings.Join(r[4:], " ")
}

func TestVaultView(t *testing.T) {
	m := vaultApp(t, tuitest.NewFake(), "linux")
	fr := tuitest.Frame(m)
	for _, want := range []string{"Vault", "state      locked", "slots      passphrase, keystore", "relay-token (not in the vault yet: m)",
		"idle lock  30 min", "[p] put relay-token or admin-token", "[a] add a recovery code", "[s] seal with the TPM2"} {
		if !strings.Contains(fr, want) {
			t.Errorf("no %q:\n%s", want, fr)
		}
	}
	tuitest.Golden(t, "vault_linux_120x40", fr)
	checkOnce(t, "vault", m)
}

func TestVaultFailsClosed(t *testing.T) {
	f := tuitest.NewFake()
	f.VaultV = api.VaultView{State: "unusable", Reason: "bad header\x1b[2J"}
	f.Err["Settings"] = apiclient.ErrServerUnreachable
	fr := tuitest.Frame(vaultApp(t, f, "linux"))
	for _, want := range []string{"FAIL unusable", "reason     bad header", "idle lock  FAIL"} {
		if !strings.Contains(fr, want) {
			t.Errorf("no %q:\n%s", want, fr)
		}
	}
	if strings.Contains(fr, "\x1b[2J") {
		t.Errorf("escape reached the frame")
	}
}

// Each vault command re-runs this jumpgate with a fixed argv, no secret on
// it, and waits for enter so its output can be read.
func TestVaultCommandsRunSelf(t *testing.T) {
	f := tuitest.NewFake()
	a := vaultApp(t, f, "linux")
	ran := recordSelf(a)
	var m tea.Model = press(t, a, "I", "m", "b")
	m = press(t, typed(t, m, "/tmp/v.bak"), "enter", "r")
	m = press(t, typed(t, m, "/tmp/v.bak"), "enter")
	if len(*ran) != 3 {
		t.Fatalf("restore ran before its confirmation: %v", *ran)
	}
	m = press(t, typed(t, m, "restore"), "enter", "a", "K", "s", "S")
	want := []string{"vault init", "vault migrate", "vault backup /tmp/v.bak", "vault restore /tmp/v.bak",
		"vault add-recovery", "vault rekdf", "vault seal --tpm2", "vault seal --allow-host-key"}
	if len(*ran) != len(want) {
		t.Fatalf("ran %v", *ran)
	}
	for i, r := range *ran {
		if got := selfArgs(t, r); got != want[i] {
			t.Errorf("ran %q, want %q", got, want[i])
		}
	}
	m, cmds := tuitest.Send(m, selfDoneMsg{args: []string{"vault", "rekdf"}, err: exitErr(t, 3)})
	if fl := m.(*App).flash; fl != "jumpgate vault rekdf: exit 3" {
		t.Errorf("flash %q", fl)
	}
	f.Calls = nil
	tuitest.Settle(m, cmds...)
	if !f.Called("Vault") {
		t.Errorf("the vault was not re-read: %v", f.Calls)
	}
}

func TestVaultSealOffLinuxAndBadPaths(t *testing.T) {
	a := vaultApp(t, tuitest.NewFake(), "darwin")
	ran := recordSelf(a)
	fr := tuitest.Frame(press(t, a, "s", "S"))
	if strings.Contains(fr, "[s] seal") || len(*ran) != 0 {
		t.Fatalf("seal off Linux: ran %v\n%s", *ran, fr)
	}
	m := press(t, typed(t, press(t, a, "b"), "--out"), "enter")
	if fr := tuitest.Frame(m); !strings.Contains(fr, "cannot start with -") || len(*ran) != 0 {
		t.Fatalf("a flag went through as a path: %v\n%s", *ran, fr)
	}
	m = press(t, m, "esc")
	if m.(*App).modal != nil {
		t.Fatalf("esc left the prompt open")
	}
}

// The guide's vault step opens the Vault view and presses I there.
func TestGuideCreatesTheVaultOnTheVaultView(t *testing.T) {
	f := tuitest.NewFake()
	f.VaultV = api.VaultView{State: "none"}
	a := dashApp(t, f, 120, 40, "ascii", "auto")
	ran := recordSelf(a)
	m := press(t, a, "g", "enter")
	if m.(*App).active != scrVault || len(*ran) != 1 || selfArgs(t, (*ran)[0]) != "vault init" {
		t.Fatalf("active %v, ran %v", m.(*App).active, *ran)
	}
}

const secretValue = "relay-SECRET-token-value"

func TestVaultPutSecret(t *testing.T) {
	f := tuitest.NewFake()
	var sent apiclient.RawJSON
	var got string
	f.Routes["PUT /api/vault/secrets/relay-token"] = func(in any) (any, error) {
		sent = in.(apiclient.RawJSON)
		var v api.SecretValue
		if err := json.Unmarshal(sent, &v); err != nil {
			return nil, err
		}
		got = string(v.Value)
		return nil, nil
	}
	m := press(t, typed(t, press(t, vaultApp(t, f, "linux"), "p"), "relay-token"), "enter")
	m = typed(t, m, secretValue)
	fr := tuitest.Frame(m)
	if strings.Contains(fr, "SECRET") || !strings.Contains(fr, strings.Repeat("*", len(secretValue))) {
		t.Fatalf("the value is not masked:\n%s", fr)
	}
	m = press(t, m, "enter")
	if got != secretValue {
		t.Fatalf("the server got %q", got)
	}
	for _, b := range sent {
		if b != 0 {
			t.Fatalf("the body was not cleared: %q", sent)
		}
	}
	fr = tuitest.Frame(m)
	if strings.Contains(fr, "SECRET") || strings.Contains(base64.StdEncoding.EncodeToString([]byte(fr)), "SECRET") || !strings.Contains(fr, "stored relay-token") {
		t.Fatalf("after:\n%s", fr)
	}
	for _, e := range m.(*App).log.ev {
		if strings.Contains(e.Text, "SECRET") {
			t.Fatalf("the log has the secret: %q", e.Text)
		}
	}
}

func TestVaultPutRefusesAnUnknownName(t *testing.T) {
	f := tuitest.NewFake()
	m := press(t, typed(t, press(t, vaultApp(t, f, "linux"), "p"), "other"), "enter")
	if fr := tuitest.Frame(m); !strings.Contains(fr, "relay-token or admin-token") {
		t.Fatalf("no refusal:\n%s", fr)
	}
	if _, ok := m.(*App).modal.(*askModal); !ok {
		t.Fatalf("the name prompt closed: %T", m.(*App).modal)
	}
	if f.Called("Do PUT") {
		t.Fatalf("sent: %v", f.Calls)
	}
}

func TestVaultIdleLock(t *testing.T) {
	f := tuitest.NewFake()
	m := press(t, vaultApp(t, f, "linux"), "t", "backspace", "backspace")
	m = press(t, typed(t, m, "9999"), "enter")
	if fr := tuitest.Frame(m); !strings.Contains(fr, "0 to 1440") || f.Called("SaveSettings") {
		t.Fatalf("9999 went through:\n%s", fr)
	}
	m = press(t, m, "backspace", "backspace", "backspace", "backspace")
	m = press(t, typed(t, m, "45"), "enter")
	if f.LastUpdate.VaultIdleLockMinutes == nil || *f.LastUpdate.VaultIdleLockMinutes != 45 {
		t.Fatalf("saved %+v", f.LastUpdate)
	}
	if fr := tuitest.Frame(m); !strings.Contains(fr, "idle lock  45 min") {
		t.Fatalf("not shown:\n%s", fr)
	}
}

func TestVaultUnlockAndLockKeys(t *testing.T) {
	f := tuitest.NewFake()
	m := press(t, vaultApp(t, f, "linux"), "u")
	if _, ok := m.(*App).modal.(*unlockModal); !ok {
		t.Fatalf("u: %T", m.(*App).modal)
	}
	m = press(t, m, "esc", "l")
	if !f.Called("Lock") {
		t.Fatalf("l did not lock: %v", f.Calls)
	}
}
