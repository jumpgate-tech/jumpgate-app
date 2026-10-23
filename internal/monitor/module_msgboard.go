package monitor

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
)

func init() { nodeDecoders["msgboard"] = parseMsgboardStatus }

// parseMsgboardStatus decodes msgboard_status: {"enabled", "count", "size",
// "headBlock"}, the numbers as 0x-hex quantities. It takes the bare result
// or the whole JSON-RPC response around it.
//
// docs:covers msgboard-status
func parseMsgboardStatus(body string) (ModuleStatus, error) {
	var env struct {
		Result json.RawMessage `json:"result"`
	}
	raw := []byte(body)
	if err := json.Unmarshal(raw, &env); err == nil && len(env.Result) > 0 {
		raw = env.Result
	}
	var r struct {
		Enabled   bool   `json:"enabled"`
		Count     string `json:"count"`
		Size      string `json:"size"`
		HeadBlock string `json:"headBlock"`
	}
	if err := json.Unmarshal(raw, &r); err != nil {
		return ModuleStatus{}, fmt.Errorf("msgboard_status: unreadable answer")
	}
	st := ModuleStatus{Served: true, Accepting: r.Enabled}
	for _, f := range []struct {
		name string
		v    string
		dst  *uint64
	}{{"count", r.Count, &st.Count}, {"size", r.Size, &st.Size}, {"headBlock", r.HeadBlock, &st.HeadBlock}} {
		n, err := parseHexQuantity(f.v)
		if err != nil {
			return ModuleStatus{Served: true}, fmt.Errorf("msgboard_status: %s %q is not a hex quantity", f.name, f.v)
		}
		*f.dst = n
	}
	return st, nil
}

func parseHexQuantity(s string) (uint64, error) {
	h, ok := strings.CutPrefix(s, "0x")
	if !ok || h == "" {
		return 0, fmt.Errorf("no 0x prefix")
	}
	return strconv.ParseUint(h, 16, 64)
}
