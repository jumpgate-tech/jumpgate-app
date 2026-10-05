package main

import (
	"os"
	"path/filepath"
	"strings"
)

// expandHome expands a leading ~ that the shell left alone: cmd.exe and
// Windows PowerShell never expand it, and bash does not in --key=~/x (M-2).
// Only "~" and "~/…" (and "~\…" on Windows) are expanded; "~user" is left as
// it is, since jumpgate cannot resolve other users' homes portably.
func expandHome(p string) (string, error) {
	if p != "~" && !strings.HasPrefix(p, "~/") && !(hostGOOS == "windows" && strings.HasPrefix(p, `~\`)) {
		return p, nil
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	rest := p[1:]
	if hostGOOS == "windows" {
		rest = strings.ReplaceAll(rest, `\`, "/") // Join then uses the host's separator
	}
	return filepath.Join(home, rest), nil
}
