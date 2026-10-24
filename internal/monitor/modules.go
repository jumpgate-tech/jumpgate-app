package monitor

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/valve-tech/jumpgate/internal/catalog"
	"github.com/valve-tech/jumpgate/internal/executor"
)

// ModuleStatus is one module's reading. A node module is probed through its
// execution client's status method; a box module through its sidecar and
// its unit.
type ModuleStatus struct {
	ID        string  `json:"id"`
	Instance  string  `json:"instance,omitempty"` // box modules
	Served    bool    `json:"served"`
	Active    bool    `json:"active"` // box: systemctl is-active
	Accepting bool    `json:"accepting"`
	Count     uint64  `json:"count"`
	Size      uint64  `json:"size"`
	HeadBlock uint64  `json:"headBlock"`
	Identity  string  `json:"identity,omitempty"` // enode:// or enr:-
	Note      string  `json:"note,omitempty"`     // advice about a reading that works but may not help (an undialable identity)
	Alerts    []Alert `json:"alerts,omitempty"`   // watch only
	Error     string  `json:"error,omitempty"`
}

// Alert is one health-watch alert, as the watch writes it to its state file.
type Alert struct {
	Key     string     `json:"key"` // "exec-down", "disk-90", "endpoint:<name>", ...
	Message string     `json:"message"`
	Since   time.Time  `json:"since"`
	Cleared *time.Time `json:"cleared,omitempty"`
}

// boxModulesGlob is where setup writes each box module's root-owned sidecar.
const boxModulesGlob = catalog.BoxModulesConfigRoot + "/*/module.json"

var (
	// validateBoxModule vets a record read back from a box before any
	// command is built from it; a var so tests can stand in modules the
	// catalog does not register yet.
	validateBoxModule = catalog.ValidateBoxModule

	// nodeModules is the registry probeModules walks; a var so tests can
	// stand in modules the catalog does not register yet.
	nodeModules = catalog.Modules

	// nodeDecoders turn a node module's status-method response into its
	// reading, keyed by module id. A module without one gets decodeServed.
	nodeDecoders = map[string]func(body string) (ModuleStatus, error){}

	// boxProbes add a box module's own reading (its identity, its alerts)
	// to the sidecar-and-unit one, keyed by module id.
	boxProbes = map[string]func(ctx context.Context, ex executor.Executor, b catalog.BoxModule) ModuleStatus{}
)

type jsonRPCError struct {
	Error *struct {
		Code    int    `json:"code"`
		Message string `json:"message"`
	} `json:"error"`
	Result json.RawMessage `json:"result"`
}

// decodeServed reads only whether the method answered.
func decodeServed(string) (ModuleStatus, error) { return ModuleStatus{Served: true}, nil }

// probeModules reads every node module that applies to w. A node that
// answers "method not found" is not serving the module, which is a reading
// rather than an error.
//
// docs:covers msgboard-status
func probeModules(ctx context.Context, ex executor.Executor, w catalog.WireConfig) []ModuleStatus {
	var out []ModuleStatus
	for _, m := range nodeModules() {
		if m.Placement != catalog.PlaceNode || m.Probe.StatusMethod == "" {
			continue
		}
		if ok, _ := m.Applies(w.ChainID, w.ExecID); !ok {
			continue
		}
		st := ModuleStatus{ID: m.ID}
		res, err := ex.Run(ctx, jsonRPCCmd(execRPCAddr(w), m.Probe.StatusMethod), nil)
		switch {
		case err != nil:
			st.Error = err.Error()
		case res.ExitCode != 0:
			st.Error = fmt.Sprintf("the node did not answer %s", m.Probe.StatusMethod)
		default:
			st = decodeNodeStatus(m, res.Stdout)
		}
		out = append(out, st)
	}
	return out
}

func decodeNodeStatus(m catalog.Module, body string) ModuleStatus {
	st := ModuleStatus{ID: m.ID}
	var resp jsonRPCError
	if err := json.Unmarshal([]byte(body), &resp); err != nil {
		st.Error = fmt.Sprintf("%s: unreadable answer", m.Probe.StatusMethod)
		return st
	}
	if resp.Error != nil {
		if resp.Error.Code != -32601 {
			st.Error = resp.Error.Message
		}
		return st
	}
	dec := nodeDecoders[m.ID]
	if dec == nil {
		dec = decodeServed
	}
	got, err := dec(body)
	got.ID = m.ID
	if err != nil {
		got.Error = err.Error()
	}
	return got
}

// ProbeBoxModules lists the box's modules from their sidecars and reads
// each. A box with no modules directory, or one the shell cannot reach,
// gives nil.
func ProbeBoxModules(ctx context.Context, ex executor.Executor) []ModuleStatus {
	res, err := ex.Run(ctx, "ls "+boxModulesGlob, nil)
	if err != nil || res.ExitCode != 0 {
		return nil
	}
	var out []ModuleStatus
	for _, path := range strings.Fields(res.Stdout) {
		cat, err := ex.Run(ctx, "cat "+shQuote(path), nil)
		if err != nil || cat.ExitCode != 0 {
			continue
		}
		var side catalog.BoxSidecar
		if err := json.Unmarshal([]byte(cat.Stdout), &side); err != nil {
			out = append(out, ModuleStatus{Error: path + ": unreadable module record"})
			continue
		}
		b := side.Module()
		if err := validateBoxModule(b); err != nil {
			out = append(out, ModuleStatus{Error: path + ": " + err.Error()})
			continue
		}
		if b.SidecarPath() != path || side.Unit != b.UnitName() {
			out = append(out, ModuleStatus{Error: path + ": the record is for " + b.Instance()})
			continue
		}
		out = append(out, ProbeBoxModule(ctx, ex, b))
	}
	return out
}

// ProbeBoxModule reads one box module: whether its unit is active, then its
// own probe, if it has one. A record that is not a known box module is an
// error and runs nothing.
func ProbeBoxModule(ctx context.Context, ex executor.Executor, b catalog.BoxModule) ModuleStatus {
	if err := validateBoxModule(b); err != nil {
		return ModuleStatus{Error: err.Error()}
	}
	st := ModuleStatus{ID: b.ID, Instance: b.Instance()}
	if p := boxProbes[b.ID]; p != nil {
		st = p(ctx, ex, b)
		st.ID, st.Instance = b.ID, b.Instance()
	}
	// systemctl is-active exits non-zero for an inactive unit: a reading.
	res, err := ex.Run(ctx, "systemctl is-active "+shQuote(b.UnitName()), nil)
	if err != nil {
		if st.Error == "" {
			st.Error = err.Error()
		}
		return st
	}
	st.Active = strings.TrimSpace(res.Stdout) == "active"
	return st
}
