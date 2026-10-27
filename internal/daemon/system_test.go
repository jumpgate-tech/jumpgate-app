package daemon

import (
	"encoding/json"
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

func TestPublishSystemIsWorldReadableAndTokenFree(t *testing.T) {
	dir := t.TempDir()
	if err := PublishSystem(dir, SystemInfo{PID: 1, UID: 999, Lock: "locked"}); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(dir, "system.json")
	fi, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	if runtime.GOOS != "windows" && fi.Mode()&0o777 != 0o644 {
		t.Errorf("mode = %o, want 644", fi.Mode()&0o777)
	}
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	var m map[string]any
	if err := json.Unmarshal(b, &m); err != nil {
		t.Fatal(err)
	}
	if _, ok := m["token"]; ok {
		t.Error("system.json has a token field")
	}
	if m["lock"] != "locked" {
		t.Errorf("lock = %v, want locked", m["lock"])
	}
	RemoveSystem(dir)
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Errorf("RemoveSystem left system.json: %v", err)
	}
}
