package config

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/valve-tech/jumpgate/internal/catalog"
)

func TestTargetModulesOmitEmpty(t *testing.T) {
	b, err := json.Marshal(Target{ID: "local", Mode: "local"})
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(b), `"modules"`) {
		t.Errorf("empty modules marshaled: %s", b)
	}
	b, _ = json.Marshal(Target{ID: "local", Modules: []catalog.BoxModule{{ID: "watch"}}})
	if !strings.Contains(string(b), `"modules":[{"id":"watch"}]`) {
		t.Errorf("modules: %s", b)
	}
}
