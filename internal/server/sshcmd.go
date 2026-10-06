package server

import (
	"errors"
	"fmt"
	"strconv"
	"strings"

	"github.com/valve-tech/jumpgate/internal/api"
	"github.com/valve-tech/jumpgate/internal/config"
)

// errSSHJump: a jump host's key would be checked by the jump child ssh under
// the user's ssh_config, not jumpgate's confirmed store, so no command is built.
var errSSHJump = errors.New("this box is reached through a jump host, which a plain ssh command cannot verify")

// sshArgv is an interactive `ssh` to t under the host keys a person
// confirmed: strict checking against the confirmed store and known_hosts,
// never trust-on-first-use.
//
// Host, user, port, jump and key values come from config, so they are
// untrusted. A target with a jump host gets no command. The result is an argv for direct exec, never a shell string:
// every option is its own "-o Key=value" element, the destination follows
// "--", and a host, user or jump value that is not a plain name (a leading
// "-", whitespace, a control character, "%" for ssh token expansion, or any
// character outside a strict set) is refused.
func sshArgv(t config.Target, knownHosts []string) ([]string, error) {
	c := t.SSH
	if c == nil {
		return nil, errors.New("no ssh address")
	}
	if c.Jump != nil {
		return nil, errSSHJump
	}
	if err := checkHost(c.Host); err != nil {
		return nil, fmt.Errorf("ssh host: %w", err)
	}
	if err := checkUser(c.User); err != nil {
		return nil, fmt.Errorf("ssh user: %w", err)
	}
	port, err := checkPort(c.Port)
	if err != nil {
		return nil, err
	}
	argv := []string{"ssh", "-p", port}
	if c.KeyPath != "" {
		if err := checkPath(c.KeyPath); err != nil {
			return nil, fmt.Errorf("ssh key path: %w", err)
		}
		if strings.ContainsAny(c.KeyPath, "\"'") {
			return nil, errors.New("ssh key path: contains a quote")
		}
		argv = append(argv, "-i", c.KeyPath)
	}
	var files []string
	for _, f := range knownHosts {
		if err := checkPath(f); err != nil {
			return nil, fmt.Errorf("known_hosts path: %w", err)
		}
		if err := api.CheckKnownHostsPath(f); err != nil {
			return nil, fmt.Errorf("known_hosts path %s: %w", f, err)
		}
		if strings.ContainsAny(f, " \t") {
			f = `"` + f + `"` // ssh's own list syntax for a path with spaces
		}
		files = append(files, f)
	}
	if len(files) == 0 {
		return nil, errors.New("no known_hosts file to check the host key against")
	}
	argv = append(argv,
		"-o", api.SSHOptStrictHostKeyChecking,
		"-o", api.SSHOptKnownHostsPrefix+strings.Join(files, " "),
		"-o", api.SSHOptGlobalKnownHosts,
		// A user's ssh_config must not add an unverified hop.
		"-o", api.SSHOptProxyJump,
		"-o", api.SSHOptProxyCommand)
	if c.User != "" {
		argv = append(argv, "-l", c.User)
	}
	return append(argv, "--", c.Host), nil
}

// checkHost accepts a DNS name, IPv4 address or bare IPv6 address, and
// nothing else. "%" (an IPv6 zone, or an ssh token) is refused.
func checkHost(h string) error {
	if h == "" || len(h) > 253 {
		return errors.New("empty or too long")
	}
	if h[0] == '-' {
		return errors.New("starts with '-'")
	}
	for _, r := range h {
		switch {
		case r >= 'a' && r <= 'z', r >= 'A' && r <= 'Z', r >= '0' && r <= '9', r == '.', r == '-', r == '_', r == ':':
		default:
			return fmt.Errorf("character %q is not allowed", r)
		}
	}
	return nil
}

// checkUser accepts a portable login name. An empty user is allowed (ssh
// then uses its own default).
func checkUser(u string) error {
	if u == "" {
		return nil
	}
	if len(u) > 64 {
		return errors.New("too long")
	}
	if u[0] == '-' {
		return errors.New("starts with '-'")
	}
	for _, r := range u {
		switch {
		case r >= 'a' && r <= 'z', r >= 'A' && r <= 'Z', r >= '0' && r <= '9', r == '.', r == '-', r == '_':
		default:
			return fmt.Errorf("character %q is not allowed", r)
		}
	}
	return nil
}

func checkPort(p int) (string, error) {
	if p == 0 {
		p = 22
	}
	if p < 1 || p > 65535 {
		return "", fmt.Errorf("port %d", p)
	}
	return strconv.Itoa(p), nil
}

// checkPath refuses what ssh would expand or what could not be one argv
// element: control characters, "%" and "$" (ssh's token and environment
// expansion), a leading "~", and a leading "-" (ssh would read it as an
// option; the client screen refuses it too).
func checkPath(p string) error {
	if strings.HasPrefix(p, "~") {
		return errors.New("starts with '~'")
	}
	if strings.HasPrefix(p, "-") {
		return errors.New("starts with '-'")
	}
	for _, r := range p {
		if r < 0x20 || r == 0x7f {
			return errors.New("contains a control character")
		}
		if r == '%' || r == '$' {
			return errors.New("contains '%' or '$'")
		}
	}
	return nil
}

// sshDisplay is argv as one line for a person to read or paste. It is for
// showing only: nothing executes it, and it is quoted for goos's shell (POSIX
// single quotes; double quotes on Windows).
func sshDisplay(argv []string, goos string) string {
	parts := make([]string, len(argv))
	for i, a := range argv {
		parts[i] = quoteArg(a, goos)
	}
	return strings.Join(parts, " ")
}

func quoteArg(a, goos string) string {
	safe := a != ""
	for _, r := range a {
		if !(r >= 'a' && r <= 'z' || r >= 'A' && r <= 'Z' || r >= '0' && r <= '9' || strings.ContainsRune("-_./:=@,+", r)) {
			safe = false
			break
		}
	}
	if safe {
		return a
	}
	if goos == "windows" {
		return `"` + strings.ReplaceAll(a, `"`, `\"`) + `"`
	}
	return `'` + strings.ReplaceAll(a, `'`, `'\''`) + `'`
}
