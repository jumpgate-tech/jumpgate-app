package api

import (
	"errors"
	"fmt"
	"path/filepath"
	"strings"
)

// sshNoArgFlags are the ssh flags a server-built shell command may use; the
// server emits only -p, -i, -l and -o. Everything else, including every flag
// that forwards, jumps, backgrounds, multiplexes or reads another config, is
// refused rather than enumerated.
const sshNoArgFlags = "46qvCtT"

var sshValueFlags = map[string]bool{"-p": true, "-i": true, "-l": true, "-o": true}

// sshForbiddenOptions can never appear in a "-o" of a server-built command.
var sshForbiddenOptions = map[string]bool{
	"localcommand": true, "permitlocalcommand": true, "controlpath": true,
	"controlmaster": true, "knownhostscommand": true, "match": true,
	"include": true, "localforward": true, "remoteforward": true,
	"dynamicforward": true, "pkcs11provider": true, "securitykeyprovider": true,
}

// CheckSSHArgv screens an ssh command received from a server before this
// machine runs it. The server is trusted to know the box, not to choose what
// runs locally, so a compromised or impersonated server must not turn
// "open a shell" into local command execution or an unverified hop. It
// accepts only the system ssh, a small set of flags, "-o" options that cannot
// run a command, jump, forward or loosen host-key checking, and a single
// destination after "--".
func CheckSSHArgv(argv []string) error {
	if len(argv) == 0 {
		return errors.New("the ssh command is empty")
	}
	for _, a := range argv {
		if strings.ContainsAny(a, "\n\r\x00") {
			return errors.New("the ssh command has a line break or NUL in an argument")
		}
	}
	if b := filepath.Base(argv[0]); b != "ssh" && b != "ssh.exe" {
		return errors.New("the program is not the system ssh")
	}
	i := 1
	for ; i < len(argv); i++ {
		a := argv[i]
		if a == "--" {
			break
		}
		if !strings.HasPrefix(a, "-") || len(a) < 2 {
			return fmt.Errorf("unexpected argument %d before --", i)
		}
		if sshValueFlags[a] {
			i++
			if i >= len(argv) || argv[i] == "--" {
				return fmt.Errorf("%s has no value", a)
			}
			if a == "-o" {
				if err := checkSSHOption(argv[i]); err != nil {
					return err
				}
			}
			continue
		}
		if strings.HasPrefix(a, "-o") { // -oKey=value
			if err := checkSSHOption(a[2:]); err != nil {
				return err
			}
			continue
		}
		if len(a) > 2 && sshValueFlags[a[:2]] { // -p22, -ifile, -lroot
			continue
		}
		for _, c := range a[1:] {
			if !strings.ContainsRune(sshNoArgFlags, c) {
				return fmt.Errorf("the flag -%c is not allowed", c)
			}
		}
	}
	if i >= len(argv) {
		return errors.New("the ssh command has no -- before the destination")
	}
	dest := argv[i+1:]
	if len(dest) != 1 || dest[0] == "" || strings.HasPrefix(dest[0], "-") {
		return errors.New("the ssh command must end with -- and one destination")
	}
	return nil
}

func checkSSHOption(opt string) error {
	k, v := opt, ""
	if j := strings.IndexAny(opt, "= \t"); j >= 0 {
		k, v = opt[:j], strings.TrimLeft(opt[j:], "= \t")
	}
	key, val := strings.ToLower(k), strings.ToLower(strings.TrimSpace(v))
	switch {
	case key == "proxyjump" || key == "proxycommand":
		if val != "none" {
			return fmt.Errorf("the option %s must be none", k)
		}
	case key == "stricthostkeychecking":
		if val != "yes" {
			return errors.New("StrictHostKeyChecking must be yes")
		}
	case sshForbiddenOptions[key]:
		return fmt.Errorf("the option %s is not allowed", k)
	}
	return nil
}
