package executor

import (
	"strconv"
	"strings"
)

// Closing an SSH session does not stop the command it started, so to stop a
// cancelled command the executor must know its process group and kill that
// group on a fresh session. wrapInProcessGroup runs the command in a group of
// its own and prints the group id as the first stdout line, which Run reads
// and strips.

// pgidMarker prefixes the first stdout line of every wrapped command. It is
// followed by the group id, or by pgidNone when no group could be made.
const pgidMarker = "jumpgate-pgid:"

// pgidNone follows pgidMarker when the command shares the wrapper's group.
// parsePgidLine rejects it, so cancellation falls back to closing the session.
const pgidNone = "none"

// markerScript is run by sh with $1 = "g" (this process leads its own group)
// or anything else (it does not), and $2 = the command. It prints the marker
// and then execs the command, which keeps the pid and therefore the group.
// It is embedded in single quotes, so it must not contain any.
const markerScript = `if [ "$1" = g ]; then printf "` + pgidMarker + `%s\n" "$$"; else printf "` +
	pgidMarker + pgidNone + `\n"; fi; exec sh -c "$2"`

// wrapInProcessGroup returns one POSIX shell line that runs cmd in a process
// group of its own, prints the marker line first, and exits with cmd's status.
// stdin, stdout and stderr all reach cmd unchanged.
//
// It picks a mechanism on the box at run time:
//  1. util-linux setsid with --wait (setsidBranch), on any modern Linux.
//  2. Otherwise shell job control (jobControlBranch), which bash and macOS sh
//     grant without a tty.
//  3. dash with no setsid refuses job control without a tty. The command then
//     runs in the wrapper's group and the marker says pgidNone. Cancelling it
//     only closes the session, which does not stop it. This gap is accepted
//     (ruling R7).
func wrapInProcessGroup(cmd string) string {
	return "if setsid -w true >/dev/null 2>&1; then " + setsidBranch(cmd) + "; fi; " + jobControlBranch(cmd)
}

// setsidBranch makes cmd a session, and so a group, leader. -w waits for it
// and returns its exit status. setsid forks only when the caller already
// leads a group, so the pid markerScript prints is the group id either way.
func setsidBranch(cmd string) string {
	return "exec setsid -w sh -c '" + markerScript + "' sh g " + shQuote(cmd)
}

// jobControlBranch runs cmd as a background job with job control on (set -m),
// which gives the job a group whose id is its pid, and waits for it. The
// shell's own stderr goes to /dev/null so job-completion notices ("[1]+
// Done") never reach the caller; the job gets the original stderr back on fd
// 3. When the shell refused job control ($- lacks m), cmd runs in the
// foreground instead: a background job without job control would get
// /dev/null as its stdin.
func jobControlBranch(cmd string) string {
	q := shQuote(cmd)
	return "exec 3>&2 2>/dev/null; set -m; case $- in *m*) ;; *) exec sh -c '" + markerScript + "' sh - " + q +
		" 2>&3 3>&-;; esac; sh -c '" + markerScript + "' sh g " + q + " 2>&3 3>&- & wait \"$!\""
}

// parsePgidLine reads the marker line. ok is false for any other line, and
// for a marker that carries no usable group id.
func parsePgidLine(line string) (int, bool) {
	rest, found := strings.CutPrefix(line, pgidMarker)
	if !found {
		return 0, false
	}
	n, err := strconv.Atoi(strings.TrimSpace(rest))
	return n, err == nil && n > 1
}

// killGroupCmd stops a process group: TERM, then up to five seconds for it to
// exit, then KILL. It uses the POSIX "kill -s SIGNAL" form because dash's
// kill builtin does not accept "-TERM" before "--" (ruling R8).
func killGroupCmd(pgid int) string {
	g := strconv.Itoa(pgid)
	return "kill -s TERM -- -" + g + " 2>/dev/null; i=0; while [ $i -lt 5 ] && kill -s 0 -- -" + g +
		" 2>/dev/null; do sleep 1; i=$((i+1)); done; kill -s KILL -- -" + g + " 2>/dev/null; exit 0"
}
