package main

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/valve-tech/jumpgate/internal/executor"
	"github.com/valve-tech/jumpgate/internal/sysunit"
)

// Seams for tests; production never reassigns them.
var (
	// runPrivileged is the executor setup's root steps run through: this
	// process, which is already root.
	runPrivileged = func() executor.Executor { return executor.NewLocal() }
	pathExists    = func(p string) bool { _, err := os.Lstat(p); return err == nil }
	statOwner     = lstatOwner
)

// checkRootOwnedPath refuses a binary root would run that anyone but root
// could replace: bin must be a regular file, not a symlink, and it and every
// directory up to / must be root-owned and not group- or other-writable.
func checkRootOwnedPath(bin string) error {
	for p := bin; ; p = filepath.Dir(p) {
		uid, mode, err := statOwner(p)
		if err != nil {
			return fmt.Errorf("%s: %v", p, err)
		}
		switch {
		case mode&os.ModeSymlink != 0:
			return fmt.Errorf("%s is a symlink; give the real path", p)
		case p == bin && !mode.IsRegular():
			return fmt.Errorf("%s is not a regular file", p)
		case uid != 0:
			return fmt.Errorf("%s is not owned by root (uid %d)", p, uid)
		case mode.Perm()&0o022 != 0:
			return fmt.Errorf("%s is writable by its group or others (%v)", p, mode.Perm())
		}
		if p == filepath.Dir(p) {
			return nil
		}
	}
}

// setupStagingDir is root-only (0700), as the box modules use: a file bound
// for a root-owned path waits here under a random name, then is installed.
const setupStagingDir = "/var/lib/jumpgate/staging"

func cmdSetup(args []string) int {
	if len(args) == 0 {
		return usage("usage: jumpgate setup install-service [--bin PATH] | jumpgate setup server …")
	}
	switch args[0] {
	case "install-service":
		return setupInstallService(context.Background(), args[1:])
	case "server":
		return setupServer(context.Background(), args[1:])
	}
	return usage("unknown setup subcommand %q", args[0])
}

// setupInstallService writes jumpgate.service and its sysusers entry for a
// jumpgate installed without the package, creates the users, and reloads
// systemd. It never enables or starts the unit: `jumpgate setup server`
// does that once the key is in place.
func setupInstallService(ctx context.Context, args []string) int {
	fset := flag.NewFlagSet("setup install-service", flag.ContinueOnError)
	bin := fset.String("bin", "", "the jumpgate binary the service runs (default: this one)")
	if err := fset.Parse(args); err != nil || fset.NArg() != 0 {
		return usage("usage: jumpgate setup install-service [--bin PATH]")
	}
	if hostGOOS != "linux" {
		return failed("the jumpgate service runs only on Linux")
	}
	if geteuid() != 0 {
		return failed("installing the jumpgate service needs root; run `sudo jumpgate setup install-service`")
	}
	if pathExists(sysunit.PackageUnitPath) {
		return failed("the jumpgate package owns the service (%s); use apt or dnf to install or upgrade it", sysunit.PackageUnitPath)
	}
	if *bin == "" {
		exe, err := os.Executable()
		if err != nil {
			return failed("find this binary: %v", err)
		}
		if exe, err = filepath.EvalSymlinks(exe); err != nil {
			return failed("find this binary: %v", err)
		}
		*bin = exe
	}
	if !filepath.IsAbs(*bin) || strings.ContainsAny(*bin, " \t\n\"'\\%$") {
		return failed("--bin must be an absolute path with no spaces, quotes or %% signs: %q", *bin)
	}
	// The unit runs this binary for every boot; whoever can replace it, or
	// any directory above it, would run code as the service and then as
	// root on the next install.
	if err := checkRootOwnedPath(filepath.Clean(*bin)); err != nil {
		return failed("refusing to point jumpgate.service at %s: %v. Install jumpgate root-owned (for example `sudo install -o root -g root -m 0755 jumpgate /usr/local/bin/jumpgate`) and pass --bin /usr/local/bin/jumpgate", *bin, err)
	}

	ex := runPrivileged()
	defer ex.Close()
	run := func(cmd string) error {
		res, err := ex.Run(ctx, cmd, nil)
		if err != nil {
			return err
		}
		if res.ExitCode != 0 {
			return fmt.Errorf("%s: exit %d: %s", cmd, res.ExitCode, strings.TrimSpace(res.Stderr))
		}
		return nil
	}
	if err := run(fmt.Sprintf(`if [ -L /var/lib/jumpgate ] || [ -L %[1]s ]; then echo "refusing: a staging path is a symlink" >&2; false; fi && %[2]s`,
		setupStagingDir, executor.QuoteArgv([]string{"install", "-d", "-o", "root", "-g", "root", "-m", "0700", setupStagingDir}))); err != nil {
		return failed("staging dir: %v", err)
	}
	for _, f := range []struct{ dst, content string }{
		{sysunit.AdminUnitPath, sysunit.Unit(*bin)},
		{sysunit.AdminSysusersPath, sysunit.Sysusers},
	} {
		if err := installFile(ctx, ex, run, f.dst, f.content); err != nil {
			return failed("%s: %v", f.dst, err)
		}
		fmt.Printf("wrote %s\n", f.dst)
	}
	if err := run("systemd-sysusers " + sysunit.AdminSysusersPath); err != nil {
		return failed("create the jumpgate-ctl user: %v", err)
	}
	if err := run("systemctl daemon-reload"); err != nil {
		return failed("%v", err)
	}
	fmt.Println("installed jumpgate.service; next: `sudo jumpgate setup server`")
	return 0
}

// installFile stages content under a random name in the root-only staging
// dir, then installs it at dst, root-owned and 0644.
func installFile(ctx context.Context, ex executor.Executor, run func(string) error, dst, content string) error {
	var b [16]byte
	if _, err := rand.Read(b[:]); err != nil {
		return fmt.Errorf("staging name: %w", err)
	}
	staged := setupStagingDir + "/" + hex.EncodeToString(b[:])
	if err := ex.WriteFile(ctx, staged, []byte(content), 0o600); err != nil {
		return fmt.Errorf("stage: %w", err)
	}
	err := run(executor.QuoteArgv([]string{"install", "-D", "-o", "root", "-g", "root", "-m", "0644", staged, dst}))
	_ = run(executor.QuoteArgv([]string{"rm", "-f", staged}))
	return err
}
