package monitor

import (
	"context"
	"fmt"
	"regexp"

	"github.com/valve-tech/jumpgate/internal/catalog"
	"github.com/valve-tech/jumpgate/internal/executor"
)

var (
	// reth logs "(Discv5) enode: enode://<128 hex>@<ip>:0?discport=<port>".
	enodeRe = regexp.MustCompile(`enode://[0-9a-fA-F]{128}@[0-9A-Za-z.:\[\]?=]+`)
	// lighthouse logs `Contact information enr="enr:-<base64url>"`.
	enrRe = regexp.MustCompile(`enr:-[A-Za-z0-9_-]+`)
)

func init() {
	boxProbes["el-bootnode"] = journalIdentityProbe(enodeRe)
	boxProbes["cl-bootnode"] = journalIdentityProbe(enrRe)
}

// journalIdentityProbe reads the last identity re matches in the unit's
// recent journal: the address other nodes put in their bootnode lists.
//
// docs:covers box-modules
func journalIdentityProbe(re *regexp.Regexp) func(context.Context, executor.Executor, catalog.BoxModule) ModuleStatus {
	return func(ctx context.Context, ex executor.Executor, b catalog.BoxModule) ModuleStatus {
		var st ModuleStatus
		res, err := ex.Run(ctx, "journalctl -u "+shQuote(b.UnitName())+" -n 200 -o cat --no-pager", nil)
		if err != nil {
			st.Error = err.Error()
			return st
		}
		if res.ExitCode != 0 {
			st.Error = fmt.Sprintf("could not read the %s journal", b.UnitName())
			return st
		}
		if all := re.FindAllString(res.Stdout, -1); len(all) > 0 {
			st.Identity = all[len(all)-1]
		}
		return st
	}
}
