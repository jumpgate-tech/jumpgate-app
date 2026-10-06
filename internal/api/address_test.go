package api

import (
	"strings"
	"testing"
)

func TestParseLogin(t *testing.T) {
	cases := map[string]struct {
		user, host string
		port       int
		ok         bool
	}{
		"root@203.0.113.7":         {"root", "203.0.113.7", 0, true},
		"ops@box.example.com:2222": {"ops", "box.example.com", 2222, true},
		"root@[2001:db8::1]:22":    {"root", "2001:db8::1", 22, true},
		"root@[::1]":               {"root", "::1", 0, true},
		"203.0.113.7":              {"", "", 0, false},
		"root@":                    {"", "", 0, false},
		"root@host:notaport":       {"", "", 0, false},
		"root@:22":                 {"", "", 0, false},
		"root@[]":                  {"", "", 0, false},
		"root@[::1":                {"", "", 0, false},
		"root@host]":               {"", "", 0, false},
		"root@[::1]]":              {"", "", 0, false},
		"root@[]:22":               {"", "", 0, false},
	}
	for in, want := range cases {
		v, err := ParseLogin(in)
		if (err == nil) != want.ok || (want.ok && (v.User != want.user || v.Host != want.host || v.Port != want.port)) {
			t.Errorf("ParseLogin(%q) = %+v %v", in, v, err)
		}
	}
}

// Hostile input: whatever ParseLogin accepts, the server's ssh argv rules
// accept too, so a box that can be added can be ssh'd into.
func TestParseLoginRefusesHostileInput(t *testing.T) {
	for _, in := range []string{
		"-oProxyCommand=x@host", "root@-oProxyCommand=x", "-root@host", "root@-host",
		"root@ho st", "ro ot@host", "root@host\t", "root@host\n", "root@host\r", "root@ho\x00st", "ro\x1bot@host",
		"root@host%eth0", "root@[fe80::1%eth0]", "root@$HOST", "root@`id`", "root@$(id)",
		`root@"host"`, "root@'host'", "root@ho@st", "a@b@host", "root@host;id", "root@host|id", "root@host&id",
		"root@host:0", "root@host:65536", "root@host:022", "root@host:+22", "root@host:-1", "root@host:2 2", "root@host:",
		"root@::1", "root@2001:db8::1", "root@2001:db8::1:22", "root@[host]", "root@[1.2.3.4]", "root@[::1]x", "root@[::1]:",
		"root@[::1] ", "root@hôst", "rööt@host", "@host", "root@@host",
		strings.Repeat("a", 65) + "@host", "root@" + strings.Repeat("a", 254),
	} {
		if v, err := ParseLogin(in); err == nil {
			t.Errorf("ParseLogin(%q) accepted: %+v", in, v)
		}
	}
	if _, err := ParseLogin("root@host:65535"); err != nil {
		t.Errorf("65535 refused: %v", err)
	}
}

func TestParseLoginAcceptsOnlyWhatTheServerCanSSHInto(t *testing.T) {
	for _, in := range []string{"root@203.0.113.7", "ops@box.example.com:2222", "a_b.c-d@h-1_x.example:1", "root@[2001:db8::1]:22"} {
		v, err := ParseLogin(in)
		if err != nil {
			t.Fatalf("%q: %v", in, err)
		}
		if !plainName(v.User, false) || !plainName(v.Host, true) {
			t.Errorf("%q parsed to %+v, which CheckSSHArgv would refuse", in, v)
		}
	}
}

func TestParseLoginErrorsNeverEchoControlBytes(t *testing.T) {
	_, err := ParseLogin("ro\x1b]0;x\x07ot@host")
	if err == nil || strings.ContainsAny(err.Error(), "\x1b\x07") {
		t.Fatalf("error %q", err)
	}
}
