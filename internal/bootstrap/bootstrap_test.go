// internal/bootstrap/bootstrap_test.go
package bootstrap

import (
	"context"
	"errors"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/valve-tech/jumpgate/internal/eip712"
	"github.com/valve-tech/jumpgate/internal/executor"
)

// fakeBox is a scripted target: each rule maps a command substring to a
// result, and written files are kept in memory.
type fakeBox struct {
	rules  []rule
	cmds   []string
	files  map[string]string
	stdins []string
}

type rule struct {
	match string
	res   executor.Result
}

func (b *fakeBox) Run(_ context.Context, cmd string, o *executor.RunOpts) (executor.Result, error) {
	b.cmds = append(b.cmds, cmd)
	if o != nil && o.Stdin != nil {
		x, _ := io.ReadAll(o.Stdin)
		b.stdins = append(b.stdins, string(x))
	}
	for _, r := range b.rules {
		if strings.Contains(cmd, r.match) {
			return r.res, nil
		}
	}
	return executor.Result{}, nil
}

func (b *fakeBox) WriteFile(_ context.Context, path string, content []byte, _ fs.FileMode) error {
	if b.files == nil {
		b.files = map[string]string{}
	}
	b.files[path] = string(content)
	return nil
}

func (b *fakeBox) ReadFile(_ context.Context, path string) ([]byte, error) {
	if c, ok := b.files[path]; ok {
		return []byte(c), nil
	}
	return nil, os.ErrNotExist
}

func (b *fakeBox) Close() error { return nil }

const agentAddrHex = "0x00000000000000000000000000000000000000aa"

func freshBox() *fakeBox {
	return &fakeBox{rules: []rule{
		{"uname -s", executor.Result{Stdout: "Linux\n"}},
		{"uname -m", executor.Result{Stdout: "x86_64\n"}},
		{"Include", executor.Result{}},                                  // drop-in include present
		{"sha256sum " + BinaryPath + " ", executor.Result{ExitCode: 1}}, // not installed yet
		{"agent init", executor.Result{Stdout: agentAddrHex + "\n"}},
		{"is-active", executor.Result{Stdout: "active\n"}},
	}}
}

func opts(t *testing.T, box *fakeBox) Options {
	t.Helper()
	bin := filepath.Join(t.TempDir(), "jumpgate-linux-amd64")
	_ = os.WriteFile(bin, []byte("binary"), 0o755)
	var ctrl eip712.Address
	ctrl[19] = 0xC
	return Options{
		Exec: box, AgentBinary: func(arch string) (string, error) {
			if arch != "amd64" {
				return "", errors.New("wrong arch " + arch)
			}
			return bin, nil
		},
		Controller: ctrl, ControllerLabel: "laptop", TransportKey: "ssh-ed25519 AAAATEST jumpgate-controller",
	}
}

func TestRunPairsAFreshBox(t *testing.T) {
	box := freshBox()
	got, err := Run(context.Background(), opts(t, box))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.EqualFold(got.Hex(), agentAddrHex) {
		t.Fatalf("agent address = %s", got.Hex())
	}
	all := strings.Join(box.cmds, "\n")
	for _, want := range []string{"groupadd --system jumpgate", "useradd --system", "sshd -t", "agent enroll --address 0x", "systemctl enable --now jumpgate-agent.service"} {
		if !strings.Contains(all, want) {
			t.Errorf("never ran %q", want)
		}
	}
	if !strings.Contains(box.files[AuthorizedKeys], `restrict,port-forwarding,command="/bin/false" ssh-ed25519 AAAATEST`) {
		t.Errorf("authorized_keys = %q", box.files[AuthorizedKeys])
	}
	if !strings.Contains(box.files[DropInPath], "AllowStreamLocalForwarding local") {
		t.Errorf("drop-in = %q", box.files[DropInPath])
	}
}

// Review Focus 4: re-pairing or pairing a second controller appends; it never
// drops an existing key.
func TestRunKeepsExistingAuthorizedKeys(t *testing.T) {
	box := freshBox()
	box.files = map[string]string{AuthorizedKeys: `restrict,port-forwarding,command="/bin/false" ssh-ed25519 AAAAOTHER other-controller` + "\n"}
	if _, err := Run(context.Background(), opts(t, box)); err != nil {
		t.Fatal(err)
	}
	ak := box.files[AuthorizedKeys]
	if !strings.Contains(ak, "AAAAOTHER") || !strings.Contains(ak, "AAAATEST") {
		t.Fatalf("authorized_keys = %q", ak)
	}
	// And pairing the same controller again adds nothing.
	if _, err := Run(context.Background(), opts(t, box)); err != nil {
		t.Fatal(err)
	}
	if strings.Count(box.files[AuthorizedKeys], "AAAATEST") != 1 {
		t.Fatalf("duplicated key: %q", box.files[AuthorizedKeys])
	}
}

func TestRunRemovesABadSSHDDropIn(t *testing.T) {
	box := freshBox()
	box.rules = append([]rule{{"sshd -t", executor.Result{ExitCode: 255, Stderr: "bad config"}}}, box.rules...)
	_, err := Run(context.Background(), opts(t, box))
	var se *StepError
	if !errors.As(err, &se) || se.Step != "sshd" {
		t.Fatalf("err = %v, want a StepError for sshd", err)
	}
	if !strings.Contains(strings.Join(box.cmds, "\n"), "rm -f "+DropInPath) {
		t.Fatal("a drop-in that failed sshd -t was left in place")
	}
}

func TestRunRefusesWithoutTheSSHDInclude(t *testing.T) {
	box := freshBox()
	box.rules = append([]rule{{"Include", executor.Result{ExitCode: 1}}}, box.rules...)
	_, err := Run(context.Background(), opts(t, box))
	var se *StepError
	if !errors.As(err, &se) || se.Step != "preflight" {
		t.Fatalf("err = %v, want preflight", err)
	}
}

func TestRunVerifiesTheUploadedBinary(t *testing.T) {
	box := freshBox()
	box.rules = append([]rule{{"sha256sum -c", executor.Result{ExitCode: 1, Stderr: "FAILED"}}}, box.rules...)
	_, err := Run(context.Background(), opts(t, box))
	var se *StepError
	if !errors.As(err, &se) || se.Step != "upload" {
		t.Fatalf("err = %v, want upload", err)
	}
}

func TestLocalSkipsSSHDAndEnrollsTheUID(t *testing.T) {
	box := freshBox()
	o := opts(t, box)
	o.Local, o.LocalUID, o.TransportKey = true, 501, ""
	if _, err := Run(context.Background(), o); err != nil {
		t.Fatal(err)
	}
	all := strings.Join(box.cmds, "\n")
	if strings.Contains(all, "sshd -t") {
		t.Error("local pairing touched sshd")
	}
	if !strings.Contains(all, "--local-uid 501") {
		t.Error("local uid not enrolled")
	}
}

func TestControllerLabelIsShellQuoted(t *testing.T) {
	box := freshBox()
	o := opts(t, box)
	o.ControllerLabel = `it's $(touch /tmp/x)`
	if _, err := Run(context.Background(), o); err != nil {
		t.Fatal(err)
	}
	want := `--label 'it'"'"'s $(touch /tmp/x)'`
	if !strings.Contains(strings.Join(box.cmds, "\n"), want) {
		t.Fatalf("label not quoted; cmds:\n%s", strings.Join(box.cmds, "\n"))
	}
}

func TestUploadStagesToNewAndVerifiesBeforeRename(t *testing.T) {
	box := freshBox()
	if _, err := Run(context.Background(), opts(t, box)); err != nil {
		t.Fatal(err)
	}
	if box.files[BinaryPath+".new"] != "binary" {
		t.Fatalf("binary not staged at .new: %v", box.files)
	}
	for _, c := range box.cmds {
		if strings.Contains(c, "sha256sum -c") {
			if strings.Index(c, "sha256sum -c") > strings.Index(c, "mv -f") {
				t.Fatalf("rename before verify: %s", c)
			}
			return
		}
	}
	t.Fatal("never verified the upload")
}

func TestSSHDValidationFindsSSHDOffPath(t *testing.T) {
	box := freshBox()
	if _, err := Run(context.Background(), opts(t, box)); err != nil {
		t.Fatal(err)
	}
	for _, c := range box.cmds {
		if strings.Contains(c, "sshd -t") {
			if !strings.Contains(c, "/usr/sbin") {
				t.Fatalf("sshd -t may not be on PATH under sudo: %s", c)
			}
			return
		}
	}
	t.Fatal("sshd -t never ran")
}
