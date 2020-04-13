package roughtime

import (
	"errors"
	"go/ast"

	"golang.org/x/tools/go/analysis"
	"golang.org/x/tools/go/analysis/passes/inspect"
	"golang.org/x/tools/go/ast/inspector"
)

// Doc explaining the tool.
const Doc = "Tool to enforce the use of roughtime.Now() rather than time.Now(). This is " +
	"critically important to certain ETH2 systems where the client / server must be in sync with " +
	"some NTP network."

// Analyzer runs static analysis.
var Analyzer = &analysis.Analyzer{
	Name:     "roughtime",
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
		(*ast.CallExpr)(nil),
	}

	inspect.Preorder(nodeFilter, func(node ast.Node) {
		if ce, ok := node.(*ast.CallExpr); ok {
			if isPkgDot(ce.Fun, "time", "Now") {
				pass.Reportf(node.Pos(), "Do not use time.Now(), use "+
					"github.com/prysmaticlabs/prysm/shared/roughtime.Now() or add an exclusion "+
					"to //:nogo.json.")
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
