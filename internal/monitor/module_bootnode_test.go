package monitor

import (
	"context"
	"strings"
	"testing"

	"github.com/valve-tech/jumpgate/internal/catalog"
	"github.com/valve-tech/jumpgate/internal/executor"
)

const (
	enodeA = "enode://2546ee2b2c03533e4da51fdf4e788430c55f8c057431d0c1915b492d52ee40f876db73ef633d8784b828282f69890238ad03142daa9fb4f33aff60625b5b74c9@203.0.113.7:0?discport=30301"
	enodeB = "enode://921821c42ceb20840f7424447d967392adcc695efe16a59f1b4ddfded88192d0db04e54176ca0c618cce548c4c02b55be5f3cf8e959acf1b4e3ff5f17be3fbca@203.0.113.8:0?discport=30301"
)

func TestBootnodeIdentityFromJournal(t *testing.T) {
	journal := "INFO Bootnode started\nINFO (Discv5) enode: " + enodeA + "\nINFO Received SIGTERM\nINFO (Discv5) enode: \x1b[0m" + enodeB + "\x1b[2m\n"
	ex := newFakeExecutor().
		script("journalctl", executor.Result{Stdout: journal}).
		script("is-active", executor.Result{Stdout: "active\n"})
	b := catalog.BoxModule{ID: "el-bootnode", ChainID: 943}
	st := ProbeBoxModule(context.Background(), ex, b)
	if st.Identity != enodeB {
		t.Fatalf("Identity = %q, want %q", st.Identity, enodeB)
	}
	if !st.Active || st.Instance != "el-bootnode-943" {
		t.Fatalf("status %+v", st)
	}
	want := "journalctl -u 'jumpgate-mod-el-bootnode-943.service' -n 200 -o cat"
	found := false
	for _, c := range ex.calls {
		found = found || strings.HasPrefix(c, want)
	}
	if !found {
		t.Fatalf("calls %v, want one starting %q", ex.calls, want)
	}
}

// An enode that advertises 0.0.0.0 cannot be dialled: the reading says to
// set ExtIP.
func TestELBootnodeWithoutExtIPNotes(t *testing.T) {
	ex := newFakeExecutor().script("journalctl", executor.Result{Stdout: "enode://" + strings.Repeat("a", 128) + "@0.0.0.0:30301\n"})
	st := ProbeBoxModule(context.Background(), ex, catalog.BoxModule{ID: "el-bootnode", ChainID: 943})
	if !strings.Contains(st.Note, "ExtIP") || st.Error != "" {
		t.Fatalf("status %+v", st)
	}
	ex = newFakeExecutor().script("journalctl", executor.Result{Stdout: enodeA + "\n"})
	if st := ProbeBoxModule(context.Background(), ex, catalog.BoxModule{ID: "el-bootnode", ChainID: 943}); st.Note != "" {
		t.Fatalf("note on a public enode: %+v", st)
	}
}

func TestCLBootnodeIdentityFromJournal(t *testing.T) {
	journal := `INFO Contact information enr="enr:-Jq4QBvJ-s6B3SDt_LCH7eZ1" x` + "\n" +
		`INFO Contact information enr="enr:-Jq4QLAST_one-x9"` + "\n"
	ex := newFakeExecutor().script("journalctl", executor.Result{Stdout: journal})
	st := ProbeBoxModule(context.Background(), ex, catalog.BoxModule{ID: "cl-bootnode", ChainID: 1})
	if st.Identity != "enr:-Jq4QLAST_one-x9" {
		t.Fatalf("Identity = %q", st.Identity)
	}
}
