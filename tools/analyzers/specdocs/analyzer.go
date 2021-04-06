// Package specdocs implements a static analyzer to ensure that pseudo code we use in our comments, when implementing
// functions defined in specs, is up to date. Reference specs documentation is cached (so that we do not need to
// download it every time build is run).
package specdocs

import (
	"errors"
	"fmt"
	"go/ast"

	"golang.org/x/tools/go/analysis"
	"golang.org/x/tools/go/analysis/passes/inspect"
	"golang.org/x/tools/go/ast/inspector"
)

// Doc explaining the tool.
const Doc = "Tool to enforce that specs pseudo code is up to date"

var errWeakCrypto = errors.New("crypto-secure RNGs are required, use CSPRNG or PRNG defined in github.com/prysmaticlabs/prysm/shared/rand")

// Analyzer runs static analysis.
var Analyzer = &analysis.Analyzer{
	Name:     "specdocs",
	Doc:      Doc,
	Requires: []*analysis.Analyzer{inspect.Analyzer},
	Run:      run,
}

func run(pass *analysis.Pass) (interface{}, error) {
	inspection, ok := pass.ResultOf[inspect.Analyzer].(*inspector.Inspector)
	if !ok {
		return nil, errors.New("analyzer is not type *inspector.Inspector")
	}

	nodeFilter := []ast.Node{
		(*ast.File)(nil),
		(*ast.Comment)(nil),
	}

	aliases := make(map[string]string)
	inspection.Preorder(nodeFilter, func(node ast.Node) {
		switch stmt := node.(type) {
		case *ast.File:
			fmt.Printf("node: %v", stmt)
			// Reset aliases (per file).
			aliases = make(map[string]string)
		case *ast.Comment:
			fmt.Printf("comment: %v", stmt)
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
