package api

import (
	"slices"
	"testing"
)

func goodArgv() []string {
	return []string{"ssh", "-p", "22", "-i", "/k/id",
		"-o", "StrictHostKeyChecking=yes",
		"-o", `UserKnownHostsFile=/h/c "/h/with space/k"`,
		"-o", "GlobalKnownHostsFile=none",
		"-o", "ProxyJump=none", "-o", "ProxyCommand=none",
		"-l", "root", "--", "10.0.0.5"}
}

func TestCheckSSHArgvAcceptsTheServersShape(t *testing.T) {
	if err := CheckSSHArgv(goodArgv()); err != nil {
		t.Fatal(err)
	}
	min := []string{"ssh.exe", "-p", "2222", "-o", "StrictHostKeyChecking=yes", "-o", "UserKnownHostsFile=C:/h/c",
		"-o", "GlobalKnownHostsFile=none", "-o", "ProxyJump=none", "-o", "ProxyCommand=none", "--", "box.example"}
	if err := CheckSSHArgv(min); err != nil {
		t.Fatal(err)
	}
}

func TestCheckSSHArgvAllowsSpacesInTheKeyPath(t *testing.T) {
	a := goodArgv()
	a[4] = `C:\Users\John Smith\.ssh\id_ed25519`
	if err := CheckSSHArgv(a); err != nil {
		t.Fatal(err)
	}
	for _, bad := range []string{"C:\\Users\\\"John Smith\"\\id", "/k/it's", "/k/a\tb"} {
		a[4] = bad
		if err := CheckSSHArgv(a); err == nil {
			t.Errorf("%q passed", bad)
		}
	}
}

func TestCheckSSHArgvRefusesEverythingElse(t *testing.T) {
	mut := func(f func(a []string) []string) []string { return f(goodArgv()) }
	replace := func(i int, v string) []string { return mut(func(a []string) []string { a[i] = v; return a }) }
	bad := map[string][]string{
		"empty":           nil,
		"not ssh":         replace(0, "/bin/sh"),
		"leading space":   replace(6, " ProxyCommand=echo PWN"),
		"leading tab":     {"ssh", "-p", "22", "-o", "\tProxyCommand echo PWN", "--", "h"},
		"quoted key":      {"ssh", "-p", "22", "-o", `"ProxyCommand"=echo PWN`, "--", "h"},
		"space form":      replace(6, "StrictHostKeyChecking yes"),
		"strict no":       replace(6, "StrictHostKeyChecking=no"),
		"strict case":     replace(6, "stricthostkeychecking=yes"),
		"proxycmd":        replace(14, "ProxyCommand=nc evil 1"),
		"proxyjump":       replace(12, "ProxyJump=evil"),
		"global kh":       replace(10, "GlobalKnownHostsFile=/dev/null"),
		"devnull kh only": {"ssh", "-p", "22", "-o", "UserKnownHostsFile=/dev/null", "--", "h"},
		"hostkeyalias":    {"ssh", "-p", "22", "-o", "HostKeyAlias=x", "-o", "Hostname=evil", "--", "h"},
		"remotecommand":   {"ssh", "-p", "22", "-o", "RemoteCommand=id", "--", "h"},
		"forwardagent":    {"ssh", "-p", "22", "-o", "ForwardAgent=yes", "--", "h"},
		"sendenv":         {"ssh", "-p", "22", "-o", "SendEnv=*", "--", "h"},
		"forkafterauth":   {"ssh", "-p", "22", "-o", "SessionType=none", "-o", "ForkAfterAuthentication=yes", "--", "h"},
		"bundled":         {"ssh", "-p", "22", "-qoProxyCommand=x", "--", "h"},
		"attached -p":     replace(1, "-p22"),
		"attached -o":     replace(5, "-oStrictHostKeyChecking=yes"),
		"-J":              {"ssh", "-p", "22", "-J", "evil", "--", "h"},
		"extra strict no": mut(func(a []string) []string { return slices.Insert(a, 1, "-o", "StrictHostKeyChecking=no") }),
		"dup option":      mut(func(a []string) []string { return slices.Insert(a, 14, "-o", "StrictHostKeyChecking=no") }),
		"extra flag":      mut(func(a []string) []string { return slices.Insert(a, 3, "-N") }),
		"after dest":      mut(func(a []string) []string { return append(a, "id") }),
		"after dest -o":   mut(func(a []string) []string { return append(a, "-o", "ProxyCommand=id") }),
		"no --":           mut(func(a []string) []string { return slices.Delete(a, 17, 18) }),
		"dest dash":       replace(18, "-oProxyCommand=id"),
		"dest chars":      replace(18, "h;id"),
		"user chars":      replace(16, "a b"),
		"user dash":       replace(16, "-x"),
		"port word":       replace(2, "22x"),
		"port zero":       replace(2, "0"),
		"port big":        replace(2, "70000"),
		"port lead zero":  replace(2, "022"),
		"key empty":       replace(4, ""),
		"key dash":        replace(4, "-F"),
		"kh quote":        replace(8, `UserKnownHostsFile=/a" -o ProxyCommand=id "/b`),
		"kh percent":      replace(8, "UserKnownHostsFile=/h/%d"),
		"kh empty":        replace(8, "UserKnownHostsFile="),
		"kh tab":          replace(8, "UserKnownHostsFile=/a\t/b"),
		"kh dollar":       replace(8, "UserKnownHostsFile=${HOME}/kh"),
		"kh tilde":        replace(8, "UserKnownHostsFile=~/kh"),
		"kh quoted bs":    replace(8, `UserKnownHostsFile="a b\"`),
		"kh quoted esc":   replace(8, `UserKnownHostsFile="a\" "b"`),
		"kh bare quote":   replace(8, `UserKnownHostsFile=a"b`),
		"key dollar":      replace(4, "/k/${HOME}"),
		"key tilde":       replace(4, "~/id"),
		"key quote":       replace(4, `/k/"id`),
		"newline":         replace(16, "a\nb"),
		"nul":             replace(18, "h\x00"),
		"missing -p":      {"ssh", "--", "h"},
		"repeat -l":       mut(func(a []string) []string { return slices.Insert(a, 17, "-l", "x") }),
	}
	for name, argv := range bad {
		if err := CheckSSHArgv(argv); err == nil {
			t.Errorf("%s: %q passed", name, argv)
		}
	}
}
