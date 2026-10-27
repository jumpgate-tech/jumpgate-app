package main

import (
	"context"
	"errors"
	"fmt"
	"os"

	"github.com/valve-tech/jumpgate/internal/api"
	"github.com/valve-tech/jumpgate/internal/apiclient"
)

// cmdUnlock is `jumpgate unlock [--stdin]`: it reads the vault passphrase from
// the terminal (or one line of stdin) and sends it to the running server. The
// passphrase is never taken from the command line or the environment, where
// other users of the machine could read it.
func cmdUnlock(args []string) int {
	fromStdin := false
	for _, a := range args {
		switch a {
		case "--stdin":
			fromStdin = true
		case "-h", "--help":
			return usage("usage: jumpgate unlock [--stdin]")
		default:
			if len(a) > 0 && a[0] == '-' {
				return usage("unlock: unknown flag %q; usage: jumpgate unlock [--stdin]", a)
			}
			// Do not echo the argument: it may be the passphrase.
			return usage("the passphrase is never taken from the command line; use --stdin or a terminal")
		}
	}
	return withVaultClient("unlock", func(ctx context.Context, c *apiclient.Client) error {
		pass, err := readPassphrase("vault passphrase: ", fromStdin, false)
		if err != nil {
			return err
		}
		v, err := c.Unlock(ctx, pass) // clears pass
		if err == nil {
			fmt.Println(vaultStateWord(v))
		}
		return err
	})
}

// cmdLock is `jumpgate lock`: the server forgets the vault key until the next
// unlock.
func cmdLock(args []string) int {
	if len(args) > 0 {
		return usage("usage: jumpgate lock")
	}
	return withVaultClient("lock", func(ctx context.Context, c *apiclient.Client) error {
		v, err := c.Lock(ctx)
		if err == nil {
			fmt.Println(vaultStateWord(v))
		}
		return err
	})
}

func vaultStateWord(v api.VaultView) string {
	if v.State == "" {
		return "unknown"
	}
	return safe(v.State)
}

// withVaultClient is withClient with what named in the error line.
func withVaultClient(what string, f func(ctx context.Context, c *apiclient.Client) error) int {
	ctx := context.Background()
	c, err := connect(ctx)
	if err != nil {
		return failed("%s: %s", what, safe(err.Error()))
	}
	apiclient.WarnSkew(os.Stderr, c)
	err = f(ctx, c)
	var e *api.Error
	if errors.As(err, &e) {
		clean := *e
		clean.Message, clean.Hint, clean.Reason = safe(e.Message), safe(e.Hint), safe(e.Reason)
		return reportServerErrorFrom(os.Stderr, what, c.Info(), clean)
	}
	if err != nil {
		return failed("%s: %s", what, safe(err.Error()))
	}
	return 0
}
