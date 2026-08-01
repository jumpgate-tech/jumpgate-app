package setup

import (
	"strings"
	"testing"
)

// The trust-store command installs a ROOT certificate authority, so the exact
// string per OS is worth pinning: a wrong keychain, a dropped quote or a
// missing elevation is the difference between "the warning is gone" and a
// silent command injection running as administrator.
func TestTrustStoreCommand_PerOS(t *testing.T) {
	const path = "/home/ops/.valve-node-app/caddy-root.crt"

	darwin, err := TrustStoreCommand("darwin", path, "default")
	if err != nil {
		t.Fatalf("darwin: %v", err)
	}
	// osascript with `administrator privileges` is what elevates a non-root GUI
	// app, so darwin must NOT be flagged as needing root itself.
	if darwin.NeedsRoot {
		t.Error("darwin should elevate via osascript, not require an already-root shell")
	}
	for _, want := range []string{
		"osascript -e '",
		"security add-trusted-cert -d -r trustRoot -k /Library/Keychains/System.keychain",
		"with administrator privileges",
		`quoted form of "` + path + `"`,
	} {
		if !strings.Contains(darwin.Command, want) {
			t.Errorf("darwin command missing %q:\n%s", want, darwin.Command)
		}
	}

	linux, err := TrustStoreCommand("linux", path, "edge")
	if err != nil {
		t.Fatalf("linux: %v", err)
	}
	if !linux.NeedsRoot {
		t.Error("linux install writes /usr/local/share and runs update-ca-certificates — it needs root")
	}
	// The path is single-quoted for sh, the destination is scoped by gateway id
	// so two gateways cannot clobber each other's root, and the bundle is rebuilt.
	for _, want := range []string{
		"cp '" + path + "'",
		"/usr/local/share/ca-certificates/valve-node-app-edge.crt",
		"&& update-ca-certificates",
	} {
		if !strings.Contains(linux.Command, want) {
			t.Errorf("linux command missing %q:\n%s", want, linux.Command)
		}
	}

	windows, err := TrustStoreCommand("windows", path, "default")
	if err != nil {
		t.Fatalf("windows: %v", err)
	}
	if !windows.NeedsRoot {
		t.Error("windows certutil -addstore ROOT needs an elevated shell")
	}
	if !strings.Contains(windows.Command, "certutil -addstore -f ROOT '"+path+"'") {
		t.Errorf("windows command wrong:\n%s", windows.Command)
	}
}

// An OS we do not automate must be an error the caller can turn into "install
// it by hand", not a half-formed command that runs the wrong thing.
func TestTrustStoreCommand_UnknownOSErrors(t *testing.T) {
	if _, err := TrustStoreCommand("plan9", "/x/caddy-root.crt", "default"); err == nil {
		t.Fatal("an unknown OS must not silently produce a command")
	}
}

// The path is interpolated into a command that runs as administrator, so a
// path carrying a shell or AppleScript metacharacter is REFUSED rather than
// escaped-and-hoped. These are the characters that would break out of the
// quoting each branch relies on.
func TestTrustStoreCommand_RejectsUnsafePaths(t *testing.T) {
	for _, bad := range []string{
		"relative/caddy-root.crt",                       // not absolute
		"/x/root.crt'; rm -rf / #",                      // single quote → breaks sh
		`/x/root.crt" with administrator privileges; (`, // double quote → breaks the AppleScript literal
		"/x/root$(id).crt",                              // command substitution
		"/x/root`id`.crt",                               // backtick substitution
		"/x/root\\.crt",                                 // backslash → breaks the AppleScript literal
		"/x/root\n.crt",                                 // newline
	} {
		for _, goos := range []string{"darwin", "linux", "windows"} {
			if _, err := TrustStoreCommand(goos, bad, "default"); err == nil {
				t.Errorf("%s: unsafe path %q was accepted", goos, bad)
			}
		}
	}
}
