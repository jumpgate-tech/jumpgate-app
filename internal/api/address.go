package api

import (
	"fmt"
	"net"
	"strconv"
	"strings"
)

// ParseLogin reads user@host[:port], with an IPv6 host in brackets. It is at
// least as strict as the server's ssh argv rules (internal/server/sshcmd.go
// and CheckSSHArgv), so a box that can be added can also be ssh'd into: the
// user is a plain login name, the host a plain DNS name, IPv4 address or
// bracketed IPv6 address, neither starts with '-', and the port is 1-65535
// written without a sign or leading zeros. Whitespace, control characters,
// quotes, '%', '$' and '@' never get through.
func ParseLogin(s string) (SSHView, error) {
	bad := func(why string) (SSHView, error) {
		return SSHView{}, fmt.Errorf("%s in %q: want user@host[:port]", why, printable(s))
	}
	at := strings.LastIndex(s, "@")
	if at <= 0 || at == len(s)-1 {
		return bad("no user or no host")
	}
	user, rest := s[:at], s[at+1:]
	if !plainName(user, false) || len(user) > 64 {
		return bad("the user is not a plain login name")
	}
	var host, portText string
	switch {
	case strings.HasPrefix(rest, "["):
		end := strings.IndexByte(rest, ']')
		if end < 0 || strings.Count(rest, "[") != 1 || strings.Count(rest, "]") != 1 {
			return bad("unbalanced brackets")
		}
		host, rest = rest[1:end], rest[end+1:]
		ip := net.ParseIP(host)
		if ip == nil || ip.To4() != nil || !strings.Contains(host, ":") || !plainName(host, true) {
			return bad("brackets are for an IPv6 address")
		}
		if rest != "" {
			if rest[0] != ':' {
				return bad("text after the brackets")
			}
			portText = rest[1:]
			if portText == "" {
				return bad("empty port")
			}
		}
	case strings.ContainsAny(rest, "[]"):
		return bad("unbalanced brackets")
	default:
		host = rest
		if i := strings.IndexByte(rest, ':'); i >= 0 {
			if strings.Count(rest, ":") > 1 {
				return bad("an IPv6 address needs brackets")
			}
			host, portText = rest[:i], rest[i+1:]
			if portText == "" {
				return bad("empty port")
			}
		}
		if host == "" {
			return bad("empty host")
		}
		if strings.Contains(host, ":") || !plainName(host, false) || len(host) > 253 {
			return bad("the host is not a plain name")
		}
	}
	port := 0
	if portText != "" {
		n, err := strconv.Atoi(portText)
		if err != nil || n < 1 || n > 65535 || strconv.Itoa(n) != portText {
			return bad("bad port")
		}
		port = n
	}
	return SSHView{User: user, Host: host, Port: port}, nil
}

// printable quotes s for an error message without echoing control bytes.
func printable(s string) string {
	q := strconv.QuoteToASCII(s)
	return q[1 : len(q)-1]
}
