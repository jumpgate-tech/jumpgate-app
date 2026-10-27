package agent

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"runtime"
	"strconv"
	"strings"

	"github.com/valve-tech/jumpgate/internal/buildinfo"
	"github.com/valve-tech/jumpgate/internal/catalog"
	"github.com/valve-tech/jumpgate/internal/eip712"
	"github.com/valve-tech/jumpgate/internal/intent"
	"github.com/valve-tech/jumpgate/internal/jobs"
	"github.com/valve-tech/jumpgate/internal/logwatch"
	"github.com/valve-tech/jumpgate/internal/monitor"
	"github.com/valve-tech/jumpgate/internal/ops"
)

// DiskResult answers disk.read.
type DiskResult struct {
	Usage     ops.DU `json:"usage"`
	FreeBytes uint64 `json:"freeBytes"`
}

// FirewallReadPayload carries the controller's trusted overlay ranges, which
// live in the controller's config, not on the box.
type FirewallReadPayload struct {
	OverlayCIDRs []string `json:"overlayCidrs"`
}

// loadNode is the node config every node kind acts on: node.json with the
// names marker applied over it (jobs.ReadNodeConfig, D-J10), so a re-pair
// that rewrote node.json with a controller's stale wire cannot point the
// agent back at units, an account or a data root a migration replaced. It is
// the agent's only reader of node.json.
func (a *Agent) loadNode() (catalog.WireConfig, *Reject) {
	w, err := jobs.ReadNodeConfig(a.cfg.NodePath, a.cfg.NamesPath)
	var marker *jobs.MarkerError
	switch {
	case errors.As(err, &marker):
		return w, reject(intent.ReasonValidation, "names.json: "+err.Error())
	case errors.Is(err, os.ErrNotExist):
		return w, reject(intent.ReasonNotSetUp, "this box has no node set up yet")
	case err != nil:
		return w, reject(intent.ReasonValidation, err.Error())
	}
	if err := catalog.ValidateDataDir(w.DataDir); err != nil {
		return w, reject(intent.ReasonValidation, err.Error())
	}
	if err := catalog.ValidateNames(w); err != nil {
		return w, reject(intent.ReasonValidation, err.Error())
	}
	return w, nil
}

func decode[T any](payload []byte) (T, *Reject) {
	var v T
	if err := json.Unmarshal(payload, &v); err != nil {
		return v, reject(intent.ReasonInvalidPayload, err.Error())
	}
	return v, nil
}

// dispatch runs one admitted intent. An operation error is a signed failure
// receipt (StatusFailed), distinct from a rejection.
func (a *Agent) dispatch(ctx context.Context, i intent.Intent, env intent.Envelope, signers []eip712.Address, p Policy) ([]byte, uint8, *Reject) {
	payload := env.Payload
	if i.Kind == intent.KindAgentInfo {
		w, notSetUp := a.loadNode()
		info := intent.AgentInfo{
			Version: buildinfo.Version(), Arch: runtime.GOARCH, Address: a.Address().Hex(), LastSeq: a.replay.LastSeq(i.Controller),
			PlanVersions: jobs.Versions(), Signers: len(p.Signers), Controllers: p.routineSigners(), SetUp: notSetUp == nil,
		}
		switch {
		case notSetUp == nil:
			// After the names marker: a migrated box reports its new names
			// and paths even when its node.json is stale.
			info.ChainID, info.Names, info.DataDir, info.JWTPath = w.ChainID, w.Names, w.DataDir, w.JWTPath
		case notSetUp.Code != intent.ReasonNotSetUp:
			// A config that exists but is refused is not a new box.
			info.SetUpError = &intent.Rejection{Code: notSetUp.Code, Message: notSetUp.Message}
		}
		return mustJSON(info), intent.StatusOK, nil
	}
	if i.Kind == intent.KindAgentUpgrade {
		return a.upgrade(ctx, payload)
	}
	// modules.read reads the box modules' sidecars, not node.json: a box
	// may run modules and no node.
	if i.Kind == intent.KindModulesRead {
		mods := monitor.ProbeBoxModules(ctx, a.cfg.Exec)
		if mods == nil {
			mods = []monitor.ModuleStatus{}
		}
		return mustJSON(mods), intent.StatusOK, nil
	}
	// firewall.read covers the box modules too, so a module-only box (no
	// node.json) still has a checklist: its modules' ports.
	if i.Kind == intent.KindFirewallRead {
		return a.firewallRead(ctx, payload)
	}
	// A job kind needs no node set up: node.migrate-names reports a missing
	// node.json itself, in its preflight.
	if isJobKind(i.Kind) {
		return a.dispatchJob(ctx, i, env, signers, p)
	}

	w, rej := a.loadNode()
	if rej != nil {
		return nil, 0, rej
	}
	ex := a.cfg.Exec

	switch i.Kind {
	case intent.KindStatusRead:
		snap := monitor.New(monitor.Config{Exec: ex, Wire: w}).PollOnce(ctx)
		return mustJSON(snap), intent.StatusOK, nil

	case intent.KindDiskRead:
		du, err := ops.DiskUsage(ctx, ex, w)
		if err != nil {
			return failed(err)
		}
		free, err := ops.FreeBytesAt(ctx, ex, w.DataDir)
		if err != nil {
			return failed(err)
		}
		return mustJSON(DiskResult{Usage: du, FreeBytes: free}), intent.StatusOK, nil

	case intent.KindEndpointsRead:
		pl, rej := decode[intent.EndpointsReadPayload](payload)
		if rej != nil {
			return nil, 0, rej
		}
		info, err := ops.Endpoints(ctx, ex, w, pl.SSHLogin != "", pl.SSHLogin)
		if err != nil {
			return failed(err)
		}
		return mustJSON(info), intent.StatusOK, nil

	case intent.KindLogsRead:
		pl, rej := decode[intent.LogsReadPayload](payload)
		if rej != nil {
			return nil, 0, rej
		}
		n := pl.N
		if n <= 0 {
			n = intent.LogsDefaultN
		}
		if n > intent.LogsMaxN {
			n = intent.LogsMaxN
		}
		var hits []logwatch.Hit
		for _, unit := range ops.NodeUnits(w.NodeNames()) {
			res, err := ex.Run(ctx, "journalctl -u "+unit+" -n "+strconv.Itoa(n)+" --no-pager -o cat", nil)
			if err != nil {
				return failed(err)
			}
			now := a.cfg.Now()
			for _, line := range strings.Split(strings.TrimRight(res.Stdout, "\n"), "\n") {
				if line == "" {
					continue
				}
				if h, ok := logwatch.Classify(unit, line, now); ok {
					hits = append(hits, h)
				} else {
					hits = append(hits, logwatch.Hit{Unit: unit, Line: line, At: now, Severity: "info"})
				}
			}
		}
		return mustJSON(hits), intent.StatusOK, nil

	case intent.KindLogsSince:
		pl, rej := decode[intent.LogsSincePayload](payload)
		if rej != nil {
			return nil, 0, rej
		}
		hits, cursor, rej, err := logsSince(ctx, ex, ops.NodeUnits(w.NodeNames()), pl.Cursor, pl.N, a.cfg.Now(),
			func(e journalEntry) logwatch.Hit { return classifyLine(e.unit, e.line, e.at) })
		if rej != nil {
			return nil, 0, rej
		}
		if err != nil {
			return failed(err)
		}
		return mustJSON(LogsSinceResult{Hits: hits, Cursor: cursor}), intent.StatusOK, nil

	case intent.KindServiceAction:
		pl, rej := decode[intent.ServiceActionPayload](payload)
		if rej != nil {
			return nil, 0, rej
		}
		if (pl.Service != "exec" && pl.Service != "beacon") || (pl.Action != "start" && pl.Action != "stop" && pl.Action != "restart") {
			return nil, 0, reject(intent.ReasonInvalidPayload, fmt.Sprintf("service %q / action %q", pl.Service, pl.Action))
		}
		// J1: the node services belong to a running job until it ends.
		if rej := a.unopenedJobStore(); rej != nil {
			return nil, 0, rej
		}
		if j, busy, err := a.activeJob(); err != nil {
			return failed(err)
		} else if busy {
			return nil, 0, reject(intent.ReasonJobActive, fmt.Sprintf("job %s (%s) is %s and owns the node services until it ends", j.ID, j.Kind, j.State))
		}
		active, err := ops.ServiceAction(ctx, ex, w.NodeNames(), pl.Service, pl.Action)
		if err != nil {
			return failed(err)
		}
		return mustJSON(struct {
			Active bool `json:"active"`
		}{active}), intent.StatusOK, nil
	}
	return nil, 0, reject(intent.ReasonUnknownKind, i.Kind)
}

// firewallRead is the node checklist, when the box has a node, plus an
// item per public box-module port. A box with neither is not set up.
func (a *Agent) firewallRead(ctx context.Context, payload []byte) ([]byte, uint8, *Reject) {
	pl, rej := decode[FirewallReadPayload](payload)
	if rej != nil {
		return nil, 0, rej
	}
	ex := a.cfg.Exec
	w, notSetUp := a.loadNode()
	if notSetUp != nil && notSetUp.Code != intent.ReasonNotSetUp {
		return nil, 0, notSetUp
	}
	var items []ops.CheckItem
	if notSetUp == nil {
		var err error
		items, err = ops.FirewallChecklist(ctx, ex, w, ops.ParseOverlayCIDRs(pl.OverlayCIDRs)...)
		if err != nil {
			return failed(err)
		}
	}
	ports := ops.ModulePortsOnBox(ctx, ex)
	if notSetUp != nil && len(ports) == 0 {
		return nil, 0, notSetUp
	}
	mods, err := ops.ModuleFirewallItems(ctx, ex, ports)
	if err != nil {
		return failed(err)
	}
	return mustJSON(append(items, mods...)), intent.StatusOK, nil
}

func failed(err error) ([]byte, uint8, *Reject) {
	return mustJSON(intent.Failure{Message: err.Error()}), intent.StatusFailed, nil
}

func mustJSON(v any) []byte {
	b, err := json.Marshal(v)
	if err != nil {
		b, _ = json.Marshal(intent.Failure{Message: "encode result: " + err.Error()})
	}
	return b
}
