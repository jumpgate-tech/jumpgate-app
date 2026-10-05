package main

import (
	"context"
	"os"
)

// cmdOpen is `jumpgate open`: start the server if needed and open the web
// app in the browser with a one-time login link.
func cmdOpen(args []string) int {
	if len(args) != 0 {
		return usage("usage: jumpgate open")
	}
	if err := openWebApp(context.Background(), os.Stdout); err != nil {
		return failed("%v", err)
	}
	return 0
}
