package setup

import (
	"context"
	"strings"
	"testing"

	"github.com/valve-tech/jumpgate/internal/catalog"
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
