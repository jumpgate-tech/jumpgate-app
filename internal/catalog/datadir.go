package catalog

import (
	"fmt"
	"path"
	"strings"
	"unicode"
)

// systemTrees are directories a DataDir may live under but never BE. Setup
// runs `chown -R` on DataDir as root and clear runs `rm -rf` inside it, so a
// DataDir of /var would re-own the system tree to the service user.
var systemTrees = map[string]bool{
	"/bin": true, "/boot": true, "/dev": true, "/etc": true, "/home": true,
	"/lib": true, "/lib64": true, "/opt": true, "/proc": true, "/root": true,
	"/run": true, "/sbin": true, "/srv": true, "/sys": true, "/tmp": true,
	"/usr": true, "/var": true, "/var/lib": true,
}

// ValidateDataDir checks a node data directory before anything acts on it.
//
// It must be absolute (a relative path resolves against the SSH login
// directory), already clean (so what is checked is what is used), at least
// two components deep, free of whitespace and control characters (the path is
// written unquoted into a systemd unit, where a newline injects a directive
// that root then runs), and not itself a system tree.
func ValidateDataDir(p string) error {
	if p == "" {
		return fmt.Errorf("data directory is empty")
	}
	if !strings.HasPrefix(p, "/") {
		return fmt.Errorf("data directory %q must be an absolute path", p)
	}
	if path.Clean(p) != p {
		return fmt.Errorf("data directory %q is not a clean path (use %q)", p, path.Clean(p))
	}
	for _, r := range p {
		if unicode.IsSpace(r) || unicode.IsControl(r) {
			return fmt.Errorf("data directory %q contains whitespace or a control character", p)
		}
	}
	if strings.Count(p, "/") < 2 {
		return fmt.Errorf("data directory %q must be at least two levels deep", p)
	}
	if systemTrees[p] {
		return fmt.Errorf("data directory %q is a system directory; use a directory inside it", p)
	}
	return nil
}
