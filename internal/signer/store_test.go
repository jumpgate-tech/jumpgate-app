// internal/signer/store_test.go
package signer

import (
	"context"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"strings"
	"testing"
)

// fakeRunner records commands. The property under test is that a key's hex
// never appears in any argument: every process's argv is readable by every
// local user.
type fakeRunner struct {
	calls   [][]string
	stdins  []string
	files   map[string]string // template files read during a call
	out     string
	modes   map[string]os.FileMode // template file modes seen during a call
	nStores int
	fn      func(stdin, name string, args []string) (string, error)
}

func (f *fakeRunner) run(_ context.Context, stdin string, name string, args ...string) (string, error) {
	f.calls = append(f.calls, append([]string{name}, args...))
	f.stdins = append(f.stdins, stdin)
	for i, a := range args {
		if a == "--template" && i+1 < len(args) {
			b, _ := os.ReadFile(args[i+1])
			if f.files == nil {
				f.files = map[string]string{}
			}
			f.files[args[i+1]] = string(b)
			if st, err := os.Stat(args[i+1]); err == nil {
				if f.modes == nil {
					f.modes = map[string]os.FileMode{}
				}
				f.modes[args[i+1]] = st.Mode().Perm()
			}
		}
	}
	if f.fn != nil {
		return f.fn(stdin, name, args)
	}
	return f.out, nil
}

func withRunner(t *testing.T, f *fakeRunner, goos string, have ...string) {
	t.Helper()
	oldRun, oldGOOS, oldLook := runCmd, hostOS, lookPath
	runCmd, hostOS = f.run, goos
	lookPath = func(name string) error {
		for _, h := range have {
			if h == name {
				return nil
			}
		}
		return errors.New("not found")
	}
	t.Cleanup(func() { runCmd, hostOS, lookPath = oldRun, oldGOOS, oldLook })
}

func assertNoSecretInArgv(t *testing.T, f *fakeRunner, secret string) {
	t.Helper()
	for _, c := range f.calls {
		if strings.Contains(strings.Join(c, " "), secret) {
			t.Fatalf("key material in argv: %v", c)
		}
	}
}

func TestKeychainMacOSKeepsTheKeyOffArgv(t *testing.T) {
	f := &fakeRunner{}
	f.fn = memStore(f)
	withRunner(t, f, "darwin", "security")
	k, err := Create(context.Background(), StoreKeychain, "controller")
	if err != nil {
		t.Fatal(err)
	}
	secret := hex.EncodeToString(k.Bytes())
	assertNoSecretInArgv(t, f, secret)
	handed := false
	for i, c := range f.calls {
		if len(c) == 2 && c[1] == "-i" && strings.Contains(f.stdins[i], secret) {
			handed = true
		}
		if strings.Contains(f.stdins[i], "-U") {
			t.Fatal("-U would replace an existing item")
		}
	}
	if !handed {
		t.Fatal("the key was not handed to `security -i` on stdin")
	}

	got, err := Open(context.Background(), StoreKeychain, "controller")
	if err != nil || got.Address() != k.Address() {
		t.Fatalf("Open = %v, %v", got, err)
	}
}

func TestKeychainLinuxUsesSecretToolStdin(t *testing.T) {
	f := &fakeRunner{}
	f.fn = memStore(f)
	withRunner(t, f, "linux", "secret-tool")
	k, err := Create(context.Background(), StoreKeychain, "controller")
	if err != nil {
		t.Fatal(err)
	}
	secret := hex.EncodeToString(k.Bytes())
	assertNoSecretInArgv(t, f, secret)
	var store []string
	for i, c := range f.calls {
		if len(c) > 1 && c[1] == "store" {
			store = c
			if f.stdins[i] != secret {
				t.Fatalf("store stdin %q", f.stdins[i])
			}
		}
	}
	if store == nil || store[0] != "secret-tool" {
		t.Fatalf("no secret-tool store call in %v", f.calls)
	}
}

func TestKeychainMissingIsAnErrorNotAFallback(t *testing.T) {
	withRunner(t, &fakeRunner{}, "linux")
	if _, err := Create(context.Background(), StoreKeychain, "controller"); !errors.Is(err, ErrNoKeychain) {
		t.Fatalf("err = %v, want ErrNoKeychain", err)
	}
}

func TestOnePasswordCreatesFromATemplateFileItThenDeletes(t *testing.T) {
	f := &fakeRunner{}
	f.fn = memStore(f)
	withRunner(t, f, "darwin", "op")
	k, err := Create(context.Background(), StoreOnePassword, "op://Private/jumpgate-controller/credential")
	if err != nil {
		t.Fatal(err)
	}
	secret := hex.EncodeToString(k.Bytes())
	assertNoSecretInArgv(t, f, secret)
	var tmpl string
	for path, body := range f.files {
		tmpl = body
		if _, err := os.Stat(path); !os.IsNotExist(err) {
			t.Errorf("template %s was left on disk", path)
		}
	}
	if !strings.Contains(tmpl, secret) || !strings.Contains(tmpl, "jumpgate-controller") {
		t.Fatalf("template = %q", tmpl)
	}

	got, err := Open(context.Background(), StoreOnePassword, "op://Private/jumpgate-controller/credential")
	if err != nil || got.Address() != k.Address() {
		t.Fatalf("Open = %v, %v", got, err)
	}
	last := f.calls[len(f.calls)-1]
	if strings.Join(last, " ") != "op read op://Private/jumpgate-controller/credential" {
		t.Fatalf("read call = %v", last)
	}
}

func TestOnePasswordRejectsAMalformedRef(t *testing.T) {
	withRunner(t, &fakeRunner{}, "darwin", "op")
	if _, err := Create(context.Background(), StoreOnePassword, "Private/jumpgate"); err == nil {
		t.Fatal("want an error for a ref without op://vault/item/field")
	}
}

func TestKeychainRejectsNamesThatCouldInjectIntoTheSecurityCommandLine(t *testing.T) {
	for _, name := range []string{"", "a b", "a\"b", "a'b", "a\nb", "a\rb", "a\\b", "a;b", "a$b", "a`b", "é", "a\x00b"} {
		f := &fakeRunner{}
		withRunner(t, f, "darwin", "security")
		if _, err := Create(context.Background(), StoreKeychain, name); err == nil {
			t.Errorf("name %q was accepted", name)
		}
		if len(f.calls) != 0 {
			t.Errorf("name %q reached the runner: %v", name, f.calls)
		}
	}
	f := &fakeRunner{}
	f.fn = memStore(f)
	withRunner(t, f, "darwin", "security")
	if _, err := Create(context.Background(), StoreKeychain, "controller-1.test_x"); err != nil {
		t.Fatal(err)
	}
}

// notFound is what each real tool reports for a missing item.
func notFound(tool string) error {
	switch tool {
	case "security":
		return &cmdError{Name: tool, ExitCode: 44, Stderr: "could not be found", Err: errors.New("exit status 44")}
	case "op":
		return &cmdError{Name: tool, ExitCode: 1, Stderr: `"x" isn't an item`, Err: errors.New("exit status 1")}
	}
	return &cmdError{Name: tool, ExitCode: 1, Err: errors.New("exit status 1")}
}

// memStore simulates the three backends in memory: stores on the write
// commands, errors like the real tools when an item is missing.
func memStore(f *fakeRunner) func(string, string, []string) (string, error) {
	items := map[string]string{}
	nextID := 0
	return func(stdin, name string, args []string) (string, error) {
		missing := notFound(name)
		switch {
		case name == "security" && args[0] == "-i":
			fs := strings.Fields(stdin)
			items["kc:"+fs[2]] = fs[len(fs)-1]
		case name == "security":
			v, ok := items["kc:"+args[2]]
			if !ok {
				return "", missing
			}
			if args[len(args)-1] == "-w" {
				return v + "\n", nil
			}
			return "", nil
		case name == "secret-tool" && args[0] == "store":
			items["kc:"+args[len(args)-1]] = stdin
		case name == "secret-tool":
			v, ok := items["kc:"+args[len(args)-1]]
			if !ok {
				return "", missing
			}
			return v, nil
		case name == "op" && args[0] == "item" && args[1] == "get":
			if _, ok := items["op:"+args[2]]; !ok {
				return "", missing
			}
		case name == "op" && args[0] == "item" && args[1] == "create":
			var tm struct {
				Title  string
				Fields []struct{ ID, Value string }
			}
			tpath := ""
			for i, a := range args {
				if a == "--template" {
					tpath = args[i+1]
				}
			}
			b, _ := os.ReadFile(tpath)
			if err := json.Unmarshal(b, &tm); err != nil {
				return "", err
			}
			items["op:"+tm.Title] = tm.Fields[0].Value
			items["opref:"+args[3]+"/"+tm.Title+"/"+tm.Fields[0].ID] = tm.Fields[0].Value
			nextID++
			id := fmt.Sprintf("id%d", nextID)
			items["opid:"+id] = tm.Title
			return `{"id":"` + id + `"}`, nil
		case name == "op" && args[0] == "item" && args[1] == "delete":
			title := items["opid:"+args[2]]
			delete(items, "op:"+title)
			for k := range items {
				if strings.HasPrefix(k, "opref:") && strings.Contains(k, "/"+title+"/") {
					delete(items, k)
				}
			}
		case name == "op" && args[0] == "read":
			v, ok := items["opref:"+strings.TrimPrefix(args[1], "op://")]
			if !ok {
				return "", missing
			}
			return v, nil
		}
		return "", nil
	}
}

func TestCreateNeverReplacesAnExistingKey(t *testing.T) {
	cases := []struct {
		store Store
		goos  string
		tool  string
		ref   string
	}{
		{StoreKeychain, "darwin", "security", "controller"},
		{StoreKeychain, "linux", "secret-tool", "controller"},
		{StoreOnePassword, "darwin", "op", "op://Private/jumpgate-controller/credential"},
	}
	for _, c := range cases {
		f := &fakeRunner{}
		f.fn = memStore(f)
		withRunner(t, f, c.goos, c.tool)
		first, err := Create(context.Background(), c.store, c.ref)
		if err != nil {
			t.Fatal(err)
		}
		before := len(f.calls)
		if _, err := Create(context.Background(), c.store, c.ref); !errors.Is(err, ErrKeyExists) {
			t.Fatalf("%s/%s: err = %v, want ErrKeyExists", c.store, c.goos, err)
		}
		for _, call := range f.calls[before:] {
			j := strings.Join(call, " ")
			if strings.Contains(j, "store") || strings.Contains(j, "item create") || strings.Contains(j, " -i") {
				t.Fatalf("%s: store command ran despite an existing key: %v", c.store, call)
			}
		}
		got, err := Open(context.Background(), c.store, c.ref)
		if err != nil || got.Address() != first.Address() {
			t.Fatalf("%s: original key was disturbed: %v %v", c.store, got, err)
		}
	}
}

func TestCreateFailsWhenTheReadBackIsEmptyOrDifferent(t *testing.T) {
	other, err := GenerateKey()
	if err != nil {
		t.Fatal(err)
	}
	otherHex := hex.EncodeToString(other.Bytes())
	cases := []struct {
		store Store
		goos  string
		tool  string
		ref   string
	}{
		{StoreKeychain, "darwin", "security", "controller"},
		{StoreKeychain, "linux", "secret-tool", "controller"},
		{StoreOnePassword, "darwin", "op", "op://Private/jumpgate-controller/credential"},
	}
	for _, c := range cases {
		for _, readBack := range []string{"", otherHex} {
			f := &fakeRunner{}
			// Existence checks fail (nothing there), the store "succeeds", and the
			// read-back returns the scripted value; for "" the read-back errors.
			f.fn = func(stdin, name string, args []string) (string, error) {
				isCheck := (name == "security" && args[0] == "find-generic-password" && args[len(args)-1] != "-w") ||
					(name == "secret-tool" && args[0] == "lookup" && f.nStores == 0) ||
					(name == "op" && args[0] == "item" && args[1] == "get")
				isRead := (name == "security" && args[0] == "find-generic-password") ||
					(name == "secret-tool" && args[0] == "lookup") || (name == "op" && args[0] == "read")
				switch {
				case isCheck && f.nStores == 0:
					return "", notFound(name)
				case isRead:
					if readBack == "" {
						return "", nil
					}
					return readBack, nil
				}
				if name == "secret-tool" && args[0] == "store" || name == "security" && args[0] == "-i" ||
					name == "op" && args[1] == "create" {
					f.nStores++
				}
				return "", nil
			}
			withRunner(t, f, c.goos, c.tool)
			if k, err := Create(context.Background(), c.store, c.ref); err == nil || k != nil {
				t.Fatalf("%s/%s readback %q: Create = %v, %v; want failure", c.store, c.goos, readBack, k, err)
			}
		}
	}
}

func TestLinuxKeychainFailureNamesTheFileStore(t *testing.T) {
	f := &fakeRunner{}
	f.fn = func(_, _ string, _ []string) (string, error) {
		return "", errors.New("Cannot autolaunch D-Bus without X11 $DISPLAY")
	}
	withRunner(t, f, "linux", "secret-tool")
	_, err := Create(context.Background(), StoreKeychain, "controller")
	if err == nil || !strings.Contains(err.Error(), "--store file") {
		t.Fatalf("err = %v, want mention of --store file", err)
	}
	_, err = Open(context.Background(), StoreKeychain, "controller")
	if err == nil || !strings.Contains(err.Error(), "--store file") {
		t.Fatalf("Open err = %v, want mention of --store file", err)
	}
}

func TestOnePasswordTemplateIs0600AndRemovedWhenOpFails(t *testing.T) {
	f := &fakeRunner{}
	mem := memStore(f)
	f.fn = func(stdin, name string, args []string) (string, error) {
		if name == "op" && args[1] == "create" {
			return "", errors.New("not signed in")
		}
		return mem(stdin, name, args)
	}
	withRunner(t, f, "darwin", "op")
	if _, err := Create(context.Background(), StoreOnePassword, "op://Private/jumpgate-controller/credential"); err == nil {
		t.Fatal("want an error when op fails")
	}
	if len(f.modes) != 1 {
		t.Fatalf("template files seen: %v", f.modes)
	}
	for path, mode := range f.modes {
		if mode != 0o600 {
			t.Errorf("template mode = %o, want 600", mode)
		}
		if _, err := os.Stat(path); !os.IsNotExist(err) {
			t.Errorf("template %s left on disk after op failed", path)
		}
	}
}

type storeCase struct {
	store Store
	goos  string
	tool  string
	ref   string
}

var allStoreCases = []storeCase{
	{StoreKeychain, "darwin", "security", "controller"},
	{StoreKeychain, "linux", "secret-tool", "controller"},
	{StoreOnePassword, "darwin", "op", "op://Private/jumpgate-controller/credential"},
}

func isWrite(c []string) bool {
	j := strings.Join(c, " ")
	return strings.HasPrefix(j, "security -i") || strings.HasPrefix(j, "secret-tool store") || strings.Contains(j, "item create")
}

func TestExistenceCheckFailureThatIsNotNotFoundAbortsCreate(t *testing.T) {
	transient := map[string]error{
		"security":    &cmdError{Name: "security", ExitCode: 36, Stderr: "User interaction is not allowed", Err: errors.New("exit status 36")},
		"secret-tool": &cmdError{Name: "secret-tool", ExitCode: 1, Stderr: "Cannot autolaunch D-Bus", Err: errors.New("exit status 1")},
		"op":          &cmdError{Name: "op", ExitCode: 1, Stderr: "You are not signed in", Err: errors.New("exit status 1")},
	}
	for _, c := range allStoreCases {
		f := &fakeRunner{}
		f.fn = func(_, _ string, _ []string) (string, error) { return "", transient[c.tool] }
		withRunner(t, f, c.goos, c.tool)
		_, err := Create(context.Background(), c.store, c.ref)
		if err == nil || errors.Is(err, ErrKeyExists) || !strings.Contains(err.Error(), "existence check failed") {
			t.Fatalf("%s/%s: err = %v", c.store, c.goos, err)
		}
		for _, call := range f.calls {
			if isWrite(call) {
				t.Fatalf("%s/%s: store command ran: %v", c.store, c.goos, call)
			}
		}
	}
}

func TestRecognisedNotFoundLetsCreateProceed(t *testing.T) {
	for _, c := range allStoreCases {
		f := &fakeRunner{}
		f.fn = memStore(f)
		withRunner(t, f, c.goos, c.tool)
		if _, err := Create(context.Background(), c.store, c.ref); err != nil {
			t.Fatalf("%s/%s: %v", c.store, c.goos, err)
		}
	}
}

func TestOnePasswordRollsBackExactlyTheCreatedItem(t *testing.T) {
	f := &fakeRunner{}
	mem := memStore(f)
	f.fn = func(stdin, name string, args []string) (string, error) {
		if name == "op" && args[0] == "read" {
			return "", notFound("op") // the read-back fails
		}
		return mem(stdin, name, args)
	}
	withRunner(t, f, "darwin", "op")
	k, err := Create(context.Background(), StoreOnePassword, "op://Private/jumpgate-controller/credential")
	if err == nil || k != nil {
		t.Fatalf("Create = %v, %v", k, err)
	}
	var deletes [][]string
	for _, c := range f.calls {
		if len(c) > 2 && c[1] == "item" && c[2] == "delete" {
			deletes = append(deletes, c)
		}
	}
	if len(deletes) != 1 || strings.Join(deletes[0], " ") != "op item delete id1 --vault Private" {
		t.Fatalf("deletes = %v", deletes)
	}
}

func TestOnePasswordReportsTheIdWhenRollbackFails(t *testing.T) {
	f := &fakeRunner{}
	mem := memStore(f)
	f.fn = func(stdin, name string, args []string) (string, error) {
		if name == "op" && args[0] == "read" {
			return "", notFound("op")
		}
		if name == "op" && args[1] == "delete" {
			return "", errors.New("vault locked")
		}
		return mem(stdin, name, args)
	}
	withRunner(t, f, "darwin", "op")
	_, err := Create(context.Background(), StoreOnePassword, "op://Private/jumpgate-controller/credential")
	if err == nil || !strings.Contains(err.Error(), "id1") || !strings.Contains(err.Error(), "by hand") {
		t.Fatalf("err = %v, want the item id and a by-hand instruction", err)
	}
}

func TestKeychainVerifyFailureSaysWhatWasWritten(t *testing.T) {
	f := &fakeRunner{}
	mem := memStore(f)
	f.fn = func(stdin, name string, args []string) (string, error) {
		if name == "security" && args[0] == "find-generic-password" && args[len(args)-1] == "-w" {
			return "", notFound("security")
		}
		return mem(stdin, name, args)
	}
	withRunner(t, f, "darwin", "security")
	_, err := Create(context.Background(), StoreKeychain, "controller")
	if err == nil || !strings.Contains(err.Error(), "was written to keychain item") {
		t.Fatalf("err = %v", err)
	}
}

func TestPlainNotFoundOnLinuxReadHasNoServiceHint(t *testing.T) {
	f := &fakeRunner{}
	f.fn = memStore(f)
	withRunner(t, f, "linux", "secret-tool")
	_, err := Open(context.Background(), StoreKeychain, "nope")
	if err == nil || strings.Contains(err.Error(), "Secret Service") {
		t.Fatalf("err = %v", err)
	}
}

// realRunCmd is the production runner, captured before any test swaps runCmd.
var realRunCmd = runCmd

// The fakes in the other tests build *cmdError by hand; this one runs real
// subprocesses so the production runner is what produces the error type the
// not-found checks depend on.
func TestRealRunnerReportsExitCodeAndStderr(t *testing.T) {
	cases := []struct {
		script     string
		code       int
		stderr     string
		darwinMiss bool
		linuxMiss  bool
	}{
		{"exit 44", 44, "", true, false},
		{"exit 1", 1, "", false, true},
		{"echo boom >&2; exit 2", 2, "boom", false, false},
		{"echo secretvalue; echo oops >&2; exit 1", 1, "oops", false, false},
	}
	for _, c := range cases {
		_, err := realRunCmd(context.Background(), "", "sh", "-c", c.script)
		var ce *cmdError
		if !errors.As(err, &ce) {
			t.Fatalf("%q: err = %T %v, want *cmdError", c.script, err, err)
		}
		if ce.ExitCode != c.code || strings.TrimSpace(ce.Stderr) != c.stderr {
			t.Errorf("%q: code %d stderr %q", c.script, ce.ExitCode, ce.Stderr)
		}
		if strings.Contains(err.Error(), "secretvalue") {
			t.Errorf("stdout leaked into the error text: %v", err)
		}
		if got := keychainNotFound("security", err); got != c.darwinMiss {
			t.Errorf("%q: darwin not-found = %v", c.script, got)
		}
		if got := keychainNotFound("secret-tool", err); got != c.linuxMiss {
			t.Errorf("%q: linux not-found = %v", c.script, got)
		}
	}
	if _, err := realRunCmd(context.Background(), "", "jumpgate-no-such-tool-xyz"); err == nil {
		t.Fatal("want an error for a missing program")
	} else if ce := new(*cmdError); errors.As(err, ce) && (*ce).ExitCode != -1 {
		t.Errorf("exit code for a process that never started = %d, want -1", (*ce).ExitCode)
	}
	if out, err := realRunCmd(context.Background(), "in", "cat"); err != nil || out != "in" {
		t.Fatalf("success path = %q, %v", out, err)
	}
}
