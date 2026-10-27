package daemon

import (
	"encoding/json"
	"os"
	"path/filepath"
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
