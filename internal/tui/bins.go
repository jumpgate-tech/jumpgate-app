package tui

import (
	"os"
	"path/filepath"
	"strings"
)

// Programs the TUI runs (jumpgate itself, ssh, a clipboard tool) are found
// once, as absolute paths. A relative path is refused, so a planted ./ssh or
// ./pbcopy in the working directory is never run.

// bins are the resolved programs; "" is not found.
type bins struct {
	jumpgate string
	ssh      string
}

// resolveBins finds jumpgate ($JUMPGATE_BIN, else self, the running
// executable) and ssh ($JUMPGATE_SSH, a sibling of self, then PATH).
func resolveBins(self string, getenv func(string) string, goos string) bins {
	return bins{
		jumpgate: resolveSelf(self, getenv, goos),
		ssh:      resolveBin("ssh", "JUMPGATE_SSH", self, getenv, goos),
	}
}

// executable: p is a regular file this user may run.
func executable(p, goos string) bool {
	st, err := os.Stat(p)
	if err != nil || !st.Mode().IsRegular() {
		return false
	}
	return goos == "windows" || st.Mode().Perm()&0o111 != 0
}

func exeName(name, goos string) string {
	if goos == "windows" && !strings.HasSuffix(strings.ToLower(name), ".exe") {
		return name + ".exe"
	}
	return name
}

// onPath finds name on PATH with its relative entries dropped: an absolute
// path, or "".
func onPath(name string, getenv func(string) string, goos string) string {
	exe := exeName(name, goos)
	for _, dir := range filepath.SplitList(getenv("PATH")) {
		if dir == "" || !filepath.IsAbs(dir) {
			continue
		}
		if p := filepath.Join(dir, exe); executable(p, goos) {
			return p
		}
	}
	return ""
}

// resolveBin finds name: $envVar when set (it must be absolute), then a
// sibling of self, then PATH. "" when none is usable.
func resolveBin(name, envVar, self string, getenv func(string) string, goos string) string {
	if v := getenv(envVar); v != "" {
		if filepath.IsAbs(v) && executable(v, goos) {
			return v
		}
		return ""
	}
	if filepath.IsAbs(self) {
		if p := filepath.Join(filepath.Dir(self), exeName(name, goos)); p != self && executable(p, goos) {
			return p
		}
	}
	return onPath(name, getenv, goos)
}

// resolveSelf is the jumpgate the TUI re-runs: $JUMPGATE_BIN when set
// (absolute only), else self when its path is absolute.
func resolveSelf(self string, getenv func(string) string, goos string) string {
	if v := getenv("JUMPGATE_BIN"); v != "" {
		if filepath.IsAbs(v) && executable(v, goos) {
			return v
		}
		return ""
	}
	if filepath.IsAbs(self) {
		return self
	}
	return ""
}
