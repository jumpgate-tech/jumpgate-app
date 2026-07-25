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
	"time"

	"golang.org/x/crypto/ssh"
)

// sshExecutor runs commands and moves files on a remote host over SSH.
type sshExecutor struct {
	client *ssh.Client
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
	done := make(chan struct{})
	defer close(done)
	go func() {
		select {
		case <-ctx.Done():
			_ = session.Close()
		case <-done:
		}
	}()

	var stdoutBuf, stderrBuf bytes.Buffer
	session.Stderr = &stderrBuf

	stdoutPipe, err := session.StdoutPipe()
	if err != nil {
		return Result{}, fmt.Errorf("ssh stdout pipe: %w", err)
	}

	if err := session.Start(cmd); err != nil {
		return Result{}, fmt.Errorf("start ssh command: %w", err)
	}

	var streamFn StreamFunc
	if opts != nil {
		streamFn = opts.Stream
	}
	w := &lineStreamer{buf: &stdoutBuf, fn: streamFn}

	copyErrCh := make(chan error, 1)
	go func() {
		_, err := io.Copy(w, stdoutPipe)
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

// WriteFile writes content to path on the remote host by piping a
// base64-encoded copy through the remote `base64 -d` and setting mode with
// `chmod`. No SFTP subsystem is required.
func (s *sshExecutor) WriteFile(ctx context.Context, path string, content []byte, mode fs.FileMode) error {
	res, err := s.Run(ctx, writeFileCmd(path, content, mode), nil)
	if err != nil {
		return err
	}
	if res.ExitCode != 0 {
		return fmt.Errorf("write remote file %s: exit %d: %s", path, res.ExitCode, res.Stderr)
	}
	return nil
}

// writeFileCmd builds the single POSIX shell line that WriteFile ships to the
// target. It is a pure function of its inputs — no client, no context — so the
// remote path construction can be asserted directly in tests, on any host OS.
// remoteDir (not filepath.Dir) is what keeps this correct when the control
// plane is Windows: filepath.Dir would emit `\var\lib\...` into the mkdir -p,
// breaking every unit/config/jwt write against every Linux target.
func writeFileCmd(remotePath string, content []byte, mode fs.FileMode) string {
	encoded := base64.StdEncoding.EncodeToString(content)
	return fmt.Sprintf(
		"mkdir -p %s && printf %%s %s | base64 -d > %s && chmod %o %s",
		shQuote(remoteDir(remotePath)),
		shQuote(encoded),
		shQuote(remotePath),
		mode.Perm(),
		shQuote(remotePath),
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

func (s *sshExecutor) Close() error {
	return s.client.Close()
}

// shQuote wraps s in single quotes for safe embedding in a `sh -c` command,
// escaping any embedded single quotes.
func shQuote(s string) string {
	return "'" + strings.ReplaceAll(s, "'", `'"'"'`) + "'"
}
