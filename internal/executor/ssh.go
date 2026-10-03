package executor

// Paths handled in this file are paths on the *remote* target and are always
// POSIX. That is why this file must not import "path/filepath" — see
// remotepath.go for the full rationale. Local-filesystem concerns (the
// known_hosts file, the private key) live in hostkey.go.
import (
	"bytes"
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"net"
	"os"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"golang.org/x/crypto/ssh"
)

// sshExecutor runs commands and moves files on a remote host over SSH.
type sshExecutor struct {
	client *ssh.Client

	// kills tracks group kills still running after a cancelled Run returned,
	// so Close can let them finish before it tears down the connection.
	// closed, under mu, stops new ones from starting once Close has begun.
	mu     sync.Mutex
	closed bool
	kills  sync.WaitGroup
}

// NewSSH dials user@host:port (default port 22) using the private key at
// cfg.KeyPath, verifying the remote host key against cfg.HostKeyFile using a
// trust-on-first-use policy: an unknown host's key is appended to
// HostKeyFile (created 0600 on first use); a known host presenting a
// different key is rejected with an error.
func NewSSH(cfg SSHConfig) (Executor, error) {
	port := cfg.Port
	if port == 0 {
		port = 22
	}
	addr := net.JoinHostPort(cfg.Host, strconv.Itoa(port))

	keyBytes, err := os.ReadFile(cfg.KeyPath)
	if err != nil {
		return nil, fmt.Errorf("read private key %s: %w", cfg.KeyPath, err)
	}
	signer, err := ssh.ParsePrivateKey(keyBytes)
	if err != nil {
		return nil, fmt.Errorf("parse private key %s: %w", cfg.KeyPath, err)
	}

	config := &ssh.ClientConfig{
		User:            cfg.User,
		Auth:            []ssh.AuthMethod{ssh.PublicKeys(signer)},
		HostKeyCallback: tofuHostKeyCallback(cfg.HostKeyFile),
		Timeout:         10 * time.Second,
	}

	client, err := ssh.Dial("tcp", addr, config)
	if err != nil {
		return nil, err
	}

	return &sshExecutor{client: client}, nil
}

func (s *sshExecutor) Run(ctx context.Context, cmd string, opts *RunOpts) (Result, error) {
	session, err := s.client.NewSession()
	if err != nil {
		return Result{}, fmt.Errorf("new ssh session: %w", err)
	}
	defer session.Close()

	// SSH sessions have no WaitDelay equivalent: unlike os/exec, there's no
	// stdlib backstop that forcibly unblocks a stuck read after ctx is
	// canceled. That guarantee instead comes from this goroutine: closing
	// the session on ctx.Done() tears down its underlying channel, which
	// unblocks the io.Copy below (it returns an error reading the now-closed
	// stdoutPipe) so Run cannot hang forever past ctx cancellation.
	//
	// Closing the session does not stop the command on the box, though, so
	// the goroutine then kills the command's process group, whose id the
	// wrapper reports on its first stdout line (see wrapInProcessGroup). If
	// that line has not arrived yet it waits up to markerWait for it first;
	// that wait is the only delay cancellation adds to Run. The kill itself
	// runs after the session is closed and Run may return before it ends.
	var pgid atomic.Int64
	firstLine := make(chan struct{}) // closed once the first stdout line is in
	copyDone := make(chan struct{})  // closed once stdout hits EOF or errors
	done := make(chan struct{})
	defer close(done)
	go func() {
		select {
		case <-ctx.Done():
		case <-done:
			return
		}
		wait := time.NewTimer(markerWait)
		defer wait.Stop()
		select {
		case <-firstLine:
		case <-copyDone:
		case <-done:
		case <-wait.C:
		}
		// Register the kill before closing the session lets Run return, so a
		// caller's Close right after Run waits for it.
		pg := pgid.Load()
		kill := pg > 1 && s.startKill()
		_ = session.Signal(ssh.SIGTERM) // honoured by OpenSSH >= 7.9; harmless otherwise
		_ = session.Close()
		if kill {
			defer s.kills.Done()
			s.killGroup(int(pg))
		}
	}()

	var stdoutBuf, stderrBuf bytes.Buffer
	session.Stderr = &stderrBuf

	stdoutPipe, err := session.StdoutPipe()
	if err != nil {
		return Result{}, fmt.Errorf("ssh stdout pipe: %w", err)
	}

	if opts != nil && opts.Stdin != nil {
		session.Stdin = opts.Stdin
	}

	if err := session.Start(wrapInProcessGroup(cmd)); err != nil {
		return Result{}, fmt.Errorf("start ssh command: %w", err)
	}

	var streamFn StreamFunc
	if opts != nil {
		streamFn = opts.Stream
	}
	w := &lineStreamer{buf: &stdoutBuf, fn: streamFn, onFirst: func(line string) bool {
		defer close(firstLine) // after the Store, so the canceller sees the pgid
		if !strings.HasPrefix(line, pgidMarker) {
			return false
		}
		if pg, ok := parsePgidLine(line); ok {
			pgid.Store(int64(pg))
		}
		return true // a marker without a group id (pgidNone) is swallowed too
	}}

	copyErrCh := make(chan error, 1)
	go func() {
		_, err := io.Copy(w, stdoutPipe)
		close(copyDone)
		copyErrCh <- err
	}()

	copyErr := <-copyErrCh
	w.Flush()
	waitErr := session.Wait()

	if ctx.Err() != nil {
		return Result{}, ctx.Err()
	}
	if copyErr != nil {
		return Result{}, copyErr
	}

	result := Result{
		Stdout: stdoutBuf.String(),
		Stderr: stderrBuf.String(),
	}

	if waitErr != nil {
		var exitErr *ssh.ExitError
		if errors.As(waitErr, &exitErr) {
			result.ExitCode = exitErr.ExitStatus()
			return result, nil
		}
		return Result{}, waitErr
	}

	return result, nil
}

// markerWait bounds how long a cancelled Run waits for the marker line before
// giving up on killing the command's group.
const markerWait = 2 * time.Second

// killGroupBudget bounds killGroup, opening the session included.
const killGroupBudget = 10 * time.Second

// startKill registers a group kill with Close. It reports false once Close
// has begun, when the connection is going away and a kill could not be sent.
func (s *sshExecutor) startKill() bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.closed {
		return false
	}
	s.kills.Add(1)
	return true
}

// killGroup stops a cancelled command's process group on a fresh session. The
// caller's context is already done, so it has its own budget, and the whole
// sequence is inside it: NewSession takes no context and blocks on a stalled
// connection. A session still stalled when the budget runs out is abandoned;
// its goroutine ends when the client is closed.
func (s *sshExecutor) killGroup(pgid int) {
	finished := make(chan struct{})
	go func() {
		defer close(finished)
		sess, err := s.client.NewSession()
		if err != nil {
			return
		}
		defer sess.Close()
		_ = sess.Run(killGroupCmd(pgid))
	}()
	budget := time.NewTimer(killGroupBudget)
	defer budget.Stop()
	select {
	case <-finished:
	case <-budget.C:
	}
}

// WriteFile writes content to path on the remote host. The content travels on
// the session's stdin, never in the command line, where any local user on the
// target could read it from /proc. It is written to a temp file created under
// umask 077, chmod'ed, then renamed into place, so the file is never readable
// wider than its final mode and a reader never sees it half-written.
func (s *sshExecutor) WriteFile(ctx context.Context, path string, content []byte, mode fs.FileMode) error {
	res, err := s.Run(ctx, writeFileCmd(path, mode), &RunOpts{Stdin: bytes.NewReader(content)})
	if err != nil {
		return err
	}
	if res.ExitCode != 0 {
		return fmt.Errorf("write remote file %s: exit %d: %s", path, res.ExitCode, res.Stderr)
	}
	return nil
}

// writeFileCmd builds the POSIX shell line WriteFile runs. mkdir -p runs
// first, under the caller's umask, so a new parent directory a service must
// traverse is not created 0700; only the temp file is created under 077.
// remoteDir (not filepath.Dir) keeps this correct on a Windows control plane.
func writeFileCmd(remotePath string, mode fs.FileMode) string {
	q := shQuote(remotePath)
	return fmt.Sprintf(
		"mkdir -p %s && (umask 077 && tmp=$(mktemp %s.XXXXXX) && cat > \"$tmp\" && chmod %o \"$tmp\" && mv -f \"$tmp\" %s || { rm -f \"$tmp\"; exit 1; })",
		shQuote(remoteDir(remotePath)), shQuote(remotePath), mode.Perm(), q,
	)
}

// ReadFile reads path from the remote host via `base64 < path` over Run.
func (s *sshExecutor) ReadFile(ctx context.Context, path string) ([]byte, error) {
	res, err := s.Run(ctx, readFileCmd(path), nil)
	if err != nil {
		return nil, err
	}
	if res.ExitCode != 0 {
		return nil, fmt.Errorf("read remote file %s: exit %d: %s", path, res.ExitCode, res.Stderr)
	}
	decoded, err := base64.StdEncoding.DecodeString(strings.ReplaceAll(strings.TrimSpace(res.Stdout), "\n", ""))
	if err != nil {
		return nil, fmt.Errorf("decode remote file %s: %w", path, err)
	}
	return decoded, nil
}

// readFileCmd builds the POSIX shell line ReadFile ships to the target. Pure,
// for the same testability reason as writeFileCmd; it takes the remote path
// verbatim, so there is no separator hazard here — only quoting.
func readFileCmd(remotePath string) string {
	return fmt.Sprintf("base64 < %s", shQuote(remotePath))
}

// Close waits for any group kill a cancelled Run left running (each bounded
// by killGroupBudget), then closes the connection.
func (s *sshExecutor) Close() error {
	s.mu.Lock()
	s.closed = true
	s.mu.Unlock()
	s.kills.Wait()
	return s.client.Close()
}

// shQuote wraps s in single quotes for safe embedding in a `sh -c` command,
// escaping any embedded single quotes.
func shQuote(s string) string {
	return "'" + strings.ReplaceAll(s, "'", `'"'"'`) + "'"
}
