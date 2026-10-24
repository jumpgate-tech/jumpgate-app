package monitor

import (
	"context"
	"encoding/json"
	"net"
	"strconv"

	"github.com/valve-tech/jumpgate/internal/catalog"
	"github.com/valve-tech/jumpgate/internal/executor"
)

func init() { boxProbes["checkpointz"] = probeCheckpointz }

// checkpointzStatusBody is the part of /checkpointz/v1/status the probe
// reads. Epochs are decimal strings, as in the beacon API.
type checkpointzStatusBody struct {
	Finality *struct {
		Finalized struct {
			Epoch string `json:"epoch"`
		} `json:"finalized"`
	} `json:"finality"`
}

// probeCheckpointz asks the box's Checkpointz for its status: Served when
// it has a finalized checkpoint to hand out, HeadBlock that checkpoint's
// epoch. The address is the sidecar's bind, a literal IP the catalog has
// checked, or loopback.
//
// docs:covers checkpointz-run
func probeCheckpointz(ctx context.Context, ex executor.Executor, b catalog.BoxModule) ModuleStatus {
	var st ModuleStatus
	host := "127.0.0.1"
	if ip := net.ParseIP(b.BindAddr); ip != nil {
		host = ip.String()
	}
	port := b.Port
	if port == 0 {
		port = 5555
	}
	url := "http://" + net.JoinHostPort(host, strconv.Itoa(port)) + "/checkpointz/v1/status"
	res, err := ex.Run(ctx, "curl -fsS --max-time 5 "+shQuote(url), nil)
	if err != nil {
		st.Error = err.Error()
		return st
	}
	if res.ExitCode != 0 {
		st.Error = "Checkpointz did not answer its status request"
		return st
	}
	var body checkpointzStatusBody
	if err := json.Unmarshal([]byte(res.Stdout), &body); err != nil {
		st.Error = "Checkpointz gave an unreadable status"
		return st
	}
	if body.Finality == nil || body.Finality.Finalized.Epoch == "" {
		st.Note = "Checkpointz has no finalized checkpoint yet. Its beacon nodes may still be syncing."
		return st
	}
	epoch, err := strconv.ParseUint(body.Finality.Finalized.Epoch, 10, 64)
	if err != nil {
		st.Error = "Checkpointz reported a finalized epoch that is not a number"
		return st
	}
	st.Served, st.HeadBlock = true, epoch
	return st
}
