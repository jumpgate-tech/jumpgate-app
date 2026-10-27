package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"os/user"
	"regexp"
	"slices"
	"strings"
	"time"

	"github.com/valve-tech/jumpgate/internal/api"
	"github.com/valve-tech/jumpgate/internal/apiclient"
	"github.com/valve-tech/jumpgate/internal/daemon"
	"github.com/valve-tech/jumpgate/internal/signer"
	"github.com/valve-tech/jumpgate/internal/sysunit"
)

// serviceStartWait is how long setup server waits for the service to
// publish system.json after starting it.
const serviceStartWait = 10 * time.Second

// Seams for tests; production never reassigns them.
var (
	// reexecSudo runs `sudo -- <this exe> setup server <args>` with this
	// terminal, and returns its exit status.
	reexecSudo = func(args []string) int {
		exe, err := os.Executable()
		if err != nil {
			return failed("find this binary: %v", err)
		}
		c := exec.Command("sudo", append([]string{"--", exe, "setup", "server"}, args...)...)
		c.Stdin, c.Stdout, c.Stderr = os.Stdin, os.Stdout, os.Stderr
		if err := c.Run(); err != nil {
			var ee *exec.ExitError
			if errors.As(err, &ee) {
				return ee.ExitCode()
			}
			return failed("sudo: %v", err)
		}
		return 0
	}
	// connectService waits for the jumpgate service to publish system.json
	// and returns a client of its admin socket.
	connectService = func(ctx context.Context) (*apiclient.Client, error) {
		deadline := time.Now().Add(serviceStartWait)
		for {
			info, ok, err := daemon.FindSystem(ctx)
			if ok {
				return apiclient.New(info), nil
			}
			if !time.Now().Before(deadline) {
				if err == nil {
					err = fmt.Errorf("the jumpgate service did not start within %s; see `journalctl -u jumpgate.service`", serviceStartWait)
				}
				return nil, err
			}
			select {
			case <-ctx.Done():
				return nil, ctx.Err()
			case <-time.After(200 * time.Millisecond):
			}
		}
	}
	// userInGroup reports whether user name is in group (from the group
	// database, not this session).
	userInGroup = func(name, group string) bool {
		u, err := user.Lookup(name)
		if err != nil {
			return false
		}
		g, err := user.LookupGroup(group)
		if err != nil {
			return false
		}
		ids, err := u.GroupIds()
		return err == nil && slices.Contains(ids, g.Gid)
	}
)

// userName is what usermod is given: a plain POSIX user name, so it needs
// no quoting.
var userName = regexp.MustCompile(`^[a-z_][a-z0-9_.-]*[$]?$`)

const setupServerUse = "usage: jumpgate setup server [--name NAME] [--store vault|1password|file] [--ref REF] [--insecure-plain-key] [--self-only] [--stdin]"

// setupServer is `jumpgate setup server`: it starts the jumpgate service,
// adds the operator to jumpgate-admin, has the service create its
// controller key, unlocks it, and pairs this machine. Each step is skipped
// when it is already done, so running it again finishes an interrupted run.
func setupServer(ctx context.Context, args []string) int {
	fset := flag.NewFlagSet("setup server", flag.ContinueOnError)
	name := fset.String("name", "", "this machine's target name (default: its hostname)")
	store := fset.String("store", "vault", "where the controller key lives: vault, 1password or file")
	ref := fset.String("ref", "", "with --store 1password: op://vault/item/field")
	insecurePlain := fset.Bool("insecure-plain-key", false, "with --store file: confirm the key is kept unencrypted on disk")
	selfOnly := fset.Bool("self-only", false, "this controller manages only this machine")
	fromStdin := fset.Bool("stdin", false, "read the passphrase as one line of standard input")
	if err := fset.Parse(args); err != nil || fset.NArg() != 0 {
		return usage(setupServerUse)
	}
	switch signer.Store(*store) {
	case signer.StoreVault:
	case signer.StoreFile:
		if !*insecurePlain {
			return usage("--store file keeps the key unencrypted on disk; add --insecure-plain-key to confirm, or use --store vault")
		}
		if *ref != "" {
			return usage("--store file keeps the key at the service's own path; leave out --ref")
		}
	case signer.StoreOnePassword:
		if !strings.HasPrefix(*ref, "op://") {
			return usage("--ref op://<vault>/<item>/<field> is required for 1password")
		}
	default:
		return usage("unknown --store %q: want vault, 1password or file", *store)
	}
	if hostGOOS != "linux" {
		return failed("the jumpgate service runs only on Linux")
	}
	if geteuid() != 0 {
		fmt.Println("setup server runs as root through sudo")
		return reexecSudo(args)
	}
	if !pathExists(sysunit.PackageUnitPath) && !pathExists(sysunit.AdminUnitPath) {
		return failed("jumpgate.service is not installed: install the jumpgate package, or run install.sh --controller, first")
	}

	ex := runPrivileged()
	defer ex.Close()
	run := func(cmd string) error {
		res, err := ex.Run(ctx, cmd, nil)
		if err != nil {
			return err
		}
		if res.ExitCode != 0 {
			return fmt.Errorf("%s: exit %d: %s", cmd, res.ExitCode, safe(strings.TrimSpace(res.Stderr)))
		}
		return nil
	}
	if err := run("systemctl enable --now jumpgate.service"); err != nil {
		return failed("start the jumpgate service: %v", err)
	}
	operator := os.Getenv("SUDO_USER")
	if operator != "" && operator != "root" && !userInGroup(operator, adminGroup) {
		if !userName.MatchString(operator) {
			return failed("SUDO_USER %q is not a user name; add yourself to %s by hand", safe(operator), adminGroup)
		}
		if err := run("usermod -aG " + adminGroup + " " + operator); err != nil {
			return failed("add %s to %s: %v", safe(operator), adminGroup, err)
		}
	}
	c, err := connectService(ctx)
	if err != nil {
		return failed("%s", safe(err.Error()))
	}
	apiclient.WarnSkew(os.Stderr, c)
	report := func(what string, err error) int {
		var e *api.Error
		if errors.As(err, &e) {
			return reportServerErrorFrom(os.Stderr, what, c.Info(), *e)
		}
		return failed("%s: %s", what, safe(err.Error()))
	}

	// The key.
	var pass []byte
	defer func() { clear(pass) }()
	cv, err := c.Controller(ctx)
	if err != nil {
		return report("controller", err)
	}
	if cv.State == "missing" {
		req := api.SystemInitKey{Store: *store, Ref: *ref, InsecurePlain: *insecurePlain}
		if *selfOnly {
			req.Scope = "self"
		}
		if signer.Store(*store) == signer.StoreVault {
			if *fromStdin {
				pass, err = readPassphrase("vault passphrase: ", true, false)
			} else {
				fmt.Fprintln(os.Stderr, "a lost passphrase cannot be recovered: jumpgate keeps no copy")
				pass, err = readPassphrase("new vault passphrase: ", false, true)
			}
			if err != nil {
				return failed("setup server: %v", err)
			}
			req.Passphrase = slices.Clone(pass) // SystemInitKey clears it
		}
		fmt.Println(api.CustodyWarning)
		if err := c.SystemInitKey(ctx, req); err != nil {
			return report("create the controller key", err)
		}
	}

	// The unlock.
	vv, err := c.Vault(ctx)
	if err != nil {
		return report("vault", err)
	}
	if vv.State == "locked" {
		if pass == nil {
			if pass, err = readPassphrase("vault passphrase: ", *fromStdin, false); err != nil {
				return failed("setup server: %v", err)
			}
		}
		if _, err := c.Unlock(ctx, slices.Clone(pass)); err != nil { // Unlock clears its copy
			var e *api.Error
			if errors.As(err, &e) && e.Code == api.CodeWrongPassphrase {
				return failed("that passphrase does not open the vault; run `jumpgate unlock` to try again, then `sudo jumpgate setup server` to finish")
			}
			return report("unlock", err)
		}
	}
	clear(pass)

	// This machine, paired.
	if *name == "" {
		h, err := os.Hostname()
		if err != nil || h == "" {
			return usage("this machine has no hostname; pass --name NAME")
		}
		*name = h
	}
	targets, err := c.Targets(ctx)
	if err != nil {
		return report("targets", err)
	}
	i := slices.IndexFunc(targets, func(t api.TargetView) bool { return t.ID == *name })
	if i < 0 || targets[i].Agent == nil {
		if i < 0 {
			if err := c.AddTarget(ctx, api.AddTarget{ID: *name, Mode: "local"}); err != nil {
				return report("add "+*name, err)
			}
		}
		f, code := factsFromServer(ctx, c, *name)
		if code != 0 {
			return code
		}
		addr, code := pairLocalForeground(ctx, os.Stdout, f)
		if code != 0 {
			return code
		}
		if code := streamPair(c, *name, api.PairRequest{Installed: addr}, true); code != 0 {
			return code
		}
	}
	host, _ := os.Hostname()
	if host == "" {
		host = *name
	}
	fmt.Printf("log out and back in so the %s group applies to you, then run `jumpgate` here or `ssh -t %s jumpgate` from your machine\n", adminGroup, host)
	return 0
}
