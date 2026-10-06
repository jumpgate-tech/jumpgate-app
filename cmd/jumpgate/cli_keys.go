package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"io/fs"
	"os"

	"github.com/valve-tech/jumpgate/internal/api"
	"github.com/valve-tech/jumpgate/internal/config"
	"github.com/valve-tech/jumpgate/internal/signer"
)

func cmdKeys(args []string) int {
	if len(args) == 0 {
		return usage("usage: jumpgate keys init|show [--recorded]")
	}
	switch args[0] {
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
	k, err := signer.Open(context.Background(), signer.Store(rec.KeyStore), rec.KeyRef)
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
	store := fset.String("store", "", "file | keychain | wincred | 1password (default: this machine's OS key store, else file)")
	ref := fset.String("ref", "", "key file path, keychain or Windows credential name, or op://vault/item/field")
	if err := fset.Parse(args); err != nil {
		return exitCode("usage")
	}
	if *store == "" {
		*store = string(signer.DefaultStore())
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
		case signer.StoreFile:
			*ref = jgFile("keys", "controller.key")
		case signer.StoreKeychain, signer.StoreWinCred:
			*ref = "controller"
		case signer.StoreOnePassword:
			return usage("--ref op://<vault>/<item>/<field> is required for 1password")
		default:
			return usage("unknown --store %q: want file, keychain, wincred or 1password", *store)
		}
	}
	k, err := signer.Create(context.Background(), signer.Store(*store), *ref)
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
		return nil
	}); err != nil {
		return failed("the key %s was created in %s at %q but could not be recorded: %v", k.Address().Hex(), *store, *ref, err)
	}
	fmt.Println(k.Address().Hex())
	fmt.Println("restart the server to use it: jumpgate stop")
	return 0
}
