package setup

// Every way the toolchain step can fail on the WIRE, as opposed to on an exit
// code. The distinction is the point: a dropped SSH connection while probing
// `command -v go` must not be read as "go is not installed", because the
// repair for the two is different and the wrong one installs a compiler over
// a link that is already broken.

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/valve-tech/valve-node-app/internal/executor"
)

// A transport failure at any probe or install must surface as itself, wrapping
// the cause, rather than being folded into "not present" and repaired.
func TestToolchain_RunSurfacesEveryTransportFailure(t *testing.T) {
	boom := errors.New("ssh: connection lost")

	tests := []struct {
		name    string
		needed  []string
		prime   func(*fakeExecutor)
		breakOn string
		wantSay string
	}{
		{
			name:    "probing cc",
			needed:  []string{"go"},
			prime:   func(e *fakeExecutor) { present(e, "git", "apt-get") },
			breakOn: "command -v cc",
			wantSay: "probe cc",
		},
		{
			name:    "probing go",
			needed:  []string{"go"},
			prime:   func(e *fakeExecutor) { present(e, "git", "cc", "apt-get") },
			breakOn: "command -v go",
			wantSay: "probe go",
		},
		{
			name:    "probing cargo",
			needed:  []string{"rust"},
			prime:   func(e *fakeExecutor) { present(e, "git", "cc", "apt-get") },
			breakOn: "command -v cargo",
			wantSay: "probe cargo",
		},
		{
			name:    "probing apt-get",
			needed:  []string{"go"},
			prime:   func(e *fakeExecutor) { present(e, "git", "cc"); absent(e, "go") },
			breakOn: "command -v apt-get",
			wantSay: "probe apt-get",
		},
		{
			name:   "installing with apt",
			needed: []string{"go"},
			prime: func(e *fakeExecutor) {
				present(e, "git", "cc", "apt-get")
				absent(e, "go")
			},
			breakOn: "apt-get update",
			wantSay: "golang-go",
		},
		{
			name:   "installing rust",
			needed: []string{"rust"},
			prime: func(e *fakeExecutor) {
				present(e, "git", "cc", "apt-get")
				absent(e, "cargo")
			},
			breakOn: "sh.rustup.rs",
			wantSay: "install rust",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			e := newFakeExecutor()
			tc.prime(e)
			e.errOn(tc.breakOn, boom)

			err := toolchainStep(tc.needed).Run(context.Background(), e, &State{Wire: testWire()})
			if err == nil {
				t.Fatalf("a broken link at %q was reported as success", tc.breakOn)
			}
			if !errors.Is(err, boom) {
				t.Errorf("error lost the transport cause: %v", err)
			}
			if !strings.Contains(err.Error(), tc.wantSay) {
				t.Errorf("error does not say what it was doing (want %q): %v", tc.wantSay, err)
			}
		})
	}
}

// The cc branch has its own apt guards, reached only when a toolchain needs a
// compiler and the target has none.
func TestToolchain_MissingCCOnANonAptTargetSaysSo(t *testing.T) {
	e := present(newFakeExecutor(), "git")
	absent(e, "cc", "apt-get")

	err := toolchainStep([]string{"go"}).Run(context.Background(), e, &State{Wire: testWire()})
	if err == nil {
		t.Fatal("a target with no compiler and no apt was accepted")
	}
	if !strings.Contains(err.Error(), "Debian/Ubuntu") {
		t.Errorf("error does not name the supported targets: %v", err)
	}
}

func TestToolchain_AFailedBuildEssentialInstallStopsThePlan(t *testing.T) {
	e := present(newFakeExecutor(), "git", "apt-get")
	absent(e, "cc")
	e.script("build-essential", executor.Result{ExitCode: 100, Stderr: "E: Unable to locate package"})

	err := toolchainStep([]string{"go"}).Run(context.Background(), e, &State{Wire: testWire()})
	if err == nil {
		t.Fatal("a failed build-essential install reported success — the next step would compile with no cc")
	}
	if !strings.Contains(err.Error(), "build-essential") {
		t.Errorf("error does not name the package: %v", err)
	}
	if !strings.Contains(err.Error(), "Unable to locate package") {
		t.Errorf("error drops apt's own explanation: %v", err)
	}
}

// Run refuses an unknown toolchain for the same reason Verify does: silently
// skipping it would report a satisfied machine that cannot build.
func TestToolchain_RunRefusesAToolchainItDoesNotKnow(t *testing.T) {
	e := present(newFakeExecutor(), "git", "cc", "apt-get")

	err := toolchainStep([]string{"zig"}).Run(context.Background(), e, &State{Wire: testWire()})
	if err == nil {
		t.Fatal("an unknown toolchain was silently skipped")
	}
	if !strings.Contains(err.Error(), "zig") {
		t.Errorf("error does not name the toolchain: %v", err)
	}
}

// ---------------------------------------------------------------------
// Verify
// ---------------------------------------------------------------------

// Verify's probes can drop the link too, and the same rule applies: a lost
// connection is not "the toolchain is missing".
func TestToolchain_VerifySurfacesEveryTransportFailure(t *testing.T) {
	boom := errors.New("ssh: connection lost")

	tests := []struct {
		name    string
		needed  []string
		breakOn string
		wantSay string
	}{
		{name: "git", needed: []string{"go"}, breakOn: "git --version", wantSay: "verify git"},
		{name: "cc", needed: []string{"go"}, breakOn: "cc --version", wantSay: "verify cc"},
		{name: "go", needed: []string{"go"}, breakOn: "go version", wantSay: "verify go"},
		{name: "cargo", needed: []string{"rust"}, breakOn: "cargo --version", wantSay: "verify cargo"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			e := newFakeExecutor().errOn(tc.breakOn, boom)

			err := toolchainStep(tc.needed).Verify(context.Background(), e, &State{Wire: testWire()})
			if err == nil {
				t.Fatalf("a broken link at %q verified clean", tc.breakOn)
			}
			if !errors.Is(err, boom) {
				t.Errorf("error lost the transport cause: %v", err)
			}
			if !strings.Contains(err.Error(), tc.wantSay) {
				t.Errorf("error does not say what it was verifying (want %q): %v", tc.wantSay, err)
			}
		})
	}
}

// Resolving on PATH is not the same as running. A `go` that exits non-zero is
// a broken install, and verify has to fail on it.
func TestToolchain_VerifyFailsWhenGoDoesNotRun(t *testing.T) {
	e := newFakeExecutor().script("go version", executor.Result{ExitCode: 1, Stderr: "cannot execute binary file"})

	err := toolchainStep([]string{"go"}).Verify(context.Background(), e, &State{Wire: testWire()})
	if err == nil {
		t.Fatal("a go that cannot run verified clean")
	}
	if !strings.Contains(err.Error(), "go not available") {
		t.Errorf("error does not name what is missing: %v", err)
	}
}
