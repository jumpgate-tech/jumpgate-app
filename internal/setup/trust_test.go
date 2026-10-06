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
	// darwin carries a run-by-hand fallback for when osascript has no GUI session
	// to prompt in ("no user interaction was possible"): a sudo command that
	// prompts in an interactive terminal instead. The path is single-quoted for
	// sh, safe because validateCertPath forbids a single quote.
	for _, want := range []string{
		"sudo security add-trusted-cert -d -r trustRoot -k /Library/Keychains/System.keychain",
		"'" + path + "'",
	} {
		if !strings.Contains(darwin.ManualCommand, want) {
			t.Errorf("darwin ManualCommand missing %q:\n%s", want, darwin.ManualCommand)
		}
	}
	// The osascript one-liner is not what a human should paste into a plain
	// shell, so the fallback must not merely echo it.
	if strings.Contains(darwin.ManualCommand, "osascript") {
		t.Errorf("darwin ManualCommand should be a plain sudo command, not the osascript form:\n%s", darwin.ManualCommand)
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

	// On Windows the root lives under the user's profile, a drive path.
	winPath := `C:\Users\O'Neil Smith\.valve-node-app\caddy-root.crt`
	windows, err := TrustStoreCommand("windows", winPath, "default")
	if err != nil {
		t.Fatalf("windows: %v", err)
	}
	if !windows.NeedsRoot {
		t.Error("windows certutil -addstore ROOT needs an elevated shell")
	}
	if !strings.Contains(windows.Command, `certutil -addstore -f ROOT "`+winPath+`"`) {
		t.Errorf("windows command wrong:\n%s", windows.Command)
	}
}

// The truthful-retry probe. darwin returns the command that reports whether the
// root is ALREADY trusted; linux and windows have no cheap probe and return "",
// which tells the caller to skip the short-circuit rather than run something
// that lies.
func TestTrustVerifyCommand_PerOS(t *testing.T) {
	const path = "/home/ops/.valve-node-app/caddy-root.crt"

	darwin, err := TrustVerifyCommand("darwin", path)
	if err != nil {
		t.Fatalf("darwin: %v", err)
	}
	// `security verify-cert -c <path>` exits 0 for a trusted root and 1 for one
	// that is not, so its exit code answers "already trusted?" directly. The path
	// is single-quoted for sh, safe because validateCertPath forbids a single
	// quote.
	for _, want := range []string{
		"security verify-cert -c ",
		"'" + path + "'",
	} {
		if !strings.Contains(darwin, want) {
			t.Errorf("darwin verify command missing %q:\n%s", want, darwin)
		}
	}

	// linux and windows have no equally cheap, side-effect-free probe, so they opt
	// out with "" — the caller then keeps its existing install-and-report path.
	for goos, p := range map[string]string{"linux": path, "windows": `C:\Users\dev\.valve-node-app\caddy-root.crt`} {
		cmd, err := TrustVerifyCommand(goos, p)
		if err != nil {
			t.Fatalf("%s: %v", goos, err)
		}
		if cmd != "" {
			t.Errorf("%s has no cheap trust probe and must return \"\", got %q", goos, cmd)
		}
	}
}

// The verify command interpolates the path into a shell command exactly as the
// install does, so it REFUSES a path with a shell metacharacter rather than
// escaping-and-hoping.
func TestTrustVerifyCommand_RejectsUnsafePaths(t *testing.T) {
	for _, bad := range []string{
		"relative/caddy-root.crt",  // not absolute
		"/x/root.crt'; rm -rf / #", // single quote → breaks sh
		"/x/root$(id).crt",         // command substitution
		"/x/root`id`.crt",          // backtick substitution
	} {
		if _, err := TrustVerifyCommand("darwin", bad); err == nil {
			t.Errorf("unsafe path %q was accepted", bad)
		}
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
		"\n/x/root.crt",                                 // a leading newline: the raw path is checked
	} {
		for _, goos := range []string{"darwin", "linux", "windows"} {
			if _, err := TrustStoreCommand(goos, bad, "default"); err == nil {
				t.Errorf("%s: unsafe path %q was accepted", goos, bad)
			}
		}
	}
}

// A Windows path is pasted into an elevated cmd.exe, double-quoted: %VAR%,
// !VAR!, ^ and a double quote would still act there, and a POSIX path is not
// a Windows one.
func TestTrustStoreCommandWindowsRefusesCmdMetacharacters(t *testing.T) {
	for _, bad := range []string{
		"/var/lib/x/caddy-root.crt",
		`relative\caddy-root.crt`,
		`C:\Users\%USERNAME%\root.crt`,
		`C:\Users\a!b!\root.crt`,
		`C:\Users\a^b\root.crt`,
		`C:\Users\a" & calc & "\root.crt`,
		"C:\\x\nroot.crt",
		// PowerShell (Windows Terminal's admin default) expands these inside
		// double quotes; the pasted command must be inert there too.
		`C:\Users\$env:USERNAME\root.crt`,
		`C:\Users\$(calc)\root.crt`,
		"C:\\Users\\a`b\\root.crt",
		// The raw path is checked: a leading line break would run the line
		// before it.
		"\nC:\\Users\\dev\\root.crt",
	} {
		if _, err := TrustStoreCommand("windows", bad, "default"); err == nil {
			t.Errorf("unsafe Windows path %q was accepted", bad)
		}
	}
}

// I-12 (Linux): Fedora and Arch have update-ca-trust, not
// update-ca-certificates; the command handles both, and the manual form is a
// sudo command a person can paste.
func TestTrustStoreCommandLinuxHandlesBothTools(t *testing.T) {
	got, err := TrustStoreCommand("linux", "/var/lib/x/caddy-root.crt", "edge")
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"update-ca-certificates", "update-ca-trust extract", "/etc/pki/ca-trust/source/anchors", "/usr/local/share/ca-certificates/valve-node-app-edge.crt"} {
		if !strings.Contains(got.Command, want) {
			t.Errorf("command lacks %q:\n%s", want, got.Command)
		}
	}
	if !strings.HasPrefix(got.ManualCommand, "sudo sh -c ") {
		t.Errorf("ManualCommand %q, want a sudo sh -c form", got.ManualCommand)
	}
}
