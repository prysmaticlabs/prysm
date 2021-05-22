// Package interfacechecker implements a static analyzer to prevent incorrect conditional checks on select interfaces.
package interfacechecker

import (
	"errors"
	"go/ast"
	"go/token"
	"go/types"
	"strings"

	"golang.org/x/tools/go/analysis"
	"golang.org/x/tools/go/analysis/passes/inspect"
	"golang.org/x/tools/go/ast/inspector"
)

// Doc explaining the tool.
const Doc = "Enforce usage of proper conditional check for interfaces"

// Analyzer runs static analysis.
var Analyzer = &analysis.Analyzer{
	Name:     "interfacechecker",
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
		(*ast.IfStmt)(nil),
	}

	inspection.Preorder(nodeFilter, func(node ast.Node) {
		stmt, ok := node.(*ast.IfStmt)
		if !ok {
			return
		}
		exp, ok := stmt.Cond.(*ast.BinaryExpr)
		if !ok {
			return
		}
		id, ok := exp.X.(*ast.Ident)
		if !ok {
			return
		}
		if _, ok := pass.TypesInfo.Types[id].Type.(*types.Slice); ok {
			return
		}
		if strings.Contains(pass.TypesInfo.Types[id].Type.String(), "interfaces.SignedBeaconBlock") {
			nval, ok := exp.Y.(*ast.Ident)
			if ok && pass.TypesInfo.Types[nval].IsNil() {
				pass.Reportf(id.Pos(), pass.TypesInfo.Types[id].Type.String())
			}
		}
		if strings.Contains(pass.TypesInfo.Types[id].Type.String(), "interface.BeaconState") {
			nval, ok := exp.Y.(*ast.Ident)
			if ok && pass.TypesInfo.Types[nval].IsNil() {
				pass.Reportf(id.Pos(), pass.TypesInfo.Types[id].Type.String())
			}
		}
		if strings.Contains(pass.TypesInfo.Types[id].Type.String(), "interface.ReadOnlyBeaconState") {
			nval, ok := exp.Y.(*ast.Ident)
			if ok && pass.TypesInfo.Types[nval].IsNil() {
				pass.Reportf(id.Pos(), pass.TypesInfo.Types[id].Type.String())
			}
		}
		if strings.Contains(pass.TypesInfo.Types[id].Type.String(), "interface.WriteOnlyBeaconState") {
			nval, ok := exp.Y.(*ast.Ident)
			if ok && pass.TypesInfo.Types[nval].IsNil() {
				pass.Reportf(id.Pos(), pass.TypesInfo.Types[id].Type.String())
			}
		}

	})

	return nil, nil
}

func reportForbiddenUsage(pass *analysis.Pass, pos token.Pos) {
	pass.Reportf(pos, "Use of featureconfig.Init is forbidden in test code. Please use "+
		"featureconfig.InitWithReset and call reset in the same test function.")
}

func reportUnhandledReset(pass *analysis.Pass, pos token.Pos) {
	pass.Reportf(pos, "Unhandled reset featureconfig not found in test "+
		"method. Be sure to call the returned reset function from featureconfig.InitWithReset "+
		"within this test method.")
}

func isPkgDot(expr ast.Expr, pkg, name string) bool {
	sel, ok := expr.(*ast.SelectorExpr)
	return ok && isIdent(sel.X, pkg) && isIdent(sel.Sel, name)
}

func isIdent(expr ast.Expr, ident string) bool {
	id, ok := expr.(*ast.Ident)
	return ok && id.Name == ident
}
