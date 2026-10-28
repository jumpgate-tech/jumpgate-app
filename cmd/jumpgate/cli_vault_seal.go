package main

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/valve-tech/jumpgate/internal/secretenv"
	"github.com/valve-tech/jumpgate/internal/secrets"
	"github.com/valve-tech/jumpgate/internal/sysunit"
	"github.com/valve-tech/jumpgate/internal/vault"
)

// credstoreOut is where systemd looks for an encrypted credential that
// LoadCredentialEncrypted= names without a path.
const credstoreOut = "/etc/credstore.encrypted/" + vaultCredName

const hostKeyNote = "--allow-host-key seals with the host key instead, which only protects against copying the vault file alone"

// runSystemdCreds runs systemd-creds with args and stdin, which is where a
// key goes: never argv, which every local user can read. A test seam.
var runSystemdCreds = func(ctx context.Context, stdin []byte, args ...string) error {
	ctx, cancel := context.WithTimeout(ctx, time.Minute)
	defer cancel()
	name, argv := systemdCredsCommand(geteuid(), args)
	c := exec.CommandContext(ctx, name, argv...)
	c.Env = secretenv.Environ()
	c.Stdin = bytes.NewReader(stdin)
	var errb bytes.Buffer
	c.Stdout, c.Stderr = io.Discard, &errb
	if err := c.Run(); err != nil {
		return fmt.Errorf("systemd-creds %s: %w: %s", args[0], err, safe(strings.TrimSpace(errb.String())))
	}
	return nil
}

// runCredFile runs a file command (mv or rm) on the credential: as root
// directly, else through sudo, as encrypt is, since root wrote the file in
// a directory root owns. argv never holds a key. A test seam.
var runCredFile = func(ctx context.Context, argv ...string) error {
	ctx, cancel := context.WithTimeout(ctx, time.Minute)
	defer cancel()
	name, args := argv[0], argv[1:]
	if geteuid() != 0 {
		name, args = "sudo", append([]string{"--"}, argv...)
	}
	c := exec.CommandContext(ctx, name, args...)
	c.Env = secretenv.Environ()
	var errb bytes.Buffer
	c.Stdout, c.Stderr = io.Discard, &errb
	if err := c.Run(); err != nil {
		return fmt.Errorf("%s: %w: %s", argv[0], err, safe(strings.TrimSpace(errb.String())))
	}
	return nil
}

// credTempPath is a fresh name beside out for the credential until its
// slot is added.
func credTempPath(out string) (string, error) {
	var b [8]byte
	if _, err := rand.Read(b[:]); err != nil {
		return "", err
	}
	return filepath.Join(filepath.Dir(out), "."+filepath.Base(out)+".tmp-"+hex.EncodeToString(b[:])), nil
}

// systemdCredsCommand is the command line for systemd-creds args. Only root
// reads the host credential secret (and the TPM), so a vault owner who is
// not root encrypts through sudo. That root step gets the key on stdin
// only, and writes only the credential: the vault is still written by its
// owner, in this process, never by root.
func systemdCredsCommand(euid int, args []string) (string, []string) {
	if euid != 0 && len(args) > 0 && args[0] == "encrypt" {
		return "sudo", append([]string{"--", "systemd-creds"}, args...)
	}
	return "systemd-creds", args
}

// vaultSeal is `jumpgate vault seal (--tpm2 | --allow-host-key) [--out
// PATH]`: it draws a random 32-byte key, has systemd-creds encrypt it into a
// credential, and adds a systemd-creds slot that key unlocks, so a service
// started with LoadCredentialEncrypted=jumpgate-vault-key unlocks at boot.
func vaultSeal(args []string) int {
	const use = "usage: jumpgate vault seal (--tpm2 | --allow-host-key) [--out PATH] [--stdin]"
	fset := vaultFlags("seal")
	tpm2 := fset.Bool("tpm2", false, "seal with this machine's TPM2")
	hostKey := fset.Bool("allow-host-key", false, "seal with the systemd host key (no TPM2)")
	out := fset.String("out", credstoreOut, "where the encrypted credential is written")
	fromStdin := fset.Bool("stdin", false, "read the passphrase as one line of standard input")
	if err := fset.Parse(args); err != nil || fset.NArg() > 0 {
		return usage(use)
	}
	if *tpm2 == *hostKey {
		return usage("vault seal needs exactly one of --tpm2 and --allow-host-key; " + use)
	}
	// The jumpgate service's vault is the service's: it adds the slot, and
	// this process seals the key for jumpgate.service.
	if c, ok := findServiceClient(context.Background()); ok {
		if *out != credstoreOut {
			return usage("--out does not apply to the jumpgate service: its credential is always %s", sysunit.SealCredPath)
		}
		return sealSystem(context.Background(), os.Stdout, c, *tpm2, *hostKey, *fromStdin)
	}
	path, err := secrets.VaultPath()
	if err != nil {
		return failed("%v", err)
	}
	v, err := vault.Open(path)
	if err != nil {
		return vaultFail("vault seal", err)
	}
	// Root never writes another user's vault: say how to seal it instead,
	// before any prompt or credential.
	if err := v.CheckWrite(); err != nil {
		code := vaultFail("vault seal", err)
		if os.Geteuid() == 0 {
			fmt.Fprintln(os.Stderr, "  run `jumpgate vault seal` as the vault's owner; it runs only systemd-creds encrypt as root, through sudo")
		}
		return code
	}
	ctx := context.Background()
	mode := "host"
	if *tpm2 {
		mode = "tpm2"
		if err := runSystemdCreds(ctx, nil, "has-tpm2"); err != nil {
			return failed("no usable TPM2 here; %s (%v)", hostKeyNote, err)
		}
	} else {
		fmt.Fprintln(os.Stderr, "jumpgate: warning: "+hostKeyNote)
	}
	pass, err := readPassphrase("vault passphrase: ", *fromStdin, false)
	if err != nil {
		return failed("vault seal: %v", err)
	}
	if _, err := unlockOffline(v, pass); err != nil {
		return vaultFail("vault seal", err)
	}
	defer v.Lock()
	kek := make([]byte, 32)
	defer clear(kek)
	if _, err := rand.Read(kek); err != nil {
		return failed("vault seal: %v", err)
	}
	// The credential goes to a temporary name and is renamed over *out only
	// once the slot it opens is in the vault: a failed seal never replaces
	// a working credential.
	tmp, err := credTempPath(*out)
	if err != nil {
		return failed("vault seal: %v", err)
	}
	dropTmp := func() {
		if rerr := runCredFile(ctx, "rm", "-f", "--", tmp); rerr != nil {
			fmt.Fprintf(os.Stderr, "  %s opens nothing; remove it as root (%v)\n", tmp, rerr)
		}
	}
	if err := runSystemdCreds(ctx, kek, "encrypt", "--with-key="+mode, "--name="+vaultCredName, "-", tmp); err != nil {
		dropTmp()
		return failed("vault seal: %v", err)
	}
	if err := v.AddKEKSlot(vault.Slot{Kind: vault.SlotSystemdCreds, Cred: vaultCredName}, kek); err != nil { // clears kek
		dropTmp()
		return vaultFail("vault seal", err)
	}
	if err := runCredFile(ctx, "mv", "-f", "--", tmp, *out); err != nil {
		dropTmp()
		return failed("vault seal: the slot was added but %s was not put in place; run `jumpgate vault seal` again: %v", *out, err)
	}
	fmt.Printf("sealed the vault key into %s (%s); a unit with LoadCredentialEncrypted=%s unlocks the vault at start\n", *out, mode, vaultCredName)
	return 0
}
