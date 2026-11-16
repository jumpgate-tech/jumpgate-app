package tui

import (
	"encoding/json"
	"fmt"
	"strings"
)

// diagLines is a diagnostics answer as one line per check, "name  ok", rather
// than the raw path-and-value lines (checks[0].name: rpc). The server's report
// lists its checks as items (ID, Title, Status pass/fail/warn/unknown, Detail,
// Fix); a plainer shape with name and ok is read too. ok is false for an
// answer with neither, which is shown the generic way.
func diagLines(raw json.RawMessage) ([]string, bool) {
	var obj map[string]json.RawMessage
	if json.Unmarshal(raw, &obj) != nil {
		return nil, false
	}
	var list []map[string]any
	for _, k := range []string{"items", "Items", "checks", "Checks"} {
		if v, ok := obj[k]; ok && json.Unmarshal(v, &list) == nil && len(list) > 0 {
			break
		}
	}
	if len(list) == 0 {
		return nil, false
	}
	str := func(m map[string]any, keys ...string) string {
		for _, k := range keys {
			if s, ok := m[k].(string); ok && s != "" {
				return sanitizeLine(s)
			}
		}
		return ""
	}
	var out []string
	for _, k := range []string{"at", "trigger"} {
		var s string
		if v, ok := obj[k]; ok && json.Unmarshal(v, &s) == nil && s != "" {
			out = append(out, k+": "+maskText(sanitizeLine(s)))
		}
	}
	for _, c := range list {
		name := str(c, "Title", "title", "name", "Name", "ID", "id")
		if name == "" {
			name = "check"
		}
		status := str(c, "Status", "status")
		switch ok, isBool := c["ok"].(bool); {
		case status != "":
		case isBool && ok:
			status = "pass"
		case isBool:
			status = "fail"
		}
		word := map[string]string{"pass": "ok", "fail": "FAIL", "warn": "warn", "unknown": "n/a", "": "n/a"}[strings.ToLower(status)]
		if word == "" {
			word = status
		}
		l := fmt.Sprintf("%s  %s", name, word)
		if d := str(c, "Detail", "detail"); d != "" {
			l += "  " + maskText(d)
		}
		out = append(out, l)
		if fix := str(c, "Fix", "fix"); fix != "" {
			out = append(out, "  fix: "+maskText(fix))
		}
	}
	if len(out) > maxResultLines {
		out = out[:maxResultLines]
	}
	return out, true
}
