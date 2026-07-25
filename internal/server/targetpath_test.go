package server

import (
	"go/ast"
	"go/parser"
	"go/token"
	"strings"
	"testing"
)

// Regression guard for target paths built with the control plane's LOCAL
// separator. valve-node-app runs on macOS, Windows, or Linux and drives Linux
// targets, so anything the target's shell or systemd will read must be a POSIX
// path regardless of the host OS. These assertions are on exact strings, not
// on the host separator, so they express the Windows-control-plane contract
// from a darwin/linux test run.

func TestDefaultTargetPathsArePOSIX(t *testing.T) {
	tests := []struct {
		name        string
		chainID     int
		wantDataDir string
		wantJWTPath string
	}{
		{name: "pulsechain mainnet", chainID: 369, wantDataDir: "/var/lib/valve-node-app/369", wantJWTPath: "/var/lib/valve-node-app/369/jwt.hex"},
		{name: "ethereum mainnet", chainID: 1, wantDataDir: "/var/lib/valve-node-app/1", wantJWTPath: "/var/lib/valve-node-app/1/jwt.hex"},
		{name: "pulsechain testnet v4", chainID: 943, wantDataDir: "/var/lib/valve-node-app/943", wantJWTPath: "/var/lib/valve-node-app/943/jwt.hex"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotDir := defaultDataDir(tt.chainID)
			if gotDir != tt.wantDataDir {
				t.Errorf("defaultDataDir(%d) = %q, want %q", tt.chainID, gotDir, tt.wantDataDir)
			}
			gotJWT := defaultJWTPath(gotDir)
			if gotJWT != tt.wantJWTPath {
				t.Errorf("defaultJWTPath(%q) = %q, want %q", gotDir, gotJWT, tt.wantJWTPath)
			}
			if strings.Contains(gotJWT, `\`) {
				t.Errorf("defaultJWTPath(%q) = %q contains a backslash — target paths must be POSIX on every control-plane OS", gotDir, gotJWT)
			}
		})
	}
}

// TestDefaultJWTPath_CustomDataDir covers an operator-supplied DataDir: the
// join must still be POSIX, and must not "clean" a path the target considers
// meaningful.
func TestDefaultJWTPath_CustomDataDir(t *testing.T) {
	tests := []struct {
		name    string
		dataDir string
		want    string
	}{
		{name: "custom mount", dataDir: "/mnt/nvme0/valve", want: "/mnt/nvme0/valve/jwt.hex"},
		{name: "trailing slash", dataDir: "/data/", want: "/data/jwt.hex"},
		{name: "root", dataDir: "/", want: "/jwt.hex"},
		{name: "backslash is a normal filename char on linux", dataDir: `/data/odd\dir`, want: `/data/odd\dir/jwt.hex`},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := defaultJWTPath(tt.dataDir); got != tt.want {
				t.Errorf("defaultJWTPath(%q) = %q, want %q", tt.dataDir, got, tt.want)
			}
		})
	}
}

// TestAPIFilepathUsesAreLocalOnly is the guard that can actually fail on this
// (POSIX) machine: filepath.Join and path.Join are indistinguishable at
// runtime here, so the rule has to be checked structurally instead. Every
// filepath reference in api.go must be building a path on the CONTROL PLANE.
// Today that is only handleAddTarget's known_hosts file; anything naming a
// location on the target belongs to path.Join (see defaultJWTPath). If a new
// site appears here, decide which side of the local/remote line it is on
// before adding it to the allowlist.
func TestAPIFilepathUsesAreLocalOnly(t *testing.T) {
	localPathFuncs := map[string]string{
		"handleAddTarget": "known_hosts lives in the operator's own config dir",
	}

	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, "api.go", nil, parser.SkipObjectResolution)
	if err != nil {
		t.Fatalf("parse api.go: %v", err)
	}

	for _, decl := range f.Decls {
		fn, ok := decl.(*ast.FuncDecl)
		if !ok {
			continue
		}
		_, allowed := localPathFuncs[fn.Name.Name]
		ast.Inspect(fn, func(n ast.Node) bool {
			sel, ok := n.(*ast.SelectorExpr)
			if !ok {
				return true
			}
			ident, ok := sel.X.(*ast.Ident)
			if !ok || ident.Name != "filepath" || allowed {
				return true
			}
			t.Errorf("%s: %s uses filepath.%s. If that path is on the TARGET it must use path.%s (POSIX, host-independent); if it is genuinely on the control plane, add %s to localPathFuncs with a reason",
				fset.Position(sel.Pos()), fn.Name.Name, sel.Sel.Name, sel.Sel.Name, fn.Name.Name)
			return true
		})
	}
}
