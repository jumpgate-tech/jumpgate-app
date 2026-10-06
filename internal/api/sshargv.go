package api

import (
	"errors"
	"fmt"
	"path/filepath"
	"strconv"
	"strings"
)

// The fixed -o options of a server-built shell command, in the order the
// server emits them. The server's sshArgv and CheckSSHArgv share these so
// the builder and the screen cannot drift.
const (
	SSHOptStrictHostKeyChecking = "StrictHostKeyChecking=yes"
	SSHOptKnownHostsPrefix      = "UserKnownHostsFile="
	SSHOptGlobalKnownHosts      = "GlobalKnownHostsFile=none"
	SSHOptProxyJump             = "ProxyJump=none"
	SSHOptProxyCommand          = "ProxyCommand=none"
)

// CheckSSHArgv screens an ssh command received from a server before this
// machine runs it. The server is trusted to know the box, not to choose what
// runs locally, so the command must match, token for token, the shape the
// server's sshArgv builds, and nothing else:
//
//	ssh|ssh.exe -p PORT [-i KEYPATH]
//	  -o StrictHostKeyChecking=yes -o UserKnownHostsFile=FILES
//	  -o GlobalKnownHostsFile=none -o ProxyJump=none -o ProxyCommand=none
//	  [-l USER] -- HOST
//
// Each flag and option is its own element: no bundled flags, no attached
// values, no repeats, nothing after the destination. An allowlist of shapes,
// not of keys, so ssh's own option parsing (whitespace, quotes, case) never
// has a chance to read something this check did not.
func CheckSSHArgv(argv []string) error {
	for _, a := range argv {
		if strings.ContainsAny(a, "\n\r\x00") {
			return errors.New("an argument has a line break or NUL")
		}
	}
	p := &argvReader{argv: argv}
	if len(argv) == 0 {
		return errors.New("the ssh command is empty")
	}
	if b := filepath.Base(argv[0]); b != "ssh" && b != "ssh.exe" {
		return errors.New("the program is not the system ssh")
	}
	p.i = 1
	if err := p.flag("-p"); err != nil {
		return err
	}
	port, err := p.next("the port")
	if err != nil {
		return err
	}
	if n, err := strconv.Atoi(port); err != nil || n < 1 || n > 65535 || strconv.Itoa(n) != port {
		return errors.New("the port is not a plain number")
	}
	if p.peek() == "-i" {
		p.i++
		path, err := p.next("the key path")
		if err != nil {
			return err
		}
		if err := checkSSHPath(path); err != nil || strings.ContainsAny(path, " \t\"'") {
			return errors.New("key path: not a plain path")
		}
	}
	if err := p.option(SSHOptStrictHostKeyChecking); err != nil {
		return err
	}
	if err := p.flag("-o"); err != nil {
		return err
	}
	kh, err := p.next("the known_hosts option")
	if err != nil {
		return err
	}
	if !strings.HasPrefix(kh, SSHOptKnownHostsPrefix) {
		return errors.New("the UserKnownHostsFile option is missing")
	}
	if err := checkKnownHostsList(strings.TrimPrefix(kh, SSHOptKnownHostsPrefix)); err != nil {
		return err
	}
	for _, o := range []string{SSHOptGlobalKnownHosts, SSHOptProxyJump, SSHOptProxyCommand} {
		if err := p.option(o); err != nil {
			return err
		}
	}
	if p.peek() == "-l" {
		p.i++
		u, err := p.next("the user")
		if err != nil {
			return err
		}
		if !plainName(u, false) || len(u) > 64 {
			return errors.New("the user is not a plain name")
		}
	}
	if err := p.flag("--"); err != nil {
		return err
	}
	host, err := p.next("the destination")
	if err != nil {
		return err
	}
	if !plainName(host, true) || len(host) > 253 {
		return errors.New("the destination is not a plain host name")
	}
	if p.i != len(argv) {
		return errors.New("the ssh command has elements after the destination")
	}
	return nil
}

type argvReader struct {
	argv []string
	i    int
}

func (p *argvReader) peek() string {
	if p.i < len(p.argv) {
		return p.argv[p.i]
	}
	return ""
}

func (p *argvReader) flag(want string) error {
	if p.peek() != want || p.i >= len(p.argv) {
		return fmt.Errorf("expected %s", want)
	}
	p.i++
	return nil
}

func (p *argvReader) next(what string) (string, error) {
	if p.i >= len(p.argv) {
		return "", fmt.Errorf("the ssh command ends before %s", what)
	}
	p.i++
	return p.argv[p.i-1], nil
}

// option wants "-o" then exactly want.
func (p *argvReader) option(want string) error {
	if err := p.flag("-o"); err != nil {
		return err
	}
	v, err := p.next("an option")
	if err != nil {
		return err
	}
	if v != want {
		return fmt.Errorf("expected the option %s", want)
	}
	return nil
}

// checkSSHPath refuses what ssh would expand or reinterpret: control
// characters, '%' and '$' (token and environment expansion) and a leading '~'.
func checkSSHPath(s string) error {
	if s == "" || s[0] == '-' || s[0] == '~' {
		return errors.New("empty, or starts with '-' or '~'")
	}
	for _, r := range s {
		if r < 0x20 || r == 0x7f || r == '%' || r == '$' {
			return errors.New("has a control character, '%' or '$'")
		}
	}
	return nil
}

// checkKnownHostsList accepts what the server writes: files separated by
// single spaces, each a bare path (no whitespace, quote or '%') or, when the
// path has spaces, one double-quoted path (no inner quote or '%').
func checkKnownHostsList(v string) error {
	if v == "" {
		return errors.New("no known_hosts file")
	}
	for v != "" {
		var tok string
		if v[0] == '"' {
			end := strings.IndexByte(v[1:], '"')
			if end < 0 {
				return errors.New("known_hosts: unterminated quote")
			}
			tok, v = v[1:1+end], v[end+2:]
			// ssh reads \" inside quotes as an escaped quote; the server
			// never writes one, so a quoted path may not end in a backslash.
			if strings.ContainsAny(tok, "\t") || strings.HasSuffix(tok, `\`) {
				return errors.New("known_hosts: tab or escaped quote in a quoted path")
			}
		} else {
			end := strings.IndexByte(v, ' ')
			if end < 0 {
				end = len(v)
			}
			tok, v = v[:end], v[end:]
			if strings.ContainsAny(tok, "\t\"") {
				return errors.New("known_hosts: whitespace or quote in a path")
			}
		}
		if v != "" {
			if v[0] != ' ' || len(v) == 1 {
				return errors.New("known_hosts: malformed list")
			}
			v = v[1:]
		}
		if err := checkSSHPath(tok); err != nil {
			return fmt.Errorf("known_hosts: %w", err)
		}
	}
	return nil
}

// plainName is a DNS name or address (host) or a login name: letters,
// digits, '.', '-', '_' (and ':' for a host), not starting with '-'.
func plainName(s string, host bool) bool {
	if s == "" || s[0] == '-' {
		return false
	}
	for _, r := range s {
		switch {
		case r >= 'a' && r <= 'z', r >= 'A' && r <= 'Z', r >= '0' && r <= '9', r == '.', r == '-', r == '_':
		case r == ':' && host:
		default:
			return false
		}
	}
	return true
}
