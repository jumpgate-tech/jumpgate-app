package daemon

import (
	"context"
	"encoding/json"
	"errors"
	"io/fs"
	"net/http"
	"os"
	"path/filepath"
	"syscall"
	"time"
)

// SystemDir is the runtime directory of the jumpgate service (`jumpgate
// serve --system` under systemd): 0750 jumpgate-ctl:jumpgate-admin. It holds
// the admin socket and system.json. Tests override it.
var SystemDir = "/run/jumpgate-ctl"

// SystemInfo is system.json, how a command finds the jumpgate service. It is
// world-readable and so carries no token, ever: the admin socket admits a
// peer by its kernel credentials, not by anything it presents.
type SystemInfo struct {
	PID       int       `json:"pid"`
	Socket    string    `json:"socket"`
	HTTPAddr  string    `json:"httpAddr"`
	Version   string    `json:"version"`
	StartedAt time.Time `json:"startedAt"`
	UID       int       `json:"uid"`
	// Lock is the vault state: none, locked, unlocked or unusable.
	Lock string `json:"lock"`
}

const systemFile = "system.json"

// PublishSystem writes dir/system.json atomically: a temp file created 0644
// in dir, synced, then renamed over the old one.
func PublishSystem(dir string, info SystemInfo) error {
	b, err := json.MarshalIndent(info, "", "  ")
	if err != nil {
		return err
	}
	f, err := os.CreateTemp(dir, ".system-*.json")
	if err != nil {
		return err
	}
	tmp := f.Name()
	ok := false
	defer func() {
		if !ok {
			_ = os.Remove(tmp)
		}
	}()
	if err := f.Chmod(0o644); err != nil {
		f.Close()
		return err
	}
	if _, err := f.Write(b); err != nil {
		f.Close()
		return err
	}
	if err := f.Sync(); err != nil {
		f.Close()
		return err
	}
	if err := f.Close(); err != nil {
		return err
	}
	if err := os.Rename(tmp, filepath.Join(dir, systemFile)); err != nil {
		return err
	}
	ok = true
	return nil
}

// RemoveSystem removes dir/system.json.
func RemoveSystem(dir string) {
	_ = os.Remove(filepath.Join(dir, systemFile))
}

// ErrNotAdmin: the jumpgate service runs here, and this login session may
// not reach its socket. A user just added to jumpgate-admin is refused too,
// until they log in again (the session lacks the group). Exact copy, D7.
var ErrNotAdmin = errors.New("a jumpgate service runs on this machine; ask an administrator to run `sudo usermod -aG jumpgate-admin $USER`, then log in again")

// ErrServiceManaged: `jumpgate stop` does not stop the jumpgate service.
var ErrServiceManaged = errors.New("daemon: the jumpgate service is managed by systemd; use `sudo systemctl stop jumpgate`")

// ErrServiceNotRunning: the jumpgate service is installed but not up, so no
// per-user server is started in its place.
var ErrServiceNotRunning = errors.New("the jumpgate service is installed but not running; run `sudo systemctl start jumpgate`, then see `journalctl -u jumpgate`")

// SystemUnitPaths are where an installed jumpgate.service lives (install.sh,
// then the package). Tests override it.
var SystemUnitPaths = []string{"/etc/systemd/system/jumpgate.service", "/usr/lib/systemd/system/jumpgate.service"}

func serviceInstalled() bool {
	for _, p := range SystemUnitPaths {
		if _, err := os.Stat(p); err == nil {
			return true
		}
	}
	return false
}

// FindSystem reports the jumpgate service. A missing system.json, or one
// owned by neither root nor the uid it names, means none. The socket must
// answer /api/health with no token; a socket the kernel will not let this
// user connect to is ErrNotAdmin, and one that is gone or refusing means the
// service is not running.
func FindSystem(ctx context.Context) (Info, bool, error) {
	path := filepath.Join(SystemDir, systemFile)
	b, err := os.ReadFile(path)
	switch {
	case errors.Is(err, fs.ErrNotExist):
		return Info{}, false, nil
	case errors.Is(err, fs.ErrPermission):
		// The runtime dir is 0750 jumpgate-ctl:jumpgate-admin.
		return Info{}, false, ErrNotAdmin
	case err != nil:
		return Info{}, false, err
	}
	var si SystemInfo
	if err := json.Unmarshal(b, &si); err != nil {
		return Info{}, false, nil
	}
	if owner, ok := fileOwner(path); !ok || (owner != 0 && owner != si.UID) {
		return Info{}, false, nil
	}
	info := Info{
		PID: si.PID, Socket: si.Socket, HTTPAddr: si.HTTPAddr, Version: si.Version, StartedAt: si.StartedAt,
		System: true, ServiceUID: si.UID, Lock: si.Lock,
	}
	hctx, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()
	req, _ := http.NewRequestWithContext(hctx, http.MethodGet, BaseURL+"/api/health", nil)
	res, err := info.Client().Do(req)
	if err != nil {
		if errors.Is(err, syscall.EACCES) || errors.Is(err, syscall.EPERM) {
			return Info{}, false, ErrNotAdmin
		}
		return Info{}, false, nil
	}
	res.Body.Close()
	if res.StatusCode != http.StatusOK {
		return Info{}, false, nil
	}
	return info, true, nil
}
