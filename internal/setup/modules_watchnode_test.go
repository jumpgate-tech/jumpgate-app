package setup

import (
	"context"
	"strings"
	"testing"

	"github.com/valve-tech/jumpgate/internal/catalog"
	"github.com/valve-tech/jumpgate/internal/executor"
)

func runWriteFiles(t *testing.T, steps []Step) *fakeExecutor {
	t.Helper()
	e := boxTarget()
	for _, s := range steps {
		if s.ID == "write-files" {
			if err := s.Run(context.Background(), e, &State{}); err != nil {
				t.Fatal(err)
			}
		}
	}
	return e
}

// The watch reads the box's node from its own config dir: the gate's
// node.json sits in a root-only dir. Setup stages a copy without secrets.
func TestPlanBoxModuleStagesWatchNodeConfig(t *testing.T) {
	b := watchModule(t)
	w := &catalog.WireConfig{ChainID: 369, ExecID: "reth", SnapshotKey: "vk_s3cret", ERPCUpstreams: []string{"https://up.example/key123"}}
	steps, err := PlanBoxModule(b, w, true)
	if err != nil {
		t.Fatal(err)
	}
	e := runWriteFiles(t, steps)
	if !ran(e, "chmod 0640", "'"+b.NodePath()+"'") {
		t.Fatalf("node.json not installed 0640; ran %q", e.callLog())
	}
	found := false
	for _, c := range e.files {
		s := string(c)
		if strings.Contains(s, `"ChainID":369`) {
			found = true
			if strings.Contains(s, "s3cret") || strings.Contains(s, "key123") {
				t.Errorf("node.json carries a secret: %s", s)
			}
		}
	}
	if !found {
		t.Fatal("node.json not staged")
	}
	steps, err = PlanBoxModule(b, nil, true)
	if err != nil {
		t.Fatal(err)
	}
	if e := runWriteFiles(t, steps); ran(e, "'"+b.NodePath()+"'") {
		t.Error("node.json staged without a node")
	}
}

// A box module on a box whose node still runs as the legacy account would
// create the current account beside it, and node.migrate-names refuses a box
// with both. The account step asks the box and refuses before any useradd;
// a box with no legacy account, or one already renamed, goes ahead.
func TestBoxModuleRefusesALegacyAccountBox(t *testing.T) {
	steps, err := PlanBoxModule(watchModule(t), nil, true)
	if err != nil {
		t.Fatal(err)
	}
	var account Step
	for _, s := range steps {
		if s.ID == "account" {
			account = s
		}
	}
	if account.Run == nil {
		t.Fatal("no account step")
	}
	legacy := boxTarget().script("! id -u jumpgate-node", executor.Result{Stdout: "legacy\n"})
	err = account.Run(context.Background(), legacy, &State{})
	if err == nil || !strings.Contains(err.Error(), "node.migrate-names") {
		t.Fatalf("legacy box: %v, want a refusal naming node.migrate-names", err)
	}
	for _, c := range legacy.callLog() {
		if strings.Contains(c, "useradd") {
			t.Fatalf("legacy box: ran %q", c)
		}
	}
	other := boxTarget()
	if err := account.Run(context.Background(), other, &State{}); err != nil {
		t.Fatalf("no legacy account: %v", err)
	}
	if !ran(other, "useradd", "jumpgate-node") {
		t.Fatalf("no legacy account: useradd not run; ran %q", other.callLog())
	}
}
