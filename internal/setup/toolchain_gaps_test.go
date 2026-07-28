package setup

// The toolchain paths the existing tests do not reach: the rust install, and
// every way an install can fail. A toolchain step that swallowed a failure
// would hand the next step a machine with no compiler, and the build that then
// fails names a missing header rather than the missing toolchain.

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/valve-tech/valve-node-app/internal/executor"
)

// present marks a command as installed; commandExists probes with `command -v`.
func present(e *fakeExecutor, cmds ...string) *fakeExecutor {
	for _, c := range cmds {
		e.script("command -v "+c, executor.Result{ExitCode: 0, Stdout: "/usr/bin/" + c + "\n"})
	}
	return e
}

func absent(e *fakeExecutor, cmds ...string) *fakeExecutor {
	for _, c := range cmds {
		e.script("command -v "+c, executor.Result{ExitCode: 1})
	}
	return e
}

// reth is built with cargo, so a machine without it gets rustup — not apt,
// which on Debian stable ships a rust too old to build reth at all.
func TestToolchain_MissingCargoIsInstalledWithRustupNotApt(t *testing.T) {
	e := present(newFakeExecutor(), "git", "cc", "apt-get")
	absent(e, "cargo")
	e.script("cargo --version", executor.Result{ExitCode: 1})

	if err := toolchainStep([]string{"rust"}).Run(context.Background(), e, &State{Wire: testWire()}); err != nil {
		t.Fatalf("Run: %v", err)
	}
	if !ran(e, "rustup.rs") {
		t.Errorf("cargo was missing and rustup never ran; ran: %q", e.callLog())
	}
	if ran(e, "apt-get install", "rust") {
		t.Error("rust was installed from apt, which ships a version too old to build reth")
	}
}

// A rustup install that failed must stop the plan. Continuing would run the
// build step against a machine with no cargo, and the operator would be shown
// a compiler error instead of "rust could not be installed".
func TestToolchain_AFailedRustInstallStopsThePlan(t *testing.T) {
	e := present(newFakeExecutor(), "git", "cc", "apt-get")
	absent(e, "cargo")
	e.script("cargo --version", executor.Result{ExitCode: 1})
	e.script("rustup.rs", executor.Result{ExitCode: 1, Stderr: "curl: (6) Could not resolve host: sh.rustup.rs\n"})

	err := toolchainStep([]string{"rust"}).Run(context.Background(), e, &State{Wire: testWire()})
	if err == nil {
		t.Fatal("a failed rustup install reported success")
	}
	if !strings.Contains(err.Error(), "Could not resolve host") {
		t.Errorf("error %q does not carry what the target said", err)
	}
}

func TestToolchain_AFailedAptInstallStopsThePlan(t *testing.T) {
	e := present(newFakeExecutor(), "apt-get")
	absent(e, "git")
	e.script("apt-get update", executor.Result{ExitCode: 100, Stderr: "E: Unable to locate package git\n"})

	err := toolchainStep([]string{"go"}).Run(context.Background(), e, &State{Wire: testWire()})
	if err == nil {
		t.Fatal("a failed apt install reported success")
	}
	if !strings.Contains(err.Error(), "Unable to locate package") {
		t.Errorf("error %q does not carry apt's own words", err)
	}
}

// A transport failure while probing is not "the tool is missing" — installing
// on top of a dropped connection would be acting on an unknown state.
func TestToolchain_ATransportFailureWhileProbingIsNotTreatedAsAbsence(t *testing.T) {
	e := newFakeExecutor().errOn("command -v git", errors.New("ssh: connection lost"))

	err := toolchainStep([]string{"go"}).Run(context.Background(), e, &State{Wire: testWire()})
	if err == nil {
		t.Fatal("a dropped connection reported success")
	}
	if !strings.Contains(err.Error(), "connection lost") {
		t.Errorf("error %q loses the transport failure", err)
	}
	if ran(e, "apt-get install") {
		t.Error("an install ran on top of a probe that never answered")
	}
}

// An unrecognized toolchain name is a programming error in the catalog, and it
// must be loud: silently skipping it produces a plan that builds nothing and
// reports success.
func TestToolchain_VerifyRefusesAToolchainItDoesNotKnow(t *testing.T) {
	e := newFakeExecutor().
		script("git --version", executor.Result{ExitCode: 0}).
		script("cc --version", executor.Result{ExitCode: 0})

	err := verifyToolchains(context.Background(), e, []string{"zig"})
	if err == nil {
		t.Fatal("an unknown toolchain verified as satisfied")
	}
	if !strings.Contains(err.Error(), "zig") {
		t.Errorf("error %q does not name the toolchain", err)
	}
}

// Verify's git check has to be able to fail on its own — it is the one tool
// every build path needs.
func TestToolchain_VerifyFailsWhenGitIsMissing(t *testing.T) {
	e := newFakeExecutor().script("git --version", executor.Result{ExitCode: 127})

	if err := verifyToolchains(context.Background(), e, []string{"go"}); err == nil {
		t.Fatal("verify passed with no git on the target")
	}
}

// A cargo installed by rustup is not on a fresh shell's PATH until its env
// file is sourced. Verifying without that prefix reports "cargo not available"
// on a machine where the install just succeeded — a plan that can never
// converge.
func TestToolchain_VerifySourcesRustupsEnvBeforeLookingForCargo(t *testing.T) {
	e := newFakeExecutor().
		script("git --version", executor.Result{ExitCode: 0}).
		script("cc --version", executor.Result{ExitCode: 0}).
		script("cargo --version", executor.Result{ExitCode: 0})

	if err := verifyToolchains(context.Background(), e, []string{"rust"}); err != nil {
		t.Fatalf("verify: %v", err)
	}
	var probe string
	for _, c := range e.callLog() {
		if strings.Contains(c, "cargo --version") {
			probe = c
		}
	}
	if !strings.Contains(probe, cargoEnvPrefix) {
		t.Errorf("the cargo probe does not source rustup's env first: %q", probe)
	}
}
