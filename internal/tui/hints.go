package tui

import (
	"strings"

	"charm.land/bubbles/v2/key"
	"charm.land/lipgloss/v2"
)

// The footer: the keys that matter here as "[k] label" buttons, the most
// useful first, with "[?] more" always last. A line that is too long loses
// buttons from its end, never the "[?] more".

// shortDesc is a binding's description as a button: help (?) and the actions
// menu say it in full, the footer in a word or two.
var shortDesc = map[string]string{
	"add a box": "add", "pair or re-pair": "pair", "pair this machine": "this machine", "this machine": "this machine",
	"remove, typed confirmation": "remove", "upgrade the gate": "upgrade", "host keys: list, forget": "host keys",
	"leftover containers": "leftovers", "table: fleet / addresses and pairing": "addresses",
	"choose columns (Settings)": "columns", "refresh now": "refresh", "sort: host, sync, disk, lag": "sort",
	"start a job on a box": "new job", "answer the selected job's question": "decide",
	"take over a job whose controller went silent": "take over", "modules of the selected gateway": "modules",
	"new gateway": "new", "delete, typed confirmation": "delete", "stop, typed confirmation": "stop",
	"replace the config (JSON)": "config", "provision, typed confirmation": "provision", "wipe its data, typed confirmation": "wipe",
	"trust its certificate here": "trust cert", "verify TLS": "verify TLS", "known set of one chain": "known set",
	"customer API keys": "keys", "new customer key (shown once)": "new key", "revoke a customer key": "revoke key",
	"chain list": "chains", "one chain": "one chain", "Docker readiness": "Docker", "start Docker here": "start Docker",
	"test signing on the selected box": "test", "create this controller's key": "create key",
	"export a keystore v3 file": "export", "restart the server (loads a new key)": "restart server",
	"unlock": "unlock", "lock": "lock", "idle lock minutes": "idle lock", "put relay-token or admin-token": "put a secret",
	"create the vault": "create the vault", "move plain secrets into the vault": "move secrets in", "back up to a file": "back up",
	"restore from a file, typed confirmation": "restore a backup", "add a recovery code": "recovery code",
	"seal with the TPM2 so the service unlocks at boot (sudo)": "seal: TPM2", "seal with the host key so the service unlocks at boot (sudo)": "seal: host key",
	"clients / servers": "clients/servers", "bring the tunnel up": "up", "take the tunnel down, asks first": "down",
	"server: endpoint": "endpoint", "server: add a peer (config shown once)": "add peer", "server: remove a peer": "remove peer",
	"check for an update now": "check for an update", "start the server": "start the server", "stop the server, typed confirmation": "stop the server",
	"open the web app in the browser": "open in the browser", "install the systemd service (sudo)": "install service",
	"set up the service: vault key, unlock, pair this machine (sudo)": "set up service", "set up the service, self-only (sudo)": "self-only",
	"SSH shell on this box": "ssh", "actions menu for the selected item": "more actions",
	"measure the disk now (Storage)": "measure", "disk usage by path (Storage)": "disk usage",
	"start (Services)": "start", "restart, typed confirmation (Services)": "restart", "stop, typed confirmation (Services)": "stop",
	"clear the service's state, typed confirmation (Services)": "clear state",
	"follow the newest line":                                   "follow", "level: all, warn+, error+": "level", "explain the errors (AI)": "explain",
	"provision the container (Docker)": "provision", "show the container's config (Docker)": "config",
	"set the container's config as JSON (Docker)": "edit config", "wipe the container, typed confirmation (Docker)": "wipe",
	"reset the devnet, typed confirmation (Docker)": "reset", "run the checks (Tools)": "run checks",
	"reload the latest checks (Tools)": "reload", "monitor on/off (Tools)": "monitor", "run an intent (Tools)": "intent",
	"node setup wizard (Tools)": "wizard", "change who can reach a node module": "reach", "copy a bootnode's enode or ENR": "copy",
	"make the passphrase harder to crack (unlock gets slower)": "harden",
	"forget, typed confirmation":                               "forget", "dismiss, typed confirmation": "dismiss", "back": "back",
}

// hintLabel is a binding's footer label: its entry in shortDesc, else its
// description cut at the first comma, colon or parenthesis.
func hintLabel(h key.Help) string {
	if s, ok := shortDesc[h.Desc]; ok {
		return s
	}
	d := h.Desc
	for _, sep := range []string{" (", ",", ":"} {
		if i := strings.Index(d, sep); i > 0 {
			d = d[:i]
		}
	}
	return d
}

// button is a binding as "[k] label".
func (a *App) button(b key.Binding) string {
	h := b.Help()
	return "[" + keyLabel(h.Key, a.gl) + "] " + hintLabel(h)
}

// buttons joins bindings as buttons, three spaces apart so the mouse finds
// where each one ends.
func (a *App) buttons(bs []key.Binding) string {
	var out []string
	for _, b := range bs {
		out = append(out, a.button(b))
	}
	return strings.Join(out, "   ")
}

// hintItems are the focused pane's buttons, most useful first, without the
// "[?] more" every line ends with.
func (a *App) hintItems() []string {
	switch {
	case a.modal != nil:
		return nil
	case a.focus == focusTree:
		return []string{a.gl.UpDown + " select", "[space] expand", "[enter] open"}
	case a.focus == focusLog:
		return []string{a.gl.UpDown + " scroll"}
	}
	return a.paneHints(a.current().keys())
}

// hints is the footer line: the pane's buttons, then "[?] more".
func (a *App) hints() string { return joinHints(a.hintItems()) }

func joinHints(items []string) string {
	return strings.Join(append(append([]string{}, items...), "[?] more"), "  ")
}

// fitHints is the footer line in room cells: buttons come off its end until
// it fits.
func fitHints(items []string, room int) string {
	for n := len(items); n >= 0; n-- {
		if s := joinHints(items[:n]); lipgloss.Width(s) <= room {
			return s
		}
	}
	return "[?] more"
}

// hintsKept is how many of items fitHints keeps in room cells.
func hintsKept(items []string, room int) int {
	for n := len(items); n > 0; n-- {
		if lipgloss.Width(joinHints(items[:n])) <= room {
			return n
		}
	}
	return 0
}
