package main

import (
	"bytes"
	"context"
	"errors"
	"flag"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"github.com/valve-tech/jumpgate/internal/apiclient"
	"github.com/valve-tech/jumpgate/internal/config"
	"github.com/valve-tech/jumpgate/internal/daemon"
	"github.com/valve-tech/jumpgate/internal/fsperm"
	"github.com/valve-tech/jumpgate/internal/secrets"
	"github.com/valve-tech/jumpgate/internal/vault"
)

const vaultUsage = "usage: jumpgate vault init|migrate|backup|restore|add-recovery|rekdf"

// findServer is daemon.Find; a test seam.
var findServer = daemon.Find

// cmdVault is `jumpgate vault ...`: administration of ~/.jumpgate/vault. It
// runs in this process on the file, under the vault's file lock; a running
// server reloads the file when it changes.
func cmdVault(args []string) int {
	if len(args) == 0 {
		return usage(vaultUsage)
	}
	rest := args[1:]
	switch args[0] {
	case "init":
		return vaultInit(rest)
	case "migrate":
		return vaultMigrate(rest)
	case "backup":
		return vaultBackup(rest)
	case "restore":
		return vaultRestore(rest)
	case "add-recovery":
		return vaultAddRecovery(rest)
	case "rekdf":
		return vaultRekdf(rest)
	}
	return usage("unknown vault subcommand %q; %s", args[0], vaultUsage)
}

// vaultFail reports a vault failure with its exit class: a wrong passphrase
// 1, no vault 2, a corrupt or unsafe file 4 (a security failure).
func vaultFail(what string, err error) int {
	fmt.Fprintf(os.Stderr, "jumpgate: %s: %s\n", what, safe(err.Error()))
	switch {
	case errors.Is(err, vault.ErrCorrupt):
		return exitCode("vault_corrupt")
	case errors.Is(err, vault.ErrUnsafeFile):
		return exitCode("vault_unsafe")
	case errors.Is(err, vault.ErrNoVault):
		return exitCode("no_vault")
	}
	return exitCode("failed")
}

func vaultFlags(name string) *flag.FlagSet {
	fset := flag.NewFlagSet("vault "+name, flag.ContinueOnError)
	fset.SetOutput(os.Stderr)
	return fset
}

// createVaultAt warns, asks for a new passphrase (twice on a terminal, one
// line with fromStdin), and creates the vault at path, unlocked. keep, when
// not nil, receives a copy of the passphrase, which the caller clears.
func createVaultAt(path string, fromStdin bool, keep *[]byte) (*vault.Vault, error) {
	fmt.Println("a lost passphrase cannot be recovered: jumpgate keeps no copy")
	pass, err := readPassphrase("new vault passphrase: ", fromStdin, !fromStdin)
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
	if keep != nil {
		*keep = bytes.Clone(pass)
	}
	v, err := vault.Create(path, pass, vault.DefaultParams) // clears pass
	if err != nil && keep != nil {
		clear(*keep)
		*keep = nil
	}
	return v, err
}

func vaultExists(path string) bool {
	_, err := os.Lstat(path)
	return !errors.Is(err, fs.ErrNotExist)
}

func vaultInit(args []string) int {
	fset := vaultFlags("init")
	fromStdin := fset.Bool("stdin", false, "read the passphrase as one line of standard input")
	if err := fset.Parse(args); err != nil || fset.NArg() > 0 {
		return exitCode("usage")
	}
	path, err := secrets.VaultPath()
	if err != nil {
		return failed("%v", err)
	}
	if vaultExists(path) {
		return failed("a vault already exists at %s", path)
	}
	v, err := createVaultAt(path, *fromStdin, nil)
	if errors.Is(err, vault.ErrExists) {
		return failed("a vault already exists at %s", path)
	}
	if err != nil {
		return vaultFail("vault init", err)
	}
	v.Lock()
	fmt.Printf("created %s\n", path)
	return 0
}

// vaultMigrate creates the vault when there is none, then unlocks the
// running server (starting one if needed), whose unlock moves every plain
// secret into the vault. It reports which fields moved; values never print.
func vaultMigrate(args []string) int {
	fset := vaultFlags("migrate")
	fromStdin := fset.Bool("stdin", false, "read the passphrase as one line of standard input")
	if err := fset.Parse(args); err != nil || fset.NArg() > 0 {
		return exitCode("usage")
	}
	path, err := secrets.VaultPath()
	if err != nil {
		return failed("%v", err)
	}
	before, err := config.Load()
	if err != nil {
		return failed("load config: %v", err)
	}
	var pass []byte
	if !vaultExists(path) {
		v, err := createVaultAt(path, *fromStdin, &pass)
		if err != nil {
			return vaultFail("vault migrate", err)
		}
		v.Lock()
		fmt.Printf("created %s\n", path)
	}
	code := withVaultClient("vault migrate", func(ctx context.Context, c *apiclient.Client) error {
		if pass == nil {
			var err error
			if pass, err = readPassphrase("vault passphrase: ", *fromStdin, false); err != nil {
				return err
			}
		}
		_, err := c.Unlock(ctx, pass) // clears pass
		return err
	})
	clear(pass)
	if code != 0 {
		return code
	}
	after, err := config.Load()
	if err != nil {
		return failed("load config: %v", err)
	}
	left := after.PlainSecrets()
	var moved []string
	for _, f := range before.PlainSecrets() {
		if !slices.Contains(left, f) {
			moved = append(moved, f)
		}
	}
	if len(moved) == 0 {
		fmt.Println("no plain secrets in config.json moved")
	} else {
		fmt.Printf("moved into the vault: %s\n", strings.Join(moved, ", "))
		fmt.Println(secrets.MigrateNotice)
	}
	if len(left) > 0 {
		fmt.Printf("still plain: %s; see the notices in `jumpgate home`\n", strings.Join(left, ", "))
		return exitCode("failed")
	}
	return 0
}

func vaultBackup(args []string) int {
	fset := vaultFlags("backup")
	if err := fset.Parse(args); err != nil || fset.NArg() != 1 {
		return usage("usage: jumpgate vault backup FILE")
	}
	out := fset.Arg(0)
	path, err := secrets.VaultPath()
	if err != nil {
		return failed("%v", err)
	}
	unlock, err := vault.LockPath(path)
	if err != nil {
		return vaultFail("vault backup", err)
	}
	raw, err := vault.ReadFile(path)
	if err == nil {
		err = writeNewPrivate(out, raw)
	}
	unlock()
	if errors.Is(err, fs.ErrExist) {
		return failed("%s already exists; jumpgate never overwrites a backup", out)
	}
	if err != nil {
		return vaultFail("vault backup", err)
	}
	fmt.Printf("backed up %s to %s\n", path, out)
	fmt.Println("the backup is useless without the passphrase; TPM and keystore slots do not travel")
	return 0
}

// writeNewPrivate writes data to a new owner-only file and refuses an
// existing one. A partial file is removed.
func writeNewPrivate(path string, data []byte) error {
	f, err := fsperm.CreatePrivate(path)
	if err != nil {
		return err
	}
	_, werr := f.Write(data)
	if werr == nil {
		werr = f.Sync()
	}
	if cerr := f.Close(); werr == nil {
		werr = cerr
	}
	if werr != nil {
		os.Remove(path)
	}
	return werr
}

func vaultRestore(args []string) int {
	fset := vaultFlags("restore")
	replace := fset.Bool("replace", false, "replace the vault that is there")
	if err := fset.Parse(args); err != nil || fset.NArg() != 1 {
		return usage("usage: jumpgate vault restore [--replace] FILE")
	}
	src := fset.Arg(0)
	if _, running, err := findServer(context.Background()); err != nil {
		return failed("vault restore: cannot tell whether a server is running: %v", err)
	} else if running {
		return failed("a jumpgate server is running and holds the vault; run `jumpgate stop` first")
	}
	path, err := secrets.VaultPath()
	if err != nil {
		return failed("%v", err)
	}
	if vaultExists(path) && !*replace {
		return failed("a vault already exists at %s; add --replace to replace it", path)
	}
	// FILE is checked as a vault (integrity and header) before anything at
	// path is touched.
	raw, err := vault.ReadFile(src)
	if err != nil {
		return vaultFail("vault restore: "+src, err)
	}
	if err := fsperm.MkdirPrivate(filepath.Dir(path)); err != nil {
		return failed("%v", err)
	}
	err = vault.Install(path, raw, *replace)
	if errors.Is(err, vault.ErrExists) {
		return failed("a vault already exists at %s; add --replace to replace it", path)
	}
	if err != nil {
		return vaultFail("vault restore", err)
	}
	fmt.Printf("restored %s from %s\n", path, src)
	return 0
}

// openVaultOffline opens the vault and unlocks it in this process with a
// passphrase from the terminal. The caller locks it.
func openVaultOffline() (*vault.Vault, error) {
	path, err := secrets.VaultPath()
	if err != nil {
		return nil, err
	}
	v, err := vault.Open(path)
	if err != nil {
		return nil, err
	}
	if err := v.CheckWrite(); err != nil {
		return nil, err
	}
	pass, err := readPassphrase("vault passphrase: ", false, false)
	if err != nil {
		return nil, err
	}
	return unlockOffline(v, pass)
}

func vaultAddRecovery(args []string) int {
	if len(args) > 0 {
		return usage("usage: jumpgate vault add-recovery")
	}
	v, err := openVaultOffline()
	if err != nil {
		return vaultFail("vault add-recovery", err)
	}
	defer v.Lock()
	code, err := v.AddRecovery()
	if err != nil {
		return vaultFail("vault add-recovery", err)
	}
	rule := strings.Repeat("-", 72)
	fmt.Println("your recovery code, shown this once; it unlocks the vault like the passphrase:")
	fmt.Println(rule)
	fmt.Println(code)
	fmt.Println(rule)
	fmt.Println("write it down and store it offline, away from this machine; running add-recovery again replaces it")
	return 0
}

func vaultRekdf(args []string) int {
	fset := vaultFlags("rekdf")
	d := vault.DefaultParams
	memory := fset.Uint("memory", uint(d.Memory), "argon2id memory in KiB")
	time := fset.Uint("time", uint(d.Time), "argon2id passes")
	threads := fset.Uint("threads", uint(d.Threads), "argon2id threads")
	newPass := fset.Bool("new-passphrase", false, "also choose a new passphrase")
	if err := fset.Parse(args); err != nil || fset.NArg() > 0 {
		return exitCode("usage")
	}
	if *memory > 1<<32-1 || *time > 1<<32-1 || *threads > 255 {
		return usage("vault rekdf: a parameter is out of range")
	}
	p := vault.Params{Memory: uint32(*memory), Time: uint32(*time), Threads: uint8(*threads)}
	if err := p.Check(); err != nil {
		return usage("vault rekdf: %v (floor m=65536 t=3 p=1)", err)
	}
	path, err := secrets.VaultPath()
	if err != nil {
		return failed("%v", err)
	}
	v, err := vault.Open(path)
	if err != nil {
		return vaultFail("vault rekdf", err)
	}
	if err := v.CheckWrite(); err != nil {
		return vaultFail("vault rekdf", err)
	}
	pass, err := readPassphrase("vault passphrase: ", false, false)
	if err != nil {
		return failed("vault rekdf: %v", err)
	}
	keep := bytes.Clone(pass)
	if _, err := unlockOffline(v, pass); err != nil {
		clear(keep)
		return vaultFail("vault rekdf", err)
	}
	defer v.Lock()
	var next []byte
	if *newPass {
		if next, err = readPassphrase("new vault passphrase: ", false, true); err != nil {
			clear(keep)
			return failed("vault rekdf: %v", err)
		}
	}
	if err := v.Rekdf(keep, next, p); err != nil { // clears both
		return vaultFail("vault rekdf", err)
	}
	fmt.Printf("the primary passphrase slot now uses argon2id m=%d t=%d p=%d\n", p.Memory, p.Time, p.Threads)
	return 0
}
