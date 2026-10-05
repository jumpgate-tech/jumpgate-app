// cmd/jumpgate/help.go
package main

import (
	"fmt"
	"io"
	"os"
)

// commandHelp is one line per command, in the order a new user needs them.
var commandHelp = []struct{ name, text string }{
	{"keys init", "create this controller's signing key"},
	{"keys show", "print this controller's address"},
	{"hosts add NAME --ssh USER@HOST", "pair a Linux box over SSH (--local pairs this Linux machine)"},
	{"hosts list", "list machines and their pairing"},
	{"status HOST", "sync, peers and disk of a paired box"},
	{"disk HOST", "disk usage of a paired box"},
	{"endpoints HOST", "RPC endpoints of a paired box"},
	{"firewall HOST", "firewall checklist of a paired box"},
	{"logs HOST [-n N]", "recent node logs"},
	{"service HOST exec|beacon start|stop|restart", "control a node service"},
	{"open", "open the web app in your browser"},
	{"serve", "run the server in the foreground"},
	{"stop", "stop the background server"},
	{"help", "this list"},
}

// printHelp prints the command list. The longest command is 44 columns, so
// each description goes on its own indented line: a two-column layout wraps
// in an 80-column terminal window, the size a double-click opens.
func printHelp(w io.Writer) {
	fmt.Fprintln(w, "commands (run as `jumpgate COMMAND`):")
	for _, c := range commandHelp {
		fmt.Fprintf(w, "  %s\n      %s\n", c.name, c.text)
	}
}

func cmdHelp([]string) int {
	printHelp(os.Stdout)
	return 0
}
