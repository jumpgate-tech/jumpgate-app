package setup

// Installing a gateway's internal-CA root into the trust store of the machine
// it runs on — the one manual step that otherwise stood between a fronted
// gateway and a browser that stops warning about it.
//
// The command that does this installs a ROOT certificate authority, so it is
// written ONCE, quoted ONCE, and unit-tested per OS here, rather than being
// assembled at a call site where a single missing quote is a shell injection
// into a privileged command. The caller is responsible for having already
// established that certPath is the gateway's OWN exported root (rootCAPath) and
// not an arbitrary file — this file only turns "that path, on this OS" into the
// exact command, and refuses a path it cannot quote safely.

import (
	"fmt"
	"strings"
)

// TrustStoreInstall is the OS-specific way to add a root certificate to the
// machine's trust store.
type TrustStoreInstall struct {
	// Command is the shell command to run on the target, already quoted for
	// `sh -c`. Callers run it verbatim.
	Command string

	// NeedsRoot is true when the command only succeeds as root and does NOT
	// elevate on its own. darwin's does elevate (osascript ... with
	// administrator privileges), so it is false there; linux and windows need
	// an already-privileged shell.
	NeedsRoot bool

	// ManualCommand, when set, is a plain command an operator runs themselves in
	// a terminal THEY opened, for when Command cannot run automatically. It
	// exists for darwin: `osascript ... with administrator privileges` can only
	// show its auth dialog inside a GUI (Aqua) login session, so when the server
	// was launched detached (over SSH, a background service, `nohup`) the
	// automatic install fails with "SecTrustSettingsSetTrustSettings: The
	// authorization was denied since no user interaction was possible." The
	// osascript one-liner is useless to paste into a plain shell in that state,
	// so this hands back the equivalent `sudo` command, which prompts normally in
	// an interactive terminal. It is quoted for `sh` like Command. Empty when
	// Command is already the run-by-hand form (linux/windows).
	ManualCommand string
}

// TrustStoreCommand builds the trust-store install for goos, installing exactly
// certPath. gatewayID scopes the installed file on Linux so two gateways' roots
// cannot overwrite each other.
//
// SECURITY: this installs a ROOT CA. certPath is validated to contain no shell
// or AppleScript metacharacter before it is interpolated; a path that fails the
// check is refused rather than escaped-and-hoped, because the cost of getting
// the escaping wrong is a command injection running with administrator rights.
func TrustStoreCommand(goos, certPath, gatewayID string) (TrustStoreInstall, error) {
	if err := validateCertPath(certPath); err != nil {
		return TrustStoreInstall{}, err
	}
	switch goos {
	case "darwin":
		// `with administrator privileges` is what lets a non-root GUI app
		// elevate through the standard macOS auth prompt, so nothing here has to
		// already be root. `quoted form of` is AppleScript's own shell-quoting
		// for the inner `do shell script`, so a path with a space survives; the
		// surrounding -e argument is single-quoted for sh, which is safe only
		// because validateCertPath forbids a single quote (and a double quote and
		// backslash, so the AppleScript string literal is safe too).
		script := `do shell script "security add-trusted-cert -d -r trustRoot -k /Library/Keychains/System.keychain " & quoted form of "` + certPath + `" with administrator privileges`
		return TrustStoreInstall{
			Command: "osascript -e '" + script + "'",
			// Fallback for when osascript has no GUI session to prompt in: the
			// same install as a sudo command, which prompts in an interactive
			// terminal. shQuote is safe here because validateCertPath already
			// forbids a single quote.
			ManualCommand: "sudo security add-trusted-cert -d -r trustRoot -k /Library/Keychains/System.keychain " + shQuote(certPath),
		}, nil

	case "linux":
		// update-ca-certificates rebuilds the system bundle from
		// /usr/local/share/ca-certificates. The file is named per gateway so two
		// gateways' roots do not clobber each other and removing one is
		// unambiguous.
		dest := "/usr/local/share/ca-certificates/valve-node-app-" + sanitizeGatewayID(gatewayID) + ".crt"
		return TrustStoreInstall{
			Command:   "cp " + shQuote(certPath) + " " + shQuote(dest) + " && update-ca-certificates",
			NeedsRoot: true,
		}, nil

	case "windows":
		// certutil -addstore ROOT writes the machine root store, which requires
		// an elevated (Administrator) shell. There is no in-band elevation as
		// darwin has, so the caller surfaces this as a command to run rather than
		// attempting it against a shell that would only be denied.
		return TrustStoreInstall{
			Command:   "certutil -addstore -f ROOT " + shQuote(certPath),
			NeedsRoot: true,
		}, nil

	default:
		return TrustStoreInstall{}, fmt.Errorf(
			"valve-node-app has no automatic trust-store install for %q — install %s into that machine's trust store by hand", goos, certPath)
	}
}

// TrustVerifyCommand returns the command that checks whether certPath is ALREADY
// trusted as a root on goos, or "" when this OS has no cheap probe for it.
//
// WHY this exists — the detached-launch false-failure trap: on darwin the
// automatic install runs through osascript, which can raise its authorization
// dialog ONLY inside a GUI (Aqua) login session. A server launched detached
// (over SSH, a background service, nohup) has none, so the install fails with
// "no user interaction was possible". The operator then runs the handed-back
// sudo command by hand, and from that point the certificate IS trusted. A naive
// retry re-runs osascript, fails the same GUI prompt again, and reports failure
// though trust already succeeded — a false negative. Running this probe FIRST
// lets the retry tell the truth: if the root already verifies, report success
// and never touch the install.
//
// darwin: `security verify-cert -c <certPath>` exits 0 for a trusted root and
// exits 1 (CSSMERR_TP_NOT_TRUSTED) for an untrusted self-signed one, so its exit
// code answers the question directly and without side effects. linux and windows
// have no equally cheap, side-effect-free check, so they return "" and the
// caller keeps its existing install-and-report path (no short-circuit).
//
// SECURITY: certPath is validated by the same validateCertPath the install uses,
// and single-quoted the same way, because it is interpolated into a shell
// command; a path that fails the check is refused, not escaped-and-hoped.
func TrustVerifyCommand(goos, certPath string) (string, error) {
	if err := validateCertPath(certPath); err != nil {
		return "", err
	}
	switch goos {
	case "darwin":
		return "security verify-cert -c " + shQuote(certPath), nil
	default:
		return "", nil
	}
}

// validateCertPath refuses a path that could break out of the quoting the
// trust-store commands rely on. It is deliberately strict: the path is one this
// app derived (rootCAPath), so a metacharacter in it is far likelier a bug than
// a real filename, and a root-CA install is the wrong place to be lenient.
func validateCertPath(p string) error {
	p = strings.TrimSpace(p)
	if p == "" {
		return fmt.Errorf("trust: empty certificate path")
	}
	if !strings.HasPrefix(p, "/") {
		return fmt.Errorf("trust: certificate path %q is not an absolute POSIX path", p)
	}
	for _, r := range p {
		switch r {
		case '\'', '"', '\\', '`', '$', '\n', '\r', 0:
			return fmt.Errorf("trust: certificate path %q contains an unsafe character %q", p, string(r))
		}
	}
	return nil
}
