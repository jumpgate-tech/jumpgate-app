// cmd/jumpgate/help.go
package main

import (
	"fmt"
	"io"
	"os"

	"github.com/valve-tech/jumpgate/internal/catalog"
)

// commandHelp is one line per command, in the order a new user needs them.
var commandHelp = []struct{ name, text string }{
	{"keys init [--store vault]", "create this controller's signing key (OS key store, else the vault)"},
	{"keys show", "print this controller's address"},
	{"keys export --keystore-v3 FILE", "write the controller key as an Ethereum keystore v3 file"},
	{"unlock", "unlock the vault so the server can sign"},
	{"lock", "drop the vault key from the server's memory"},
	{"vault init|migrate|backup FILE|restore FILE", "create the vault, move plain secrets into it, copy it out and back"},
	{"vault add-recovery|rekdf", "add a recovery code; raise the passphrase's argon2id cost"},
	{"vault seal --tpm2|--allow-host-key", "seal the vault key with systemd-creds, so the service unlocks at boot"},
	{"vault put relay-token|admin-token", "store the relay's token in the unlocked vault"},
	{"hosts add NAME --ssh USER@HOST", "pair a Linux box over SSH (--local pairs this Linux machine)"},
	{"hosts list", "list machines and their pairing"},
	{"hosts invite NAME", "print the one-liner that sets up a box's gate to pair with this controller"},
	{"hosts upgrade NAME", "replace a box's gate with this controller's, over the pairing"},
	{"status HOST", "sync, peers and disk of a paired box"},
	{"disk HOST", "disk usage of a paired box"},
	{"endpoints HOST", "RPC endpoints of a paired box"},
	{"firewall HOST", "firewall checklist of a paired box"},
	{"logs HOST [-n N]", "recent node logs"},
	{"service HOST exec|beacon start|stop|restart", "control a node service"},
	{"jobs [HOST]", "durable jobs; also jobs show|decide|adopt|output"},
	{"migrate-names HOST --yes", "rename " + catalog.LegacyNames.Prefix + " to " + catalog.CurrentNames.Prefix + " on HOST (stops its node)"},
	{"ssh HOST", "open a shell on a box, under the host keys you confirmed"},
	{"hosts forget-key NAME [--jump]", "forget the host key recorded for a box"},
	{"open", "open the web app in your browser"},
	{"home", "the plain status screen (bare jumpgate opens the terminal UI)"},
	{"tui", "the terminal UI"},
	{"serve", "run the server in the foreground"},
	{"setup install-service [--bin PATH]", "install jumpgate.service on this Linux server (as root)"},
	{"setup server [--self-only]", "create the service's vault key, unlock it and pair this server"},
	{"relay", "run only the metered RPC data plane, in the foreground"},
	{"stop", "stop the background server"},
	{"agent setup --controller ADDR --transport-key KEY", "on a box, as root: install its gate for a controller (pull pairing)"},
	{"agent watch --config FILE", "the box's health watch (its gate's watch module runs it)"},
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
