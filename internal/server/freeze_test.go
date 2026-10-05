package server

import (
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"testing"
)

// frozenTargetRoutes is every per-target route. The legacy executor path is
// frozen (spec A5): a new node operation is a new intent kind, reached through
// the routes below by the server's transport choice, not a new route. Views
// over existing operations live under /api/fleet/{id}.
var frozenTargetRoutes = []string{
	"DELETE /api/targets/{id}",
	"GET /api/targets/{id}/containers",
	"GET /api/targets/{id}/containers/{svc}",
	"GET /api/targets/{id}/containers/{svc}/config",
	"GET /api/targets/{id}/diagnostics",
	"GET /api/targets/{id}/diagnostics/latest",
	"GET /api/targets/{id}/disk",
	"GET /api/targets/{id}/du",
	"GET /api/targets/{id}/endpoints",
	"GET /api/targets/{id}/firewall",
	"GET /api/targets/{id}/logs",
	"GET /api/targets/{id}/logs/stream",
	"GET /api/targets/{id}/monitor/stream",
	"GET /api/targets/{id}/setup/stream",
	"POST /api/targets/{id}/containers/{svc}/provision",
	"POST /api/targets/{id}/containers/{svc}/reset",
	"POST /api/targets/{id}/containers/{svc}/wipe",
	"POST /api/targets/{id}/containers/{svc}/{action}",
	"POST /api/targets/{id}/explain",
	"POST /api/targets/{id}/intent/{kind}",
	"POST /api/targets/{id}/pair",
	"POST /api/targets/{id}/services/{svc}/clear",
	"POST /api/targets/{id}/services/{svc}/{action}",
	"POST /api/targets/{id}/setup",
	"PUT /api/targets/{id}/containers/{svc}/config",
}

func TestLegacyTargetRoutesAreFrozen(t *testing.T) {
	// Handle and HandleFunc both register a route, with or without a method,
	// and any wildcard name counts: /api/targets/{name}/x is still per-target.
	re := regexp.MustCompile(`\.Handle(?:Func)?\(\s*"((?:[A-Z]+ )?/api/targets/[^"]*)"`)
	files, err := filepath.Glob("*.go")
	if err != nil {
		t.Fatalf("glob: %v", err)
	}
	if len(files) == 0 {
		t.Fatal("no Go files found; the test must run in internal/server")
	}
	var got []string
	for _, f := range files {
		if strings.HasSuffix(f, "_test.go") {
			continue
		}
		b, err := os.ReadFile(f)
		if err != nil {
			t.Fatal(err)
		}
		for _, m := range re.FindAllStringSubmatch(string(b), -1) {
			got = append(got, m[1])
		}
	}
	if len(got) == 0 {
		t.Fatal("found no per-target routes; the scan no longer matches how routes are registered")
	}
	sort.Strings(got)
	want := append([]string(nil), frozenTargetRoutes...)
	sort.Strings(want)
	if strings.Join(got, "\n") != strings.Join(want, "\n") {
		t.Fatalf("the per-target routes changed.\nA new node operation is an intent kind (spec A5), not a route; a new view belongs under /api/fleet/{id}.\ngot:\n%s\nwant:\n%s", strings.Join(got, "\n"), strings.Join(want, "\n"))
	}
}
