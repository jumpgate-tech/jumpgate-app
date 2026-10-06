package server

import (
	"strings"
	"testing"

	"github.com/valve-tech/jumpgate/internal/config"
	"github.com/valve-tech/jumpgate/internal/executor"
)

func sshTarget(host, user string, jump *executor.SSHConfig) config.Target {
	return config.Target{ID: "t", Mode: "ssh", SSH: &executor.SSHConfig{Host: host, User: user, Port: 2222, KeyPath: "/k/id", Jump: jump}}
}

func TestSSHArgvShape(t *testing.T) {
	argv, err := sshArgv(sshTarget("10.0.0.5", "root", &executor.SSHConfig{Host: "bastion", User: "ops"}),
		[]string{"/h/confirmed_hosts", "/h/with space/known_hosts"})
	if err != nil {
		t.Fatal(err)
	}
	want := []string{"ssh", "-p", "2222", "-i", "/k/id",
		"-o", "StrictHostKeyChecking=yes",
		"-o", `UserKnownHostsFile=/h/confirmed_hosts "/h/with space/known_hosts"`,
		"-o", "GlobalKnownHostsFile=none",
		"-J", "ops@bastion:22", "-l", "root", "--", "10.0.0.5"}
	if strings.Join(argv, "\x00") != strings.Join(want, "\x00") {
		t.Fatalf("argv\n got %q\nwant %q", argv, want)
	}
	// The destination is the last element and follows "--".
	if argv[len(argv)-2] != "--" {
		t.Fatal("destination is not behind --")
	}
	for _, a := range argv {
		if strings.Contains(a, "accept-new") || a == "StrictHostKeyChecking=no" {
			t.Fatalf("weak host key option %q", a)
		}
	}
}

func TestSSHArgvJumpChainAndIPv6(t *testing.T) {
	inner := &executor.SSHConfig{Host: "edge", User: "a"}
	mid := &executor.SSHConfig{Host: "fe80::1", Port: 2200, Jump: inner}
	argv, err := sshArgv(sshTarget("::1", "root", mid), []string{"/h/c"})
	if err != nil {
		t.Fatal(err)
	}
	joined := strings.Join(argv, " ")
	// Outermost hop first.
	if !strings.Contains(joined, "-J a@edge:22,[fe80::1]:2200") {
		t.Fatalf("argv %q", joined)
	}
}

func TestSSHArgvRefusesHostileValues(t *testing.T) {
	bad := []string{
		"-oProxyCommand=touch /tmp/x", "-v", "a b", "a\tb", "h%n", "h%h", "a\nb", "a\x00b", "host;rm -rf", "$(id)", "`id`",
		"a'b", `a"b`, "", "h‮x", "host/../x", "u@h", "[::1]", "fe80::1%eth0",
	}
	for _, h := range bad {
		if argv, err := sshArgv(sshTarget(h, "root", nil), []string{"/h/c"}); err == nil {
			t.Errorf("host %q accepted: %q", h, argv)
		}
		if argv, err := sshArgv(sshTarget("10.0.0.5", "root", &executor.SSHConfig{Host: h}), []string{"/h/c"}); err == nil {
			t.Errorf("jump host %q accepted: %q", h, argv)
		}
	}
	for _, u := range []string{"-oProxyCommand=x", "-l", "a b", "u%u", "a\nb", "a@b", "root;id", "a:b"} {
		if argv, err := sshArgv(sshTarget("10.0.0.5", u, nil), []string{"/h/c"}); err == nil {
			t.Errorf("user %q accepted: %q", u, argv)
		}
		if argv, err := sshArgv(sshTarget("10.0.0.5", "root", &executor.SSHConfig{Host: "j", User: u}), []string{"/h/c"}); err == nil {
			t.Errorf("jump user %q accepted: %q", u, argv)
		}
	}
	for _, k := range []string{"/k/%d/id", "/k/a\nb"} {
		tg := sshTarget("10.0.0.5", "root", nil)
		tg.SSH.KeyPath = k
		if _, err := sshArgv(tg, []string{"/h/c"}); err == nil {
			t.Errorf("key path %q accepted", k)
		}
	}
	for _, p := range []int{-1, 65536} {
		tg := sshTarget("10.0.0.5", "root", nil)
		tg.SSH.Port = p
		if _, err := sshArgv(tg, []string{"/h/c"}); err == nil {
			t.Errorf("port %d accepted", p)
		}
	}
	for _, kh := range [][]string{nil, {"/h/%d/known"}, {"/h/a\"b"}, {"/h/a\nb"}} {
		if _, err := sshArgv(sshTarget("10.0.0.5", "root", nil), kh); err == nil {
			t.Errorf("known_hosts %q accepted", kh)
		}
	}
}

func TestSSHArgvDefaultPortAndNoUser(t *testing.T) {
	argv, err := sshArgv(config.Target{SSH: &executor.SSHConfig{Host: "box.example"}}, []string{"/h/c"})
	if err != nil {
		t.Fatal(err)
	}
	if argv[1] != "-p" || argv[2] != "22" || strings.Contains(strings.Join(argv, " "), "-l") {
		t.Fatalf("argv %q", argv)
	}
}

func TestSSHDisplayQuotesForThePlatform(t *testing.T) {
	argv := []string{"ssh", "-o", `UserKnownHostsFile=/h/a b/known`, "-i", "/k/it's", "--", "host"}
	if got, want := sshDisplay(argv, "linux"), `ssh -o 'UserKnownHostsFile=/h/a b/known' -i '/k/it'\''s' -- host`; got != want {
		t.Errorf("posix:\n got %s\nwant %s", got, want)
	}
	if got, want := sshDisplay(argv, "windows"), `ssh -o "UserKnownHostsFile=/h/a b/known" -i "/k/it's" -- host`; got != want {
		t.Errorf("windows:\n got %s\nwant %s", got, want)
	}
}

func TestSSHCommandRefusesAHostileAddress(t *testing.T) {
	ts, token := contractServer(t, config.Target{ID: "evil", Mode: "ssh", SSH: &executor.SSHConfig{Host: "-oProxyCommand=id", User: "root"}})
	res, e := do(t, ts, token, "GET", "/api/fleet/evil/ssh", "")
	if res.StatusCode < 400 || e.Message == "" {
		t.Fatalf("hostile host answered %d %+v", res.StatusCode, e)
	}
}
