// internal/bootstrap/bootstrap_test.go
package bootstrap

import (
	"context"
	"crypto/ed25519"
	"crypto/rand"
	"errors"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/valve-tech/jumpgate/internal/eip712"
	"github.com/valve-tech/jumpgate/internal/executor"
	"golang.org/x/crypto/ssh"
)

// fakeBox is a scripted target: each rule maps a command substring to a
// result, and written files are kept in memory.
type fakeBox struct {
	rules  []rule
	cmds   []string
	files  map[string]string
	stdins []string
	writes []string
	// readErr, when set, is what ReadFile returns for an existing file.
	readErr error
}

// testKey returns a fresh valid public key line and its base64 blob.
func testKey(t *testing.T) (line, blob string) {
	t.Helper()
	pub, _, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	sp, err := ssh.NewPublicKey(pub)
	if err != nil {
		t.Fatal(err)
	}
	line = strings.TrimSpace(string(ssh.MarshalAuthorizedKey(sp)))
	return line, strings.Fields(line)[1]
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
	if p, ok := strings.CutPrefix(cmd, "test -e "); ok {
		if _, has := b.files[p]; has {
			return executor.Result{}, nil
		}
		return executor.Result{ExitCode: 1}, nil
	}
	if strings.Contains(cmd, "mv -f "+AuthorizedKeys+".new "+AuthorizedKeys) {
		b.files[AuthorizedKeys] = b.files[AuthorizedKeys+".new"]
		delete(b.files, AuthorizedKeys+".new")
		return executor.Result{}, nil
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
	b.writes = append(b.writes, path)
	return nil
}

func (b *fakeBox) ReadFile(_ context.Context, path string) ([]byte, error) {
	if c, ok := b.files[path]; ok {
		if b.readErr != nil {
			return nil, b.readErr
		}
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
	key, _ := testKey(t)
	return optsWithKey(t, box, key)
}

func optsWithKey(t *testing.T, box *fakeBox, key string) Options {
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
		Controller: ctrl, ControllerLabel: "laptop", TransportKey: key,
	}
}

func TestRunPairsAFreshBox(t *testing.T) {
	box := freshBox()
	key, _ := testKey(t)
	got, err := Run(context.Background(), optsWithKey(t, box, key))
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
	if !strings.Contains(box.files[AuthorizedKeys], `restrict,port-forwarding,command="/bin/false" `+key) {
		t.Errorf("authorized_keys = %q", box.files[AuthorizedKeys])
	}
	if !strings.Contains(box.files[DropInPath], "AllowStreamLocalForwarding local") {
		t.Errorf("drop-in = %q", box.files[DropInPath])
	}
}

// OpenSSH gates direct-streamlocal on the same local-forward permission set as
// TCP: `AllowTcpForwarding no` refuses the agent socket too. The drop-in must
// allow local forwarding and pin it to the agent socket alone.
func TestDropInAdmitsOnlyTheAgentSocket(t *testing.T) {
	box := freshBox()
	if _, err := Run(context.Background(), opts(t, box)); err != nil {
		t.Fatal(err)
	}
	d := box.files[DropInPath]
	if strings.Contains(d, "AllowTcpForwarding no") {
		t.Errorf("AllowTcpForwarding no also refuses direct-streamlocal: %q", d)
	}
	for _, want := range []string{"AllowTcpForwarding local", "PermitOpen [/run/jumpgate/agent.sock]:*"} {
		if !strings.Contains(d, want) {
			t.Errorf("drop-in lacks %q: %q", want, d)
		}
	}
}

// Review Focus 4: re-pairing or pairing a second controller appends; it never
// drops an existing key.
func TestRunKeepsExistingAuthorizedKeys(t *testing.T) {
	box := freshBox()
	other, otherBlob := testKey(t)
	box.files = map[string]string{AuthorizedKeys: `restrict,port-forwarding,command="/bin/false" ` + other + " other-controller\n"}
	key, blob := testKey(t)
	if _, err := Run(context.Background(), optsWithKey(t, box, key)); err != nil {
		t.Fatal(err)
	}
	ak := box.files[AuthorizedKeys]
	if !strings.Contains(ak, otherBlob) || !strings.Contains(ak, blob) {
		t.Fatalf("authorized_keys = %q", ak)
	}
	// And pairing the same controller again adds nothing.
	if _, err := Run(context.Background(), optsWithKey(t, box, key)); err != nil {
		t.Fatal(err)
	}
	if strings.Count(box.files[AuthorizedKeys], blob) != 1 {
		t.Fatalf("duplicated key: %q", box.files[AuthorizedKeys])
	}
}

// A failed read of an existing authorized_keys must stop the step, never be
// taken for "no keys" and overwrite another controller's key.
func TestRunRefusesToOverwriteAnUnreadableAuthorizedKeys(t *testing.T) {
	box := freshBox()
	other, _ := testKey(t)
	box.files = map[string]string{AuthorizedKeys: other + "\n"}
	box.readErr = errors.New("garbled base64")
	_, err := Run(context.Background(), opts(t, box))
	var se *StepError
	if !errors.As(err, &se) || se.Step != "sshd" {
		t.Fatalf("err = %v, want a StepError for sshd", err)
	}
	for _, w := range box.writes {
		if strings.HasPrefix(w, AuthorizedKeys) {
			t.Fatalf("wrote %s after a failed read", w)
		}
	}
	if box.files[AuthorizedKeys] != other+"\n" {
		t.Fatalf("authorized_keys changed: %q", box.files[AuthorizedKeys])
	}
}

// The final authorized_keys must never be root-owned: stage, chown, rename.
func TestAuthorizedKeysIsStagedAndOwnedBeforeRename(t *testing.T) {
	box := freshBox()
	if _, err := Run(context.Background(), opts(t, box)); err != nil {
		t.Fatal(err)
	}
	for _, w := range box.writes {
		if w == AuthorizedKeys {
			t.Fatal("authorized_keys written in place as root")
		}
	}
	for _, c := range box.cmds {
		if strings.Contains(c, "mv -f "+AuthorizedKeys+".new") {
			i, j := strings.Index(c, "chown jumpgate:jumpgate "+AuthorizedKeys+".new"), strings.Index(c, "mv -f")
			if i < 0 || i > j {
				t.Fatalf("rename before chown: %s", c)
			}
			return
		}
	}
	t.Fatal("never renamed the staged file")
}

// Re-pairing repairs ownership even when the key is already present.
func TestRepairRepairsOwnershipWhenKeyAlreadyPresent(t *testing.T) {
	box := freshBox()
	key, _ := testKey(t)
	box.files = map[string]string{AuthorizedKeys: `restrict,port-forwarding,command="/bin/false" ` + key + "\n"}
	if _, err := Run(context.Background(), optsWithKey(t, box, key)); err != nil {
		t.Fatal(err)
	}
	all := strings.Join(box.cmds, "\n")
	for _, want := range []string{"install -d -m 0700 -o jumpgate -g jumpgate", "chown jumpgate:jumpgate " + AuthorizedKeys, "chmod 0600 " + AuthorizedKeys} {
		if !strings.Contains(all, want) {
			t.Errorf("never ran %q", want)
		}
	}
}

func TestTransportKeyMustBeOneBareKey(t *testing.T) {
	k1, _ := testKey(t)
	k2, _ := testKey(t)
	for name, key := range map[string]string{
		"two lines":   k1 + "\n" + k2,
		"options":     `command="/bin/true" ` + k1,
		"garbage":     "ssh-ed25519 AAAATEST",
		"empty":       "",
		"embedded nl": k1 + "\nrestrict " + k2,
	} {
		box := freshBox()
		_, err := Run(context.Background(), optsWithKey(t, box, key))
		var se *StepError
		if !errors.As(err, &se) || se.Step != "sshd" {
			t.Errorf("%s: err = %v, want sshd StepError", name, err)
		}
		if box.files[AuthorizedKeys] != "" || box.files[AuthorizedKeys+".new"] != "" {
			t.Errorf("%s: wrote authorized_keys", name)
		}
	}
}

func TestTransportKeyIsWrittenCanonically(t *testing.T) {
	box := freshBox()
	k, _ := testKey(t)
	if _, err := Run(context.Background(), optsWithKey(t, box, k+" laptop comment\n")); err != nil {
		t.Fatal(err)
	}
	if want := `restrict,port-forwarding,command="/bin/false" ` + k + "\n"; box.files[AuthorizedKeys] != want {
		t.Fatalf("authorized_keys = %q, want %q", box.files[AuthorizedKeys], want)
	}
}

// A drop-in from an earlier good pairing survives a failed sshd -t.
func TestAFailedSSHDCheckRestoresAnExistingDropIn(t *testing.T) {
	box := freshBox()
	box.rules = append([]rule{{"sshd -t", executor.Result{ExitCode: 255, Stderr: "bad"}}}, box.rules...)
	if _, err := Run(context.Background(), opts(t, box)); err == nil {
		t.Fatal("want an error")
	}
	backup := DropInPath + ".jumpgate-bak"
	var backedUp, restored bool
	for _, c := range box.cmds {
		if strings.Contains(c, "cp -p "+DropInPath+" "+backup) {
			backedUp = true
		}
		if strings.Contains(c, "then mv -f "+backup+" "+DropInPath+"; else rm -f "+DropInPath+"; fi") {
			restored = true
		}
	}
	if !backedUp || !restored {
		t.Fatalf("backup %v restore %v; cmds:\n%s", backedUp, restored, strings.Join(box.cmds, "\n"))
	}
	// Order: backup, then write, then sshd -t, then restore.
	idx := func(sub string) int {
		for i, c := range box.cmds {
			if strings.Contains(c, sub) {
				return i
			}
		}
		return -1
	}
	if !(idx("cp -p") < idx("sshd -t") && idx("sshd -t") < idx("then mv -f "+backup)) {
		t.Fatalf("wrong order:\n%s", strings.Join(box.cmds, "\n"))
	}
}

func TestUserStepCreatesTheParentDirectoryFirst(t *testing.T) {
	box := freshBox()
	if _, err := Run(context.Background(), opts(t, box)); err != nil {
		t.Fatal(err)
	}
	for _, c := range box.cmds {
		if strings.Contains(c, "useradd") {
			if !strings.HasPrefix(c, "install -d -m 0755 /var/lib/jumpgate;") {
				t.Fatalf("useradd without the parent dir: %s", c)
			}
			return
		}
	}
	t.Fatal("no useradd")
}

func TestAnInactiveServiceGetsTheJournalHint(t *testing.T) {
	box := freshBox()
	box.rules = append([]rule{{"is-active", executor.Result{Stdout: "inactive\n", ExitCode: 3}}}, box.rules...)
	_, err := Run(context.Background(), opts(t, box))
	var se *StepError
	if !errors.As(err, &se) || se.Step != "service" || !strings.Contains(err.Error(), "journalctl -u jumpgate-agent") || !strings.Contains(err.Error(), "inactive") {
		t.Fatalf("err = %v", err)
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

// M9: a failed pairing says what it left on the box, step by step, so the
// operator knows what a re-run will find (and what to remove by hand).
func TestAFailedStepSaysWhatWasLeftInPlace(t *testing.T) {
	cases := []struct {
		step  string
		fail  rule
		local bool
		want  []string
		not   []string
	}{
		{"preflight", rule{"uname -s", executor.Result{Stdout: "Darwin\n"}}, false, []string{"nothing"}, nil},
		{"upload", rule{"sha256sum -c", executor.Result{ExitCode: 1}}, false, []string{BinaryPath + ".new"}, nil},
		{"user", rule{"useradd", executor.Result{ExitCode: 1}}, false, []string{BinaryPath}, []string{DropInPath}},
		{"sshd", rule{"sshd -t", executor.Result{ExitCode: 255}}, false, []string{BinaryPath, "jumpgate user", "restored"}, nil},
		{"identity", rule{"agent init", executor.Result{ExitCode: 1}}, false, []string{BinaryPath, DropInPath, AuthorizedKeys}, []string{"agent key"}},
		{"identity", rule{"agent init", executor.Result{ExitCode: 1}}, true, []string{BinaryPath}, []string{DropInPath, AuthorizedKeys}},
		{"policy", rule{"agent enroll", executor.Result{ExitCode: 1}}, false, []string{"agent key", "/var/lib/jumpgate"}, []string{"policy.json"}},
		{"service", rule{"is-active", executor.Result{Stdout: "failed\n", ExitCode: 3}}, false, []string{"policy.json", NodePath, UnitPath}, nil},
	}
	for _, c := range cases {
		box := freshBox()
		box.rules = append([]rule{c.fail}, box.rules...)
		o := opts(t, box)
		if c.local {
			o.Local, o.TransportKey, o.LocalUID = true, "", 1000
		}
		_, err := Run(context.Background(), o)
		var se *StepError
		if !errors.As(err, &se) || se.Step != c.step {
			t.Errorf("%s: err = %v", c.step, err)
			continue
		}
		msg := err.Error()
		_, left, ok := strings.Cut(msg, "left in place: ")
		if !ok {
			t.Errorf("%s (local %v): no left-in-place summary: %q", c.step, c.local, msg)
			continue
		}
		for _, w := range c.want {
			if !strings.Contains(left, w) {
				t.Errorf("%s (local %v): %q does not mention %q", c.step, c.local, left, w)
			}
		}
		for _, n := range c.not {
			if strings.Contains(left, n) {
				t.Errorf("%s (local %v): %q mentions %q", c.step, c.local, left, n)
			}
		}
	}
}
