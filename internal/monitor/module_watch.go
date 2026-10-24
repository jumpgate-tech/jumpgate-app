package monitor

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/valve-tech/jumpgate/internal/catalog"
	"github.com/valve-tech/jumpgate/internal/executor"
)

// watchFresh is how old the watch's last reading may be: three of its
// default 30s polls.
const watchFresh = 90 * time.Second

// watchNow is the probe's clock; a var for tests.
var watchNow = time.Now

// watchState is the part of the watch's state.json the probe reads
// (internal/watch.State; monitor cannot import watch, which imports it).
type watchState struct {
	At   time.Time `json:"at"`
	Open []Alert   `json:"open"`
}

func init() { boxProbes["watch"] = watchProbe }

// watchProbe reads the watch's state.json: its open alerts, and whether
// it is still writing.
//
// docs:covers box-modules
func watchProbe(ctx context.Context, ex executor.Executor, b catalog.BoxModule) ModuleStatus {
	var st ModuleStatus
	res, err := ex.Run(ctx, "cat "+shQuote(b.DataDir()+"/state.json"), nil)
	if err != nil {
		st.Error = err.Error()
		return st
	}
	if res.ExitCode != 0 {
		st.Error = "The health watch has not written a reading yet."
		return st
	}
	var s watchState
	if err := json.Unmarshal([]byte(res.Stdout), &s); err != nil {
		st.Error = "The health watch's state file does not parse."
		return st
	}
	st.Alerts = s.Open
	if age := watchNow().Sub(s.At); age > watchFresh {
		st.Error = fmt.Sprintf("The health watch has not written a reading for %d minutes.", int(age/time.Minute))
		return st
	}
	st.Served = true
	return st
}
