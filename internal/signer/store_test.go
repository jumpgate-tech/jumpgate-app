// internal/signer/store_test.go
package signer

import (
	"context"
	"encoding/hex"
	"errors"
	"os"
	"strings"
	"testing"
)

// fakeRunner records commands. The property under test is that a key's hex
// never appears in any argument: every process's argv is readable by every
// local user.
type fakeRunner struct {
	calls  [][]string
	stdins []string
	files  map[string]string // template files read during a call
	out    string
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
		}
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
	withRunner(t, f, "darwin", "security")
	k, err := Create(context.Background(), StoreKeychain, "controller")
	if err != nil {
		t.Fatal(err)
	}
	secret := hex.EncodeToString(k.Bytes())
	assertNoSecretInArgv(t, f, secret)
	if !strings.Contains(f.stdins[0], secret) {
		t.Fatal("the key was not handed to `security -i` on stdin")
	}

	f.out = secret + "\n"
	got, err := Open(context.Background(), StoreKeychain, "controller")
	if err != nil || got.Address() != k.Address() {
		t.Fatalf("Open = %v, %v", got, err)
	}
}

func TestKeychainLinuxUsesSecretToolStdin(t *testing.T) {
	f := &fakeRunner{}
	withRunner(t, f, "linux", "secret-tool")
	k, err := Create(context.Background(), StoreKeychain, "controller")
	if err != nil {
		t.Fatal(err)
	}
	secret := hex.EncodeToString(k.Bytes())
	assertNoSecretInArgv(t, f, secret)
	if f.calls[0][0] != "secret-tool" || f.calls[0][1] != "store" || f.stdins[0] != secret {
		t.Fatalf("store call = %v stdin %q", f.calls[0], f.stdins[0])
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

	f.out = secret
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
	withRunner(t, f, "darwin", "security")
	if _, err := Create(context.Background(), StoreKeychain, "controller-1.test_x"); err != nil {
		t.Fatal(err)
	}
}
