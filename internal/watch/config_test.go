package watch

import (
	"os"
	"path/filepath"
	"testing"
)

// LoadConfig reads the node copy setup stages beside config.json.
func TestLoadConfigReadsStagedNode(t *testing.T) {
	dir := t.TempDir()
	conf := filepath.Join(dir, "config.json")
	if err := os.WriteFile(conf, []byte(`{"id":"watch","upstreams":["gw=https://rpc.example"]}`), 0o600); err != nil {
		t.Fatal(err)
	}
	cfg, err := LoadConfig(conf)
	if err != nil || cfg.Wire != nil || len(cfg.Endpoints) != 1 {
		t.Fatalf("no node: cfg %+v err %v", cfg, err)
	}
	if err := os.WriteFile(filepath.Join(dir, "node.json"), []byte(`{"ChainID":369,"ExecID":"reth"}`), 0o600); err != nil {
		t.Fatal(err)
	}
	cfg, err = LoadConfig(conf)
	if err != nil || cfg.Wire == nil || cfg.Wire.ChainID != 369 {
		t.Fatalf("node: cfg %+v err %v", cfg, err)
	}
}
