package executor

import "path"

// Remote paths are POSIX paths, always — the targets valve-node-app drives are
// Linux hosts, no matter what the control plane runs on. The stdlib's
// "path/filepath" is deliberately host-dependent: on a Windows control plane
// filepath.Dir("/var/lib/valve-node-app/369/jwt.hex") returns
// `\var\lib\valve-node-app\369`, which is meaningless to the `mkdir -p` we ship
// over SSH. Every path that will be interpreted by the *target's* shell must
// therefore be built with "path" (slash-only, host-independent), never
// "path/filepath". internal/setup/steps.go and internal/catalog already follow
// this rule; these helpers exist so the executor can follow it by construction
// and so the rule is unit-testable.
//
// The mirror-image rule also holds: anything touching the operator's own
// filesystem (the config dir, the known_hosts file, an SSH private key) is a
// local path and must keep using "path/filepath". That code lives in
// hostkey.go and local.go; nothing in this file or ssh.go may import filepath.

// remoteDir returns the parent directory of a POSIX path on the target host.
// It is path.Dir, named to make the remote/local distinction explicit at every
// call site (and to keep filepath.Dir from ever creeping back in).
func remoteDir(p string) string {
	return path.Dir(p)
}
