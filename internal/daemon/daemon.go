// internal/daemon/daemon.go

// Package daemon keeps one controller server per user and lets every jumpgate
// command find it. The server outlives the command that started it, so a TUI
// or CLI can quit and come back to the same server and the same jobs.
package daemon

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"time"

	"github.com/valve-tech/jumpgate/internal/config"
	"github.com/valve-tech/jumpgate/internal/filelock"
)

// ErrAlreadyRunning means another server holds the lock.
var ErrAlreadyRunning = errors.New("daemon: a jumpgate server is already running for this user")

// BaseURL is the authority used for requests over the unix socket; the
// transport ignores it.
const BaseURL = "http://jumpgate"

// Info is server.json.
type Info struct {
	PID       int       `json:"pid"`
	Socket    string    `json:"socket"`
	HTTPAddr  string    `json:"httpAddr"`
	Token     string    `json:"token"`
	Version   string    `json:"version"`
	StartedAt time.Time `json:"startedAt"`
}

// RunDir is ~/.jumpgate/run, created 0700.
func RunDir() (string, error) {
	base, err := config.Dir()
	if err != nil {
		return "", err
	}
	dir := filepath.Join(base, "run")
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return "", err
	}
	return dir, nil
}

// Holder owns the single-instance lock for the server's lifetime.
type Holder struct {
	lock *filelock.Handle
	dir  string
}

// Acquire takes the lock or reports ErrAlreadyRunning.
func Acquire() (*Holder, error) {
	dir, err := RunDir()
	if err != nil {
		return nil, err
	}
	h, err := filelock.TryLock(filepath.Join(dir, "server.lock"))
	if errors.Is(err, filelock.ErrLocked) {
		return nil, alreadyRunning(dir)
	}
	if err != nil {
		return nil, err
	}
	return &Holder{lock: h, dir: dir}, nil
}

// alreadyRunning wraps ErrAlreadyRunning with the holder's pid from
// server.json when there is one. The pid is for the message only; nothing
// signals it.
func alreadyRunning(dir string) error {
	b, err := os.ReadFile(filepath.Join(dir, "server.json"))
	if err != nil {
		return ErrAlreadyRunning
	}
	var info Info
	if json.Unmarshal(b, &info) != nil || info.PID <= 0 {
		return ErrAlreadyRunning
	}
	return fmt.Errorf("%w, pid %d", ErrAlreadyRunning, info.PID)
}

// Publish writes server.json (0600: it carries the session token) once the
// listeners are up.
func (h *Holder) Publish(info Info) error {
	b, err := json.MarshalIndent(info, "", "  ")
	if err != nil {
		return err
	}
	tmp := filepath.Join(h.dir, "server.json.tmp")
	if err := os.WriteFile(tmp, b, 0o600); err != nil {
		return err
	}
	return os.Rename(tmp, filepath.Join(h.dir, "server.json"))
}

// Release removes server.json and drops the lock.
func (h *Holder) Release() {
	_ = os.Remove(filepath.Join(h.dir, "server.json"))
	_ = h.lock.Unlock()
}

// Client talks HTTP to the server over its unix socket.
func (i Info) Client() *http.Client {
	return &http.Client{Transport: &http.Transport{DialContext: func(ctx context.Context, _, _ string) (net.Conn, error) {
		return (&net.Dialer{Timeout: 2 * time.Second}).DialContext(ctx, "unix", i.Socket)
	}}}
}

// Find reports a running server. Neither the file nor its pid is trusted: a
// server is running only if the lock is held and its socket answers
// /api/health with the token. A file failing either test is stale and removed.
func Find(ctx context.Context) (Info, bool, error) {
	dir, err := RunDir()
	if err != nil {
		return Info{}, false, err
	}
	path := filepath.Join(dir, "server.json")
	b, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return Info{}, false, nil
	}
	if err != nil {
		return Info{}, false, err
	}
	var info Info
	if err := json.Unmarshal(b, &info); err != nil {
		_ = os.Remove(path)
		return Info{}, false, nil
	}

	if h, err := filelock.TryLock(filepath.Join(dir, "server.lock")); err == nil {
		// Nobody holds the lock: the file is left over from a dead server.
		_ = h.Unlock()
		_ = os.Remove(path)
		return Info{}, false, nil
	}

	hctx, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()
	req, _ := http.NewRequestWithContext(hctx, http.MethodGet, BaseURL+"/api/health", nil)
	req.Header.Set("Authorization", "Bearer "+info.Token)
	res, err := info.Client().Do(req)
	if err != nil {
		return info, false, nil
	}
	res.Body.Close()
	return info, res.StatusCode == http.StatusOK, nil
}

// EnsureRunning returns the running server, starting `exe serve` detached if
// there is none.
func EnsureRunning(ctx context.Context, exe string) (Info, error) {
	if info, ok, err := Find(ctx); err != nil || ok {
		return info, err
	}
	dir, err := RunDir()
	if err != nil {
		return Info{}, err
	}
	logf, err := os.OpenFile(filepath.Join(dir, "server.log"), os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o600)
	if err != nil {
		return Info{}, err
	}
	defer logf.Close()
	cmd := exec.Command(exe, "serve", "--no-open")
	cmd.Stdout, cmd.Stderr = logf, logf
	detach(cmd)
	if err := cmd.Start(); err != nil {
		return Info{}, fmt.Errorf("daemon: start server: %w", err)
	}
	_ = cmd.Process.Release()

	deadline := time.Now().Add(10 * time.Second)
	for time.Now().Before(deadline) {
		if info, ok, err := Find(ctx); err == nil && ok {
			return info, nil
		}
		time.Sleep(100 * time.Millisecond)
	}
	return Info{}, fmt.Errorf("daemon: the server did not come up within 10s; see %s", filepath.Join(dir, "server.log"))
}

// Stop asks the server to shut down over its authenticated API. It never
// signals a pid: a pid from a file may by now belong to another process.
func Stop(ctx context.Context) error {
	info, ok, err := Find(ctx)
	if err != nil {
		return err
	}
	if !ok {
		return errors.New("daemon: no jumpgate server is running")
	}
	req, _ := http.NewRequestWithContext(ctx, http.MethodPost, BaseURL+"/api/shutdown", nil)
	req.Header.Set("Authorization", "Bearer "+info.Token)
	res, err := info.Client().Do(req)
	if err != nil {
		return err
	}
	res.Body.Close()
	if res.StatusCode != http.StatusAccepted {
		return fmt.Errorf("daemon: shutdown answered %d", res.StatusCode)
	}
	return nil
}
