package main

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/valve-tech/jumpgate/internal/api"
	"github.com/valve-tech/jumpgate/internal/apiclient"
	"github.com/valve-tech/jumpgate/internal/daemon"
	"github.com/valve-tech/jumpgate/internal/executor"
	"github.com/valve-tech/jumpgate/internal/sysunit"
)

// sealDropIn is the jumpgate.service drop-in that hands the sealed key to
// the service at start, where vault-auto-unlock reads it.
const sealDropIn = "[Service]\nLoadCredentialEncrypted=" + vaultCredName + ":" + sysunit.SealCredPath + "\n"

// Seams for tests; production never reassigns them.
var (
	// findServiceClient is a client of the jumpgate service's admin socket,
	// when this machine runs one this user may reach.
	findServiceClient = func(ctx context.Context) (*apiclient.Client, bool) {
		info, ok, err := daemon.FindSystem(ctx)
		if err != nil || !ok {
			return nil, false
		}
		return apiclient.New(info), true
	}
	// sealPrivileged runs the seal's root file steps: directly as root,
	// else through sudo.
	sealPrivileged = func() executor.Executor {
		if geteuid() == 0 {
			return executor.NewLocal()
		}
		return executor.Sudo(executor.NewLocal())
	}
)

// sealSystem is `jumpgate vault seal` against the jumpgate service. The
// service owns its vault, so it adds the systemd-creds slot and hands back
// the slot key (POST /api/system/seal, which wants the passphrase). This
// process, run by root or a jumpgate-admin member, then has root seal the
// key with systemd-creds (the key on stdin only), installs the credential
// root 0600 and the unit drop-in through the root-only staging dir, and
// reloads systemd. The key is cleared before it returns and never printed.
func sealSystem(ctx context.Context, out io.Writer, c *apiclient.Client, tpm2, allowHostKey, fromStdin bool) int {
	if tpm2 == allowHostKey {
		return usage("vault seal needs exactly one of --tpm2 and --allow-host-key")
	}
	mode := "host"
	if tpm2 {
		mode = "tpm2"
		if err := runSystemdCreds(ctx, nil, "has-tpm2"); err != nil {
			return failed("this machine has no TPM2; %s (%v)", hostKeyNote, err)
		}
	} else {
		fmt.Fprintln(os.Stderr, "jumpgate: warning: "+hostKeyNote)
	}
	pass, err := readPassphrase("vault passphrase: ", fromStdin, false)
	if err != nil {
		return failed("vault seal: %v", err)
	}
	// sudo asks now, on the terminal, before the key exists: systemd-creds
	// reads the key from stdin, so sudo must not need to prompt there.
	if geteuid() != 0 {
		if err := sudoNonInteractive(ctx); err != nil {
			fmt.Fprintln(out, "sealing the service's vault runs systemd-creds as root; sudo may ask for your password")
			if err := sudoPrompt(ctx); err != nil {
				clear(pass)
				return failed("sudo: %v. Sealing needs root: ask an administrator, or run it as root", err)
			}
		}
	}
	kek, err := c.SystemSeal(ctx, pass) // clears pass
	defer clear(kek)
	if err != nil {
		var e *api.Error
		if errors.As(err, &e) {
			return reportServerErrorFrom(os.Stderr, "vault seal", c.Info(), *e)
		}
		return failed("vault seal: %s", safe(err.Error()))
	}
	// From here the vault has the new slot; a failure says how to finish.
	half := func(step string, err error) int {
		return failed("vault seal: the service's vault has a new systemd-creds slot, but %s failed: %v. Run `jumpgate vault seal` again; the passphrase still unlocks", step, err)
	}

	ex := sealPrivileged()
	defer ex.Close()
	run := func(cmd string) error {
		res, err := ex.Run(ctx, cmd, nil)
		if err != nil {
			return err
		}
		if res.ExitCode != 0 {
			return fmt.Errorf("exit %d: %s", res.ExitCode, safe(strings.TrimSpace(res.Stderr)))
		}
		return nil
	}
	if err := ensureStagingDir(run); err != nil {
		return half("the staging dir", err)
	}
	staged, err := stagingName()
	if err != nil {
		return half("the staging name", err)
	}
	drop := func() { _ = run(executor.QuoteArgv([]string{"rm", "-f", "--", staged})) }
	if err := runSystemdCreds(ctx, kek, "encrypt", "--with-key="+mode, "--name="+vaultCredName, "-", staged); err != nil {
		drop()
		return half("systemd-creds encrypt", err)
	}
	clear(kek)
	credDir := filepath.Dir(sysunit.SealCredPath)
	mkdir := fmt.Sprintf(`if [ -L %[1]s ]; then echo "refusing: %[1]s is a symlink" >&2; false; fi && %[2]s`,
		credDir, executor.QuoteArgv([]string{"install", "-d", "-o", "root", "-g", "root", "-m", "0700", credDir}))
	if err := run(mkdir); err != nil {
		drop()
		return half(credDir, err)
	}
	err = run(executor.QuoteArgv([]string{"install", "-o", "root", "-g", "root", "-m", "0600", staged, sysunit.SealCredPath}))
	drop()
	if err != nil {
		return half(sysunit.SealCredPath, err)
	}
	if err := installFile(ctx, ex, run, sysunit.SealDropInPath, sealDropIn); err != nil {
		return half(sysunit.SealDropInPath, err)
	}
	if err := run("systemctl daemon-reload"); err != nil {
		return half("systemctl daemon-reload", err)
	}
	fmt.Fprintf(out, "sealed the service's vault key into %s (%s); from its next start jumpgate.service unlocks without the passphrase\n", sysunit.SealCredPath, mode)
	return 0
}
