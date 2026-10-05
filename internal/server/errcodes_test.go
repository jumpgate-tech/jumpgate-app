package server

import (
	"go/ast"
	"go/parser"
	"go/token"
	"path/filepath"
	"strings"
	"testing"
)

// Every code the server writes is an api.Code constant. A string literal in
// a code position is how the two dialects (kebab and snake) grew, so it is
// refused here: in writeErrorDetail's code argument, in an api.Error's Code
// field, and in a pairing event's Code field.
func TestNoLiteralErrorCodes(t *testing.T) {
	files, _ := filepath.Glob("*.go")
	fset := token.NewFileSet()
	for _, f := range files {
		if strings.HasSuffix(f, "_test.go") {
			continue
		}
		file, err := parser.ParseFile(fset, f, nil, 0)
		if err != nil {
			t.Fatal(err)
		}
		ast.Inspect(file, func(n ast.Node) bool {
			switch n := n.(type) {
			case *ast.CallExpr:
				if id, ok := n.Fun.(*ast.Ident); ok && id.Name == "writeErrorDetail" && len(n.Args) == 5 {
					if lit, ok := n.Args[4].(*ast.BasicLit); ok && lit.Kind == token.STRING {
						t.Errorf("%s: writeErrorDetail with a literal code; use an api.Code constant", fset.Position(n.Pos()))
					}
				}
			case *ast.KeyValueExpr:
				if k, ok := n.Key.(*ast.Ident); ok && k.Name == "Code" {
					if lit, ok := n.Value.(*ast.BasicLit); ok && lit.Kind == token.STRING {
						t.Errorf("%s: Code: with a literal; use an api.Code constant", fset.Position(n.Pos()))
					}
				}
			}
			return true
		})
	}
}
