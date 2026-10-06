package server

import (
	"errors"
	"net/http"
	"strings"
	"testing"

	"github.com/valve-tech/jumpgate/internal/api"
	"github.com/valve-tech/jumpgate/internal/config"
	"github.com/valve-tech/jumpgate/internal/executor"
)

func sshTarget(host, user string, jump *executor.SSHConfig) config.Target {
	return config.Target{ID: "t", Mode: "ssh", SSH: &executor.SSHConfig{Host: host, User: user, Port: 2222, KeyPath: "/k/id", Jump: jump}}
}

func TestSSHArgvShape(t *testing.T) {
	argv, err := sshArgv(sshTarget("10.0.0.5", "root", nil),
		[]string{"/h/confirmed_hosts", "/h/with space/known_hosts"})
	if err != nil {
		t.Fatal(err)
	}
	want := []string{"ssh", "-p", "2222", "-i", "/k/id",
		"-o", "StrictHostKeyChecking=yes",
		"-o", `UserKnownHostsFile=/h/confirmed_hosts "/h/with space/known_hosts"`,
		"-o", "GlobalKnownHostsFile=none",
		"-o", "ProxyJump=none", "-o", "ProxyCommand=none",
		"-l", "root", "--", "10.0.0.5"}
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

// A target with a jump host cannot get a plain ssh command that verifies the
// jump host too, so it gets none, and no argv carries a jump.
func TestSSHArgvRefusesAJumpAndNeverEmitsOne(t *testing.T) {
	_, err := sshArgv(sshTarget("10.0.0.5", "root", &executor.SSHConfig{Host: "bastion", User: "ops"}), []string{"/h/c"})
	if !errors.Is(err, errSSHJump) {
		t.Fatalf("err %v", err)
	}
	argv, err := sshArgv(sshTarget("10.0.0.5", "root", nil), []string{"/h/c"})
	if err != nil {
		t.Fatal(err)
	}
	for _, a := range argv {
		if a == "-J" || (strings.HasPrefix(a, "ProxyJump=") && a != "ProxyJump=none") || (strings.HasPrefix(a, "ProxyCommand=") && a != "ProxyCommand=none") {
			t.Fatalf("argv carries %q", a)
		}
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
	}
	for _, u := range []string{"-oProxyCommand=x", "-l", "a b", "u%u", "a\nb", "a@b", "root;id", "a:b"} {
		if argv, err := sshArgv(sshTarget("10.0.0.5", u, nil), []string{"/h/c"}); err == nil {
			t.Errorf("user %q accepted: %q", u, argv)
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

func TestSSHCommandForAJumpTargetIs422WithItsOwnCode(t *testing.T) {
	ts, token := contractServer(t, config.Target{ID: "far", Mode: "ssh", SSH: &executor.SSHConfig{Host: "10.0.0.5", User: "root", Jump: &executor.SSHConfig{Host: "bastion", User: "ops"}}})
	res, e := do(t, ts, token, "GET", "/api/fleet/far/ssh", "")
	if res.StatusCode != http.StatusUnprocessableEntity || e.Code != api.CodeSSHJumpUnsupported {
		t.Fatalf("%d %+v", res.StatusCode, e)
	}
	if !strings.Contains(api.HintFor(api.CodeSSHJumpUnsupported), "TUI") {
		t.Fatal("no hint")
	}
}

// The client-side screen must accept what this server really builds, with
// and without -i and -l, and for a path with spaces.
func TestSSHArgvPassesTheClientScreen(t *testing.T) {
	keyless := config.Target{SSH: &executor.SSHConfig{Host: "box.example"}}
	for _, tg := range []config.Target{sshTarget("10.0.0.5", "root", nil), sshTarget("box.example", "", nil), keyless} {
		for _, kh := range [][]string{{"/h/c"}, {"/h/c", "/h/with space/k"}} {
			argv, err := sshArgv(tg, kh)
			if err != nil {
				t.Fatal(err)
			}
			if err := api.CheckSSHArgv(argv); err != nil {
				t.Fatalf("%q: %v", argv, err)
			}
		}
	}
}

func TestSSHArgvRefusesExpandingAndQuotedPaths(t *testing.T) {
	for _, kp := range []string{"/k/${HOME}/id", "~/id", "/k/$X", `/k/"id`, "/k/it's"} {
		tg := sshTarget("10.0.0.5", "root", nil)
		tg.SSH.KeyPath = kp
		if _, err := sshArgv(tg, []string{"/h/c"}); err == nil {
			t.Errorf("key path %q accepted", kp)
		}
	}
	for _, kh := range []string{"${HOME}/kh", "~/kh", "/h/$X"} {
		if _, err := sshArgv(sshTarget("10.0.0.5", "root", nil), []string{kh}); err == nil {
			t.Errorf("known_hosts %q accepted", kh)
		}
	}
}

func TestSSHArgvKeyPathWithSpacesIsBare(t *testing.T) {
	tg := sshTarget("10.0.0.5", "root", nil)
	tg.SSH.KeyPath = `C:\Users\John Smith\.ssh\id_ed25519`
	argv, err := sshArgv(tg, []string{"/h/c"})
	if err != nil {
		t.Fatal(err)
	}
	if argv[3] != "-i" || argv[4] != tg.SSH.KeyPath {
		t.Fatalf("argv %q", argv)
	}
	if err := api.CheckSSHArgv(argv); err != nil {
		t.Fatal(err)
	}
}

func TestSSHArgvRefusesALeadingDashPath(t *testing.T) {
	tg := sshTarget("10.0.0.5", "root", nil)
	tg.SSH.KeyPath = "-oProxyCommand=x"
	if _, err := sshArgv(tg, []string{"/h/known_hosts"}); err == nil {
		t.Fatal("a key path starting with '-' was accepted")
	}
	tg = sshTarget("10.0.0.5", "root", nil)
	if _, err := sshArgv(tg, []string{"-h/known_hosts"}); err == nil {
		t.Fatal("a known_hosts path starting with '-' was accepted")
	}
}

func TestSSHArgvAcceptsAMidPathTilde(t *testing.T) {
	tg := sshTarget("10.0.0.5", "root", nil)
	tg.SSH.KeyPath = `C:\Users\JOHNSM~1\.ssh\id_ed25519`
	argv, err := sshArgv(tg, []string{`C:\Users\JOHNSM~1\.ssh\known_hosts`})
	if err != nil {
		t.Fatalf("an 8.3 short path was refused: %v", err)
	}
	if err := api.CheckSSHArgv(argv); err != nil {
		t.Fatalf("client screen refused %q: %v", argv, err)
	}
}

func TestSSHArgvUserAndSpacedKeyPassesTheClientScreen(t *testing.T) {
	tg := sshTarget("10.0.0.5", "root", nil)
	tg.SSH.KeyPath = `/home/john smith/.ssh/id_ed25519`
	argv, err := sshArgv(tg, []string{"/h/known_hosts"})
	if err != nil {
		t.Fatal(err)
	}
	joined := strings.Join(argv, "\x00")
	if !strings.Contains(joined, "\x00-l\x00root\x00") || !strings.Contains(joined, "\x00-i\x00"+tg.SSH.KeyPath+"\x00") {
		t.Fatalf("argv lacks -l or -i: %q", argv)
	}
	if err := api.CheckSSHArgv(argv); err != nil {
		t.Fatalf("client screen refused %q: %v", argv, err)
	}
}
