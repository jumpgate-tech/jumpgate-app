package main

import (
	"context"
	"os"
)

// cmdOpen is `jumpgate open`: start the server if needed and open the web
// app in the browser with a one-time login link. --print prints the link
// instead, for a browser that cannot read the redirect file under
// ~/.jumpgate (a snap- or flatpak-confined one) or one on another session.
func cmdOpen(args []string) int {
	open := true
	switch {
	case len(args) == 1 && args[0] == "--print":
		open = false
	case len(args) != 0:
		return usage("usage: jumpgate open [--print]")
	}
	if err := openWebAppWith(context.Background(), os.Stdout, open); err != nil {
		return failed("%v", err)
	}
	return 0
}
