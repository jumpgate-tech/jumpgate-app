package main

import (
	"bytes"
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"

	"github.com/valve-tech/jumpgate/internal/api"
	"github.com/valve-tech/jumpgate/internal/config"
	"github.com/valve-tech/jumpgate/internal/fsperm"
	"github.com/valve-tech/jumpgate/internal/secrets"
	"github.com/valve-tech/jumpgate/internal/signer"
	"github.com/valve-tech/jumpgate/internal/vault"
)

func cmdKeys(args []string) int {
	if len(args) == 0 {
		return usage("usage: jumpgate keys init|show [--recorded]|export --keystore-v3 FILE")
	}
	switch args[0] {
	case "export":
		return keysExport(args[1:])
	case "show":
		return keysShow(args[1:], os.Stdout, os.Stderr)
	case "init":
		return keysInit(args[1:])
	}
	return usage("unknown keys subcommand %q", args[0])
}

// keysShow opens the controller key and prints its REAL address, so a
// replaced key cannot hide behind the address config.json recorded. A
// mismatch is a security failure (exit 4); a key that will not open is a
// failure (exit 1) that still shows what is recorded. With --recorded it
// prints only what config.json records and never opens the key, so it cannot
// prompt a keychain or 1Password.
func keysShow(args []string, stdout, stderr io.Writer) int {
	fset := flag.NewFlagSet("keys show", flag.ContinueOnError)
	fset.SetOutput(stderr)
	recorded := fset.Bool("recorded", false, "print the address config.json records, without opening the key")
	if err := fset.Parse(args); err != nil || fset.NArg() > 0 {
		return exitCode("usage")
	}
	c, err := config.Load()
	if err != nil {
		fmt.Fprintf(stderr, "jumpgate: load config: %v\n", err)
		return exitCode("failed")
	}
	if c.Controller == nil {
		fmt.Fprintln(stderr, "jumpgate: no controller key; run `jumpgate keys init`")
		return exitCode("failed")
	}
	rec := c.Controller
	if *recorded {
		fmt.Fprintf(stdout, "%s (%s: %s)\n", rec.Address, rec.KeyStore, rec.KeyRef)
		return 0
	}
	var k *signer.Key
	if signer.Store(rec.KeyStore) == signer.StoreVault {
		k, err = openVaultKey(rec.KeyRef)
	} else {
		k, err = signer.Open(context.Background(), signer.Store(rec.KeyStore), rec.KeyRef)
	}
	if err != nil {
		fmt.Fprintf(stderr, "jumpgate: config.json records %s (%s: %s), but the key could not be opened: %v\n", rec.Address, rec.KeyStore, rec.KeyRef, err)
		return exitCode("failed")
	}
	fmt.Fprintf(stdout, "%s (%s: %s)\n", k.Address().Hex(), rec.KeyStore, rec.KeyRef)
	if err := signer.CheckAddress(k, rec.Address); err != nil {
		fmt.Fprintf(stderr, "jumpgate: SECURITY: the controller key in %s at %q is %s, but config.json records %s\n  -> %s\n",
			rec.KeyStore, rec.KeyRef, k.Address().Hex(), rec.Address, api.HintFor(api.CodeControllerKeyMismatch))
		return exitCode("controller_key_mismatch")
	}
	return 0
}

func keysInit(args []string) int {
	fset := flag.NewFlagSet("keys init", flag.ContinueOnError)
	// The default is resolved only when --store is not given, so an explicit
	// store never runs the Secret Service probe.
	store := fset.String("store", "", "vault | keychain | wincred | 1password | file (file needs --insecure-plain) (default: this machine's OS key store, else vault)")
	insecurePlain := fset.Bool("insecure-plain", false, "with --store file: confirm the key is kept unencrypted on disk")
	ref := fset.String("ref", "", "key file path, keychain or Windows credential name, or op://vault/item/field")
	if err := fset.Parse(args); err != nil {
		return exitCode("usage")
	}
	if *store == "" {
		*store = string(signer.DefaultStore())
	}
	if signer.Store(*store) == signer.StoreFile && !*insecurePlain {
		return usage("--store file keeps the key unencrypted on disk; add --insecure-plain to confirm, or use --store vault")
	}
	c, err := config.Load()
	if err != nil {
		return failed("load config: %v", err)
	}
	if c.Controller != nil {
		return failed("a controller key already exists (%s); jumpgate never replaces one silently", c.Controller.Address)
	}
	if *ref == "" {
		switch signer.Store(*store) {
		case signer.StoreVault:
			*ref = "controller-key"
		case signer.StoreFile:
			*ref = jgFile("keys", "controller.key")
		case signer.StoreKeychain, signer.StoreWinCred:
			*ref = "controller"
		case signer.StoreOnePassword:
			return usage("--ref op://<vault>/<item>/<field> is required for 1password")
		default:
			return usage("unknown --store %q: want vault, keychain, wincred, 1password or file", *store)
		}
	}
	var k *signer.Key
	if signer.Store(*store) == signer.StoreVault {
		k, err = createVaultKey(*ref)
	} else {
		k, err = signer.Create(context.Background(), signer.Store(*store), *ref)
	}
	if errors.Is(err, signer.ErrKeyExists) || errors.Is(err, fs.ErrExist) { // the file store reports an existing file as ErrExist
		return failed("a key already exists in %s at %q and jumpgate will not replace it: paired boxes trust that key's address. "+
			"Choose another --ref, or remove the old key yourself if you are sure nothing is paired with it", *store, *ref)
	}
	if err != nil {
		return failed("%v", err)
	}
	if _, err := config.Update(func(c *config.Config) error {
		if c.Controller != nil {
			return fmt.Errorf("a controller key was recorded while this one was being created (%s)", c.Controller.Address)
		}
		c.Controller = &config.Controller{KeyStore: *store, KeyRef: *ref, Address: k.Address().Hex()}
		// Save refuses plain secrets under the vault; those stay where they
		// are until they are migrated.
		if signer.Store(*store) == signer.StoreVault && len(c.PlainSecrets()) == 0 {
			c.SecretStore = config.SecretStoreVault
		}
		if signer.Store(*store) == signer.StoreFile {
			c.SecretStore = config.SecretStoreFile
		}
		return nil
	}); err != nil {
		return failed("the key %s was created in %s at %q but could not be recorded: %v", k.Address().Hex(), *store, *ref, err)
	}
	fmt.Println(k.Address().Hex())
	if signer.Store(*store) == signer.StoreVault {
		fmt.Println("run `jumpgate unlock` so the server uses it")
		return 0
	}
	fmt.Println("restart the server to use it: jumpgate stop")
	return 0
}

// unlockVault opens the vault at secrets.VaultPath() and unlocks it with a
// passphrase read from the terminal. With create, a missing vault is created
// first, after a warning and a confirmed passphrase.
func unlockVault(create bool) (*vault.Vault, error) {
	path, err := secrets.VaultPath()
	if err != nil {
		return nil, err
	}
	if _, err := os.Lstat(path); errors.Is(err, fs.ErrNotExist) {
		if !create {
			return nil, fmt.Errorf("no vault at %s", path)
		}
		fmt.Fprintln(os.Stderr, "a lost passphrase cannot be recovered: jumpgate keeps no copy")
		pass, err := readNewPassphrase("new vault passphrase: ", false, true)
		if err != nil {
			return nil, err
		}
		if err := noCoreDumps(); err != nil {
			clear(pass)
			return nil, err
		}
		if err := fsperm.MkdirPrivate(filepath.Dir(path)); err != nil {
			clear(pass)
			return nil, err
		}
		return vault.Create(path, pass, vault.DefaultParams) // clears pass
	}
	v, err := vault.Open(path)
	if err != nil {
		return nil, err
	}
	pass, err := readPassphrase("vault passphrase: ", false, false)
	if err != nil {
		return nil, err
	}
	return unlockOffline(v, pass)
}

// disableCoreDumps is vault.DisableCoreDumps; a test seam.
var disableCoreDumps = vault.DisableCoreDumps

// noCoreDumps runs before this process derives or holds a vault key: a core
// file or a ptrace would otherwise expose the data key. If that cannot be
// guaranteed, the unlock is refused.
func noCoreDumps() error {
	if err := disableCoreDumps(); err != nil {
		return fmt.Errorf("refusing to unlock the vault: cannot disable core dumps: %w", err)
	}
	return nil
}

// unlockOffline unlocks v in this process, after core dumps are off. It
// clears pass.
func unlockOffline(v *vault.Vault, pass []byte) (*vault.Vault, error) {
	defer clear(pass)
	if err := noCoreDumps(); err != nil {
		return nil, err
	}
	if err := v.Unlock(pass); err != nil {
		return nil, err
	}
	return v, nil
}

// openVaultKey unlocks the vault offline, reads the key, and locks it again.
func openVaultKey(ref string) (*signer.Key, error) {
	v, err := unlockVault(false)
	if err != nil {
		return nil, err
	}
	defer v.Lock()
	return signer.OpenVault(v, ref)
}

// exportScryptN is the keystore v3 scrypt cost; tests lower it.
var exportScryptN = 262144

// keysExport is `jumpgate keys export --keystore-v3 FILE`: it writes the
// vault's controller key as an Ethereum keystore v3 file a wallet can
// import. Every passphrase comes from the terminal; there is no --stdin.
func keysExport(args []string) int {
	fset := flag.NewFlagSet("keys export", flag.ContinueOnError)
	fset.SetOutput(os.Stderr)
	v3 := fset.Bool("keystore-v3", false, "write an Ethereum keystore v3 file")
	if err := fset.Parse(args); err != nil || !*v3 || fset.NArg() != 1 {
		return usage("usage: jumpgate keys export --keystore-v3 FILE")
	}
	out := fset.Arg(0)
	c, err := config.Load()
	if err != nil {
		return failed("load config: %v", err)
	}
	if c.Controller == nil {
		return failed("no controller key; run `jumpgate keys init`")
	}
	if signer.Store(c.Controller.KeyStore) != signer.StoreVault {
		return usage("only a vault key is exported; the other stores already hold it where a wallet or tool can reach it")
	}
	if _, err := os.Lstat(out); err == nil {
		return failed("%s already exists; jumpgate never overwrites it", out)
	}
	k, err := openVaultKey(c.Controller.KeyRef) // core dumps off first
	if err != nil {
		return vaultFail("keys export", err)
	}
	if err := signer.CheckAddress(k, c.Controller.Address); err != nil {
		fmt.Fprintf(os.Stderr, "jumpgate: SECURITY: the vault key is %s, but config.json records %s\n", k.Address().Hex(), c.Controller.Address)
		return exitCode("controller_key_mismatch")
	}
	pass, err := readPassphrase("keystore file passphrase: ", false, true)
	if err != nil {
		return failed("keys export: %v", err)
	}
	check := bytes.Clone(pass)
	defer clear(check)
	b, err := signer.EncryptKeystoreV3(k, pass, exportScryptN) // clears pass
	if err != nil {
		return failed("keys export: %v", err)
	}
	err = writeNewPrivate(out, b)
	if errors.Is(err, fs.ErrExist) {
		return failed("%s already exists; jumpgate never overwrites it", out)
	}
	if err != nil {
		return failed("keys export: %v", err)
	}
	written, err := os.ReadFile(out)
	if err == nil {
		var back *signer.Key
		if back, err = signer.DecryptKeystoreV3(written, bytes.Clone(check)); err == nil && back.Address() != k.Address() {
			err = errors.New("another address")
		}
	}
	if err != nil {
		os.Remove(out)
		return failed("keys export: the written file does not read back to %s (%v); it was removed", k.Address().Hex(), err)
	}
	fmt.Printf("wrote %s (%s); anyone with this file and its passphrase controls the key\n", out, k.Address().Hex())
	return 0
}

// createVaultKey creates the vault if needed and stores a new key in it.
func createVaultKey(ref string) (*signer.Key, error) {
	v, err := unlockVault(true)
	if err != nil {
		return nil, err
	}
	defer v.Lock()
	return signer.CreateVault(v, ref)
}
