package agent

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/valve-tech/jumpgate/internal/buildinfo"
	"github.com/valve-tech/jumpgate/internal/catalog"
	"github.com/valve-tech/jumpgate/internal/intent"
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

// PlanVersions lists the plan versions this agent can run. Sub-project 1 runs
// no plans; sub-project 2 adds the first.
var PlanVersions = []string{}

func (a *Agent) loadNode() (catalog.WireConfig, *Reject) {
	var w catalog.WireConfig
	b, err := os.ReadFile(a.cfg.NodePath)
	if errors.Is(err, os.ErrNotExist) {
		return w, reject(intent.ReasonNotSetUp, "this box has no node set up yet")
	}
	if err != nil {
		return w, reject(intent.ReasonValidation, err.Error())
	}
	if err := json.Unmarshal(b, &w); err != nil {
		return w, reject(intent.ReasonValidation, "node.json does not parse: "+err.Error())
	}
	if err := catalog.ValidateDataDir(w.DataDir); err != nil {
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
func (a *Agent) dispatch(ctx context.Context, i intent.Intent, payload []byte, p Policy) ([]byte, uint8, *Reject) {
	if i.Kind == intent.KindAgentInfo {
		w, notSetUp := a.loadNode()
		info := intent.AgentInfo{
			Version: buildinfo.Version(), Address: a.Address().Hex(), LastSeq: a.replay.LastSeq(i.Controller),
			PlanVersions: PlanVersions, Signers: len(p.Signers), SetUp: notSetUp == nil,
		}
		if notSetUp == nil {
			info.ChainID = w.ChainID
		}
		return mustJSON(info), intent.StatusOK, nil
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

	case intent.KindFirewallRead:
		pl, rej := decode[FirewallReadPayload](payload)
		if rej != nil {
			return nil, 0, rej
		}
		items, err := ops.FirewallChecklist(ctx, ex, w, ops.ParseOverlayCIDRs(pl.OverlayCIDRs)...)
		if err != nil {
			return failed(err)
		}
		return mustJSON(items), intent.StatusOK, nil

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
		for _, unit := range ops.NodeUnits() {
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
		res, rej, err := logsSince(ctx, ex, pl, a.cfg.Now())
		if rej != nil {
			return nil, 0, rej
		}
		if err != nil {
			return failed(err)
		}
		return mustJSON(res), intent.StatusOK, nil

	case intent.KindServiceAction:
		pl, rej := decode[intent.ServiceActionPayload](payload)
		if rej != nil {
			return nil, 0, rej
		}
		if (pl.Service != "exec" && pl.Service != "beacon") || (pl.Action != "start" && pl.Action != "stop" && pl.Action != "restart") {
			return nil, 0, reject(intent.ReasonInvalidPayload, fmt.Sprintf("service %q / action %q", pl.Service, pl.Action))
		}
		active, err := ops.ServiceAction(ctx, ex, pl.Service, pl.Action)
		if err != nil {
			return failed(err)
		}
		return mustJSON(struct {
			Active bool `json:"active"`
		}{active}), intent.StatusOK, nil
	}
	return nil, 0, reject(intent.ReasonUnknownKind, i.Kind)
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
