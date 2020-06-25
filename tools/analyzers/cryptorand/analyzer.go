package cryptorand

import (
	"errors"
	"fmt"
	"go/ast"
	"strings"

	"golang.org/x/tools/go/analysis"
	"golang.org/x/tools/go/analysis/passes/inspect"
	"golang.org/x/tools/go/ast/inspector"
)

// Doc explaining the tool.
const Doc = "Tool to enforce the use of stronger crypto: crypto/rand instead of math/rand"

var errWeakCrypto = errors.New("weak cryptography, use crypto/rand, or add an exclusion to //:nogo.json")

// Analyzer runs static analysis.
var Analyzer = &analysis.Analyzer{
	Name:     "cryptorand",
	Doc:      Doc,
	Requires: []*analysis.Analyzer{inspect.Analyzer},
	Run:      run,
}

func run(pass *analysis.Pass) (interface{}, error) {
	inspect, ok := pass.ResultOf[inspect.Analyzer].(*inspector.Inspector)
	if !ok {
		return nil, errors.New("analyzer is not type *inspector.Inspector")
	}

	nodeFilter := []ast.Node{
		(*ast.File)(nil),
		(*ast.ImportSpec)(nil),
		(*ast.CallExpr)(nil),
	}

	aliases := make(map[string]string)
	disallowedFns := []string{"NewSource", "New", "Seed", "Int63", "Uint32", "Uint64", "Int31", "Int",
		"Int63n", "Int31n", "Intn", "Float64", "Float32", "Perm", "Shuffle", "Read"}

	inspect.Preorder(nodeFilter, func(node ast.Node) {
		switch stmt := node.(type) {
		case *ast.File:
			// Reset aliases (per file).
			aliases = make(map[string]string)
		case *ast.ImportSpec:
			// Collect aliases to rand packages.
			pkg := stmt.Path.Value
			if strings.HasSuffix(pkg, "/rand\"") && !strings.Contains(pkg, "crypto/rand") {
				if stmt.Name != nil {
					aliases[stmt.Name.Name] = stmt.Path.Value
				} else {
					aliases["rand"] = stmt.Path.Value
				}
			}
		case *ast.CallExpr:
			// Check if any of disallowed functions have been used.
			for pkg, path := range aliases {
				for _, fn := range disallowedFns {
					if isPkgDot(stmt.Fun, pkg, fn) {
						pass.Reportf(node.Pos(), fmt.Sprintf(
							"%s: %s.%s() (from %s)", errWeakCrypto.Error(), pkg, fn, path))
					}
				}
			}
		}
	})

	return nil, nil
}

func isPkgDot(expr ast.Expr, pkg, name string) bool {
	sel, ok := expr.(*ast.SelectorExpr)
	return ok && isIdent(sel.X, pkg) && isIdent(sel.Sel, name)
}

func isIdent(expr ast.Expr, ident string) bool {
	id, ok := expr.(*ast.Ident)
	return ok && id.Name == ident
}
